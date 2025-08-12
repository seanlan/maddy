package docs

import (
	"embed"
	"net/http"

	"github.com/gorilla/mux"
)

//go:embed *
var Docs embed.FS

// RegisterOpenAPIService registers the OpenAPI service
func RegisterOpenAPIService(name string, router *mux.Router) {
	// Create a simple handler for OpenAPI docs
	router.PathPrefix("/swagger/").Handler(http.StripPrefix("/swagger/", http.FileServer(http.FS(Docs))))
}