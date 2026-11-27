# Arquitectura de WayraFlow

Este documento describe cómo se organiza el sistema y por qué. El README fija el
problema, el modelo de tránsito y los supuestos; acá se explica la estructura del
software que los implementa.

## 1. Contexto

WayraFlow lo usan dos perfiles:

- **Gestor municipal de tránsito**: elige un escenario, calcula un plan y descarga
  los tiempos para llevarlos al controlador.
- **Evaluador técnico**: revisa supuestos, compara estrategias y audita los KPIs.

El sistema no habla con controladores semafóricos ni con sensores. La única
dependencia externa es OpenStreetMap, y se resuelve **una sola vez**: el GeoJSON
se versiona en el repositorio y se carga en PostGIS al iniciar. La demo corre sin
internet.

```
Gestor / Evaluador
        │  HTTPS
        ▼
┌───────────────────┐        ┌──────────────────┐
│  Angular (SPA)    │  JSON  │   API Go         │
│  mapa + canvas    │───────▶│  cálculo y plan  │
└───────────────────┘        └────────┬─────────┘
                                      │ SQL
                                      ▼
                             ┌──────────────────┐
                             │ PostgreSQL+PostGIS│
                             └──────────────────┘
```

## 2. Contenedores

| Contenedor | Responsabilidad | No hace |
|---|---|---|
| `web` (Angular + nginx) — *propuesto* | Mapa, vista de nodo con IDM en Canvas, tablero de KPIs, descarga del plan | No calcula el plan ni guarda estado |
| `backend` (Go) | Simulación mesoscópica, algoritmo genético, Webster, emisiones, KPIs, validación y exportación | No renderiza ni conoce el DOM |
| `db` (PostGIS) | Red vial como geometrías, escenarios, corridas, planes y KPIs | No ejecuta lógica de negocio |

El reparto es deliberado: **Go calcula y persiste, Angular visualiza**. La única
excepción es la micro-simulación IDM de la vista de nodo, que corre en el
navegador porque necesita 60 fps y no requiere persistencia.

## 3. Componentes del backend

Implementado hoy:

```
cmd/api                  arranque y configuración
cmd/osmextract           extracción de la red desde Overpass a GeoJSON
internal/db              pool pgx y migraciones embebidas
internal/db/migrations   esquema versionado (0001_init.sql)
internal/httpapi         router y escritura de respuestas JSON
internal/osm             parseo de Overpass y generación de GeoJSON
```

Propuesto, todavía no existe:

```
internal/domain          modelos y reglas puras (sin HTTP, sin SQL)
internal/sim             modelo mesoscópico de ciudad
internal/optim           algoritmo genético, Webster, reparación de verdes
internal/emission        factores de consumo y CO₂
internal/plan            contrato wayraflow.plan/v1 y validación
internal/store           repositorios sobre pgx
```

Regla de dependencias, de adentro hacia afuera:

1. `domain` no importa nada del proyecto.
2. `sim`, `optim`, `emission` y `plan` dependen solo de `domain`.
3. `store` implementa interfaces declaradas por los servicios; nunca al revés.
4. `httpapi` traduce HTTP a llamadas de servicio y no contiene reglas.

Esto permite probar el motor completo sin base de datos ni servidor.

## 4. Componentes del frontend

```
src/app/core      cliente HTTP tipado, manejo de errores, estado de sesión
src/app/features  map, node-sim, plan, kpis
src/app/shared    componentes de presentación, formato, tokens
```

- `features/node-sim` es el único módulo con lógica de simulación: IDM en Canvas.
- `core` es el único que conoce las URLs de la API.
- Los componentes de `shared` no importan servicios.

## 5. Flujo de una corrida

1. El usuario elige escenario y estrategia en la SPA.
2. `POST /api/v1/plans` con `scenarioId`, `strategy` y `seed`.
3. La API carga la red desde PostGIS (nodos, corredores, distancias de tramo).
4. Simula el plan de ciclo fijo y guarda la línea base.
5. Ejecuta Webster como referencia y el algoritmo genético como candidato.
6. Valida el plan contra el contrato `wayraflow.plan/v1`.
7. Persiste el plan y su corrida con los KPIs; devuelve ambos identificadores.
8. La SPA pinta el mapa, la tabla y los KPIs; el botón de descarga sirve el JSON
   ya validado.

La corrida es **síncrona** en el MVP: el presupuesto del optimizador es de 3 s y
un `POST` que tarda eso es aceptable para una demo. Si el presupuesto crece, el
contrato ya prevé `202 Accepted` con `runId` y consulta posterior.

## 6. Decisiones que sostienen el diseño

| Decisión | Motivo | ADR |
|---|---|---|
| Angular + Go + PostGIS | El cálculo pesa más que la interfaz; PostGIS evita reimplementar geometría | 0001 |
| Motor en Go, no en el navegador | Simulación de 3 600 s × 8 nodos y GA de 150 generaciones | 0002 |
| Red en PostGIS con SRID 4326 | Consultas espaciales y fuente única de verdad | 0003 |
| Contrato JSON propio y versionado | Exportación estable y testeable | 0004 |
| Sin servicios externos en la demo | El jurado no depende de la red ni de claves | 0005 |

## 7. Riesgos técnicos

| Riesgo | Impacto | Mitigación |
|---|---|---|
| El GA no entra en 3 s | La demo se corta | Presupuesto por reloj, elitismo y Webster como respaldo |
| Supuestos de demanda | KPIs poco creíbles | Etiqueta `Supuesto` visible y rango P10–P90 |
| PostGIS no disponible | Nada arranca | `docker compose up` con healthcheck y volumen inicializado |
| Plan inválido exportado | Pérdida de confianza | Validación en el servidor antes de persistir |

## 8. Fuera de alcance

Backend de usuarios, autenticación, integración con controladores, SUMO,
adaptativo en tiempo real, peatones, ciclistas y prioridad de emergencias. Está
declarado en el README, sección 9, y no se implementa en el MVP.
