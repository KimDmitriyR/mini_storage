package metadata

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestSQLiteRepositoryCreateGetListDelete(t *testing.T) {
	t.Parallel()

	repo, err := NewSQLite(filepath.Join(t.TempDir(), "metadata.db"))
	if err != nil {
		t.Fatalf("NewSQLite() error = %v", err)
	}
	defer repo.Close()

	firstFile := FileMetadata{
		ID:           "file-1",
		OriginalName: "alpha.txt",
		StoredName:   "stored-alpha.txt",
		ContentType:  "text/plain",
		Size:         10,
		CreatedAt:    time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC),
	}
	secondFile := FileMetadata{
		ID:           "file-2",
		OriginalName: "beta.txt",
		StoredName:   "stored-beta.txt",
		ContentType:  "text/plain",
		Size:         20,
		CreatedAt:    time.Date(2026, 4, 26, 12, 1, 0, 0, time.UTC),
	}

	if err := repo.Create(context.Background(), firstFile); err != nil {
		t.Fatalf("Create(firstFile) error = %v", err)
	}

	if err := repo.Create(context.Background(), secondFile); err != nil {
		t.Fatalf("Create(secondFile) error = %v", err)
	}

	storedFile, err := repo.GetByID(context.Background(), firstFile.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if storedFile.StoredName != firstFile.StoredName {
		t.Fatalf("GetByID() stored file name = %q, want %q", storedFile.StoredName, firstFile.StoredName)
	}

	files, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("List() length = %d, want 2", len(files))
	}

	if files[0].ID != secondFile.ID {
		t.Fatalf("List() first item = %q, want %q", files[0].ID, secondFile.ID)
	}

	if err := repo.Delete(context.Background(), firstFile.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = repo.GetByID(context.Background(), firstFile.ID)
	if !errors.Is(err, ErrMetadataNotFound) {
		t.Fatalf("GetByID() after delete error = %v, want ErrMetadataNotFound", err)
	}
}

func TestSQLiteRepositoryDeleteMissingFile(t *testing.T) {
	t.Parallel()

	repo, err := NewSQLite(filepath.Join(t.TempDir(), "metadata.db"))
	if err != nil {
		t.Fatalf("NewSQLite() error = %v", err)
	}
	defer repo.Close()

	err = repo.Delete(context.Background(), "missing")
	if !errors.Is(err, ErrMetadataNotFound) {
		t.Fatalf("Delete() missing file error = %v, want ErrMetadataNotFound", err)
	}
}
