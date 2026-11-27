# ADR 0002 — El motor de cálculo vive en Go

- **Estado**: aceptado
- **Fecha**: 2026-11-27

## Contexto

El algoritmo genético evalúa 60 individuos por generación y cada evaluación
simula la ciudad. En el navegador eso compite con el hilo de la interfaz.

## Decisión

Implementar simulación, Webster, algoritmo genético, emisiones y KPIs en Go, como
paquetes puros dentro de `internal/`. La API solo los orquesta.

## Consecuencias

- Los paquetes se prueban sin base de datos ni HTTP.
- El presupuesto de 3 s se controla con reloj y no con heurísticas de UI.
- La semilla se guarda en `runs` y garantiza reproducibilidad.
- Angular queda como cliente delgado: no duplica reglas de negocio.

## Alternativas descartadas

- **Web Worker con TypeScript**: ya estaba prototipado, pero obligaba a duplicar
  el modelo y a cargar la red en cada pestaña.
