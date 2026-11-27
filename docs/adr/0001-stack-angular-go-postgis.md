# ADR 0001 — Angular, Go y PostgreSQL/PostGIS

- **Estado**: aceptado
- **Fecha**: 2026-11-27

## Contexto

El sistema calcula una simulación mesoscópica de 3 600 s sobre 8 nodos y un
algoritmo genético de hasta 150 generaciones. Además necesita comparar
estrategias y persistir corridas para la defensa.

## Decisión

Angular para la interfaz, Go para el motor y PostgreSQL con PostGIS para la red y
los resultados. Docker Compose como forma única de levantar el stack.

## Consecuencias

- El cálculo no compite con el render del mapa ni con la animación del nodo.
- PostGIS resuelve distancias, pertenencia y recortes sin reimplementar geometría.
- Hay dos lenguajes y un contenedor más: el equipo asume ese costo operativo.
- La vista de nodo sigue en el navegador porque necesita 60 fps y no persiste.

## Alternativas descartadas

- **Todo en el navegador**: simple de desplegar, pero la simulación bloquea la
  interfaz y no hay historial de corridas.
- **Python + FastAPI**: válido para el modelo, más lento en el ciclo de
  optimización y con otro runtime que operar.
