import { Component, input, computed, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { LoadSnapshot } from '@core/models/admin.model';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { formatSolvDate, parseDateValue } from '@shared/pipes/date-text.pipe';
import { LucideActivity } from '@lucide/angular';

@Component({
  selector: 'load-history-chart',
  standalone: true,
  imports: [CommonModule, MachineDataDirective, LucideActivity],
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
            <span class="legend-value" machineData>{{ currentRAM() }}%</span>
          </div>
          <div class="legend-item">
            <span class="legend-indicator cpu"></span>
            <span class="legend-label">Carga vCPU</span>
            <span class="legend-value" machineData>{{ currentCPU() }}%</span>
          </div>
          <div class="legend-item">
            <span class="legend-indicator threshold"></span>
            <span class="legend-label">Límite QoS ({{ threshold() }}%)</span>
          </div>
        </div>
      </div>

      <!-- Contenedor Principal del Gráfico con Eje Y HTML desacoplado -->
      @if (hasEnoughData()) {
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
          <!-- Badge de Umbral Crítico posicionado con el techo real de la política -->
          <div class="threshold-badge">
            Umbral Crítico ({{ threshold() }}%)
          </div>

          <svg viewBox="0 0 800 160" preserveAspectRatio="none" class="chart-svg">
            <!-- Grid Lines Horizontales (0%, 25%, 50%, 75%, 100%) -->
            <line x1="0" y1="0" x2="800" y2="0" class="grid-line" />
            <line x1="0" y1="40" x2="800" y2="40" class="grid-line" />
            <line x1="0" y1="80" x2="800" y2="80" class="grid-line" />
            <line x1="0" y1="120" x2="800" y2="120" class="grid-line" />
            <line x1="0" y1="159" x2="800" y2="159" class="grid-line baseline" />

            <!-- Línea de Umbral QoS: la Y sale del techo de política vigente,
                 no de un 85 hardcodeado: 160 - (techo/100)*160 -->
            <line x1="0" [attr.y1]="thresholdY()" x2="800" [attr.y2]="thresholdY()" class="threshold-line" />

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
              <div class="tooltip-time">
                <time [attr.datetime]="snap.timestamp">{{ formatTime(snap.timestamp) }}</time>
                (Hace <span machineData>{{ minutesAgo(snap.timestamp) }}</span>m)
              </div>
              <div class="tooltip-row">
                <span class="dot ram-bg"></span>
                <span class="tooltip-name">RAM Host:</span>
                <span class="tooltip-val"><span machineData>{{ snap.ram_percent }}%</span> (<span machineData>{{ snap.ram_used_gb }} GB</span>)</span>
              </div>
              <div class="tooltip-row">
                <span class="dot cpu-bg"></span>
                <span class="tooltip-name">vCPU:</span>
                <span class="tooltip-val" machineData>{{ snap.cpu_percent }}%</span>
              </div>
            </div>
          }
        </div>
      </div>

      <!-- Eje X Inferior: marcas proporcionales al rango real de los samples
           (el span cubre exactamente entre el primer y el último sample) -->
      <div class="chart-timeline">
        @for (label of xAxisLabels(); track $index) {
          <span>{{ label }}</span>
        }
      </div>
      } @else {
      <!-- Estado vacío honesto: sin al menos 2 samples reales no hay línea que dibujar. -->
      <div class="chart-empty">
        <svg lucideActivity class="empty-icon"></svg>
        <span>Recopilando telemetría: el historial se llena con muestras reales cada minuto.</span>
      </div>
      }
    </div>
  `,
  styleUrl: './load-history-chart.component.scss',
})
export class LoadHistoryChartComponent {
  history = input<LoadSnapshot[]>([]);
  // Techo de seguridad vigente: lo pasa el dashboard desde la política del
  // tenant (antes eran tres 85 hardcodeados en esta plantilla).
  threshold = input<number>(85);

  hoveredIndex = signal<number | null>(null);

  // Con menos de 2 puntos reales el gráfico muestra su estado vacío explícito:
  // no se dibuja una línea con un solo sample ni se inventa historia.
  hasEnoughData = computed(() => (this.history()?.length ?? 0) >= 2);

  thresholdY = computed(() => 160 - (Math.min(Math.max(this.threshold(), 0), 100) / 100) * 160);

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

  // Marcas del eje X proporcionales al rango temporal real de los samples: el
  // span total es (último - primer timestamp) y cada marca cae en su fracción.
  xAxisLabels = computed<string[]>(() => {
    const list = this.history() ?? [];
    if (list.length < 2) {
      return ['-60 min', '-45 min', '-30 min', '-15 min', 'Ahora'];
    }
    const first = parseDateValue(list[0].timestamp);
    const last = parseDateValue(list[list.length - 1].timestamp);
    if (!first || !last || last.getTime() <= first.getTime()) {
      return ['-60 min', '-45 min', '-30 min', '-15 min', 'Ahora'];
    }
    const spanMin = Math.max(1, Math.round((last.getTime() - first.getTime()) / 60000));
    const fractions = [0, 0.25, 0.5, 0.75, 1];
    return fractions.map((f) => {
      if (f === 1) return 'Ahora';
      const ago = Math.round(spanMin * (1 - f));
      return `-${ago} min`;
    });
  });

  // Delta real desde el timestamp del sample, no un "59 - índice" inventado.
  minutesAgo(timestamp: string): number {
    const d = parseDateValue(timestamp);
    if (!d) return 0;
    const minutes = Math.round((Date.now() - d.getTime()) / 60000);
    return Math.max(0, minutes);
  }

  formatTime(timestamp: string): string {
    return formatSolvDate(timestamp, 'time') ?? '—';
  }

  points = computed(() => {
    const list = this.history() || [];
    if (list.length === 0) return [];

    const startX = 0;
    const endX = 800;
    const baseY = 160;
    const height = 160;

    const count = list.length;
    const step = (endX - startX) / (count > 1 ? count - 1 : 1);

    // clampPercent acota 0..100: un sample medido fuera de rango no debe
    // dibujarse por debajo del baseline (negativo) ni por encima del techo.
    const clampPercent = (v: number) => Math.min(Math.max(v, 0), 100);

    return list.map((item, idx) => {
      const x = startX + idx * step;
      const ramY = baseY - (clampPercent(item.ram_percent) / 100) * height;
      const cpuY = baseY - (clampPercent(item.cpu_percent) / 100) * height;
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
