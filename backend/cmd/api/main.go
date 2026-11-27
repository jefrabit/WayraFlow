// Command api starts the WayraFlow backend HTTP server.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"wayraflow/backend/internal/db"
	"wayraflow/backend/internal/httpapi"
)

func main() {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		pool, err := db.Connect(ctx, url)
		if err != nil {
			log.Fatalf("database: %v", err)
		}
		if err := db.Migrate(ctx, pool); err != nil {
			log.Fatalf("migrations: %v", err)
		}
		log.Print("database migrations applied")
		if path := os.Getenv("NETWORK_FILE"); path != "" {
			if err := loadNetwork(ctx, pool, path); err != nil {
				log.Fatalf("network: %v", err)
			}
		}
		cancel()
		defer pool.Close()
	} else {
		log.Print("DATABASE_URL not set; starting without database")
	}

	addr := ":" + envOr("PORT", "8080")
	srv := &http.Server{
		Addr:              addr,
		Handler:           httpapi.NewRouter(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("wayraflow api listening on %s", addr)
	log.Fatal(srv.ListenAndServe())
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
	log.Printf("road network loaded: %d segments from %s", n, path)
	return nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
