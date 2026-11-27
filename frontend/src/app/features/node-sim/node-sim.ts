import { DecimalPipe } from '@angular/common';
import {
  Component,
  ElementRef,
  OnDestroy,
  afterNextRender,
  input,
  viewChild,
} from '@angular/core';
import { Mix, NodeKPI, NodePlan } from '../../core/api';

interface Vehicle {
  x: number;
  v: number;
  length: number;
  v0: number;
  color: string;
}

const WIDTH = 900;
const HEIGHT = 260;
const STOP_X = 610;
const ROAD_TOP = 96;
const ROAD_BOTTOM = 184;

@Component({
  selector: 'app-node-sim',
  imports: [DecimalPipe],
  template: `
    <div class="head">
      <div>
        <h3>Nodo {{ node()?.id ?? '—' }}</h3>
        <p>{{ label() }}</p>
      </div>
      <div class="stats">
        @if (kpi(); as k) {
          <span><b>{{ k.avgDelay_s | number: '1.1-1' }}</b> s de espera</span>
          <span><b>{{ k.maxQueue_veh | number: '1.1-1' }}</b> veh de cola</span>
          <span><b>{{ k.stopsPerVeh | number: '1.2-2' }}</b> paradas/veh</span>
        } @else {
          <span>Sin corrida todavía</span>
        }
      </div>
    </div>
    <canvas #canvas [width]="width" [height]="height"></canvas>
    <div class="foot">
      <span>Modelo IDM · 60 fps · semáforo como vehículo virtual</span>
      <span>{{ cycleLabel() }}</span>
    </div>
  `,
  styles: `
    :host {
      display: block;
      border: 1px solid #1e3145;
      border-radius: 16px;
      background: #0b1420;
      overflow: hidden;
    }
    .head {
      display: flex;
      align-items: flex-start;
      justify-content: space-between;
      gap: 16px;
      padding: 14px 18px;
      border-bottom: 1px solid #1e3145;
    }
    h3 {
      margin: 0;
      font-size: 15px;
      color: #e6f1ff;
    }
    p {
      margin: 3px 0 0;
      color: #6d8aa6;
      font-size: 12px;
      max-width: 420px;
    }
    .stats {
      display: flex;
      gap: 18px;
      color: #8aa4bd;
      font-size: 12px;
      font-family: ui-monospace, 'Cascadia Code', monospace;
      text-align: right;
    }
    .stats b {
      color: #57c4a8;
      font-size: 15px;
    }
    canvas {
      display: block;
      width: 100%;
      height: auto;
    }
    .foot {
      display: flex;
      justify-content: space-between;
      padding: 9px 18px;
      border-top: 1px solid #1e3145;
      color: #6d8aa6;
      font-size: 11px;
      font-family: ui-monospace, 'Cascadia Code', monospace;
    }
  `,
})
export class NodeSim implements OnDestroy {
  readonly node = input<NodePlan | null>(null);
  readonly kpi = input<NodeKPI | null>(null);
  readonly flow = input<number>(0);
  readonly mix = input<Mix | null>(null);
  readonly label = input<string>('');

  readonly width = WIDTH;
  readonly height = HEIGHT;
  private readonly canvas = viewChild.required<ElementRef<HTMLCanvasElement>>('canvas');

  private ctx: CanvasRenderingContext2D | null = null;
  private vehicles: Vehicle[] = [];
  private simTime = 0;
  private spawnAcc = 0;
  private lastId = '';
  private lastFrame = 0;
  private raf = 0;

  constructor() {
    afterNextRender(() => {
      this.ctx = this.canvas().nativeElement.getContext('2d');
      this.lastFrame = performance.now();
      this.raf = requestAnimationFrame(this.frame);
    });
  }

  ngOnDestroy(): void {
    cancelAnimationFrame(this.raf);
  }

  cycleLabel(): string {
    const node = this.node();
    if (!node) {
      return '';
    }
    return `Ciclo ${node.cycle_s.toFixed(0)} s · verde ${node.phases[0].green_s.toFixed(0)} s`;
  }

  private readonly frame = (now: number): void => {
    const dt = Math.min((now - this.lastFrame) / 1000, 0.05);
    this.lastFrame = now;
    this.tick(dt);
    this.draw();
    this.raf = requestAnimationFrame(this.frame);
  };

  private tick(dt: number): void {
    const node = this.node();
    if (!node) {
      return;
    }
    if (node.id !== this.lastId) {
      this.lastId = node.id;
      this.vehicles = [];
      this.simTime = 0;
      this.spawnAcc = 0;
      this.seedVehicles();
    }
    this.simTime += dt;
    const green = this.isGreen(node);
    const flow = this.flow();
    if (flow > 0) {
      this.spawnAcc += dt;
      const headway = 3600 / flow;
      const tail = this.vehicles[this.vehicles.length - 1];
      if (this.spawnAcc >= headway && (!tail || tail.x > 70)) {
        this.spawnAcc = 0;
        this.vehicles.push(this.makeVehicle(-40));
      }
    }

    for (let i = 0; i < this.vehicles.length; i++) {
      const v = this.vehicles[i];
      const leader = i === 0 ? null : this.vehicles[i - 1];
      const leaderX = leader ? leader.x - leader.length : green ? Infinity : STOP_X;
      const gap = leaderX - v.x;
      const dv = leader ? v.v - leader.v : v.v;
      const a = 80;
      const b = 80;
      const T = 1.1;
      const s0 = 13;
      const sStar = s0 + Math.max(0, v.v * T + (v.v * dv) / (2 * Math.sqrt(a * b)));
      const free = 1 - Math.pow(v.v / v.v0, 4);
      const interaction = Number.isFinite(leaderX) ? Math.pow(sStar / Math.max(gap, 1), 2) : 0;
      const acc = Math.max(-240, Math.min(a, a * (free - interaction)));
      v.v = Math.max(0, v.v + acc * dt);
      v.x += v.v * dt;
    }
    this.vehicles = this.vehicles.filter((v) => v.x < WIDTH + 80);
  }

  private draw(): void {
    const ctx = this.ctx;
    const node = this.node();
    if (!ctx || !node) {
      return;
    }
    ctx.clearRect(0, 0, WIDTH, HEIGHT);
    ctx.fillStyle = '#0b1420';
    ctx.fillRect(0, 0, WIDTH, HEIGHT);

    ctx.strokeStyle = '#14243a';
    ctx.lineWidth = 1;
    for (let x = 0; x < WIDTH; x += 40) {
      ctx.beginPath();
      ctx.moveTo(x, 0);
      ctx.lineTo(x, HEIGHT);
      ctx.stroke();
    }

    ctx.fillStyle = '#101e2e';
    ctx.fillRect(0, ROAD_TOP, WIDTH, ROAD_BOTTOM - ROAD_TOP);
    ctx.strokeStyle = '#28415a';
    ctx.setLineDash([18, 14]);
    ctx.beginPath();
    ctx.moveTo(0, (ROAD_TOP + ROAD_BOTTOM) / 2);
    ctx.lineTo(WIDTH, (ROAD_TOP + ROAD_BOTTOM) / 2);
    ctx.stroke();
    ctx.setLineDash([]);

    ctx.fillStyle = '#16283c';
    ctx.fillRect(STOP_X, ROAD_TOP - 8, 96, ROAD_BOTTOM - ROAD_TOP + 16);
    ctx.strokeStyle = '#2f4a63';
    ctx.strokeRect(STOP_X, ROAD_TOP - 8, 96, ROAD_BOTTOM - ROAD_TOP + 16);

    ctx.strokeStyle = '#e6f1ff';
    ctx.lineWidth = 3;
    ctx.beginPath();
    ctx.moveTo(STOP_X, ROAD_TOP);
    ctx.lineTo(STOP_X, ROAD_BOTTOM);
    ctx.stroke();

    const phase = this.phase(node);
    this.drawSignal(ctx, STOP_X + 48, 44, phase);

    for (const v of this.vehicles) {
      ctx.fillStyle = v.color;
      this.roundRect(ctx, v.x - v.length, 122, v.length, 28, 6);
      ctx.fill();
    }

    ctx.fillStyle = '#6d8aa6';
    ctx.font = '12px ui-monospace, monospace';
    ctx.fillText(`cola ${this.queueLength().toFixed(1)} veh`, 18, 32);
    ctx.fillText(`${phase}`, WIDTH - 150, 32);
  }

  private drawSignal(ctx: CanvasRenderingContext2D, x: number, y: number, phase: string): void {
    ctx.fillStyle = '#0b1420';
    this.roundRect(ctx, x - 18, y - 18, 36, 92, 10);
    ctx.fill();
    ctx.strokeStyle = '#2f4a63';
    ctx.stroke();
    const colors: [string, boolean][] = [
      ['#ef476f', phase === 'ROJO'],
      ['#ffb703', phase === 'ÁMBAR'],
      ['#3ddc97', phase === 'VERDE'],
    ];
    colors.forEach(([color, on], i) => {
      ctx.beginPath();
      ctx.arc(x, y + i * 28, 9, 0, Math.PI * 2);
      ctx.fillStyle = on ? color : '#1b2a3a';
      ctx.fill();
      if (on) {
        ctx.shadowColor = color;
        ctx.shadowBlur = 14;
        ctx.fill();
        ctx.shadowBlur = 0;
      }
    });
  }

  private roundRect(
    ctx: CanvasRenderingContext2D,
    x: number,
    y: number,
    w: number,
    h: number,
    r: number,
  ): void {
    ctx.beginPath();
    ctx.moveTo(x + r, y);
    ctx.arcTo(x + w, y, x + w, y + h, r);
    ctx.arcTo(x + w, y + h, x, y + h, r);
    ctx.arcTo(x, y + h, x, y, r);
    ctx.arcTo(x, y, x + w, y, r);
    ctx.closePath();
  }

  private phase(node: NodePlan): 'VERDE' | 'ÁMBAR' | 'ROJO' {
    const local = this.simTime % node.cycle_s;
    const green = node.phases[0].green_s;
    const amber = node.phases[0].amber_s;
    if (local < green) {
      return 'VERDE';
    }
    if (local < green + amber) {
      return 'ÁMBAR';
    }
    return 'ROJO';
  }

  private isGreen(node: NodePlan): boolean {
    return this.phase(node) === 'VERDE';
  }

  private queueLength(): number {
    return this.vehicles.filter((v) => v.v < 8).length;
  }

  private seedVehicles(): void {
    for (let i = 0; i < 4; i++) {
      const v = this.makeVehicle(40 + i * 95);
      v.v = 60;
      this.vehicles.push(v);
    }
  }

  private makeVehicle(x: number): Vehicle {
    const mix = this.mix() ?? { L5: 0.35, M1: 0.35, M2: 0.2, N: 0.1 };
    const roll = Math.random();
    if (roll < mix.L5) {
      return { x, v: 90, length: 24, v0: 105, color: '#ffd166' };
    }
    if (roll < mix.L5 + mix.M1) {
      return { x, v: 110, length: 34, v0: 150, color: '#4cc9f0' };
    }
    if (roll < mix.L5 + mix.M1 + mix.M2) {
      return { x, v: 90, length: 44, v0: 135, color: '#ef476f' };
    }
    return { x, v: 85, length: 48, v0: 130, color: '#adb5bd' };
  }
}
