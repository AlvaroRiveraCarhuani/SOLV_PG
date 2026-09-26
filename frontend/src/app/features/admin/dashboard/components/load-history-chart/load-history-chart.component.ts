import { Component, input, computed, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { LoadSnapshot } from '@core/models/admin.model';
import { LucideActivity } from '@lucide/angular';

@Component({
  selector: 'load-history-chart',
  standalone: true,
  imports: [CommonModule, LucideActivity],
  template: `
    <div class="chart-card">
      <div class="chart-header">
        <div class="header-left">
          <svg lucideActivity class="header-icon"></svg>
          <div class="title-group">
            <div class="title-row">
              <h3 class="chart-title">Consumo de Recursos en Tiempo Real</h3>
              <span class="badge-live">
                <span class="live-dot"></span>
                ÚLTIMOS 60 MINUTOS
              </span>
            </div>
            <span class="chart-subtitle">Telemetría de memoria RAM y procesamiento vCPU del servidor físico</span>
          </div>
        </div>

        <div class="chart-legend">
          <div class="legend-item">
            <span class="legend-indicator ram"></span>
            <span class="legend-label">Memoria RAM</span>
            <span class="legend-value">{{ currentRAM() }}%</span>
          </div>
          <div class="legend-item">
            <span class="legend-indicator cpu"></span>
            <span class="legend-label">Carga vCPU</span>
            <span class="legend-value">{{ currentCPU() }}%</span>
          </div>
          <div class="legend-item">
            <span class="legend-indicator threshold"></span>
            <span class="legend-label">Límite QoS (85%)</span>
          </div>
        </div>
      </div>

      <!-- Contenedor Principal del Gráfico con Eje Y HTML desacoplado -->
      <div class="chart-body-wrapper">
        <!-- Eje Y en HTML: tipografía nítida sin distorsión vectorial -->
        <div class="y-axis">
          <span>100%</span>
          <span>75%</span>
          <span>50%</span>
          <span>25%</span>
          <span>0%</span>
        </div>

        <!-- Área Gráfica: SVG Geométrico + Overlays HTML -->
        <div class="chart-plot-area" (mouseleave)="hoveredIndex.set(null)">
          <!-- Badge de Umbral Crítico posicionado con HTML en el 85% -->
          <div class="threshold-badge">
            Umbral Crítico (85%)
          </div>

          <svg viewBox="0 0 800 160" preserveAspectRatio="none" class="chart-svg">
            <!-- Grid Lines Horizontales (0%, 25%, 50%, 75%, 100%) -->
            <line x1="0" y1="0" x2="800" y2="0" class="grid-line" />
            <line x1="0" y1="40" x2="800" y2="40" class="grid-line" />
            <line x1="0" y1="80" x2="800" y2="80" class="grid-line" />
            <line x1="0" y1="120" x2="800" y2="120" class="grid-line" />
            <line x1="0" y1="159" x2="800" y2="159" class="grid-line baseline" />

            <!-- Línea de Umbral QoS (85% = Y: 160 - 0.85*160 = 24) -->
            <line x1="0" y1="24" x2="800" y2="24" class="threshold-line" />

            <!-- Curvas de Carga -->
            @if (cpuLinePoints()) {
              <polyline [attr.points]="cpuLinePoints()" class="chart-line cpu-line" />
            }
            @if (ramLinePoints()) {
              <polyline [attr.points]="ramLinePoints()" class="chart-line ram-line" />
            }

            <!-- Zonas de Hover Interactivas -->
            @for (pt of points(); track $index) {
              <rect 
                [attr.x]="pt.x - 5" 
                y="0" 
                width="10" 
                height="160" 
                class="hover-trigger"
                (mouseenter)="hoveredIndex.set($index)"
              />
            }

            <!-- Marcador Activo en Hover -->
            @if (activePoint(); as ap) {
              <line [attr.x1]="ap.x" y1="0" [attr.x2]="ap.x" y2="160" class="cursor-line" />
              <circle [attr.cx]="ap.x" [attr.cy]="ap.ramY" r="4.5" class="dot-marker ram-dot" />
              <circle [attr.cx]="ap.x" [attr.cy]="ap.cpuY" r="4" class="dot-marker cpu-dot" />
            }
          </svg>

          <!-- Tooltip Flotante de Ingeniería -->
          @if (activeSnapshot(); as snap) {
            <div 
              class="chart-tooltip"
              [style.left.%]="tooltipPercent()"
              [style.top.px]="15"
            >
              <div class="tooltip-time">{{ snap.timestamp }} (Hace {{ 59 - (hoveredIndex() ?? 0) }}m)</div>
              <div class="tooltip-row">
                <span class="dot ram-bg"></span>
                <span class="tooltip-name">RAM Host:</span>
                <span class="tooltip-val">{{ snap.ram_percent }}% ({{ snap.ram_used_gb }} GB)</span>
              </div>
              <div class="tooltip-row">
                <span class="dot cpu-bg"></span>
                <span class="tooltip-name">vCPU:</span>
                <span class="tooltip-val">{{ snap.cpu_percent }}%</span>
              </div>
            </div>
          }
        </div>
      </div>

      <!-- Eje X Inferior (Marcas temporales) perfectamente alineado -->
      <div class="chart-timeline">
        <span>-60 min</span>
        <span>-45 min</span>
        <span>-30 min</span>
        <span>-15 min</span>
        <span>Ahora</span>
      </div>
    </div>
  `,
  styles: [`
    .chart-card {
      background-color: var(--bg-surface, #FFFFFF);
      border: 1px solid var(--border-subtle, #E2E8F0);
      border-radius: var(--radius-lg, 8px);
      padding: var(--space-4, 16px);
      display: flex;
      flex-direction: column;
      gap: var(--space-3, 12px);
    }

    .chart-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      flex-wrap: wrap;
      gap: var(--space-3, 12px);
    }

    .header-left {
      display: flex;
      align-items: center;
      gap: var(--space-2-5, 10px);
    }

    .header-icon {
      width: 18px;
      height: 18px;
      color: var(--text-secondary, #64748B);
    }

    .title-group {
      display: flex;
      flex-direction: column;
      gap: 2px;
    }

    .title-row {
      display: flex;
      align-items: center;
      gap: var(--space-2, 8px);
    }

    .chart-title {
      font-size: var(--font-size-sm, 14px);
      font-weight: 700;
      color: var(--text-primary, #0F172A);
      margin: 0;
    }

    .badge-live {
      display: inline-flex;
      align-items: center;
      gap: 5px;
      font-size: 10px;
      font-weight: 600;
      letter-spacing: 0.5px;
      color: var(--text-secondary, #64748B);
      background-color: var(--bg-canvas, #F1F5F9);
      padding: 1px 6px;
      border-radius: var(--radius-sm, 4px);
      border: 1px solid var(--border-subtle, #E2E8F0);
    }

    .live-dot {
      width: 5px;
      height: 5px;
      border-radius: 50%;
      background-color: var(--state-running, #16A34A);
    }

    .chart-subtitle {
      font-size: 11px;
      color: var(--text-secondary, #64748B);
    }

    .chart-legend {
      display: flex;
      align-items: center;
      gap: var(--space-4, 16px);
      flex-wrap: wrap;
    }

    .legend-item {
      display: flex;
      align-items: center;
      gap: 6px;
      font-size: 11px;
      color: var(--text-secondary, #64748B);
    }

    .legend-indicator {
      width: 8px;
      height: 8px;
      border-radius: 2px;

      &.ram { background-color: var(--tenant-primary, #2563EB); }
      &.cpu { background-color: #64748B; }
      &.threshold { 
        height: 2px; 
        background-color: var(--verdict-wa, #DC2626); 
        border-top: 1px dashed var(--verdict-wa, #DC2626); 
      }
    }

    .legend-value {
      font-weight: 700;
      color: var(--text-primary, #0F172A);
      font-family: var(--font-mono, monospace);
      font-size: 11px;
    }

    /* Contenedor macro: Eje Y + Área gráfica */
    .chart-body-wrapper {
      display: flex;
      gap: 8px;
      height: 160px;
      position: relative;
      margin-top: 6px;
    }

    /* Eje Y HTML libre de distorsión */
    .y-axis {
      display: flex;
      flex-direction: column;
      justify-content: space-between;
      width: 32px;
      height: 160px;
      text-align: right;
      padding: 0 4px 0 0;
      user-select: none;

      span {
        font-size: 10px;
        font-family: var(--font-mono, monospace);
        color: var(--text-muted, #94A3B8);
        line-height: 1;
        transform: translateY(-50%);

        &:first-child { transform: translateY(0); }
        &:last-child { transform: translateY(-100%); }
      }
    }

    /* Área interactiva del gráfico */
    .chart-plot-area {
      position: relative;
      flex: 1;
      height: 160px;
    }

    .chart-svg {
      width: 100%;
      height: 100%;
      overflow: visible;
      display: block;
    }

    /* Badge de Umbral Crítico: HTML con tipografía perfecta */
    .threshold-badge {
      position: absolute;
      top: 15%;
      right: 8px;
      transform: translateY(-100%);
      font-size: 10px;
      font-weight: 600;
      color: var(--verdict-wa, #DC2626);
      background-color: #FEF2F2;
      border: 1px solid #FECACA;
      padding: 1px 6px;
      border-radius: 4px;
      pointer-events: none;
      z-index: 5;
      letter-spacing: 0.2px;
    }

    .grid-line {
      stroke: var(--border-subtle, #E2E8F0);
      stroke-width: 1;
      stroke-dasharray: 4 4;

      &.baseline {
        stroke-dasharray: none;
        stroke: var(--border-subtle, #CBD5E1);
      }
    }

    .threshold-line {
      stroke: var(--verdict-wa, #DC2626);
      stroke-width: 1.25;
      stroke-dasharray: 5 3;
      opacity: 0.85;
    }

    .chart-line {
      fill: none;
      stroke-linecap: round;
      stroke-linejoin: round;

      &.ram-line {
        stroke: var(--tenant-primary, #2563EB);
        stroke-width: 2.25;
      }

      &.cpu-line {
        stroke: #64748B;
        stroke-width: 1.75;
      }
    }

    .hover-trigger {
      fill: transparent;
      cursor: crosshair;
    }

    .cursor-line {
      stroke: var(--border-subtle, #94A3B8);
      stroke-width: 1;
      stroke-dasharray: 2 2;
    }

    .dot-marker {
      stroke-width: 2;
      stroke: #FFFFFF;

      &.ram-dot { fill: var(--tenant-primary, #2563EB); }
      &.cpu-dot { fill: #64748B; }
    }

    .chart-tooltip {
      position: absolute;
      background-color: var(--bg-surface, #FFFFFF);
      border: 1px solid var(--border-subtle, #CBD5E1);
      color: var(--text-primary, #0F172A);
      padding: 6px 10px;
      border-radius: var(--radius-md, 6px);
      box-shadow: 0 4px 12px -2px rgba(0, 0, 0, 0.08);
      font-size: 11px;
      pointer-events: none;
      transform: translateX(-50%);
      z-index: 20;
      white-space: nowrap;
      display: flex;
      flex-direction: column;
      gap: 2px;
    }

    .tooltip-time {
      font-size: 10px;
      color: var(--text-muted, #94A3B8);
      font-family: var(--font-mono, monospace);
      margin-bottom: 2px;
      border-bottom: 1px solid var(--border-subtle, #F1F5F9);
      padding-bottom: 2px;
    }

    .tooltip-row {
      display: flex;
      align-items: center;
      gap: 6px;
    }

    .dot {
      width: 6px;
      height: 6px;
      border-radius: 50%;

      &.ram-bg { background-color: var(--tenant-primary, #2563EB); }
      &.cpu-bg { background-color: #64748B; }
    }

    .tooltip-name {
      color: var(--text-secondary, #64748B);
    }

    .tooltip-val {
      font-weight: 700;
      color: var(--text-primary, #0F172A);
      font-family: var(--font-mono, monospace);
    }

    .chart-timeline {
      display: flex;
      justify-content: space-between;
      margin-left: 40px; /* Alineado con el área de graficado (32px eje Y + 8px gap) */
      font-size: 10px;
      font-weight: 500;
      color: var(--text-muted, #94A3B8);
      font-family: var(--font-mono, monospace);
      border-top: 1px solid var(--border-subtle, #F1F5F9);
      padding-top: 6px;
    }
  `]
})
export class LoadHistoryChartComponent {
  history = input<LoadSnapshot[]>([]);

  hoveredIndex = signal<number | null>(null);

  currentRAM = computed(() => {
    const list = this.history();
    if (!list || list.length === 0) return 0;
    return Math.round(list[list.length - 1].ram_percent);
  });

  currentCPU = computed(() => {
    const list = this.history();
    if (!list || list.length === 0) return 0;
    return Math.round(list[list.length - 1].cpu_percent);
  });

  points = computed(() => {
    const list = this.history() || [];
    if (list.length === 0) return [];

    const startX = 0;
    const endX = 800;
    const baseY = 160;
    const height = 160;

    const count = list.length;
    const step = (endX - startX) / (count > 1 ? count - 1 : 1);

    return list.map((item, idx) => {
      const x = startX + idx * step;
      const ramY = baseY - (Math.min(item.ram_percent, 100) / 100) * height;
      const cpuY = baseY - (Math.min(item.cpu_percent, 100) / 100) * height;
      return { x, ramY, cpuY, item };
    });
  });

  ramLinePoints = computed(() => {
    return this.points().map(p => `${p.x.toFixed(1)},${p.ramY.toFixed(1)}`).join(' ');
  });

  cpuLinePoints = computed(() => {
    return this.points().map(p => `${p.x.toFixed(1)},${p.cpuY.toFixed(1)}`).join(' ');
  });

  activePoint = computed(() => {
    const idx = this.hoveredIndex();
    if (idx === null) return null;
    const pts = this.points();
    return pts[idx] || null;
  });

  activeSnapshot = computed(() => {
    const ap = this.activePoint();
    return ap ? ap.item : null;
  });

  tooltipPercent = computed(() => {
    const ap = this.activePoint();
    if (!ap) return 0;
    return (ap.x / 800) * 100;
  });
}
