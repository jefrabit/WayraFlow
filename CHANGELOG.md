# Changelog

Formato basado en [Keep a Changelog](https://keepachangelog.com/es-ES/1.1.0/).
El proyecto usa versionado semántico.

## [0.1.0] - 2026-11-27

Primera versión funcional presentada a LA COPA SISTEMAS.

### Agregado

- Motor de optimización semafórica con tres estrategias comparables: ciclo fijo,
  Webster y algoritmo genético con selección por simulador.
- Simulador mesoscópico determinista de 3600 s con propagación de pelotones,
  colas por nodo y KPIs de espera, paradas, cola máxima y CO₂.
- Catálogo de la red de Juliaca: 8 nodos, 3 corredores y 3 escenarios
  (base, hora punta regional, día de feria) disponible sin base de datos.
- API HTTP/JSON con `GET /health`, `GET /api/v1/network` (GeoJSON),
  `GET /api/v1/scenarios`, `POST /api/v1/plans`, `GET /api/v1/plans`,
  `GET /api/v1/plans/{id}` y `GET /api/v1/runs/{id}`.
- Persistencia en PostgreSQL/PostGIS con migraciones embebidas y respaldo en
  memoria para demos y tests.
- Tablero Angular con mapa SVG de la red, comparación de KPIs por estrategia y
  simulación IDM de un nodo sobre canvas.
- Docker Compose con PostGIS, backend y frontend servido por nginx.
- Integración continua con formato, `go vet`, tests con `-race`, build de Go,
  build de Angular y build de ambas imágenes.
- Documentación de requerimientos, diagramas C4 y de secuencia, modelo de datos
  y contrato de la API.

### Notas

- Las demandas y los tiempos de verde son supuestos declarados: no existen
  aforos vehiculares públicos de Juliaca.
- El escenario de hora punta no mejora respecto del ciclo fijo; se reporta el
  resultado real en lugar de ocultarlo.
