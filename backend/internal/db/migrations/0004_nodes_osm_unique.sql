-- One OSM node cannot appear twice. Closes the debt noted in docs/modelo-datos.md.
CREATE UNIQUE INDEX IF NOT EXISTS nodes_osm_node_id_key ON nodes (osm_node_id);
