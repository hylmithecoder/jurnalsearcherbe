package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type RequestBody struct {
	Model              string                 `json:"model"`
	Messages           []Message              `json:"messages"`
	Temperature        float64                `json:"temperature"`
	TopP               float64                `json:"top_p"`
	MaxTokens          int                    `json:"max_tokens"`
	Seed               int                    `json:"seed"`
	Stream             bool                   `json:"stream"`
	ChatTemplateKwargs map[string]interface{} `json:"chat_template_kwargs"`
}

type SearchRequest struct {
	Query  string `json:"query"`
	Stream bool   `json:"stream"`
}

type JournalResult struct {
	Source     string   `json:"source"`
	Title      string   `json:"title"`
	Authors    string   `json:"authors"`
	Year       string   `json:"year"`
	University string   `json:"university"`
	DOI        string   `json:"doi"`
	PdfLinks   []string `json:"pdf_links"`
	Summary    string   `json:"summary"`
}

type FinalResponse struct {
	Query     string          `json:"query"`
	Status    string          `json:"status"`
	Results   []JournalResult `json:"results"`
	AISummary string          `json:"ai_summary,omitempty"`
}

type GoogleSearchResult struct {
	Items []struct {
		Title   string `json:"title"`
		Link    string `json:"link"`
		Snippet string `json:"snippet"`
	} `json:"items"`
}

// ===== OpenAlex Struct =====
type OpenAlexResult struct {
	Results []struct {
		ID              string `json:"id"`
		Title           string `json:"title"`
		DOI             string `json:"doi"`
		PublicationYear int    `json:"publication_year"`
		OpenAccess      struct {
			IsOA   bool   `json:"is_oa"`
			PdfUrl string `json:"pdf_url"`
		} `json:"open_access"`
		Authorships []struct {
			Author struct {
				DisplayName string `json:"display_name"`
			} `json:"author"`
		} `json:"authorships"`
		Locations []struct {
			PdfUrl string `json:"pdf_url"`
		} `json:"locations"`
		AbstractInvertedIndex map[string][]int `json:"abstract_inverted_index"`
	} `json:"results"`
}

// ReconstructAbstract mengubah Inverted Index OpenAlex menjadi teks biasa
func ReconstructAbstract(index map[string][]int) string {
	if len(index) == 0 {
		return ""
	}
	type wordPos struct {
		word string
		pos  int
	}
	var words []wordPos
	for word, positions := range index {
		for _, pos := range positions {
			words = append(words, wordPos{word, pos})
		}
	}
	sort.Slice(words, func(i, j int) bool {
		return words[i].pos < words[j].pos
	})
	var res []string
	for _, w := range words {
		res = append(res, w.word)
	}
	return strings.Join(res, " ")
}

// ===== OpenAlex Fetch =====
func SearchOpenAlex(query string) (*OpenAlexResult, error) {
	// OpenAlex API: Mencari paper yang punya link PDF (is_oa=true)
	searchURL := fmt.Sprintf("https://api.openalex.org/works?search=%s&filter=is_oa:true,has_pdf_url:true&per-page=10",
		url.QueryEscape(query))

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	// Polite Pool: Mengidentifikasi aplikasi kita ke OpenAlex
	req.Header.Set("User-Agent", "JurnalSearcher/1.0 (mailto:admin@example.com)")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openalex search failed: %s", string(body))
	}

	var result OpenAlexResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ScrapForGetPDFUrl mencoba mencari semua link PDF yang tersedia di sebuah halaman
func ScrapForGetPDFUrl(targetURL string) ([]string, error) {
	var pdfLinks []string
	seen := make(map[string]bool)

	addLink := func(link string) {
		if link != "" && !seen[link] {
			// Transformasi OJS: Ubah /download/ menjadi /view/ agar tidak paksa download di browser
			if strings.Contains(link, "/article/download/") {
				viewLink := strings.Replace(link, "/article/download/", "/article/view/", 1)
				if !seen[viewLink] {
					pdfLinks = append(pdfLinks, viewLink)
					seen[viewLink] = true
				}
			}
			pdfLinks = append(pdfLinks, link)
			seen[link] = true
		}
	}

	// 1. Coba Unpaywall dulu jika DOI
	if strings.Contains(targetURL, "doi.org/") {
		doi := strings.Split(targetURL, "doi.org/")[1]
		unpaywallURL := fmt.Sprintf("https://api.unpaywall.org/v2/%s?email=admin@example.com", doi)
		resp, err := http.Get(unpaywallURL)
		if err == nil {
			defer resp.Body.Close()
			var result struct {
				BestOALocation struct {
					PdfUrl string `json:"pdf_url"`
				} `json:"best_oa_location"`
				OALocations []struct {
					PdfUrl string `json:"pdf_url"`
				} `json:"oa_locations"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
				addLink(result.BestOALocation.PdfUrl)
				for _, loc := range result.OALocations {
					addLink(loc.PdfUrl)
				}
			}
		}
	}

	// 2. Scraping Manual untuk mencari link tambahan di HTML
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return pdfLinks, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return pdfLinks, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	// Cari meta tags (Highwire Press, Dublin Core)
	metaPatterns := []string{"citation_pdf_url", "dc.identifier", "prism.url"}
	for _, pattern := range metaPatterns {
		if strings.Contains(html, pattern) {
			parts := strings.Split(html, pattern)
			for i := 1; i < len(parts); i++ {
				contentParts := strings.Split(parts[i], "content=\"")
				if len(contentParts) > 1 {
					link := strings.Split(contentParts[1], "\"")[0]
					if strings.Contains(link, "http") {
						addLink(link)
					}
				}
			}
		}
	}

	if len(pdfLinks) == 0 {
		return nil, fmt.Errorf("no pdf links found")
	}

	return pdfLinks, nil
}

func SearchGoogle(query string) (*GoogleSearchResult, error) {
	key := os.Getenv("SEARCH_ENGINE_KEY")
	cx := os.Getenv("SEARCH_ENGINE_ID")

	if key == "" || cx == "" {
		return nil, fmt.Errorf("SEARCH_ENGINE_KEY or SEARCH_ENGINE_ID not set")
	}

	Debug("Key %s, Cx %s", key, cx)
	searchURL := fmt.Sprintf("https://cse.google.com/cse/element/v1?cx=%s&q=%s&callback=a",
		cx, url.QueryEscape(query))

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	// Browser simulation headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Referer", "https://cse.google.com/cse?cx="+cx)
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	sBody := string(body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google search failed: %s", sBody)
	}

	// Clean JSONP: a( ... );
	sBody = strings.TrimPrefix(sBody, "a(")
	sBody = strings.TrimSuffix(sBody, ");")
	sBody = strings.TrimSuffix(sBody, ")")

	var result GoogleSearchResult
	if err := json.Unmarshal([]byte(sBody), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON (might be blocked): %s", sBody[:100])
	}

	return &result, nil
}

func SearchJurnalHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	Debug("Parallel Search started for: %s", reqBody.Query)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var journalResults []JournalResult
	var contextParts []string

	// A. Thread untuk Google Search
	wg.Add(1)
	go func() {
		defer wg.Done()
		res, err := SearchGoogle(reqBody.Query + " filetype:pdf")
		if err == nil {
			var innerWg sync.WaitGroup
			for _, item := range res.Items {
				innerWg.Add(1)
				itemCopy := item // Capture for goroutine
				go func() {
					defer innerWg.Done()
					links, _ := ScrapForGetPDFUrl(itemCopy.Link)
					if len(links) == 0 {
						links = []string{itemCopy.Link}
					}

					mu.Lock()
					journalResults = append(journalResults, JournalResult{
						Source: "Google", Title: itemCopy.Title, PdfLinks: links, Summary: itemCopy.Snippet,
					})
					contextParts = append(contextParts, fmt.Sprintf("Title: %s\nSnippet: %s", itemCopy.Title, itemCopy.Snippet))
					mu.Unlock()
				}()
			}
			innerWg.Wait()
		}
	}()

	// B. Thread untuk OpenAlex
	wg.Add(1)
	go func() {
		defer wg.Done()
		res, err := SearchOpenAlex(reqBody.Query)
		if err == nil {
			var innerWg sync.WaitGroup
			for _, item := range res.Results {
				innerWg.Add(1)
				itemCopy := item // Capture for goroutine
				go func() {
					defer innerWg.Done()

					abstract := ReconstructAbstract(itemCopy.AbstractInvertedIndex)
					allPdfs, _ := ScrapForGetPDFUrl(itemCopy.DOI)
					if itemCopy.OpenAccess.PdfUrl != "" {
						allPdfs = append([]string{itemCopy.OpenAccess.PdfUrl}, allPdfs...)
					}

					authors := "Unknown"
					if len(itemCopy.Authorships) > 0 {
						authors = itemCopy.Authorships[0].Author.DisplayName
					}

					mu.Lock()
					journalResults = append(journalResults, JournalResult{
						Source: "OpenAlex", Title: itemCopy.Title, Authors: authors, Year: fmt.Sprintf("%d", itemCopy.PublicationYear),
						DOI: itemCopy.DOI, PdfLinks: allPdfs, Summary: abstract,
					})
					contextParts = append(contextParts, fmt.Sprintf("Title: %s\nAbstract: %s", itemCopy.Title, abstract))
					mu.Unlock()
				}()
				}
			innerWg.Wait()
		}
	}()

	// Tunggu semua pencarian dan scraping selesai
	wg.Wait()

	// 4. Summarization via AI dengan Context yang lebih kaya
	fullContext := strings.Join(contextParts, "\n---\n")
	aiSummary := AiSummary(reqBody.Query, fullContext)

	// 5. Send Final Response
	finalResp := FinalResponse{
		Query: reqBody.Query, Status: "success", Results: journalResults, AISummary: aiSummary,
	}
	Debug("%s", finalResp)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(finalResp)
}

func AiSummary(query, context string) string{
	apiKey := os.Getenv("NVIDIA_API_KEY")
	aiSummary := "AI analysis skipped (API key missing)"
	
	if apiKey != "" {
		prompt := fmt.Sprintf(`Act as an academic assistant. Analyze these journal results for the topic: "%s".
Summarize the key findings from these papers in 3-4 sentences.
Data:
%s`, query, context)

		aiBody := RequestBody{
			Model: "z-ai/glm4.7",
			Messages: []Message{{Role: "user", Content: prompt}},
			Temperature: 0.7,
			TopP: 1,
			MaxTokens: 1000,
		}
		
		jsonAi, _ := json.Marshal(aiBody)
		aiReq, _ := http.NewRequest("POST", "https://integrate.api.nvidia.com/v1/chat/completions", bytes.NewBuffer(jsonAi))
		aiReq.Header.Set("Authorization", "Bearer "+apiKey)
		aiReq.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		aiResp, err := client.Do(aiReq)
		if err == nil {
			defer aiResp.Body.Close()
			var nResp struct {
				Choices []struct {
					Message struct{ Content string } `json:"message"`
				} `json:"choices"`
			}
			if err := json.NewDecoder(aiResp.Body).Decode(&nResp); err == nil && len(nResp.Choices) > 0 {
				aiSummary = nResp.Choices[0].Message.Content
			}
		}
	}
	return aiSummary
}

// TestSearch runs a one-time test and prints to console
func TestSearch(query string) {
	LogInfo("Running TestSearch for: %s", query)
	res, err := SearchGoogle(query + " filetype:pdf")
	if err != nil {
		LogErr("TestSearch Google Error: %v", err)
	} else {
		LogInfo("TestSearch found %d results from Google", len(res.Items))
		for _, item := range res.Items {
			LogInfo("- %s (%s)", item.Title, item.Link)
		}
	}
}
