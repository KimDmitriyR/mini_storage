package httpapi

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/KimDmitriyR/mini_storage/internal/metadata"
	"github.com/KimDmitriyR/mini_storage/internal/storage"
)

func TestFileAPIFlow(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t, 1024*1024)

	uploadBody := &bytes.Buffer{}
	writer := multipart.NewWriter(uploadBody)
	fileWriter, err := writer.CreateFormFile("file", "hello.txt")
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}

	if _, err := fileWriter.Write([]byte("hello api")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Close() writer error = %v", err)
	}

	uploadRequest := httptest.NewRequest(http.MethodPost, "/files", uploadBody)
	uploadRequest.Header.Set("Content-Type", writer.FormDataContentType())
	uploadResponse := httptest.NewRecorder()
	router.ServeHTTP(uploadResponse, uploadRequest)

	if uploadResponse.Code != http.StatusCreated {
		t.Fatalf("upload status = %d, want %d, body = %s", uploadResponse.Code, http.StatusCreated, uploadResponse.Body.String())
	}

	var uploadedFile metadata.FileMetadata
	if err := json.NewDecoder(uploadResponse.Body).Decode(&uploadedFile); err != nil {
		t.Fatalf("decode upload response error = %v", err)
	}

	listRequest := httptest.NewRequest(http.MethodGet, "/files", nil)
	listResponse := httptest.NewRecorder()
	router.ServeHTTP(listResponse, listRequest)

	if listResponse.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listResponse.Code, http.StatusOK)
	}

	var listPayload struct {
		Files []metadata.FileMetadata `json:"files"`
	}
	if err := json.NewDecoder(listResponse.Body).Decode(&listPayload); err != nil {
		t.Fatalf("decode list response error = %v", err)
	}

	if len(listPayload.Files) != 1 {
		t.Fatalf("list files count = %d, want 1", len(listPayload.Files))
	}

	metaRequest := httptest.NewRequest(http.MethodGet, "/files/"+uploadedFile.ID+"/meta", nil)
	metaResponse := httptest.NewRecorder()
	router.ServeHTTP(metaResponse, metaRequest)

	if metaResponse.Code != http.StatusOK {
		t.Fatalf("metadata status = %d, want %d", metaResponse.Code, http.StatusOK)
	}

	downloadRequest := httptest.NewRequest(http.MethodGet, "/files/"+uploadedFile.ID, nil)
	downloadResponse := httptest.NewRecorder()
	router.ServeHTTP(downloadResponse, downloadRequest)

	if downloadResponse.Code != http.StatusOK {
		t.Fatalf("download status = %d, want %d", downloadResponse.Code, http.StatusOK)
	}

	if downloadResponse.Body.String() != "hello api" {
		t.Fatalf("download body = %q, want %q", downloadResponse.Body.String(), "hello api")
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/files/"+uploadedFile.ID, nil)
	deleteResponse := httptest.NewRecorder()
	router.ServeHTTP(deleteResponse, deleteRequest)

	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", deleteResponse.Code, http.StatusNoContent)
	}

	missingMetaRequest := httptest.NewRequest(http.MethodGet, "/files/"+uploadedFile.ID+"/meta", nil)
	missingMetaResponse := httptest.NewRecorder()
	router.ServeHTTP(missingMetaResponse, missingMetaRequest)

	if missingMetaResponse.Code != http.StatusNotFound {
		t.Fatalf("metadata after delete status = %d, want %d", missingMetaResponse.Code, http.StatusNotFound)
	}
}

func TestUploadRejectsLargeFiles(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t, 128)

	uploadBody := &bytes.Buffer{}
	writer := multipart.NewWriter(uploadBody)
	fileWriter, err := writer.CreateFormFile("file", "too-big.txt")
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}

	if _, err := fileWriter.Write(bytes.Repeat([]byte("a"), 1024)); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Close() writer error = %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/files", uploadBody)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("upload large file status = %d, want %d, body = %s", response.Code, http.StatusRequestEntityTooLarge, response.Body.String())
	}

	var payload map[string]string
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode error response error = %v", err)
	}

	if payload["error"] == "" {
		t.Fatalf("error response should include message")
	}
}

func TestDownloadMissingFileReturnsJSONError(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t, 1024*1024)

	request := httptest.NewRequest(http.MethodGet, "/files/missing-id", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("download missing file status = %d, want %d", response.Code, http.StatusNotFound)
	}

	var payload map[string]string
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode error response error = %v", err)
	}

	if payload["error"] != "file metadata not found" {
		t.Fatalf("error message = %q, want %q", payload["error"], "file metadata not found")
	}
}

func newTestRouter(t *testing.T, maxUploadSizeBytes int64) http.Handler {
	t.Helper()

	fileStorage, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocal() error = %v", err)
	}

	metadataRepository, err := metadata.NewSQLite(filepath.Join(t.TempDir(), "metadata.db"))
	if err != nil {
		t.Fatalf("NewSQLite() error = %v", err)
	}

	t.Cleanup(func() {
		if err := metadataRepository.Close(); err != nil {
			t.Fatalf("Close() metadata repository error = %v", err)
		}
	})

	return NewRouter(RouterOptions{
		Handler:            NewHandler(fileStorage, metadataRepository, maxUploadSizeBytes),
		MaxUploadSizeBytes: maxUploadSizeBytes,
	})
}
