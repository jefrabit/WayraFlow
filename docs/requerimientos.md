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

## 2. Requerimientos funcionales

| ID | Requerimiento | Criterio de aceptación | README | Módulo |
|---|---|---|---|---|
| FR-01 | Cargar la red vial de Juliaca sin internet | Con la red desconectada, el mapa muestra los 8 nodos y los 3 corredores, con atribución ODbL visible | 3.1, 4.2 | `db/init`, `features/map` |
| FR-02 | Representar el plan actual de ciclo fijo | Para cualquier escenario, el ciclo fijo produce 90 s y verdes iguales en todos los nodos | 4.6 | `internal/optim` |
| FR-03 | Calcular el ciclo de Webster | Con `L=10` y `Y` del escenario, el ciclo sale de `(1,5L+5)/(1−Y)` y se recorta a 40–120 s | 4.6 | `internal/optim` |
| FR-04 | Optimizar verdes y desfases con GA | El GA respeta verdes mínimos de 7 s, `Σg + L = C` y `0 ≤ θ < C`; termina dentro de 3 s | 4.5 | `internal/optim` |
| FR-05 | Simular la ciudad a paso de 1 s | El horizonte de 3 600 s corre completo y reporta cola máxima y spillback | 4.3 | `internal/sim` |
| FR-06 | Calcular KPIs por nodo y por ciudad | Espera, cola, paradas y CO₂ coinciden entre corridas con la misma semilla | 7 | `internal/sim` |
| FR-07 | Exportar el plan en `wayraflow.plan/v1` | El JSON descargado pasa la validación de la sección 8 y falla si `Σ(g+a+r) ≠ C` | 8 | `internal/plan` |
| FR-08 | Comparar estrategias en una vista | El tablero muestra ciclo fijo, Webster y GA con la mejora porcentual | 7, 10 | `features/kpis` |
| FR-09 | Elegir entre los 3 escenarios | Cambiar de escenario recalcula la corrida y actualiza los KPIs | 6 | `features/plan` |
| FR-10 | Micro-simular un nodo con IDM | La vista de nodo anima los 3 tipos de vehículo a 60 fps con el semáforo como vehículo virtual | 4.4 | `features/node-sim` |

## 3. Requerimientos no funcionales

| ID | Requerimiento | Criterio de aceptación |
|---|---|---|
| NFR-01 | Rendimiento del optimizador | Una corrida completa responde en menos de 5 s en hardware de laptop; el GA se corta solo a los 3 s |
| NFR-02 | Reproducibilidad | La misma semilla y el mismo escenario producen el mismo plan y los mismos KPIs |
| NFR-03 | Operación local | `docker compose up` deja base, API y web disponibles sin pasos manuales |
| NFR-04 | Integridad del contrato | Ningún plan inválido se persiste ni se descarga |
| NFR-05 | Privacidad | No se envía telemetría ni datos a terceros; no hay claves de API |
| NFR-06 | Accesibilidad | Contraste AA, navegación por teclado y textos alternativos en el mapa |
| NFR-07 | Licenciamiento | Atribución ODbL visible; código con licencia MIT |
| NFR-08 | Trazabilidad de supuestos | Todo valor sin fuente se marca `Supuesto` en la interfaz y en el JSON |

## 4. Trazabilidad de pruebas

| Requerimiento | Prueba |
|---|---|
| FR-02, FR-03, FR-04 | `internal/optim/*_test.go` (tabla de casos por escenario) |
| FR-05, FR-06 | `internal/sim/*_test.go` (semilla fija, comparación de agregados) |
| FR-07 | `internal/plan/*_test.go` (plan válido, suma de fases, offset fuera de rango) |
| FR-08, FR-09 | `frontend` pruebas de componente del tablero |
| FR-10 | Prueba manual con cronómetro y `requestAnimationFrame` |
| NFR-01 | `go test -bench` y medición en la demo |
| NFR-02 | Prueba de determinismo con semilla repetida |

## 5. Criterios de evaluación del concurso

La rúbrica asigna 15 % a funcionalidad, 15 % a calidad técnica y 15 % a
repositorio. Este documento y `docs/arquitectura.md` existen para que esos puntos
se puedan verificar sin leer todo el código: cada requerimiento apunta a un
módulo concreto y a una prueba concreta.

| Criterio | Evidencia en el repositorio |
|---|---|
| Repositorio | Historial de commits durante la ventana, README, estructura por capas |
| Funcionalidad | FR-01 a FR-10 con criterios verificables |
| Calidad técnica | Regla de dependencias, pruebas, lint, CI, contrato versionado |
| Uso efectivo de IA | `docs/ia.md` con el registro de decisiones asistidas |
| Proyecto publicado | `docker-compose.yml` y despliegue estático de la SPA |
