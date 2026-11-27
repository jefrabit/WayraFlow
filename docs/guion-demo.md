# Guion de la demo

Duración objetivo: **3 minutos**. La sustentación completa es de 1 hora para
todos los equipos, así que asumimos 5 minutos por equipo y dejamos margen.

## Antes de empezar

- Stack levantado: `docker compose up -d` y `docker compose ps` en verde.
- Escenario `peak-regional` cargado y una corrida ya hecha (no calcular en vivo
  si el jurado tiene prisa).
- Pestaña del plan JSON abierta, por si falla la descarga.
- Video de respaldo de 60 s, grabado antes.

## 0:00 – 0:30 · El problema

> "Juliaca tiene unos 45 cruces semaforizados y un plan de ciclo fijo que no
> responde a la demanda real. El domingo, día de feria, la ciudad se mueve
> distinto y el semáforo no lo sabe. No proponemos cambiar los semáforos:
> proponemos cambiar los tiempos."

Mostrar el mapa con los 8 nodos y los 3 corredores. Señalar la atribución ODbL.

## 0:30 – 1:15 · La comparación

> "Corremos el mismo escenario con tres estrategias: el ciclo fijo actual,
> Webster como referencia analítica y nuestro algoritmo genético."

Cambiar de estrategia y leer los KPIs: espera promedio, paradas, CO₂ y el
porcentaje de mejora. Aclarar en una frase que son estimaciones del modelo, no
mediciones de calle.

## 1:15 – 2:00 · El plan

Abrir la tabla: ciclo, verdes por fase y desfase por corredor. Mostrar el estado
"contrato v1 válido" y descargar el JSON.

> "El plan es exportable y validado. Si la suma de fases no da el ciclo, el
> sistema no lo deja salir."

## 2:00 – 2:40 · La vista de nodo

Abrir la micro-simulación IDM: los tres tipos de vehículo, el semáforo como
vehículo virtual, la cola descargando durante el verde.

> "Acá se ve por qué el ciclo importa: el mototaxi y el camión no arrancan igual."

## 2:40 – 3:00 · Cierre

> "Todo corre local, sin claves ni servicios externos: `docker compose up`. El
> código, los supuestos y las pruebas están en el repositorio."

## Preguntas previsibles

| Pregunta | Respuesta corta |
|---|---|
| ¿De dónde salen los aforos? | No hay aforos públicos; son supuestos declarados y el plan es calibrarlos en campo. |
| ¿Por qué no SUMO? | SUMO es microscópico y pesado para 8 nodos; el modelo mesoscópico alcanza para comparar planes y corre en segundos. |
| ¿Cómo validan el ahorro? | Con un piloto: km, tiempo, combustible y CO₂ antes y después. Hoy comparamos modelos entre sí. |
| ¿Por qué Go y no Python? | El GA evalúa 60 individuos por generación; Go mantiene el presupuesto de 3 s sin pelear con el GIL. |
| ¿Funciona sin internet? | Sí: la red está versionada y el mapa no usa teselas externas. |
| ¿Qué falta? | Calibración en campo, vista de nodo pulida y el informe de resultados. |
