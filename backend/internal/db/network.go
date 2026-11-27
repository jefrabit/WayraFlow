package db

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RoadSegment is one OSM way read from data/juliaca.geojson.
type RoadSegment struct {
	OSMWayID int64
	Name     string
	Ref      string
	Highway  string
	Geometry json.RawMessage // GeoJSON LineString
}

type networkFile struct {
	Features []struct {
		Geometry   json.RawMessage `json:"geometry"`
		Properties struct {
			OSMWayID int64  `json:"osm_way_id"`
			Name     string `json:"name"`
			Ref      string `json:"ref"`
			Highway  string `json:"highway"`
		} `json:"properties"`
	} `json:"features"`
}

// parseRoadSegments returns the LineString features of the network file.
func parseRoadSegments(r io.Reader) ([]RoadSegment, error) {
	var f networkFile
	if err := json.NewDecoder(r).Decode(&f); err != nil {
		return nil, fmt.Errorf("decode network file: %w", err)
	}
	var segs []RoadSegment
	for i, feat := range f.Features {
		var g struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(feat.Geometry, &g); err != nil {
			return nil, fmt.Errorf("feature %d: %w", i, err)
		}
		if g.Type != "LineString" {
			continue
		}
		p := feat.Properties
		if p.OSMWayID == 0 || p.Highway == "" {
			return nil, fmt.Errorf("feature %d: LineString needs osm_way_id and highway", i)
		}
		segs = append(segs, RoadSegment{OSMWayID: p.OSMWayID, Name: p.Name, Ref: p.Ref, Highway: p.Highway, Geometry: feat.Geometry})
	}
	return segs, nil
}

// LoadRoadNetwork upserts every road segment of the GeoJSON file in a single
// transaction: either the whole file is loaded or nothing changes.
func LoadRoadNetwork(ctx context.Context, pool *pgxpool.Pool, r io.Reader) (int, error) {
	segs, err := parseRoadSegments(r)
	if err != nil {
		return 0, err
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	const upsert = `
		INSERT INTO road_segments (osm_way_id, name, ref, highway, geom)
		VALUES ($1, NULLIF($2, ''), NULLIF($3, ''), $4, ST_SetSRID(ST_GeomFromGeoJSON($5), 4326))
		ON CONFLICT (osm_way_id) DO UPDATE
		SET name = EXCLUDED.name, ref = EXCLUDED.ref, highway = EXCLUDED.highway, geom = EXCLUDED.geom`
	for _, s := range segs {
		if _, err := tx.Exec(ctx, upsert, s.OSMWayID, s.Name, s.Ref, s.Highway, string(s.Geometry)); err != nil {
			return 0, fmt.Errorf("upsert way %d: %w", s.OSMWayID, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return len(segs), nil
}
