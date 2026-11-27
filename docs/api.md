# API HTTP

Base: `/api/v1`. Formato: JSON UTF-8. Errores con RFC 7807
(`application/problem+json`).

## Convenciones

- Fechas en ISO 8601 UTC (`2026-11-27T20:15:00Z`).
- Identificadores de corrida: UUID v4.
- Toda respuesta de error incluye `type`, `title`, `status`, `detail` e `instance`.
- Los números de demanda van en veh/h y los tiempos en segundos, con sufijo en el
  nombre del campo (`cycle_s`, `distance_m`, `co2_kg_h`).

## Endpoints

### `GET /healthz`

```json
{ "status": "ok", "version": "0.1.0", "database": "ok" }
```

### `GET /network`

Devuelve la red en GeoJSON (`FeatureCollection`) con nodos y corredores. Acepta
`?bbox=minLon,minLat,maxLon,maxLat` para recortar.

### `GET /scenarios`

```json
[
  {
    "id": "base",
    "label": "Tráfico base",
    "demandVehPerHour": { "C1": 700, "C2": 600, "C3": 650, "N8": 400 },
    "mix": { "L5": 0.35, "M1": 0.35, "M2": 0.2, "N": 0.1 },
    "assumption": true
  }
]
```

### `POST /runs`

Cuerpo:

```json
{
  "scenarioId": "peak-regional",
  "strategy": "genetic",
  "seed": 42,
  "config": { "populationSize": 60, "maxGenerations": 150, "budgetMs": 3000 }
}
```

Respuesta `201 Created` con el resultado completo de la corrida:

```json
{
  "runId": "0f0f7f7e-6a1e-4b3e-9a2e-2f0f0f0f0f0f",
  "scenarioId": "peak-regional",
  "strategy": "genetic",
  "durationMs": 2740,
  "plan": { "schema": "wayraflow.plan/v1", "...": "..." },
  "kpis": {
    "avgDelay_s": { "fixed": 38.4, "webster": 31.2, "genetic": 27.9 },
    "co2_kg_h": { "fixed": 412.5, "webster": 361.0, "genetic": 340.2 }
  }
}
```

Errores:

| Código | Cuándo |
|---|---|
| 400 | Escenario desconocido, `strategy` inválida o configuración fuera de rango |
| 422 | El plan generado no cumple el contrato `wayraflow.plan/v1` |
| 500 | Fallo interno; el `detail` no expone trazas |

### `GET /runs/{runId}`

Metadatos de la corrida y estado (`queued`, `running`, `done`, `failed`).

### `GET /runs/{runId}/plan`

El plan validado, listo para descargar. `Content-Disposition: attachment` cuando
se pide con `?download=1`.

### `GET /runs/{runId}/kpis`

Serie de KPIs por nodo y agregado de ciudad, con la línea base y la mejora.

## Contrato del plan

El plan exportado es la interfaz pública del sistema. Su esquema vive en
`internal/plan` y su validación es obligatoria antes de persistir o descargar:

- `schema` debe ser `wayraflow.plan/v1`.
- En cada nodo, `Σ(green_s + amber_s + allRed_s) === cycle_s`.
- En cada corredor, `0 ≤ offset_s < cycle_s`.
- Los nodos de un corredor comparten el ciclo del corredor.
- Todo nodo referenciado existe; todo corredor referenciado existe.

Un plan que no cumple se rechaza con `422` y no se guarda. Es preferible una
corrida fallida a un plan inválido en manos del municipio.

## Versionado

Cambios incompatibles suben el sufijo del esquema (`v2`). Cambios aditivos no
suben versión, pero se documentan en `CHANGELOG.md`. La API no rompe clientes
dentro de la misma versión mayor.
