import { DecimalPipe } from '@angular/common';
import { Component, computed, input, output } from '@angular/core';
import { FeatureCollection, GeoFeature, NodeKPI } from '../../core/api';

interface Point {
  x: number;
  y: number;
}

@Component({
  selector: 'app-network-map',
  imports: [DecimalPipe],
  template: `
    <svg viewBox="0 0 800 520" class="map" role="img" aria-label="Red vial de Juliaca">
      <defs>
        <pattern id="grid" width="32" height="32" patternUnits="userSpaceOnUse">
          <path d="M 32 0 L 0 0 0 32" fill="none" stroke="#1b2a3a" stroke-width="1" />
        </pattern>
        <filter id="glow" x="-50%" y="-50%" width="200%" height="200%">
          <feGaussianBlur stdDeviation="6" result="blur" />
          <feMerge>
            <feMergeNode in="blur" />
            <feMergeNode in="SourceGraphic" />
          </feMerge>
        </filter>
      </defs>

      <rect width="800" height="520" fill="#0b1420" />
      <rect width="800" height="520" fill="url(#grid)" opacity="0.5" />

      @for (c of corridors(); track c.properties.id) {
        <polyline
          [attr.points]="linePoints(c)"
          fill="none"
          stroke="#2f4a63"
          stroke-width="10"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
        <polyline
          [attr.points]="linePoints(c)"
          fill="none"
          stroke="#57c4a8"
          stroke-width="2"
          stroke-dasharray="6 8"
          opacity="0.7"
        />
        <text
          [attr.x]="labelX(c)"
          [attr.y]="labelY(c)"
          class="corridor-label"
        >{{ c.properties.id }} · {{ c.properties.name }}</text>
      }

      @for (n of nodes(); track n.properties.id) {
        <g
          class="node"
          (click)="select.emit(n.properties.id)"
          (keydown.enter)="select.emit(n.properties.id)"
          tabindex="0"
          role="button"
          [attr.aria-label]="'Nodo ' + n.properties.id"
        >
          <circle
            [attr.cx]="x(n)"
            [attr.cy]="y(n)"
            [attr.r]="radius(n)"
            [attr.fill]="color(n)"
            [attr.stroke]="selected() === n.properties.id ? '#ffffff' : '#0b1420'"
            [attr.stroke-width]="selected() === n.properties.id ? 3 : 2"
            [attr.filter]="selected() === n.properties.id ? 'url(#glow)' : null"
          />
          <text [attr.x]="x(n)" [attr.y]="y(n) - radius(n) - 8" class="node-label">
            {{ n.properties.id }}
          </text>
          @if (kpiFor(n.properties.id); as k) {
            <text [attr.x]="x(n)" [attr.y]="y(n) + radius(n) + 15" class="node-kpi">
              {{ k.avgDelay_s | number: '1.0-0' }} s
            </text>
          }
        </g>
      }
    </svg>
    <div class="legend">
      <span><i class="dot low"></i> &lt; 10 s</span>
      <span><i class="dot mid"></i> 10–20 s</span>
      <span><i class="dot high"></i> &gt; 20 s</span>
      <span class="attribution">{{ network()?.attribution }}</span>
    </div>
  `,
  styles: `
    :host {
      display: block;
      position: relative;
      background: #0b1420;
      border: 1px solid #1e3145;
      border-radius: 16px;
      overflow: hidden;
    }
    .map {
      display: block;
      width: 100%;
      height: auto;
    }
    .node {
      cursor: pointer;
      outline: none;
    }
    .node:hover circle {
      stroke: #ffffff;
      stroke-width: 3;
    }
    .node-label {
      fill: #e6f1ff;
      font-size: 13px;
      font-weight: 700;
      text-anchor: middle;
      font-family: ui-monospace, 'Cascadia Code', monospace;
    }
    .node-kpi {
      fill: #8aa4bd;
      font-size: 11px;
      text-anchor: middle;
      font-family: ui-monospace, 'Cascadia Code', monospace;
    }
    .corridor-label {
      fill: #6d8aa6;
      font-size: 11px;
      font-family: ui-monospace, 'Cascadia Code', monospace;
    }
    .legend {
      display: flex;
      align-items: center;
      gap: 18px;
      padding: 10px 16px;
      border-top: 1px solid #1e3145;
      color: #8aa4bd;
      font-size: 12px;
      font-family: ui-monospace, 'Cascadia Code', monospace;
    }
    .dot {
      display: inline-block;
      width: 9px;
      height: 9px;
      border-radius: 50%;
      margin-right: 5px;
    }
    .low {
      background: #3ddc97;
    }
    .mid {
      background: #ffb703;
    }
    .high {
      background: #ef476f;
    }
    .attribution {
      margin-left: auto;
      opacity: 0.7;
    }
  `,
})
export class NetworkMap {
  readonly network = input<FeatureCollection | null>(null);
  readonly nodeKpis = input<NodeKPI[]>([]);
  readonly selected = input<string>('');
  readonly select = output<string>();

  readonly nodes = computed<GeoFeature[]>(
    () => this.network()?.features.filter((f) => f.properties.kind === 'node') ?? [],
  );
  readonly corridors = computed<GeoFeature[]>(
    () => this.network()?.features.filter((f) => f.properties.kind === 'corridor') ?? [],
  );

  private readonly bounds = computed(() => {
    const points: [number, number][] = [];
    for (const feature of this.network()?.features ?? []) {
      if (feature.geometry.type === 'Point') {
        points.push(feature.geometry.coordinates);
      } else {
        points.push(...feature.geometry.coordinates);
      }
    }
    const lons = points.map((p) => p[0]);
    const lats = points.map((p) => p[1]);
    const minLon = Math.min(...lons);
    const maxLon = Math.max(...lons);
    const minLat = Math.min(...lats);
    const maxLat = Math.max(...lats);
    return { minLon, maxLon, minLat, maxLat };
  });

  x(feature: GeoFeature): number {
    const b = this.bounds();
    const [lon] = this.firstPoint(feature);
    const pad = 0.12;
    return ((lon - b.minLon) / (b.maxLon - b.minLon || 1)) * (800 * (1 - pad * 2)) + 800 * pad;
  }

  y(feature: GeoFeature): number {
    const b = this.bounds();
    const [, lat] = this.firstPoint(feature);
    const pad = 0.12;
    return ((b.maxLat - lat) / (b.maxLat - b.minLat || 1)) * (520 * (1 - pad * 2)) + 520 * pad;
  }

  linePoints(feature: GeoFeature): string {
    if (feature.geometry.type !== 'LineString') {
      return '';
    }
    const b = this.bounds();
    const pad = 0.12;
    return feature.geometry.coordinates
      .map(([lon, lat]) => {
        const x = ((lon - b.minLon) / (b.maxLon - b.minLon || 1)) * (800 * (1 - pad * 2)) + 800 * pad;
        const y = ((b.maxLat - lat) / (b.maxLat - b.minLat || 1)) * (520 * (1 - pad * 2)) + 520 * pad;
        return `${x.toFixed(1)},${y.toFixed(1)}`;
      })
      .join(' ');
  }

  labelX(feature: GeoFeature): number {
    return this.x(feature) + 10;
  }

  labelY(feature: GeoFeature): number {
    return this.y(feature) - 10;
  }

  radius(feature: GeoFeature): number {
    const kpi = this.kpiFor(feature.properties.id);
    if (!kpi) {
      return 9;
    }
    return 9 + Math.min(kpi.maxQueue_veh, 20) * 0.55;
  }

  color(feature: GeoFeature): string {
    const kpi = this.kpiFor(feature.properties.id);
    if (!kpi) {
      return '#3ddc97';
    }
    if (kpi.avgDelay_s >= 20) {
      return '#ef476f';
    }
    if (kpi.avgDelay_s >= 10) {
      return '#ffb703';
    }
    return '#3ddc97';
  }

  kpiFor(nodeId: string): NodeKPI | undefined {
    return this.nodeKpis().find((k) => k.nodeId === nodeId);
  }

  private firstPoint(feature: GeoFeature): [number, number] {
    if (feature.geometry.type === 'Point') {
      return feature.geometry.coordinates;
    }
    return feature.geometry.coordinates[0] ?? [0, 0];
  }
}
