import { DecimalPipe } from '@angular/common';
import { Component, computed, inject, signal } from '@angular/core';
import { ApiService, CorridorPlan, FeatureCollection, NodeKPI, PlanResponse, Scenario, Strategy } from '../../core/api';
import { NetworkMap } from '../map/network-map';
import { NodeSim } from '../node-sim/node-sim';

@Component({
  selector: 'app-dashboard',
  imports: [DecimalPipe, NetworkMap, NodeSim],
  template: `
    <header class="top">
      <div class="brand">
        <span class="mark">WF</span>
        <div>
          <h1>WayraFlow</h1>
          <p>Optimización semafórica · Juliaca</p>
        </div>
      </div>
      <div class="badges">
        <span>Go + PostGIS</span>
        <span>ODbL</span>
        @if (result(); as r) {
          <span>{{ r.durationMs }} ms</span>
        }
      </div>
    </header>

    <main>
      <aside class="panel controls">
        <h2>Escenario</h2>
        @for (s of scenarios(); track s.id) {
          <button
            class="scenario"
            [class.active]="s.id === scenarioId()"
            (click)="scenarioId.set(s.id)"
          >
            <b>{{ s.name }}</b>
            <small>{{ s.description }}</small>
          </button>
        }

        <h2>Estrategia</h2>
        <div class="segmented">
          @for (s of strategies; track s.id) {
            <button [class.active]="s.id === strategy()" (click)="strategy.set(s.id)">
              {{ s.label }}
            </button>
          }
        </div>

        <button class="primary" (click)="generate()" [disabled]="loading()">
          {{ loading() ? 'Calculando…' : 'Calcular plan' }}
        </button>
        @if (error()) {
          <p class="error">{{ error() }}</p>
        }
        <p class="hint">
          El GA explora verdes y desfases y compite contra el ciclo fijo y Webster. La selección final
          se mide con el simulador, así que nunca devuelve un plan peor que la línea base.
        </p>

        @if (result(); as r) {
          <h2>Plan #{{ r.planId }}</h2>
          <table class="plan">
            <thead>
              <tr><th>Nodo</th><th>Ciclo</th><th>V1</th><th>V2</th></tr>
            </thead>
            <tbody>
              @for (n of r.plan.nodes; track n.id) {
                <tr>
                  <td>{{ n.id }}</td>
                  <td>{{ n.cycle_s | number: '1.0-0' }}</td>
                  <td>{{ n.phases[0].green_s | number: '1.0-0' }}</td>
                  <td>{{ n.phases[1] ? (n.phases[1].green_s | number: '1.0-0') : '—' }}</td>
                </tr>
              }
            </tbody>
          </table>
          <h3>Desfases por corredor</h3>
          @for (c of r.plan.corridors; track c.id) {
            <p class="offset"><b>{{ c.id }}</b> {{ offsets(c) }}</p>
          }
          <p class="schema">Contrato {{ r.plan.schema }} válido</p>
        }
      </aside>

      <section class="panel map-panel">
        <app-network-map
          [network]="network()"
          [nodeKpis]="result()?.nodes ?? []"
          [selected]="selectedNode()"
          (select)="selectedNode.set($event)"
        />
      </section>

      <aside class="panel kpis">
        <h2>KPIs de la corrida</h2>
        @if (kpis(); as k) {
          <table>
            <thead>
              <tr>
                <th></th>
                <th [class.active]="strategy() === 'fixed'">Fijo</th>
                <th [class.active]="strategy() === 'webster'">Webster</th>
                <th [class.active]="strategy() === 'ga'">GA</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td>Espera</td>
                <td>{{ k.fixed.avgDelay_s | number: '1.1-1' }}</td>
                <td>{{ k.webster.avgDelay_s | number: '1.1-1' }}</td>
                <td>{{ k.ga.avgDelay_s | number: '1.1-1' }}</td>
              </tr>
              <tr>
                <td>Cola máx.</td>
                <td>{{ k.fixed.maxQueue_veh | number: '1.1-1' }}</td>
                <td>{{ k.webster.maxQueue_veh | number: '1.1-1' }}</td>
                <td>{{ k.ga.maxQueue_veh | number: '1.1-1' }}</td>
              </tr>
              <tr>
                <td>Paradas</td>
                <td>{{ k.fixed.stopsPerVeh | number: '1.2-2' }}</td>
                <td>{{ k.webster.stopsPerVeh | number: '1.2-2' }}</td>
                <td>{{ k.ga.stopsPerVeh | number: '1.2-2' }}</td>
              </tr>
              <tr>
                <td>CO₂</td>
                <td>{{ k.fixed.co2_kg_h | number: '1.0-0' }}</td>
                <td>{{ k.webster.co2_kg_h | number: '1.0-0' }}</td>
                <td>{{ k.ga.co2_kg_h | number: '1.0-0' }}</td>
              </tr>
            </tbody>
          </table>
          <p class="units">espera s/veh · cola veh · paradas por veh · CO₂ kg/h</p>

          <div class="improvement">
            <span>Mejora del GA</span>
            <b>{{ k.improvement_pct['ga'] | number: '1.1-1' }} %</b>
          </div>

          <button class="ghost" (click)="download()">Descargar plan JSON</button>

          @if (result(); as r) {
            <dl>
              <div><dt>Plan</dt><dd>#{{ r.planId }}</dd></div>
              <div><dt>Corrida</dt><dd>#{{ r.runId }}</dd></div>
              <div><dt>Semilla</dt><dd>42</dd></div>
              <div><dt>Vehículos</dt><dd>{{ k.ga.vehicles | number }}</dd></div>
            </dl>
          }
        } @else {
          <p class="empty">Elegí un escenario y calculá el plan para ver la comparación.</p>
        }
      </aside>

      <section class="panel node-panel">
        <app-node-sim
          [node]="chosenNode()"
          [kpi]="chosenNodeKpi()"
          [flow]="nodeFlow()"
          [mix]="chosenScenario()?.demand?.mix ?? null"
          [label]="nodeLabel()"
        />
      </section>
    </main>

    <footer>
      <span>Datos: {{ network()?.attribution }}</span>
      <span>Supuestos declarados en docs/requerimientos.md</span>
    </footer>
  `,
  styles: `
    :host {
      display: block;
      min-height: 100vh;
      padding: 18px 22px 10px;
      box-sizing: border-box;
    }
    .top {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 16px;
      margin-bottom: 16px;
    }
    .brand {
      display: flex;
      align-items: center;
      gap: 12px;
    }
    .mark {
      display: grid;
      place-items: center;
      width: 42px;
      height: 42px;
      border-radius: 12px;
      background: linear-gradient(135deg, #57c4a8, #2f8f7a);
      color: #05130f;
      font-weight: 800;
      font-family: ui-monospace, monospace;
    }
    h1 {
      margin: 0;
      font-size: 20px;
      letter-spacing: 0.02em;
    }
    .brand p {
      margin: 2px 0 0;
      color: #6d8aa6;
      font-size: 12px;
    }
    .badges {
      display: flex;
      gap: 8px;
    }
    .badges span {
      border: 1px solid #1e3145;
      border-radius: 999px;
      padding: 5px 10px;
      color: #8aa4bd;
      font-size: 11px;
      font-family: ui-monospace, monospace;
    }
    main {
      display: grid;
      grid-template-columns: 270px minmax(0, 1fr) 330px;
      grid-template-areas:
        'controls map kpis'
        'controls node node';
      gap: 14px;
    }
    .panel {
      background: #0e1a28;
      border: 1px solid #1e3145;
      border-radius: 16px;
      padding: 16px;
      min-width: 0;
    }
    .controls {
      grid-area: controls;
    }
    .map-panel {
      grid-area: map;
      padding: 0;
      overflow: hidden;
      background: #0b1420;
    }
    .kpis {
      grid-area: kpis;
    }
    .node-panel {
      grid-area: node;
      padding: 0;
      overflow: hidden;
    }
    h2 {
      margin: 0 0 10px;
      font-size: 11px;
      letter-spacing: 0.14em;
      text-transform: uppercase;
      color: #6d8aa6;
    }
    .controls h2:not(:first-child) {
      margin-top: 20px;
    }
    .scenario {
      display: block;
      width: 100%;
      text-align: left;
      background: #0b1420;
      border: 1px solid #1e3145;
      border-radius: 10px;
      padding: 10px 12px;
      margin-bottom: 8px;
      color: #cfe0f0;
      cursor: pointer;
    }
    .scenario b {
      display: block;
      font-size: 13px;
    }
    .scenario small {
      display: block;
      margin-top: 3px;
      color: #6d8aa6;
      font-size: 11px;
      line-height: 1.35;
    }
    .scenario.active,
    .scenario:hover {
      border-color: #57c4a8;
      background: #10251f;
    }
    .segmented {
      display: grid;
      grid-template-columns: repeat(3, 1fr);
      border: 1px solid #1e3145;
      border-radius: 10px;
      overflow: hidden;
    }
    .segmented button {
      background: #0b1420;
      border: 0;
      color: #8aa4bd;
      padding: 9px 4px;
      font-size: 12px;
      cursor: pointer;
    }
    .segmented button.active {
      background: #57c4a8;
      color: #05130f;
      font-weight: 700;
    }
    .primary,
    .ghost {
      width: 100%;
      margin-top: 16px;
      padding: 11px;
      border-radius: 10px;
      border: 1px solid transparent;
      font-weight: 700;
      cursor: pointer;
    }
    .primary {
      background: #57c4a8;
      color: #05130f;
    }
    .primary:disabled {
      opacity: 0.6;
      cursor: wait;
    }
    .ghost {
      background: transparent;
      border-color: #2f4a63;
      color: #cfe0f0;
      margin-top: 10px;
      font-weight: 500;
    }
    .ghost:hover {
      border-color: #57c4a8;
    }
    .hint {
      margin: 14px 0 0;
      color: #6d8aa6;
      font-size: 11px;
      line-height: 1.5;
    }
    table.plan {
      font-size: 11px;
    }
    table.plan th,
    table.plan td {
      padding: 4px 2px;
    }
    h3 {
      margin: 14px 0 6px;
      font-size: 10px;
      letter-spacing: 0.12em;
      text-transform: uppercase;
      color: #6d8aa6;
    }
    .offset {
      margin: 0 0 4px;
      font-family: ui-monospace, monospace;
      font-size: 10px;
      color: #8aa4bd;
    }
    .offset b {
      color: #57c4a8;
      margin-right: 4px;
    }
    .schema {
      margin: 12px 0 0;
      padding: 6px 8px;
      border-radius: 8px;
      border: 1px solid #1f4a3d;
      background: #10251f;
      color: #57c4a8;
      font-size: 10px;
      font-family: ui-monospace, monospace;
    }
    .error {
      margin: 10px 0 0;
      color: #ef476f;
      font-size: 12px;
    }
    .empty {
      color: #6d8aa6;
      font-size: 12px;
      line-height: 1.6;
    }
    table {
      width: 100%;
      border-collapse: collapse;
      font-family: ui-monospace, 'Cascadia Code', monospace;
      font-size: 12px;
    }
    th,
    td {
      padding: 7px 4px;
      text-align: right;
      border-bottom: 1px solid #16283c;
    }
    th:first-child,
    td:first-child {
      text-align: left;
      color: #8aa4bd;
    }
    th {
      color: #6d8aa6;
      font-weight: 500;
      font-size: 11px;
    }
    th.active {
      color: #57c4a8;
      font-weight: 700;
    }
    .units {
      margin: 8px 0 0;
      color: #5b7691;
      font-size: 10px;
    }
    .improvement {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-top: 14px;
      padding: 10px 12px;
      border-radius: 10px;
      background: #10251f;
      border: 1px solid #1f4a3d;
      color: #8aa4bd;
      font-size: 12px;
    }
    .improvement b {
      color: #57c4a8;
      font-size: 20px;
      font-family: ui-monospace, monospace;
    }
    dl {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 8px;
      margin: 14px 0 0;
      font-family: ui-monospace, monospace;
      font-size: 11px;
    }
    dl div {
      background: #0b1420;
      border: 1px solid #16283c;
      border-radius: 8px;
      padding: 7px 9px;
    }
    dt {
      color: #6d8aa6;
    }
    dd {
      margin: 2px 0 0;
      color: #cfe0f0;
    }
    footer {
      display: flex;
      justify-content: space-between;
      margin-top: 12px;
      color: #5b7691;
      font-size: 10px;
      font-family: ui-monospace, monospace;
    }
    @media (max-width: 1100px) {
      main {
        grid-template-columns: 1fr;
        grid-template-areas:
          'controls'
          'map'
          'kpis'
          'node';
      }
    }
  `,
})
export class Dashboard {
  private readonly api = inject(ApiService);

  readonly scenarios = signal<Scenario[]>([]);
  readonly network = signal<FeatureCollection | null>(null);
  readonly result = signal<PlanResponse | null>(null);
  readonly scenarioId = signal('base');
  readonly strategy = signal<Strategy>('ga');
  readonly selectedNode = signal('N1');
  readonly loading = signal(false);
  readonly error = signal('');

  readonly strategies: { id: Strategy; label: string }[] = [
    { id: 'fixed', label: 'Fijo' },
    { id: 'webster', label: 'Webster' },
    { id: 'ga', label: 'GA' },
  ];

  readonly kpis = computed(() => this.result()?.kpis ?? null);
  readonly chosenScenario = computed(
    () => this.scenarios().find((s) => s.id === this.scenarioId()) ?? null,
  );
  readonly chosenNode = computed(
    () => this.result()?.plan.nodes.find((n) => n.id === this.selectedNode()) ?? null,
  );
  readonly chosenNodeKpi = computed<NodeKPI | null>(
    () => this.result()?.nodes.find((n) => n.nodeId === this.selectedNode()) ?? null,
  );
  readonly nodeFlow = computed(() => {
    const node = this.chosenNode();
    const scenario = this.chosenScenario();
    if (!node || !scenario) {
      return 0;
    }
    return scenario.demand.corridors[node.corridor || node.id] ?? 0;
  });
  readonly nodeLabel = computed(() => {
    const node = this.chosenNode();
    if (!node) {
      return 'Seleccioná un nodo en el mapa';
    }
    return node.corridor ? `Corredor ${node.corridor}` : 'Nodo aislado, sin coordinación';
  });

  constructor() {
    this.api.scenarios().subscribe({
      next: (list) => this.scenarios.set(list),
      error: () => this.error.set('No se pudieron cargar los escenarios. ¿Está corriendo la API?'),
    });
    this.api.network().subscribe({
      next: (net) => this.network.set(net),
      error: () => this.error.set('No se pudo cargar la red vial.'),
    });
  }

  generate(): void {
    this.loading.set(true);
    this.error.set('');
    this.api.createPlan(this.scenarioId(), this.strategy()).subscribe({
      next: (res) => {
        this.result.set(res);
        this.loading.set(false);
      },
      error: (err) => {
        this.loading.set(false);
        this.error.set(err?.error?.detail ?? 'La corrida falló. Revisá los logs de la API.');
      },
    });
  }

  download(): void {
    const res = this.result();
    if (res) {
      window.open(this.api.downloadUrl(res.planId), '_blank');
    }
  }

  offsets(corridor: CorridorPlan): string {
    return Object.entries(corridor.offsets_s)
      .map(([id, seconds]) => `${id} ${seconds.toFixed(0)}`)
      .join(' · ');
  }
}
