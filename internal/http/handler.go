package httpapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"

	"github.com/KimDmitriyR/mini_storage/internal/metadata"
	"github.com/KimDmitriyR/mini_storage/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type fileStorage interface {
	Save(ctx context.Context, originalName string, src io.Reader) (storage.SaveResult, error)
	Open(storedName string) (storage.File, error)
	Delete(storedName string) error
}

type metadataRepository interface {
	Create(ctx context.Context, file metadata.FileMetadata) error
	GetByID(ctx context.Context, id string) (metadata.FileMetadata, error)
	List(ctx context.Context) ([]metadata.FileMetadata, error)
	Delete(ctx context.Context, id string) error
}

type Handler struct {
	storage  fileStorage
	metadata metadataRepository
}

func NewHandler(fileStorage fileStorage, metadataRepo metadataRepository) *Handler {
	return &Handler{
		storage:  fileStorage,
		metadata: metadataRepo,
	}
}

func (h *Handler) UploadFile(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, fmt.Sprintf("parse multipart form: %v", err), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, fmt.Sprintf("read multipart file: %v", err), http.StatusBadRequest)
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(header.Filename))
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	savedFile, err := h.storage.Save(r.Context(), header.Filename, file)
	if err != nil {
		http.Error(w, fmt.Sprintf("save file: %v", err), http.StatusInternalServerError)
		return
	}

	fileMetadata := metadata.FileMetadata{
		ID:           uuid.NewString(),
		OriginalName: header.Filename,
		StoredName:   savedFile.StoredName,
		ContentType:  contentType,
		Size:         savedFile.Size,
	}

	if err := h.metadata.Create(r.Context(), fileMetadata); err != nil {
		_ = h.storage.Delete(savedFile.StoredName)
		http.Error(w, fmt.Sprintf("store metadata: %v", err), http.StatusInternalServerError)
		return
	}

	storedFile, err := h.metadata.GetByID(r.Context(), fileMetadata.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("fetch stored metadata: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, storedFile)
}

func (h *Handler) ListFiles(w http.ResponseWriter, r *http.Request) {
	files, err := h.metadata.List(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("list metadata: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"files": files})
}

func (h *Handler) DownloadFile(w http.ResponseWriter, r *http.Request) {
	fileMetadata, err := h.metadata.GetByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, metadata.ErrMetadataNotFound) {
			statusCode = http.StatusNotFound
		}

		http.Error(w, fmt.Sprintf("get metadata: %v", err), statusCode)
		return
	}

	storedFile, err := h.storage.Open(fileMetadata.StoredName)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, storage.ErrFileNotFound) {
			statusCode = http.StatusNotFound
		}

		http.Error(w, fmt.Sprintf("open file: %v", err), statusCode)
		return
	}
	defer storedFile.Reader.Close()

	w.Header().Set("Content-Type", fileMetadata.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileMetadata.OriginalName))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", storedFile.Size))
	w.WriteHeader(http.StatusOK)

	if _, err := io.Copy(w, storedFile.Reader); err != nil {
		http.Error(w, fmt.Sprintf("stream file: %v", err), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) GetFileMetadata(w http.ResponseWriter, r *http.Request) {
	fileMetadata, err := h.metadata.GetByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, metadata.ErrMetadataNotFound) {
			statusCode = http.StatusNotFound
		}

		http.Error(w, fmt.Sprintf("get metadata: %v", err), statusCode)
		return
	}

	writeJSON(w, http.StatusOK, fileMetadata)
}

func (h *Handler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	fileMetadata, err := h.metadata.GetByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, metadata.ErrMetadataNotFound) {
			statusCode = http.StatusNotFound
		}

		http.Error(w, fmt.Sprintf("get metadata: %v", err), statusCode)
		return
	}

	if err := h.storage.Delete(fileMetadata.StoredName); err != nil && !errors.Is(err, storage.ErrFileNotFound) {
		http.Error(w, fmt.Sprintf("delete file: %v", err), http.StatusInternalServerError)
		return
	}

	if err := h.metadata.Delete(r.Context(), fileMetadata.ID); err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, metadata.ErrMetadataNotFound) {
			statusCode = http.StatusNotFound
		}

		http.Error(w, fmt.Sprintf("delete metadata: %v", err), statusCode)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
