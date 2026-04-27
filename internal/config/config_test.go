package config

import "testing"

func TestLoadUsesDefaults(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Port != "8080" {
		t.Fatalf("Port = %q, want %q", cfg.Port, "8080")
	}

	if cfg.StorageDir != "storage" {
		t.Fatalf("StorageDir = %q, want %q", cfg.StorageDir, "storage")
	}

	if cfg.MaxUploadSizeMB != 10 {
		t.Fatalf("MaxUploadSizeMB = %d, want %d", cfg.MaxUploadSizeMB, 10)
	}

	if cfg.DatabasePath != "storage/metadata.db" {
		t.Fatalf("DatabasePath = %q, want %q", cfg.DatabasePath, "storage/metadata.db")
	}
}

func TestLoadUsesEnvironmentValues(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("STORAGE_DIR", "/tmp/files")
	t.Setenv("MAX_UPLOAD_SIZE_MB", "25")
	t.Setenv("DATABASE_PATH", "/tmp/meta.db")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Port != "9090" {
		t.Fatalf("Port = %q, want %q", cfg.Port, "9090")
	}

	if cfg.StorageDir != "/tmp/files" {
		t.Fatalf("StorageDir = %q, want %q", cfg.StorageDir, "/tmp/files")
	}

	if cfg.MaxUploadSizeMB != 25 {
		t.Fatalf("MaxUploadSizeMB = %d, want %d", cfg.MaxUploadSizeMB, 25)
	}

	if cfg.DatabasePath != "/tmp/meta.db" {
		t.Fatalf("DatabasePath = %q, want %q", cfg.DatabasePath, "/tmp/meta.db")
	}
}

func TestLoadRejectsNonPositiveUploadLimit(t *testing.T) {
	t.Setenv("MAX_UPLOAD_SIZE_MB", "0")

	if _, err := Load(); err == nil {
		t.Fatalf("Load() error = nil, want validation error")
	}
}
