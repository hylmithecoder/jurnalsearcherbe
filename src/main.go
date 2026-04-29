package main

import (
	"fmt"
	"hylmi/jurnalsearcher/src/handler"
	"hylmi/jurnalsearcher/src/routes"
	"log"
	"net/http"

	"github.com/joho/godotenv"
)

func TestSearchAndAi() {
	handler.LogInfo("--- Starting Startup Test ---")
	
	query := "deep learning architecture"
	handler.LogInfo("Testing Search for: %s", query)
	
	// res, err := handler.SearchGoogle(query + " filetype:pdf")
	// if err != nil {
	// 	handler.LogErr("Startup Search Test Failed: %v", err)
	// } else {
	// 	handler.LogInfo("Startup Search Test Success: found %d results", len(res.Items))
	// 	if len(res.Items) > 0 {
	// 		handler.LogInfo("First result: %s", res.Items[0].Title)
	// 	}
	// }

	response, err := handler.SearchOpenAlex(query)
	if err != nil {
		handler.LogErr("Startup Search Test Failed: %v", err)
	} else {
		handler.LogInfo("Startup Search Test Success: found %d results", len(response.Results))
		for i, item := range response.Results {
			handler.LogInfo("[%d] Title: %s\nAuthors: %s\nYear: %d\nPDF: %s\nDOI: %s\n\n", i+1, item.Title, item.Authorships[0].Author.DisplayName, item.PublicationYear, item.OpenAccess.PdfUrl, item.DOI)
			responsePDFs, err := handler.ScrapForGetPDFUrl(item.DOI)
			if err != nil {
				handler.LogErr("Scraping PDF Test Failed: %v", err)
			} else {
				handler.LogInfo("Scraping PDF Test Success: found %d links", len(responsePDFs))
				for _, link := range responsePDFs {
					handler.LogInfo("  -> %s", link)
				}
			}
		}
	}

	resAI := handler.AiSummary(query, "so what the meaning of deep learning architecture?")
	handler.LogInfo("AI Summary: %s", resAI)
	
	handler.LogInfo("--- Startup Test Finished ---")
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, using default environment variables")
	}

	routes.RegisterRoutes()

	// Run startup test
	TestSearchAndAi()

	fmt.Println("🚀 Server running on :3001")
	log.Fatal(http.ListenAndServe(":3001", nil))
}
