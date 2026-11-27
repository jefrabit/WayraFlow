# Modelo de datos

PostgreSQL 16 + PostGIS 3.4. El esquema vive en
`backend/internal/db/migrations/` y se aplica solo, al arrancar la API.

Diagrama entidad-relación: [diagramas.md](diagramas.md#4-modelo-de-datos).

## Cómo se aplican las migraciones

- Los archivos se llaman `NNNN_descripcion.sql` y se embeben en el binario con
  `//go:embed`, así que el contenedor no necesita el directorio de migraciones.
- `db.Migrate` toma un advisory lock (`72617119`) antes de aplicar nada: dos
  réplicas de la API no pueden migrar a la vez.
- El avance se registra en `schema_migrations (version, name, applied_at)`.
- Cada migración corre dentro de una transacción: si falla, no deja el esquema a
  medias.
- Una migración ya aplicada no se edita. El cambio va en la siguiente.

La `0002` siembra los 8 nodos, los 3 corredores y el orden con sus distancias de
tramo. Los tramos de la red se cargan al arrancar desde `NETWORK_FILE`.

## Tablas

| Tabla | Columnas relevantes | Para qué |
|---|---|---|
| `road_segments` | `id`, `osm_way_id` (único), `name`, `ref`, `highway`, `geom LineString(4326)` | La red cruda que devuelve Overpass: 566 vías |
| `nodes` | `id` (`N1`…`N8`), `name`, `osm_node_id`, `signalized`, `geom Point(4326)` | Los cruces críticos del README 3.3 |
| `corridors` | `id` (`C1`…`C3`), `name` | Los corredores coordinables |
| `corridor_nodes` | `corridor_id`, `node_id`, `position`, `distance_m` | Orden y distancia de tramo dentro del corredor |
| `scenarios` | `id`, `name`, `description`, `demand jsonb` | Los tres escenarios del README 6 |
| `plans` | `id`, `scenario_id`, `strategy`, `plan jsonb`, `created_at` | El plan generado, tal como se exporta |
| `runs` | `id`, `plan_id`, `kpis jsonb`, `duration_ms`, `created_at` | La evaluación de un plan |

## Decisiones

- **`road_segments` separado de `corridors`**: OSM entrega vías sueltas; un
  corredor es una agrupación lógica de tres o cuatro de ellas. Guardar la
  geometría una sola vez evita duplicarla y desincronizarla.
- **`corridors` sin geometría**: si hace falta dibujar el corredor, se unen sus
  tramos. La fuente de verdad geométrica es `road_segments`.
- **`demand` como `jsonb`**: incluye demanda por corredor y mezcla vehicular. Son
  tres escenarios; normalizarlos en dos tablas más agrega joins sin beneficio.
- **`plans` y `runs` separados**: un mismo plan puede evaluarse en varias
  corridas (otra semilla, otro horizonte). La corrida es la medición; el plan es
  el artefacto.
- **`distance_m` es distancia de tramo**, no acumulada desde el inicio del
  corredor. El acumulado se obtiene sumando en orden de `position`, que es lo que
  necesita el cálculo de desfases.

## Invariantes

Los sostiene la base:

- `road_segments.osm_way_id` es único.
- `(corridor_id, position)` es único: no hay dos nodos en el mismo lugar del
  corredor.
- `distance_m >= 0`.
- `strategy` está en `fixed`, `webster`, `ga`.

Los valida la aplicación, porque son del dominio:

- `Σ(green + amber + allRed) = cycle_s` por nodo.
- `0 ≤ offset < cycle_s` por corredor.
- La mezcla vehicular suma 1.

## Deuda técnica anotada

- `nodes.osm_node_id` no tiene restricción de unicidad. Un nodo de OSM no debería
  repetirse; conviene un índice único en la migración `0002`.
- `plans` no guarda si el plan pasó la validación. Hoy se valida antes de
  insertar; si se quiere auditar después, falta una columna `valid`.
- No hay tabla de usuarios ni roles: está fuera de alcance del MVP.

## Índices

- GIST sobre `road_segments.geom` y `nodes.geom` para recortes por caja.
- Clave primaria compuesta en `corridor_nodes` y en el resto de tablas por `id`.
- `plans.scenario_id` y `runs.plan_id` para las lecturas por corrida.
