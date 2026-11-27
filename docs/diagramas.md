# Diagramas

Los diagramas viven en el repositorio como texto Mermaid: GitHub los renderiza,
se versionan con el código y no se desactualizan como un PNG.

## 1. Contexto (C4 nivel 1)

Quién usa el sistema y de qué depende.

```mermaid
flowchart LR
    G["Gestor municipal<br/>de tránsito"]
    E["Evaluador técnico"]
    W["WayraFlow<br/>optimización semafórica"]
    O["OpenStreetMap<br/>Overpass API"]
    P["Plan semafórico<br/>wayraflow.plan/v1"]

    G -->|"elige escenario<br/>y descarga el plan"| W
    E -->|"revisa supuestos y KPIs"| W
    O -.->|"red vial, una sola vez"| W
    W -->|"JSON validado"| P
```

La línea punteada es una dependencia que se resuelve una vez: la red se extrae y
se versiona. En operación, el sistema no consulta Overpass.

## 2. Contenedores (C4 nivel 2)

Qué corre y dónde.

```mermaid
flowchart TB
    subgraph cliente["Cliente"]
        SPA["Angular SPA<br/>MapLibre + canvas IDM"]
    end

    subgraph servidor["Servidor"]
        API["API Go<br/>cmd/api"]
        EXT["Extractor OSM<br/>cmd/osmextract"]
    end

    DB[("PostgreSQL 16<br/>+ PostGIS")]
    GEO["data/juliaca.geojson"]

    SPA -->|"HTTP/JSON"| API
    API -->|"pgx"| DB
    EXT -->|"genera"| GEO
    GEO -->|"carga al arrancar<br/>(NETWORK_FILE)"| DB
```

Implementado hoy: la API, las migraciones, la semilla de nodos y corredores, la
carga del GeoJSON y el extractor. Pendiente: la SPA.

## 3. Secuencia de una corrida

El camino de un `POST /api/v1/plans` hasta el plan validado. Es el contrato
objetivo: los endpoints todavía no existen.

```mermaid
sequenceDiagram
    autonumber
    actor U as Usuario
    participant SPA as Angular
    participant API as API Go
    participant DB as PostGIS

    U->>SPA: elige escenario y estrategia
    SPA->>API: POST /api/v1/plans
    API->>DB: SELECT nodos, corredores, distancias
    DB-->>API: red y escenario
    API->>API: simula ciclo fijo (línea base)
    API->>API: Webster + GA (presupuesto 3 s)
    API->>API: valida wayraflow.plan/v1
    alt plan inválido
        API-->>SPA: 422 problem+json
    else plan válido
        API->>DB: INSERT plan + run
        API-->>SPA: 201 planId, runId, KPIs
    end
    SPA-->>U: mapa, tabla y KPIs
```

## 4. Modelo de datos

Las siete tablas del esquema y sus relaciones.

```mermaid
erDiagram
    road_segments {
        bigint id PK
        bigint osm_way_id UK
        text name
        text ref
        text highway
        geometry geom
    }

    nodes {
        text id PK
        text name
        bigint osm_node_id
        boolean signalized
        geometry geom
    }

    corridors {
        text id PK
        text name
    }

    corridor_nodes {
        text corridor_id PK, FK
        text node_id PK, FK
        int position
        real distance_m
    }

    scenarios {
        text id PK
        text name
        text description
        jsonb demand
    }

    plans {
        bigint id PK
        text scenario_id FK
        text strategy
        jsonb plan
        timestamptz created_at
    }

    runs {
        bigint id PK
        bigint plan_id FK
        jsonb kpis
        int duration_ms
        timestamptz created_at
    }

    corridors ||--o{ corridor_nodes : "ordena"
    nodes ||--o{ corridor_nodes : "pertenece"
    scenarios ||--o{ plans : "origina"
    plans ||--o{ runs : "se evalúa en"
```

`road_segments` no se relaciona por clave foránea con `corridors`: la pertenencia
geográfica se resuelve por intersección de geometrías, no por integridad
referencial. Es una decisión consciente, no un olvido.
