# ADR 0004 — El plan exportado es un contrato versionado

- **Estado**: aceptado
- **Fecha**: 2026-11-27

## Contexto

El valor del proyecto es que el municipio pueda llevarse los tiempos a un
controlador. Ese JSON es la interfaz más importante del sistema y no puede
cambiar sin aviso.

## Decisión

Definir `wayraflow.plan/v1` como esquema propio, validarlo antes de persistir y
antes de descargar, y versionar el sufijo ante cambios incompatibles.

## Consecuencias

- Un plan inválido no llega nunca al usuario: la corrida falla con `422`.
- El esquema se prueba con casos límite (suma de fases, offsets, referencias).
- No se promete compatibilidad con controladores reales: el README lo dice
  explícitamente.
