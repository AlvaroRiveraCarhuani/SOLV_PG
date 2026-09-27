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
  styleUrl: './load-history-chart.component.scss',
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
