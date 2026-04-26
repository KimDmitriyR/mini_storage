package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KimDmitriyR/mini_storage/internal/config"
	httpapi "github.com/KimDmitriyR/mini_storage/internal/http"
	"github.com/KimDmitriyR/mini_storage/internal/metadata"
	"github.com/KimDmitriyR/mini_storage/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	fileStorage, err := storage.NewLocal(cfg.StorageDir)
	if err != nil {
		log.Fatalf("init file storage: %v", err)
	}

	metadataRepository, err := metadata.NewSQLite(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("init metadata repository: %v", err)
	}
	defer func() {
		if err := metadataRepository.Close(); err != nil {
			log.Printf("close metadata repository: %v", err)
		}
	}()

	server := &http.Server{
		Addr: cfg.Address(),
		Handler: httpapi.NewRouter(httpapi.RouterOptions{
			Handler: httpapi.NewHandler(fileStorage, metadataRepository),
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("starting server on %s", cfg.Address())

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen and serve: %v", err)
		}
	}()

	shutdownCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-shutdownCtx.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown server: %v", err)
	}
}
