package osm

import (
	"encoding/json"
	"strings"
	"testing"
)

const overpassFixture = `{
  "osm3s": {"timestamp_osm_base": "2026-11-27T20:10:23Z"},
  "elements": [
    {"type": "way", "id": 20, "nodes": [1, 2, 3], "tags": {"highway": "primary", "name": "Avenida Ferrocarril"}},
    {"type": "way", "id": 10, "nodes": [3, 4], "tags": {"highway": "trunk", "name": "Avenida Independencia", "ref": "PE-3S"}},
    {"type": "way", "id": 30, "nodes": [1, 99], "tags": {"highway": "tertiary"}},
    {"type": "node", "id": 1, "lat": -15.48, "lon": -70.13},
    {"type": "node", "id": 2, "lat": -15.481, "lon": -70.131},
    {"type": "node", "id": 3, "lat": -15.482, "lon": -70.132},
    {"type": "node", "id": 4, "lat": -15.483, "lon": -70.133},
    {"type": "node", "id": 7, "lat": -15.49, "lon": -70.14, "tags": {"highway": "traffic_signals"}}
  ]
}`

func convertFixture(t *testing.T) FeatureCollection {
	t.Helper()
	fc, err := ToGeoJSON(strings.NewReader(overpassFixture))
	if err != nil {
		t.Fatal(err)
	}
	return fc
}

func TestToGeoJSONBuildsSortedRoadLines(t *testing.T) {
	fc := convertFixture(t)
	if fc.Type != "FeatureCollection" {
		t.Fatalf("type = %q", fc.Type)
	}
	var roads []Feature
	for _, f := range fc.Features {
		if f.Geometry.Type == "LineString" {
			roads = append(roads, f)
		}
	}
	// Way 30 references missing node 99 and must be dropped.
	if len(roads) != 2 {
		t.Fatalf("roads = %d, want 2", len(roads))
	}
	if roads[0].Properties["osm_way_id"] != int64(10) || roads[1].Properties["osm_way_id"] != int64(20) {
		t.Fatalf("roads not sorted by osm_way_id: %v, %v", roads[0].Properties, roads[1].Properties)
	}
	if roads[0].Properties["ref"] != "PE-3S" || roads[0].Properties["highway"] != "trunk" || roads[0].Properties["name"] != "Avenida Independencia" {
		t.Fatalf("unexpected properties %v", roads[0].Properties)
	}
	coords := roads[1].Geometry.Coordinates.([][2]float64)
	if len(coords) != 3 || coords[0] != [2]float64{-70.13, -15.48} {
		t.Fatalf("coordinates must be [lon, lat] in node order, got %v", coords)
	}
}

func TestToGeoJSONIncludesSignalsAndMetadata(t *testing.T) {
	fc := convertFixture(t)
	var signals []Feature
	for _, f := range fc.Features {
		if f.Properties["kind"] == "traffic_signals" {
			signals = append(signals, f)
		}
	}
	if len(signals) != 1 || signals[0].Properties["osm_node_id"] != int64(7) {
		t.Fatalf("signals = %v, want one with osm_node_id 7", signals)
	}
	if p := signals[0].Geometry.Coordinates.([2]float64); p != [2]float64{-70.14, -15.49} {
		t.Fatalf("signal point = %v", p)
	}
	if fc.Metadata.OSMTimestamp != "2026-11-27T20:10:23Z" {
		t.Fatalf("timestamp = %q", fc.Metadata.OSMTimestamp)
	}
	if !strings.Contains(fc.Metadata.Attribution, "OpenStreetMap") || fc.Metadata.License != "ODbL-1.0" {
		t.Fatalf("missing attribution/license: %+v", fc.Metadata)
	}
}

func TestToGeoJSONMarshalsAsValidGeoJSON(t *testing.T) {
	out, err := json.Marshal(convertFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	var generic struct {
		Type     string `json:"type"`
		Features []struct {
			Geometry struct {
				Type string `json:"type"`
			} `json:"geometry"`
		} `json:"features"`
	}
	if err := json.Unmarshal(out, &generic); err != nil || generic.Type != "FeatureCollection" || len(generic.Features) != 3 {
		t.Fatalf("round trip failed: %v %s", err, out)
	}
}

func TestToGeoJSONRejectsInvalidInput(t *testing.T) {
	if _, err := ToGeoJSON(strings.NewReader("<html>busy</html>")); err == nil {
		t.Fatal("expected error for non-JSON Overpass response")
	}
	if _, err := ToGeoJSON(strings.NewReader(`{"elements": []}`)); err == nil {
		t.Fatal("expected error for empty element list")
	}
}
