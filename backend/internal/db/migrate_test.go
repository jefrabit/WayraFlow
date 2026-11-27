package db

import (
	"context"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMigrationFilesAreOrderedByVersion(t *testing.T) {
	fsys := fstest.MapFS{
		"migrations/0002_b.sql": {Data: []byte("SELECT 2;")},
		"migrations/0001_a.sql": {Data: []byte("SELECT 1;")},
		"migrations/README.md":  {Data: []byte("ignored")},
	}
	got, err := loadMigrations(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].version != 1 || got[1].version != 2 {
		t.Fatalf("migrations = %+v, want versions [1 2]", got)
	}
}

func TestMigrationFilesRejectInvalidName(t *testing.T) {
	fsys := fstest.MapFS{"migrations/init.sql": {Data: []byte("SELECT 1;")}}
	if _, err := loadMigrations(fsys); err == nil {
		t.Fatal("expected error for migration without numeric version prefix")
	}
}

func TestMigrationFilesRejectDuplicateVersion(t *testing.T) {
	fsys := fstest.MapFS{
		"migrations/0001_a.sql": {Data: []byte("SELECT 1;")},
		"migrations/1_b.sql":    {Data: []byte("SELECT 1;")},
	}
	_, err := loadMigrations(fsys)
	if err == nil || !strings.Contains(err.Error(), "duplicate migration version 1") {
		t.Fatalf("err = %v, want duplicate migration version 1", err)
	}
}

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping PostGIS integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestMigrateCreatesSchemaAndIsIdempotent(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		if err := Migrate(ctx, pool); err != nil {
			t.Fatalf("Migrate run %d: %v", i+1, err)
		}
	}

	var applied int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM schema_migrations WHERE version = 1`).Scan(&applied); err != nil {
		t.Fatal(err)
	}
	if applied != 1 {
		t.Fatalf("version 1 recorded %d times, want 1", applied)
	}

	for _, table := range []string{"road_segments", "nodes", "corridors", "corridor_nodes", "scenarios", "plans", "runs"} {
		var exists bool
		if err := pool.QueryRow(ctx, `SELECT to_regclass('public.' || $1) IS NOT NULL`, table).Scan(&exists); err != nil {
			t.Fatal(err)
		}
		if !exists {
			t.Errorf("table %s missing", table)
		}
	}

	var srid int
	if err := pool.QueryRow(ctx, `SELECT srid FROM geometry_columns WHERE f_table_name = 'nodes' AND f_geometry_column = 'geom'`).Scan(&srid); err != nil {
		t.Fatal(err)
	}
	if srid != 4326 {
		t.Fatalf("nodes.geom SRID = %d, want 4326", srid)
	}
}

func TestPlansRejectUnknownStrategy(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	if err := Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `INSERT INTO scenarios (id, name, demand) VALUES ('t', 't', '{}')`); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO plans (scenario_id, strategy, plan) VALUES ('t', 'random', '{}')`); err == nil {
		t.Fatal("expected CHECK violation for unknown strategy")
	}
}
