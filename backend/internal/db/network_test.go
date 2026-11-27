package db

import (
	"context"
	"strings"
	"testing"
)

const networkFixture = `{"type":"FeatureCollection","features":[
 {"type":"Feature","geometry":{"type":"LineString","coordinates":[[-70.13,-15.48],[-70.131,-15.481]]},"properties":{"osm_way_id":10,"highway":"trunk","name":"Avenida Independencia","ref":"PE-3S"}},
 {"type":"Feature","geometry":{"type":"LineString","coordinates":[[-70.14,-15.49],[-70.141,-15.491]]},"properties":{"osm_way_id":20,"highway":"primary"}},
 {"type":"Feature","geometry":{"type":"Point","coordinates":[-70.14,-15.49]},"properties":{"osm_node_id":7,"kind":"traffic_signals"}}
]}`

func TestParseRoadSegmentsKeepsOnlyLines(t *testing.T) {
	segs, err := parseRoadSegments(strings.NewReader(networkFixture))
	if err != nil {
		t.Fatal(err)
	}
	if len(segs) != 2 || segs[0].OSMWayID != 10 || segs[0].Ref != "PE-3S" || segs[1].Name != "" {
		t.Fatalf("segments = %+v", segs)
	}
}

func TestParseRoadSegmentsRejectsMissingWayID(t *testing.T) {
	bad := `{"type":"FeatureCollection","features":[{"type":"Feature","geometry":{"type":"LineString","coordinates":[[0,0],[1,1]]},"properties":{"highway":"primary"}}]}`
	if _, err := parseRoadSegments(strings.NewReader(bad)); err == nil {
		t.Fatal("expected error for LineString without osm_way_id")
	}
}

func TestLoadRoadNetworkUpsertsAndIsIdempotent(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	if err := Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `TRUNCATE road_segments`); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		n, err := LoadRoadNetwork(ctx, pool, strings.NewReader(networkFixture))
		if err != nil {
			t.Fatalf("load %d: %v", i+1, err)
		}
		if n != 2 {
			t.Fatalf("load %d upserted %d, want 2", i+1, n)
		}
	}
	var count, srid int
	var name, ref string
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM road_segments`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT ST_SRID(geom), name, ref FROM road_segments WHERE osm_way_id = 10`).Scan(&srid, &name, &ref); err != nil {
		t.Fatal(err)
	}
	if count != 2 || srid != 4326 || name != "Avenida Independencia" || ref != "PE-3S" {
		t.Fatalf("count=%d srid=%d name=%q ref=%q", count, srid, name, ref)
	}
}

func TestLoadRoadNetworkRejectsInvalidFileWithoutChanges(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	if err := Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadRoadNetwork(ctx, pool, strings.NewReader(networkFixture)); err != nil {
		t.Fatal(err)
	}
	var before int
	pool.QueryRow(ctx, `SELECT count(*) FROM road_segments`).Scan(&before)

	// Second feature has an invalid geometry: the whole load must roll back.
	bad := `{"type":"FeatureCollection","features":[
	 {"type":"Feature","geometry":{"type":"LineString","coordinates":[[-70.2,-15.5],[-70.21,-15.51]]},"properties":{"osm_way_id":30,"highway":"primary"}},
	 {"type":"Feature","geometry":{"type":"LineString","coordinates":[[-70.2]]},"properties":{"osm_way_id":31,"highway":"primary"}}]}`
	if _, err := LoadRoadNetwork(ctx, pool, strings.NewReader(bad)); err == nil {
		t.Fatal("expected error for invalid geometry")
	}
	var after int
	pool.QueryRow(ctx, `SELECT count(*) FROM road_segments`).Scan(&after)
	if after != before {
		t.Fatalf("road_segments changed from %d to %d after failed load", before, after)
	}
}

func TestSeedProvidesCriticalNodesAndCorridors(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	if err := Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	var nodes, signalized, corridors int
	if err := pool.QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE signalized) FROM nodes`).Scan(&nodes, &signalized); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM corridors`).Scan(&corridors); err != nil {
		t.Fatal(err)
	}
	if nodes != 8 || signalized != 6 || corridors != 3 {
		t.Fatalf("nodes=%d signalized=%d corridors=%d, want 8/6/3", nodes, signalized, corridors)
	}
	var osmID int64
	if err := pool.QueryRow(ctx, `SELECT osm_node_id FROM nodes WHERE id = 'N1'`).Scan(&osmID); err != nil || osmID != 257687118 {
		t.Fatalf("N1 osm_node_id = %d (%v), want 257687118", osmID, err)
	}
	rows, err := pool.Query(ctx, `SELECT node_id FROM corridor_nodes WHERE corridor_id = 'C1' ORDER BY position`)
	if err != nil {
		t.Fatal(err)
	}
	var order []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		order = append(order, id)
	}
	if strings.Join(order, ",") != "N2,N1,N3" {
		t.Fatalf("C1 order = %v, want N2,N1,N3", order)
	}
}
