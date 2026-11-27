-- Initial schema: road network, critical nodes, corridors and optimization history.
CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE road_segments (
    id          BIGSERIAL PRIMARY KEY,
    osm_way_id  BIGINT NOT NULL UNIQUE,
    name        TEXT,
    ref         TEXT,
    highway     TEXT NOT NULL,
    geom        geometry(LineString, 4326) NOT NULL
);
CREATE INDEX road_segments_geom_idx ON road_segments USING GIST (geom);

CREATE TABLE nodes (
    id           TEXT PRIMARY KEY,               -- e.g. N1
    name         TEXT NOT NULL,
    osm_node_id  BIGINT NOT NULL,
    signalized   BOOLEAN NOT NULL,
    geom         geometry(Point, 4326) NOT NULL
);
CREATE INDEX nodes_geom_idx ON nodes USING GIST (geom);

CREATE TABLE corridors (
    id    TEXT PRIMARY KEY,                      -- e.g. C1
    name  TEXT NOT NULL
);

CREATE TABLE corridor_nodes (
    corridor_id  TEXT NOT NULL REFERENCES corridors (id) ON DELETE CASCADE,
    node_id      TEXT NOT NULL REFERENCES nodes (id) ON DELETE RESTRICT,
    position     INT  NOT NULL CHECK (position >= 0),
    distance_m   REAL CHECK (distance_m >= 0),   -- distance from previous node
    PRIMARY KEY (corridor_id, node_id),
    UNIQUE (corridor_id, position)
);

CREATE TABLE scenarios (
    id           TEXT PRIMARY KEY,               -- e.g. base, feria
    name         TEXT NOT NULL,
    description  TEXT,
    demand       JSONB NOT NULL                  -- veh/h and mix per corridor
);

CREATE TABLE plans (
    id           BIGSERIAL PRIMARY KEY,
    scenario_id  TEXT NOT NULL REFERENCES scenarios (id) ON DELETE CASCADE,
    strategy     TEXT NOT NULL CHECK (strategy IN ('fixed', 'webster', 'ga')),
    plan         JSONB NOT NULL,                 -- wayraflow.plan/v1 document
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE runs (
    id           BIGSERIAL PRIMARY KEY,
    plan_id      BIGINT NOT NULL REFERENCES plans (id) ON DELETE CASCADE,
    kpis         JSONB NOT NULL,
    duration_ms  INT CHECK (duration_ms >= 0),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
