# Registro de uso de IA

Este archivo existe porque la rúbrica evalúa el **uso efectivo de IA**: no que la
herramienta haya escrito código, sino que el equipo haya dirigido, revisado y
corregido lo que produjo. El registro se escribe durante el trabajo, no después.

## Herramientas

- OpenCode con modelos de lenguaje para análisis, redacción técnica y generación
  de código.
- ChatGPT y Claude para contraste de fuentes y revisión de textos.

## Método

1. **Especificar antes de generar.** Ningún prompt pide "hazme la app". Se pide
   una pieza concreta con su criterio de aceptación.
2. **Verificar antes de aceptar.** Todo lo que entra se compila, se prueba o se
   contrasta contra una fuente.
3. **Registrar el rechazo.** Lo que la IA propuso y se descartó vale tanto como
   lo aceptado: muestra criterio.

## Bitácora

| Momento | Objetivo | Qué propuso la IA | Decisión del equipo | Evidencia |
|---|---|---|---|---|
| 27 nov, 12:00 | Especificación técnica | README con problema, red, modelo mesoscópico, IDM, GA y KPIs, con fuentes enlazadas | Aceptado con correcciones: se marcaron como `Supuesto` los valores sin aforo y se agregó la sección de pendientes | `README.md` |
| 27 nov, 15:10 | Cambio de stack | La IA había armado un cliente 100 % en el navegador (Vite + TypeScript + Worker) | **Rechazado.** El equipo decidió Angular + Go + PostGIS porque la simulación y el GA pesan más que la interfaz y hace falta historial de corridas | `docs/adr/0001`, `docs/adr/0002` |
| 27 nov, 15:20 | Motor | Implementación del contrato del plan y del ciclo de Webster en TypeScript, con pruebas | Aceptado como referencia conceptual; se reimplementa en Go dentro de `internal/optim` | `docs/api.md` |
| 27 nov, 15:45 | Arquitectura | Diagrama de contenedores, regla de dependencias y riesgos técnicos | Aceptado; el equipo ajustó el reparto: Go calcula, Angular solo visualiza y micro-simula | `docs/arquitectura.md` |
| 27 nov, 15:50 | Base de datos | Esquema PostGIS con geometrías y `jsonb` para escenarios y planes | Aceptado con un cambio: las distancias se calculan con `::geography`, no con grados | `docs/modelo-datos.md`, `docs/adr/0003` |
| 27 nov, 16:05 | Motor | Simulador mesoscópico con pelotones, spillback y KPIs por nodo | Aceptado tras corregir la capacidad de la vía transversal: 1 800 veh/h, no 3 600 | `internal/sim`, `internal/optim` |
| 27 nov, 16:25 | Optimizador | GA que elige el mejor plan con el modelo analítico | **Rechazado en parte.** El GA perdía contra el ciclo fijo en hora punta; ahora la población y las dos líneas base compiten y la selección se mide con el mismo simulador que reporta los KPIs | `internal/optim/optim.go`, `cmd/bench` |
| 27 nov, 16:45 | API y persistencia | Endpoints y store sobre PostgreSQL/PostGIS | Aceptado; se agregó el store en memoria para que la demo corra sin base y los tests no dependan de Docker | `internal/httpapi`, `internal/store` |
| 27 nov, 17:05 | Tablero | Dashboard Angular con mapa SVG, tabla de KPIs y simulación IDM en canvas | Aceptado con correcciones: faltaba `DecimalPipe` en dos componentes y el proxy de `/api` para `ng serve` | `frontend/src/app` |

## Qué no se delegó

- La definición del problema y el alcance.
- Los supuestos de demanda, mezcla vehicular y ciclo fijo.
- La decisión de stack y el reparto de responsabilidades.
- La validación de las fuentes del README.

## Cómo se verifica

- `go test ./...` y `go vet ./...` en cada push.
- Integración continua en `.github/workflows/ci.yml`: formato, `vet`, tests con
  `-race`, build de Go, build de Angular y build de ambas imágenes.
- Pruebas del contrato del plan con casos límite.
- Revisión humana de cada documento antes del commit.
- Historial de commits dentro de la ventana del concurso.

## Riesgos asumidos

La IA puede producir texto plausible y falso. Por eso toda cifra del README tiene
fuente o etiqueta `Supuesto`, y ningún KPI se presenta como medición de campo.
