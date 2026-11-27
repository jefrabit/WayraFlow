# API HTTP

Formato: JSON UTF-8. Errores con RFC 7807 (`application/problem+json`).

## Estado actual

Todo lo que sigue está implementado en `backend/internal/httpapi` sobre
`http.ServeMux` y verificado por `router_test.go`.

| Endpoint | Respuesta |
|---|---|
| `GET /health` | `{"status":"ok"}`; `503` si la base no responde |
| `GET /api/v1/network` | `FeatureCollection` GeoJSON con nodos y corredores |
| `GET /api/v1/scenarios` | Los tres escenarios del catálogo |
| `GET /api/v1/plans?limit=20` | Resúmenes de planes, del más nuevo al más viejo |
| `POST /api/v1/plans` | `201` con plan, KPIs y KPIs por nodo |
| `GET /api/v1/plans/{id}` | El plan guardado; `?download=1` fuerza descarga |
| `GET /api/v1/runs/{id}` | Los KPIs de la corrida |

El recorrido completo de una corrida está en
[diagramas.md](diagramas.md#3-secuencia-de-una-corrida).

## Convenciones

- Fechas en ISO 8601 UTC (`2026-11-27T20:15:00Z`).
- Tiempos en segundos y distancias en metros, con sufijo en el nombre
  (`cycle_s`, `distance_m`, `co2_kg_h`).
- Estrategias válidas: `fixed`, `webster`, `ga` (así está en el `CHECK` de la
  tabla `plans`).
- Los IDs de escenario son los del catálogo: `base`, `hora-punta-regional`,
  `dia-de-feria`.

## Endpoints

### `GET /health`

```json
{ "status": "ok" }
```

Si `DATABASE_URL` está definida y PostGIS no responde, devuelve `503` con
`application/problem+json`. Sin base de datos la API arranca con el store en
memoria y el healthcheck siempre responde `200`.

### `GET /api/v1/network`

Red en GeoJSON. Ocho `Point` (nodos) y tres `LineString` (corredores), más la
atribución ODbL en la propiedad `attribution`.

```json
{
  "type": "FeatureCollection",
  "features": [
    {
      "type": "Feature",
      "geometry": { "type": "Point", "coordinates": [-70.1333, -15.4997] },
      "properties": { "id": "N1", "name": "Salida Puno", "kind": "node", "signalized": true }
    }
  ],
  "attribution": "OpenStreetMap contributors, ODbL 1.0"
}
```

El parámetro `bbox` todavía **no** está implementado: la red completa son once
features y no hay necesidad de recortarla.

### `GET /api/v1/scenarios`

```json
[
  {
    "id": "base",
    "name": "Tráfico base",
    "description": "Hora de análisis con la demanda observada como supuesto.",
    "demand": {
      "corridors": { "C1": 700, "C2": 600, "C3": 650, "N8": 400 },
      "mix": { "L5": 0.35, "M1": 0.35, "M2": 0.2, "N": 0.1 }
    }
  }
]
```

### `POST /api/v1/plans`

Genera las tres estrategias, las simula, persiste el plan elegido y su corrida.
El campo `strategy` decide cuál de los tres planes se guarda y se devuelve; los
KPIs siempre incluyen las tres para que la comparación sea justa.

```json
{
  "scenarioId": "hora-punta-regional",
  "strategy": "ga",
  "seed": 42,
  "config": { "populationSize": 60, "maxGenerations": 150, "budgetMs": 3000 }
}
```

`seed` y `config` son opcionales. `config` existe para la demo y los tests; en
producción se usan los valores por defecto. Rangos: `populationSize` en
`[4, 500]`, `maxGenerations` en `[1, 1000]`, `budgetMs` en `[100, 10000]`.

Responde `201 Created`:

```json
{
  "planId": 17,
  "runId": 23,
  "durationMs": 2740,
  "plan": {
    "schema": "wayraflow.plan/v1",
    "city": "Juliaca",
    "scenario": "hora-punta-regional",
    "strategy": "ga",
    "generatedAt": "2026-11-27T20:15:00Z",
    "corridors": [{ "id": "C1", "cycle_s": 64, "offsets_s": { "N1": 0, "N3": 21, "N5": 43 } }],
    "nodes": [
      {
        "id": "N1",
        "osmNode": 123456,
        "corridor": "C1",
        "cycle_s": 64,
        "phases": [{ "id": 1, "movements": ["C1"], "green_s": 27, "amber_s": 3, "allRed_s": 2 }]
      }
    ]
  },
  "kpis": {
    "fixed": { "avgDelay_s": 18.03, "maxQueue_veh": 14.72, "maxQueue_m": 88.3, "stopsPerVeh": 0.46, "co2_kg_h": 493.5, "vehicles": 2210 },
    "webster": { "avgDelay_s": 20.6, "maxQueue_veh": 20, "maxQueue_m": 120, "stopsPerVeh": 0.52, "co2_kg_h": 556.7, "vehicles": 2210 },
    "ga": { "avgDelay_s": 18.03, "maxQueue_veh": 14.72, "maxQueue_m": 88.3, "stopsPerVeh": 0.46, "co2_kg_h": 493.5, "vehicles": 2210 },
    "improvement_pct": { "webster": -14.3, "ga": 0 }
  },
  "nodes": [
    { "nodeId": "N1", "avgDelay_s": 12.4, "maxQueue_veh": 9.1, "maxQueue_m": 54.6, "stopsPerVeh": 0.41, "co2_kg_h": 61.2, "vehicles": 312 }
  ]
}
```

`improvement_pct` se calcula contra el ciclo fijo. Un valor negativo es una
regresión y se reporta igual: el escenario de hora punta no mejora con el GA.

Errores:

| Código | Cuándo |
|---|---|
| 400 | JSON inválido, campo desconocido, escenario inexistente, estrategia inválida o `config` fuera de rango |
| 404 | Plan o corrida inexistente |
| 500 | Fallo interno; el `detail` no expone trazas |

No se usa `422`: el plan se valida dentro del optimizador, antes de guardarse.
Un plan generado que no cumple el contrato es un defecto del servidor, no del
cliente, y se reporta como `500` sin persistir nada.

### `GET /api/v1/plans?limit=20`

Lista los planes guardados. `limit` se recorta a `[1, 100]`; el valor por
defecto es 20.

```json
[{ "id": 17, "scenarioId": "base", "strategy": "ga", "createdAt": "2026-11-27T20:15:00Z" }]
```

### `GET /api/v1/plans/{id}`

El plan validado. Con `?download=1` agrega
`Content-Disposition: attachment; filename="wayraflow-plan-17.json"`.

### `GET /api/v1/runs/{id}`

Los KPIs de la corrida, con la misma forma que el objeto `kpis` del `POST`.

## Contrato del plan

El plan exportado es la interfaz pública del sistema. Se valida antes de
insertarlo en `plans` y antes de descargarlo:

- `schema` debe ser `wayraflow.plan/v1`.
- En cada nodo, `Σ(green_s + amber_s + allRed_s) === cycle_s` con tolerancia
  `1e-6`.
- `cycle_s` en `[40, 120]`; verde mínimo de 7 s; ámbar mayor que 0.
- En cada corredor, `0 ≤ offset_s < cycle_s`.
- Los nodos de un corredor comparten el ciclo del corredor.
- Toda referencia a nodo o corredor existe.

Un plan que no cumple no se guarda. Es preferible una corrida fallida a un plan
inválido en manos del municipio.

## Versionado

Cambios incompatibles suben el sufijo (`v2`). Los aditivos no suben versión, pero
se anotan en `CHANGELOG.md`. La API no rompe clientes dentro de la misma versión
mayor.
