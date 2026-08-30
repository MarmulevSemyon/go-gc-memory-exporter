package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/MarmulevSemyon/go-gc-memory-exporter/internal/config"
	"github.com/MarmulevSemyon/go-gc-memory-exporter/internal/httpapi"
	"github.com/MarmulevSemyon/go-gc-memory-exporter/internal/metrics"
)

const shutdownTimeout = 5 * time.Second

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	previousGCPercent := debug.SetGCPercent(cfg.GCPercent)
	collector := metrics.NewCollector(cfg.GCPercent)
	api := httpapi.NewServer(collector)

	server := &http.Server{
		Addr:              cfg.Address,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("server started on %s", cfg.Address)
		log.Printf("gc percent set to %d, previous value was %d", cfg.GCPercent, previousGCPercent)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen and serve: %v", err)
		}
	}()

	shutdown(server)
}

func shutdown(server *http.Server) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	log.Println("server is shutting down")
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
		return
	}

	log.Println("server stopped")
}
