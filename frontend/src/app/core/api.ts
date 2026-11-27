import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';

export type Strategy = 'fixed' | 'webster' | 'ga';

export interface Mix {
  L5: number;
  M1: number;
  M2: number;
  N: number;
}

export interface Scenario {
  id: string;
  name: string;
  description?: string;
  demand: { corridors: Record<string, number>; mix: Mix };
}

export interface Phase {
  id: number;
  movements: string[];
  green_s: number;
  amber_s: number;
  allRed_s: number;
}

export interface NodePlan {
  id: string;
  osmNode: number;
  corridor: string;
  cycle_s: number;
  phases: Phase[];
}

export interface CorridorPlan {
  id: string;
  cycle_s: number;
  offsets_s: Record<string, number>;
}

export interface KPI {
  avgDelay_s: number;
  maxQueue_veh: number;
  maxQueue_m: number;
  stopsPerVeh: number;
  co2_kg_h: number;
  vehicles: number;
  spillback_s?: number;
}

export interface KPIReport {
  fixed: KPI;
  webster: KPI;
  ga: KPI;
  improvement_pct: Record<string, number>;
}

export interface NodeKPI {
  nodeId: string;
  avgDelay_s: number;
  maxQueue_veh: number;
  maxQueue_m: number;
  stopsPerVeh: number;
  co2_kg_h: number;
  vehicles: number;
}

export interface Plan {
  schema: string;
  city: string;
  scenario: string;
  strategy: Strategy;
  generatedAt: string;
  source: string;
  corridors: CorridorPlan[];
  nodes: NodePlan[];
  kpis?: KPIReport;
}

export interface PlanResponse {
  planId: number;
  runId: number;
  durationMs: number;
  plan: Plan;
  kpis: KPIReport;
  nodes: NodeKPI[];
}

export interface GeoFeature {
  type: 'Feature';
  geometry:
    | { type: 'Point'; coordinates: [number, number] }
    | { type: 'LineString'; coordinates: [number, number][] };
  properties: {
    id: string;
    name: string;
    kind: 'node' | 'corridor';
    signalized?: boolean;
    osmNode?: number;
  };
}

export interface FeatureCollection {
  type: 'FeatureCollection';
  features: GeoFeature[];
  attribution: string;
}

@Injectable({ providedIn: 'root' })
export class ApiService {
  private readonly http = inject(HttpClient);
  private readonly base = '/api/v1';

  scenarios(): Observable<Scenario[]> {
    return this.http.get<Scenario[]>(`${this.base}/scenarios`);
  }

  network(): Observable<FeatureCollection> {
    return this.http.get<FeatureCollection>(`${this.base}/network`);
  }

  createPlan(scenarioId: string, strategy: Strategy, seed = 42): Observable<PlanResponse> {
    return this.http.post<PlanResponse>(`${this.base}/plans`, { scenarioId, strategy, seed });
  }

  downloadUrl(planId: number): string {
    return `${this.base}/plans/${planId}?download=1`;
  }
}
