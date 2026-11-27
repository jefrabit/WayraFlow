-- Critical nodes and coordinable corridors from README sections 3.3 and 3.4.
-- Coordinates are the OSM junction node shared by the named ways (OSM 2026-11-27).
INSERT INTO nodes (id, name, osm_node_id, signalized, geom) VALUES
 ('N1', 'Av. Circunvalación Norte × Av. Independencia (PE-3S)',                        257687118,  true,  ST_SetSRID(ST_MakePoint(-70.13584, -15.48129), 4326)),
 ('N2', 'Av. Circunvalación Norte (PE-34A) × Av. Ferrocarril',                          1282294742, true,  ST_SetSRID(ST_MakePoint(-70.14279, -15.48646), 4326)),
 ('N3', 'Av. Circunvalación Norte × Av. Circunvalación Este × Av. Triunfo',             1281999136, true,  ST_SetSRID(ST_MakePoint(-70.12695, -15.48068), 4326)),
 ('N4', 'Av. Circunvalación Este (PE-3S) × Av. Huancané (PE-34H)',                      257687482,  true,  ST_SetSRID(ST_MakePoint(-70.12446, -15.48482), 4326)),
 ('N5', 'Av. Circunvalación Este × Jr. Huancané',                                       443135956,  false, ST_SetSRID(ST_MakePoint(-70.12342, -15.48803), 4326)),
 ('N6', 'Autopista Mártires del 4 de Noviembre (PE-3S) × Av. Tacna × Jr. M. Núñez Butrón', 2420987313, true,  ST_SetSRID(ST_MakePoint(-70.12799, -15.50306), 4326)),
 ('N7', 'Autopista Mártires del 4 de Noviembre × Av. Américas × Jr. J. D. Choquehuanca', 3226623702, true,  ST_SetSRID(ST_MakePoint(-70.12502, -15.51141), 4326)),
 ('N8', 'Av. Circunvalación Oeste (PE-34A) × Héroes del Pacífico',                      255459726,  false, ST_SetSRID(ST_MakePoint(-70.14661, -15.49811), 4326));

INSERT INTO corridors (id, name) VALUES
 ('C1', 'Circunvalación Norte'),
 ('C2', 'Circunvalación Este'),
 ('C3', 'Eje sur PE-3S');

-- distance_m: straight-line distance from the previous node (README 3.4).
INSERT INTO corridor_nodes (corridor_id, node_id, position, distance_m) VALUES
 ('C1', 'N2', 0, NULL), ('C1', 'N1', 1, 941), ('C1', 'N3', 2, 955),
 ('C2', 'N3', 0, NULL), ('C2', 'N4', 1, 532), ('C2', 'N5', 2, 374),
 ('C3', 'N6', 0, NULL), ('C3', 'N7', 1, 984);

CREATE INDEX corridor_nodes_node_idx ON corridor_nodes (node_id);
