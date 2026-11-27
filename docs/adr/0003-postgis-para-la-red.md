# ADR 0003 — La red vial se guarda en PostGIS

- **Estado**: aceptado
- **Fecha**: 2026-11-27

## Contexto

Los nodos y corredores vienen de OpenStreetMap. La distancia entre nodos, el
recorte por caja y la validación de pertenencia son operaciones espaciales.

## Decisión

Guardar la red como geometrías con SRID 4326 e indexarlas con GIST: los tramos
crudos de OSM en `road_segments.geom` y los cruces en `nodes.geom`. `corridors`
guarda solo identidad y nombre, porque su trazado se reconstruye uniendo los
tramos y duplicarlo sería una segunda fuente de verdad.

La distancia de cada tramo se materializa en `corridor_nodes.distance_m` para que
el optimizador no la recalcule en cada corrida; el acumulado se obtiene sumando
en orden de `position`.

## Consecuencias

- La red es dato, no código: se puede corregir sin recompilar.
- El GeoJSON se carga una vez y la demo funciona sin internet.
- El SRID 4326 es geográfico; las distancias en metros se calculan con
  `ST_Length(geom::geography)` y no con `ST_Length(geom)`.
