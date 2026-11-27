# Requerimientos

Cada requerimiento tiene criterio de aceptación verificable y trazabilidad al
README, al módulo que lo implementa y a la prueba que lo cubre. Si algo no se
puede verificar, no es un requerimiento: es un deseo.

## 1. Alcance

**Dentro**: red vial de Juliaca, 8 nodos críticos, 3 corredores, 3 escenarios,
ciclo fijo vs. Webster vs. algoritmo genético, KPIs de espera/colas/paradas/CO₂,
exportación de plan y tablero web.

**Fuera**: controladores reales, sensores, usuarios y roles, adaptativo en
tiempo real, peatones y ciclistas.

## 1.1 Actores

| Actor | Qué necesita | Dónde lo resuelve |
|---|---|---|
| Gestor municipal de tránsito | Elegir escenario, calcular un plan y llevarse los tiempos | Tablero y descarga del JSON |
| Evaluador técnico | Revisar supuestos, comparar estrategias y auditar KPIs | Tablero y `docs/` |
| Equipo de desarrollo | Levantar el stack y verificar que nada se rompió | `docker compose`, migraciones, CI |

Casos de uso principales: calcular un plan para un escenario, comparar las tres
estrategias, descargar el plan validado y micro-simular un nodo.

## 2. Requerimientos funcionales

Prioridad según MoSCoW: **D** debe, **S** debería, **P** podría.

| ID | Requerimiento | Prioridad | Criterio de aceptación | README | Módulo |
|---|---|---|---|---|---|
| FR-01 | Cargar la red vial de Juliaca sin internet | D | Con la red desconectada, el mapa muestra los 8 nodos y los 3 corredores, con atribución ODbL visible | 3.1, 4.2 | `internal/db`, `features/map` |
| FR-02 | Representar el plan actual de ciclo fijo | D | Para cualquier escenario, el ciclo fijo produce 90 s y verdes iguales en todos los nodos | 4.6 | `internal/optim` |
| FR-03 | Calcular el ciclo de Webster | D | Con `L=10` y `Y` del escenario, el ciclo sale de `(1,5L+5)/(1−Y)` y se recorta a 40–120 s | 4.6 | `internal/optim` |
| FR-04 | Optimizar verdes y desfases con GA | D | El GA respeta verdes mínimos de 7 s, `Σg + L = C` y `0 ≤ θ < C`; termina dentro de 3 s | 4.5 | `internal/optim` |
| FR-05 | Simular la ciudad a paso de 1 s | D | El horizonte de 3 600 s corre completo y reporta cola máxima y spillback | 4.3 | `internal/sim` |
| FR-06 | Calcular KPIs por nodo y por ciudad | D | Espera, cola, paradas y CO₂ coinciden entre corridas con la misma semilla | 7 | `internal/sim` |
| FR-07 | Exportar el plan en `wayraflow.plan/v1` | D | El JSON descargado pasa la validación de la sección 8 y falla si `Σ(g+a+r) ≠ C` | 8 | `internal/plan` |
| FR-08 | Comparar estrategias en una vista | S | El tablero muestra ciclo fijo, Webster y GA con la mejora porcentual | 7, 10 | `features/kpis` |
| FR-09 | Elegir entre los 3 escenarios | D | Cambiar de escenario recalcula la corrida y actualiza los KPIs | 6 | `features/plan` |
| FR-10 | Micro-simular un nodo con IDM | S | La vista de nodo anima los 3 tipos de vehículo a 60 fps con el semáforo como vehículo virtual | 4.4 | `features/node-sim` |
| FR-11 | Exponer el estado del servicio | D | `GET /health` responde 200 con `{"status":"ok"}` y la API no arranca si la base no responde | — | `internal/httpapi` |
| FR-12 | Cargar el GeoJSON a PostGIS | D | Tras la carga, `nodes` tiene 8 filas y `corridor_nodes` respeta el orden y las distancias del README | 3.1, 3.4 | `internal/db`, `cmd/osmextract` |
| FR-13 | Consultar corridas anteriores | S | Se puede listar y abrir una corrida pasada con sus KPIs y su plan | — | `internal/store` |
| FR-14 | Exportar los KPIs a CSV | P | El tablero descarga un CSV con una fila por nodo y las tres estrategias | 7 | `features/kpis` |

**Estado de implementación**: hoy existen `internal/db` (esquema, migraciones
embebidas, semilla de nodos y corredores, y carga de la red desde GeoJSON),
`internal/httpapi` (`GET /health`), `internal/osm` y `cmd/osmextract`.
Los módulos `internal/optim`, `internal/sim`, `internal/plan`, `internal/store` y
`features/*` son el destino del MVP, no código existente. Este documento describe
el sistema completo, no solo lo ya escrito.

## 3. Requerimientos no funcionales

| ID | Requerimiento | Criterio de aceptación |
|---|---|---|
| NFR-01 | Rendimiento del optimizador | Una corrida completa responde en menos de 5 s en hardware de laptop; el GA se corta solo a los 3 s |
| NFR-02 | Reproducibilidad | La misma semilla y el mismo escenario producen el mismo plan y los mismos KPIs |
| NFR-03 | Operación local | `docker compose up` deja base y API disponibles sin pasos manuales; el frontend se suma con su workspace |
| NFR-04 | Integridad del contrato | Ningún plan inválido se persiste ni se descarga |
| NFR-05 | Privacidad | No se envía telemetría ni datos a terceros; no hay claves de API |
| NFR-06 | Accesibilidad | Contraste AA, navegación por teclado y textos alternativos en el mapa |
| NFR-07 | Licenciamiento | Atribución ODbL visible; código con licencia MIT |
| NFR-08 | Trazabilidad de supuestos | Todo valor sin fuente se marca `Supuesto` en la interfaz y en el JSON |
| NFR-09 | Seguridad de credenciales | Las credenciales de desarrollo viven en `.env`, nunca en el repositorio; `.env.example` documenta las variables |
| NFR-10 | Mantenibilidad | `domain` no importa HTTP ni SQL; cobertura de pruebas del motor ≥ 80 % en las ramas críticas |
| NFR-11 | Portabilidad | Backend en imagen multi-stage sin dependencias del host; la base se levanta con el compose |
| NFR-12 | Observabilidad | Logs estructurados con `slog`, una línea por request y duración por corrida |
| NFR-13 | Compatibilidad | Navegadores con WebGL2 (MapLibre) y Canvas 2D; sin plugins |

## 4. Trazabilidad

| Requerimiento | Módulo | Prueba | Estado |
|---|---|---|---|
| FR-01 | `frontend/src/app/features/map` | Verificación manual con la red real servida por la API | Implementado |
| FR-02, FR-03, FR-04 | `internal/optim` | `optim_test.go`: tabla por escenario y contrato del plan | Implementado |
| FR-05, FR-06 | `internal/sim` | `sim_test.go`: determinismo y comparación de agregados | Implementado |
| FR-07 | `internal/plan` | `validate_test.go`: plan válido, suma de fases, offset fuera de rango | Implementado |
| FR-08, FR-09 | `frontend/src/app/features/dashboard` | Verificación manual contra la API; sin prueba de componente | Implementado |
| FR-10 | `frontend/src/app/features/node-sim` | Verificación manual con `requestAnimationFrame` | Implementado |
| FR-11 | `internal/httpapi` | `router_test.go` | Implementado |
| FR-12 | `internal/db`, `cmd/osmextract` | `migrate_test.go`, `network_test.go`, `geojson_test.go` | Implementado |
| FR-13 | `internal/store`, `internal/httpapi` | `router_test.go`: listado y lectura de corridas | Implementado |
| FR-14 | — | — | No implementado |
| NFR-01 | `cmd/bench` | `go run ./cmd/bench` con la medición de la demo | Implementado |
| NFR-02 | `internal/optim`, `internal/sim` | `sim_test.go` repite la corrida con la misma semilla | Implementado |
| NFR-09 | `docker-compose.yml` | Revisión: sin secretos versionados | Implementado |
| NFR-12 | `cmd/api` | Log de request en `main.go` | Implementado |

## 5. Criterios de evaluación del concurso

La rúbrica asigna 15 % a funcionalidad, 15 % a calidad técnica y 15 % a
repositorio. Este documento y `docs/arquitectura.md` existen para que esos puntos
se puedan verificar sin leer todo el código: cada requerimiento apunta a un
módulo concreto y a una prueba concreta.

| Criterio | Evidencia en el repositorio |
|---|---|
| Repositorio | Historial de commits durante la ventana, README, estructura por capas |
| Funcionalidad | FR-01 a FR-14 con criterios verificables y estado explícito |
| Calidad técnica | Regla de dependencias, pruebas, CI, contrato versionado |
| Uso efectivo de IA | `docs/ia.md` con el registro de decisiones asistidas |
| Proyecto publicado | `docker-compose.yml` y despliegue de la SPA |
