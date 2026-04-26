package httpapi

import (
	"net/http"

	appmiddleware "github.com/KimDmitriyR/mini_storage/internal/middleware"
	"github.com/go-chi/chi/v5"
)

type RouterOptions struct {
	Handler            *Handler
	MaxUploadSizeBytes int64
}

func NewRouter(options RouterOptions) http.Handler {
	router := chi.NewRouter()
	router.Use(appmiddleware.RequestLogger)
	router.Use(appmiddleware.Recovery)
	router.Get("/health", healthHandler)

	if options.Handler != nil {
		router.Route("/files", func(r chi.Router) {
			r.With(appmiddleware.RequestBodyLimit(options.MaxUploadSizeBytes)).Post("/", options.Handler.UploadFile)
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
