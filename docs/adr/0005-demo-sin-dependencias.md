# ADR 0005 — La demo no depende de servicios externos

- **Estado**: aceptado
- **Fecha**: 2026-11-27

## Contexto

La sustentación ocurre en una sala con red inestable. Una demo que necesita
teselas, geocodificación o una API con clave se cae en el peor momento.

## Decisión

Versionar el GeoJSON, servir el mapa base desde el propio bundle y no usar
ninguna clave de API. `docker compose up` debe dejar todo funcionando sin
internet.

## Consecuencias

- El repositorio pesa más, pero la demo es reproducible.
- La atribución ODbL se mantiene visible porque la licencia lo exige.
- Si se agrega un servicio externo, entra con caché local y bandera de respaldo.
