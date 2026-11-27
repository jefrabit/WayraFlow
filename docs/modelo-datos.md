# Modelo de datos

PostgreSQL 16 + PostGIS 3.4. La red vial se guarda como geometría porque las
distancias de corredor, la pertenencia de nodos y el recorte por caja son
consultas espaciales, no aritmética de listas.

## Tablas

| Tabla | Qué guarda | Por qué |
|---|---|---|
| `corridors` | Corredores coordinables con su geometría | Permite calcular longitud real y validar que los nodos caen sobre el trazado |
| `network_nodes` | Los 8 nodos críticos, su nodo OSM y su corredor | Fuente única de verdad de la topología |
| `corridor_nodes` | Orden y distancia acumulada de cada nodo en su corredor | El optimizador necesita la secuencia y la distancia para los desfases |
| `scenarios` | Demanda por corredor y mezcla vehicular | Cada escenario es un dato, no una constante en el código |
| `runs` | Cada corrida con su semilla y configuración | Reproducibilidad: misma semilla, mismo resultado |
| `plans` | El plan validado en JSON | Se conserva el contrato exacto que se exportó |
| `kpis` | Métrica, línea base y candidato | Comparación sin recalcular |

## Decisiones

- **`jsonb` para demanda y mezcla**: la forma cambia si se agrega un corredor o
  una clase vehicular. Normalizarlo en dos tablas más agrega joins sin beneficio
  para un volumen de tres escenarios.
- **`jsonb` para el plan**: el contrato `wayraflow.plan/v1` es la interfaz
  pública. Guardarlo tal cual se exporta evita divergencias entre lo persistido y
  lo descargado.
- **`numeric` para KPIs**: se comparan y se muestran; el punto flotante agrega
  ruido innecesario en la base.
- **Sin `updated_at`**: las corridas y los planes son inmutables. Si un plan
  cambia, es otra corrida.

## Invariantes

Los que la base puede sostener sola:

- `network_nodes.osm_node` es único.
- `corridor_nodes` no repite posición dentro de un corredor.
- `distance_m` no es negativo.
- `strategy` y `status` están en dominios cerrados.

Los que valida la aplicación, porque son del dominio y no de la tabla:

- `Σ(green + amber + allRed) = cycle_s` por nodo.
- `0 ≤ offset < cycle_s` por corredor.
- La mezcla vehicular suma 1.

## Índices

- GIST sobre `network_nodes.geom` y `corridors.geom` para recortes por caja.
- B-tree sobre `runs.created_at` para listar corridas recientes.
- Clave primaria compuesta en `kpis` para lectura por corrida.

## Migraciones

Los scripts de `db/` se aplican en orden con `psql -f` y solo cuando el volumen
está vacío. Los cambios posteriores van en archivos numerados de forma
incremental. No se edita un script ya aplicado.
