// Command api starts the WayraFlow backend HTTP server.
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"wayraflow/backend/internal/httpapi"
)

func main() {
	addr := ":" + envOr("PORT", "8080")
	srv := &http.Server{
		Addr:              addr,
		Handler:           httpapi.NewRouter(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("wayraflow api listening on %s", addr)
	log.Fatal(srv.ListenAndServe())
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
