// Package osm converts Overpass API responses into the GeoJSON network file
// stored in the repository (data/juliaca.geojson).
package osm

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
)

// Attribution required by the ODbL for OpenStreetMap-derived data.
const (
	Attribution = "© OpenStreetMap contributors"
	License     = "ODbL-1.0"
)

type element struct {
	Type  string            `json:"type"`
	ID    int64             `json:"id"`
	Lat   float64           `json:"lat"`
	Lon   float64           `json:"lon"`
	Nodes []int64           `json:"nodes"`
	Tags  map[string]string `json:"tags"`
}

type overpassResponse struct {
	OSM3S struct {
		TimestampOSMBase string `json:"timestamp_osm_base"`
	} `json:"osm3s"`
	Elements []element `json:"elements"`
}

// Geometry is a GeoJSON geometry; Coordinates is [2]float64 for a Point and
// [][2]float64 for a LineString, always in [lon, lat] order.
type Geometry struct {
	Type        string `json:"type"`
	Coordinates any    `json:"coordinates"`
}

// Feature is a GeoJSON feature.
type Feature struct {
	Type       string         `json:"type"`
	Geometry   Geometry       `json:"geometry"`
	Properties map[string]any `json:"properties"`
}

// Metadata records provenance and licensing of the extract.
type Metadata struct {
	Source       string `json:"source"`
	OSMTimestamp string `json:"osm_timestamp"`
	Attribution  string `json:"attribution"`
	License      string `json:"license"`
}

// FeatureCollection is the root GeoJSON object written to disk.
type FeatureCollection struct {
	Type     string    `json:"type"`
	Metadata Metadata  `json:"metadata"`
	Features []Feature `json:"features"`
}

// ToGeoJSON converts an Overpass JSON response ("out body; >; out skel qt;")
// into road LineStrings and traffic-signal Points, sorted by OSM id so the
// output is deterministic. Ways that reference missing nodes are dropped.
func ToGeoJSON(r io.Reader) (FeatureCollection, error) {
	var resp overpassResponse
	if err := json.NewDecoder(r).Decode(&resp); err != nil {
		return FeatureCollection{}, fmt.Errorf("decode overpass response: %w", err)
	}
	if len(resp.Elements) == 0 {
		return FeatureCollection{}, errors.New("overpass response has no elements")
	}

	coords := make(map[int64][2]float64)
	var ways, signals []element
	for _, e := range resp.Elements {
		switch e.Type {
		case "node":
			coords[e.ID] = [2]float64{e.Lon, e.Lat}
			if e.Tags["highway"] == "traffic_signals" {
				signals = append(signals, e)
			}
		case "way":
			ways = append(ways, e)
		}
	}
	sort.Slice(ways, func(i, j int) bool { return ways[i].ID < ways[j].ID })
	sort.Slice(signals, func(i, j int) bool { return signals[i].ID < signals[j].ID })

	fc := FeatureCollection{
		Type: "FeatureCollection",
		Metadata: Metadata{
			Source:       "Overpass API",
			OSMTimestamp: resp.OSM3S.TimestampOSMBase,
			Attribution:  Attribution,
			License:      License,
		},
		Features: []Feature{},
	}
	for _, w := range ways {
		line := make([][2]float64, 0, len(w.Nodes))
		for _, id := range w.Nodes {
			c, ok := coords[id]
			if !ok {
				line = nil
				break
			}
			line = append(line, c)
		}
		if len(line) < 2 {
			continue
		}
		props := map[string]any{"osm_way_id": w.ID, "highway": w.Tags["highway"]}
		for _, k := range []string{"name", "ref"} {
			if v := w.Tags[k]; v != "" {
				props[k] = v
			}
		}
		fc.Features = append(fc.Features, Feature{
			Type:       "Feature",
			Geometry:   Geometry{Type: "LineString", Coordinates: line},
			Properties: props,
		})
	}
	for _, s := range signals {
		fc.Features = append(fc.Features, Feature{
			Type:       "Feature",
			Geometry:   Geometry{Type: "Point", Coordinates: coords[s.ID]},
			Properties: map[string]any{"osm_node_id": s.ID, "kind": "traffic_signals"},
		})
	}
	return fc, nil
}
