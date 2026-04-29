# ScholarLink Backend - Academic Journal Discovery Engine

A high-performance, multithreaded backend built with **Go** to power advanced academic research. This engine performs hybrid searches across multiple databases, verifies PDF accessibility in real-time, and generates intelligent summaries using AI.

🚀 **Frontend Repository:** [jurnalsearcherfe](https://github.com/hylmithecoder/jurnalsearcherfe)

## 🌟 Features

*   **Hybrid Search Engine**: Combines **OpenAlex API** (Academic metadata) with **Google Custom Search** for maximum coverage.
*   **High Concurrency**: Utilizes Go Routines and `sync.WaitGroup` to execute search and PDF scraping tasks in parallel.
*   **Deep PDF Discovery**: Custom scraping engine that extracts verified PDF links from landing pages using citation meta-tags.
*   **OJS Link Fixer**: Automatically detects Open Journal Systems (OJS) "download" links and converts them to browser-friendly "view" links.
*   **AI Synthesis**: Generates a 3-4 sentence summary of the search results using **NVIDIA GLM-4.7** LLM.
*   **Abstract Reconstruction**: Reassembles OpenAlex's Inverted Index into readable abstracts for better AI context.

## 🛠 Tech Stack

*   **Language**: Go (Golang)
*   **APIs**: 
    *   OpenAlex (Academic Research Data)
    *   Google Custom Search (Fallback discovery)
    *   NVIDIA Build (LLM API for GLM-4.7)
*   **Networking**: Standard `net/http` with browser simulation headers.

## 🚀 Getting Started

### Prerequisites

*   Go 1.22+ installed
*   An active NVIDIA API Key (for summarization)
*   Google Custom Search Engine ID & API Key (Optional)

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/hylmithecoder/jurnalsearcherhelper.git
   cd jurnalsearcherhelper
   ```

2. Create a `.env` file in the root directory:
   ```env
   SEARCH_ENGINE_KEY=your_google_api_key
   SEARCH_ENGINE_ID=your_cse_id
   NVIDIA_API_KEY=nvapi-xxxxxxxxxxxx
   ```

3. Run the server:
   ```bash
   go run src/*.go
   ```
   The server will start on `http://localhost:3001`.

## 📖 API Documentation

Detailed API documentation, including request/response structures, can be found in [docs.md](./docs.md).

### Main Endpoint:
`POST /api/searchjurnal`
*   **Payload**: `{ "query": "Your Search Topic" }`
*   **Response**: Rich JSON containing verified journal metadata and AI summary.

## 🤝 Relation with Frontend

This backend is designed to work seamlessly with the [ScholarLink Next.js Frontend](https://github.com/hylmithecoder/jurnalsearcherfe). The frontend uses Next.js rewrites to proxy requests to this backend, ensuring a smooth, CORS-free experience.

## 📄 License

Distributed under the MIT License. See `LICENSE` for more information.
