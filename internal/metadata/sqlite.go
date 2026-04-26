package metadata

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const createFilesTableSQL = `
CREATE TABLE IF NOT EXISTS files (
    id TEXT PRIMARY KEY,
    original_name TEXT NOT NULL,
    stored_name TEXT NOT NULL UNIQUE,
    content_type TEXT NOT NULL,
    size INTEGER NOT NULL,
    created_at TEXT NOT NULL
);
`

var ErrMetadataNotFound = errors.New("file metadata not found")

type FileMetadata struct {
	ID           string    `json:"id"`
	OriginalName string    `json:"original_name"`
	StoredName   string    `json:"stored_name"`
	ContentType  string    `json:"content_type"`
	Size         int64     `json:"size"`
	CreatedAt    time.Time `json:"created_at"`
}

type Repository struct {
	db *sql.DB
}

func NewSQLite(databasePath string) (*Repository, error) {
	if databasePath == "" {
		return nil, fmt.Errorf("database path is required")
	}

	if err := os.MkdirAll(filepath.Dir(databasePath), 0o755); err != nil {
		return nil, fmt.Errorf("create database dir: %w", err)
	}

	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	db.SetMaxOpenConns(1)

	repo := &Repository{db: db}
	if err := repo.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return repo, nil
}

func (r *Repository) Close() error {
	return r.db.Close()
}

func (r *Repository) Create(ctx context.Context, file FileMetadata) error {
	if file.CreatedAt.IsZero() {
		file.CreatedAt = time.Now().UTC()
	}

	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO files (id, original_name, stored_name, content_type, size, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		file.ID,
		file.OriginalName,
		file.StoredName,
		file.ContentType,
		file.Size,
		file.CreatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("insert file metadata: %w", err)
	}

	return nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (FileMetadata, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, original_name, stored_name, content_type, size, created_at
		 FROM files
		 WHERE id = ?`,
		id,
	)

	file, err := scanFile(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return FileMetadata{}, ErrMetadataNotFound
		}

		return FileMetadata{}, fmt.Errorf("get file metadata: %w", err)
	}

	return file, nil
}

func (r *Repository) List(ctx context.Context) ([]FileMetadata, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, original_name, stored_name, content_type, size, created_at
		 FROM files
		 ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list file metadata: %w", err)
	}
	defer rows.Close()

	files := make([]FileMetadata, 0)
	for rows.Next() {
		file, err := scanFile(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("scan file metadata: %w", err)
		}

		files = append(files, file)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate file metadata: %w", err)
	}

	return files, nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM files WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete file metadata: %w", err)
	}

	affectedRows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get affected rows: %w", err)
	}

	if affectedRows == 0 {
		return ErrMetadataNotFound
	}

	return nil
}

func (r *Repository) migrate() error {
	if _, err := r.db.Exec(createFilesTableSQL); err != nil {
		return fmt.Errorf("migrate sqlite schema: %w", err)
	}

	return nil
}

type scanner func(dest ...any) error

func scanFile(scan scanner) (FileMetadata, error) {
	var (
		file      FileMetadata
		createdAt string
	)

	if err := scan(
		&file.ID,
		&file.OriginalName,
		&file.StoredName,
		&file.ContentType,
		&file.Size,
		&createdAt,
	); err != nil {
		return FileMetadata{}, err
	}

	parsedCreatedAt, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return FileMetadata{}, fmt.Errorf("parse created_at: %w", err)
	}

	file.CreatedAt = parsedCreatedAt
	return file, nil
}
