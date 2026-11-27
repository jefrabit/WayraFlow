// Command osmextract downloads the Juliaca road network from the Overpass API
// and writes it as GeoJSON. It is run manually, once; the output file is
// committed so the demo never needs internet access.
//
//	go run ./cmd/osmextract -out ../data/juliaca.geojson
//	go run ./cmd/osmextract -in overpass.json -out ../data/juliaca.geojson
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wayraflow/backend/internal/osm"
)

// Query matches README section 3.1: bbox (south, west, north, east).
const Query = `[out:json][timeout:120];
(
  way["highway"~"^(trunk|primary|secondary|tertiary)(_link)?$"](-15.56,-70.20,-15.44,-70.08);
  node["highway"="traffic_signals"](-15.56,-70.20,-15.44,-70.08);
);
out body; >; out skel qt;`

var endpoints = []string{
	"https://overpass-api.de/api/interpreter",
	"https://maps.mail.ru/osm/tools/overpass/api/interpreter",
}

func main() {
	in := flag.String("in", "", "read a saved Overpass JSON response instead of downloading")
	out := flag.String("out", "../data/juliaca.geojson", "output GeoJSON path")
	flag.Parse()

	raw, err := load(*in)
	if err != nil {
		log.Fatal(err)
	}
	fc, err := osm.ToGeoJSON(bytes.NewReader(raw))
	if err != nil {
		log.Fatal(err)
	}
	if err := write(*out, fc); err != nil {
		log.Fatal(err)
	}
	roads, signals := 0, 0
	for _, f := range fc.Features {
		if f.Geometry.Type == "LineString" {
			roads++
		} else {
			signals++
		}
	}
	fmt.Printf("wrote %s: %d road segments, %d traffic signals (OSM %s)\n", *out, roads, signals, fc.Metadata.OSMTimestamp)
}

func load(path string) ([]byte, error) {
	if path != "" {
		return os.ReadFile(path)
	}
	client := &http.Client{Timeout: 150 * time.Second}
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		for _, ep := range endpoints {
			body, err := fetch(client, ep)
			if err == nil {
				return body, nil
			}
			lastErr = err
			log.Printf("overpass %s: %v", ep, err)
		}
		time.Sleep(time.Duration(attempt+1) * 10 * time.Second)
	}
	return nil, fmt.Errorf("all overpass endpoints failed: %w", lastErr)
}

func fetch(client *http.Client, endpoint string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(url.Values{"data": {Query}}.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "WayraFlow-osmextract/0.1")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK || !bytes.HasPrefix(bytes.TrimSpace(body), []byte("{")) {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return body, nil
}

func write(path string, fc osm.FeatureCollection) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(fc)
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
