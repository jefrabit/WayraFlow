// Command api starts the WayraFlow backend HTTP server.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"wayraflow/backend/internal/db"
	"wayraflow/backend/internal/httpapi"
	"wayraflow/backend/internal/store"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	var (
		st   store.Store
		pool *pgxpool.Pool
	)
	if url := os.Getenv("DATABASE_URL"); url != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		p, err := db.Connect(ctx, url)
		if err != nil {
			slog.Error("database", "error", err)
			os.Exit(1)
		}
		if err := db.Migrate(ctx, p); err != nil {
			slog.Error("migrations", "error", err)
			os.Exit(1)
		}
		slog.Info("database migrations applied")
		if path := os.Getenv("NETWORK_FILE"); path != "" {
			if err := loadNetwork(ctx, p, path); err != nil {
				slog.Error("network", "error", err)
				os.Exit(1)
			}
		}
		pool = p
		st = store.NewPostgres(p)
	} else {
		slog.Warn("DATABASE_URL not set; using the in-memory store")
		st = store.NewMemory()
	}
	defer st.Close()

	addr := ":" + envOr("PORT", "8080")
	srv := &http.Server{
		Addr:              addr,
		Handler:           httpapi.NewRouter(st),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		<-sig
		slog.Info("shutting down")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	slog.Info("wayraflow api listening", "addr", addr, "storage", storageName(pool))
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server", "error", err)
		os.Exit(1)
	}
}

func storageName(pool *pgxpool.Pool) string {
	if pool == nil {
		return "memory"
	}
	return "postgres"
}

func loadNetwork(ctx context.Context, pool *pgxpool.Pool, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	n, err := db.LoadRoadNetwork(ctx, pool, f)
	if err != nil {
		return err
	}
	slog.Info("road network loaded", "segments", n, "file", path)
	return nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
