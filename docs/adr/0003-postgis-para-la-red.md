# ADR 0003 — La red vial se guarda en PostGIS

- **Estado**: aceptado
- **Fecha**: 2026-11-27

## Contexto

Los nodos y corredores vienen de OpenStreetMap. La distancia entre nodos, el
recorte por caja y la validación de pertenencia son operaciones espaciales.

## Decisión

Guardar `network_nodes.geom` y `corridors.geom` como geometrías con SRID 4326, e
indexar con GIST. La distancia de corredor se materializa en `corridor_nodes`
para que el optimizador no la recalcule en cada corrida.

## Consecuencias

- La red es dato, no código: se puede corregir sin recompilar.
- El GeoJSON se carga una vez y la demo funciona sin internet.
- El SRID 4326 es geográfico; las distancias en metros se calculan con
  `ST_Length(geom::geography)` y no con `ST_Length(geom)`.
