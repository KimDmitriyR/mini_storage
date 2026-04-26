package storage

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestLocalStorageSaveOpenDelete(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store, err := NewLocal(dir)
	if err != nil {
		t.Fatalf("NewLocal() error = %v", err)
	}

	savedFile, err := store.Save(context.Background(), "report.txt", strings.NewReader("hello storage"))
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if !strings.HasSuffix(savedFile.StoredName, ".txt") {
		t.Fatalf("Save() stored file name = %q, want .txt suffix", savedFile.StoredName)
	}

	openedFile, err := store.Open(savedFile.StoredName)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	content, err := io.ReadAll(openedFile.Reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if err := openedFile.Reader.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if string(content) != "hello storage" {
		t.Fatalf("stored content = %q, want %q", string(content), "hello storage")
	}

	if err := store.Delete(savedFile.StoredName); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = store.Open(savedFile.StoredName)
	if !errors.Is(err, ErrFileNotFound) {
		t.Fatalf("Open() after delete error = %v, want ErrFileNotFound", err)
	}
}

func TestLocalStorageRejectsPathTraversal(t *testing.T) {
	t.Parallel()

	store, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocal() error = %v", err)
	}

	_, err = store.Open("../secret.txt")
	if !errors.Is(err, ErrInvalidFileName) {
		t.Fatalf("Open() traversal error = %v, want ErrInvalidFileName", err)
	}

	err = store.Delete("nested/file.txt")
	if !errors.Is(err, ErrInvalidFileName) {
		t.Fatalf("Delete() nested path error = %v, want ErrInvalidFileName", err)
	}
}
