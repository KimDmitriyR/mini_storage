package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrFileNotFound     = errors.New("file not found")
	ErrInvalidFileName  = errors.New("invalid file name")
	ErrStorageViolation = errors.New("resolved path escapes storage directory")
)

type SaveResult struct {
	StoredName string
	Size       int64
}

type File struct {
	Reader io.ReadCloser
	Size   int64
}

type LocalStorage struct {
	baseDir string
}

func NewLocal(baseDir string) (*LocalStorage, error) {
	if strings.TrimSpace(baseDir) == "" {
		return nil, fmt.Errorf("base dir is required")
	}

	absoluteBaseDir, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("resolve base dir: %w", err)
	}

	if err := os.MkdirAll(absoluteBaseDir, 0o755); err != nil {
		return nil, fmt.Errorf("create base dir: %w", err)
	}

	return &LocalStorage{baseDir: absoluteBaseDir}, nil
}

func (s *LocalStorage) Save(ctx context.Context, originalName string, src io.Reader) (SaveResult, error) {
	storedName := uuid.NewString() + filepath.Ext(filepath.Base(originalName))
	filePath, err := s.resolvePath(storedName)
	if err != nil {
		return SaveResult{}, err
	}

	dst, err := os.Create(filePath)
	if err != nil {
		return SaveResult{}, fmt.Errorf("create file: %w", err)
	}

	written, copyErr := copyWithContext(ctx, dst, src)
	closeErr := dst.Close()
	if copyErr != nil {
		_ = os.Remove(filePath)
		return SaveResult{}, fmt.Errorf("write file: %w", copyErr)
	}

	if closeErr != nil {
		_ = os.Remove(filePath)
		return SaveResult{}, fmt.Errorf("close file: %w", closeErr)
	}

	return SaveResult{
		StoredName: storedName,
		Size:       written,
	}, nil
}

func (s *LocalStorage) Open(storedName string) (File, error) {
	filePath, err := s.resolvePath(storedName)
	if err != nil {
		return File{}, err
	}

	file, err := os.Open(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return File{}, ErrFileNotFound
		}

		return File{}, fmt.Errorf("open file: %w", err)
	}

	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return File{}, fmt.Errorf("stat file: %w", err)
	}

	return File{
		Reader: file,
		Size:   info.Size(),
	}, nil
}

func (s *LocalStorage) Delete(storedName string) error {
	filePath, err := s.resolvePath(storedName)
	if err != nil {
		return err
	}

	if err := os.Remove(filePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrFileNotFound
		}

		return fmt.Errorf("remove file: %w", err)
	}

	return nil
}

func (s *LocalStorage) resolvePath(storedName string) (string, error) {
	if storedName == "" {
		return "", ErrInvalidFileName
	}

	cleanName := filepath.Clean(storedName)
	if cleanName == "." || cleanName == ".." || cleanName != filepath.Base(cleanName) {
		return "", ErrInvalidFileName
	}

	resolvedPath := filepath.Join(s.baseDir, cleanName)
	absolutePath, err := filepath.Abs(resolvedPath)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}

	relativePath, err := filepath.Rel(s.baseDir, absolutePath)
	if err != nil {
		return "", fmt.Errorf("check relative path: %w", err)
	}

	if relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(os.PathSeparator)) {
		return "", ErrStorageViolation
	}

	return absolutePath, nil
}

func copyWithContext(ctx context.Context, dst io.Writer, src io.Reader) (int64, error) {
	if ctx == nil {
		return io.Copy(dst, src)
	}

	return io.Copy(dst, &contextReader{
		ctx: ctx,
		src: src,
	})
}

type contextReader struct {
	ctx context.Context
	src io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}

	return r.src.Read(p)
}
