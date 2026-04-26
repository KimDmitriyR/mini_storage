package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type RouterOptions struct {
	Handler *Handler
}

func NewRouter(options RouterOptions) http.Handler {
	router := chi.NewRouter()
	router.Get("/health", healthHandler)

	if options.Handler != nil {
		router.Route("/files", func(r chi.Router) {
			r.Post("/", options.Handler.UploadFile)
			r.Get("/", options.Handler.ListFiles)
			r.Get("/{id}", options.Handler.DownloadFile)
			r.Get("/{id}/meta", options.Handler.GetFileMetadata)
			r.Delete("/{id}", options.Handler.DeleteFile)
		})
	}

	return router
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	_ = json.NewEncoder(w).Encode(payload)
}
