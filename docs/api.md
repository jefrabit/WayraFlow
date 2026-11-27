# API HTTP

Formato: JSON UTF-8. Errores con RFC 7807 (`application/problem+json`).

## Estado actual

| Endpoint | Estado | Respuesta |
|---|---|---|
| `GET /health` | **Implementado** | `{"status":"ok"}` |

El resto de este documento es el contrato objetivo. Se implementa en
`backend/internal/httpapi` sobre el mismo `http.ServeMux`; nada de lo que sigue
existe todavía en el binario.

## Convenciones

- Fechas en ISO 8601 UTC (`2026-11-27T20:15:00Z`).
- Tiempos en segundos y distancias en metros, con sufijo en el nombre
  (`cycle_s`, `distance_m`, `co2_kg_h`).
- Estrategias válidas: `fixed`, `webster`, `ga` (así está en el `CHECK` de la
  tabla `plans`).

## Endpoints objetivo

### `GET /api/v1/network`

Red en GeoJSON (`FeatureCollection`) con nodos y corredores. Acepta
`?bbox=minLon,minLat,maxLon,maxLat` para recortar.

### `GET /api/v1/scenarios`

```json
[
  {
    "id": "base",
    "name": "Tráfico base",
    "demand": {
      "corridors": { "C1": 700, "C2": 600, "C3": 650, "N8": 400 },
      "mix": { "L5": 0.35, "M1": 0.35, "M2": 0.2, "N": 0.1 }
    }
  }
]
```

### `POST /api/v1/plans`

Genera y persiste un plan.

```json
{
  "scenarioId": "peak-regional",
  "strategy": "ga",
  "seed": 42,
  "config": { "populationSize": 60, "maxGenerations": 150, "budgetMs": 3000 }
}
```

Responde `201 Created` con el plan y su corrida:

```json
{
  "planId": 17,
  "runId": 23,
  "durationMs": 2740,
  "plan": { "schema": "wayraflow.plan/v1", "city": "Juliaca" },
  "kpis": {
    "avgDelay_s": { "fixed": 38.4, "webster": 31.2, "ga": 27.9 },
    "co2_kg_h": { "fixed": 412.5, "webster": 361.0, "ga": 340.2 }
  }
}
```

Errores:

| Código | Cuándo |
|---|---|
| 400 | Escenario desconocido, estrategia inválida o configuración fuera de rango |
| 422 | El plan generado no cumple `wayraflow.plan/v1` |
| 500 | Fallo interno; el `detail` no expone trazas |

### `GET /api/v1/plans/{planId}`

El plan validado. Con `?download=1` agrega
`Content-Disposition: attachment`.

### `GET /api/v1/runs/{runId}`

Los KPIs de la corrida y su duración.

## Contrato del plan

El plan exportado es la interfaz pública del sistema. Se valida antes de
insertarlo en `plans` y antes de descargarlo:

- `schema` debe ser `wayraflow.plan/v1`.
- En cada nodo, `Σ(green_s + amber_s + allRed_s) === cycle_s`.
- En cada corredor, `0 ≤ offset_s < cycle_s`.
- Los nodos de un corredor comparten el ciclo del corredor.
- Toda referencia a nodo o corredor existe.

Un plan que no cumple se rechaza con `422` y no se guarda. Es preferible una
corrida fallida a un plan inválido en manos del municipio.

## Versionado

Cambios incompatibles suben el sufijo (`v2`). Los aditivos no suben versión, pero
se anotan en `CHANGELOG.md`. La API no rompe clientes dentro de la misma versión
mayor.
