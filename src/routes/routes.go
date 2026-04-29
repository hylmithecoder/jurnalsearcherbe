package routes

import (
	"net/http"

	"hylmi/jurnalsearcher/src/handler"
)

func RegisterRoutes() {
	http.HandleFunc("/api/searchjurnal", handler.SearchJurnalHandler)
}
