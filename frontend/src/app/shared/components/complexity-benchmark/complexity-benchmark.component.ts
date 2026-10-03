import { 
  Component, 
  Input, 
  Output, 
  EventEmitter, 
  signal, 
  computed,
  ChangeDetectionStrategy 
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { 
  LucideTrendingUp, 
  LucideClock, 
  LucideHardDrive, 
  LucidePlay, 
  LucideCheckCircle, 
  LucideAlertTriangle, 
  LucideInfo, 
  LucideDownload,
  LucideX,
  LucideZap
} from '@lucide/angular';
import { BenchmarkReport, BenchmarkSample, ComplexityClass } from '../../../features/teacher/models/teacher.models';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';

interface ChartPoint {
  x: number;
  y: number;
  sample: BenchmarkSample;
  value: number;
  label: string;
}

@Component({
  selector: 'complexity-benchmark',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    CommonModule,
    FormsModule,
    LucideTrendingUp,
    LucideClock,
    LucideHardDrive,
    LucidePlay,
    LucideCheckCircle,
    LucideAlertTriangle,
    LucideInfo,
    LucideDownload,
    LucideX,
    LucideZap,
    MachineDataDirective
  ],
  templateUrl: './complexity-benchmark.component.html',
  styleUrl: './complexity-benchmark.component.scss'
})
export class ComplexityBenchmarkComponent {
  @Input() report: BenchmarkReport | null = null;
  @Input() submissionId: string = '';
  @Input() isLoading: boolean = false;
  @Input() isModal: boolean = false;

  @Output() runBenchmark = new EventEmitter<'small' | 'standard' | 'stress'>();
  @Output() close = new EventEmitter<void>();

  readonly activeMetric = signal<'time' | 'memory'>('time');
  readonly scaleType = signal<'linear' | 'log'>('linear');
  readonly showTheoreticalBounds = signal<boolean>(true);
  readonly selectedPreset = signal<'small' | 'standard' | 'stress'>('standard');
  readonly hoveredPoint = signal<ChartPoint | null>(null);

  // SVG Chart Dimensions
  readonly chartWidth = 600;
  readonly chartHeight = 240;
  readonly padding = { top: 20, right: 30, bottom: 40, left: 60 };

  readonly samples = computed<BenchmarkSample[]>(() => {
    return this.report?.samples || [];
  });

  readonly hasData = computed<boolean>(() => {
    return (this.samples()?.length || 0) > 0;
  });

  readonly chartPoints = computed<ChartPoint[]>(() => {
    const list = this.samples();
    if (list.length === 0) return [];

    const isTime = this.activeMetric() === 'time';
    const isLog = this.scaleType() === 'log';

    const rawX = list.map(s => s.input_size);
    const rawY = list.map(s => isTime ? s.execution_time_ms : s.memory_used_kb);

    const minX = Math.min(...rawX);
    const maxX = Math.max(...rawX);
    const minY = 0;
    const maxY = Math.max(...rawY, 1);

    const plotW = this.chartWidth - this.padding.left - this.padding.right;
    const plotH = this.chartHeight - this.padding.top - this.padding.bottom;

    return list.map(s => {
      const valY = isTime ? s.execution_time_ms : s.memory_used_kb;
      let normX = 0;
      let normY = 0;

      if (isLog) {
        const logMinX = Math.log10(Math.max(1, minX));
        const logMaxX = Math.log10(Math.max(10, maxX));
        const logX = Math.log10(Math.max(1, s.input_size));
        normX = (logX - logMinX) / (logMaxX - logMinX || 1);

        const logMaxY = Math.log10(Math.max(10, maxY));
        const logY = Math.log10(Math.max(0.1, valY));
        normY = logY / (logMaxY || 1);
      } else {
        normX = (s.input_size - minX) / (maxX - minX || 1);
        normY = (valY - minY) / (maxY - minY || 1);
      }

      normX = Math.max(0, Math.min(1, normX));
      normY = Math.max(0, Math.min(1, normY));

      const x = this.padding.left + normX * plotW;
      const y = this.padding.top + (1 - normY) * plotH;

      return {
        x: Math.round(x * 10) / 10,
        y: Math.round(y * 10) / 10,
        sample: s,
        value: valY,
        label: isTime ? `${valY.toFixed(2)} ms` : `${valY} KB`
      };
    });
  });

  readonly empiricalPath = computed<string>(() => {
    const pts = this.chartPoints();
    if (pts.length === 0) return '';
    return pts.reduce((acc, p, idx) => {
      return idx === 0 ? `M ${p.x} ${p.y}` : `${acc} L ${p.x} ${p.y}`;
    }, '');
  });

  readonly areaPath = computed<string>(() => {
    const pts = this.chartPoints();
    if (pts.length === 0) return '';
    const bottomY = this.chartHeight - this.padding.bottom;
    const startX = pts[0].x;
    const endX = pts[pts.length - 1].x;
    const linePath = pts.reduce((acc, p, idx) => {
      return idx === 0 ? `M ${p.x} ${p.y}` : `${acc} L ${p.x} ${p.y}`;
    }, '');
    return `${linePath} L ${endX} ${bottomY} L ${startX} ${bottomY} Z`;
  });

  readonly theoreticalCurves = computed(() => {
    const pts = this.chartPoints();
    if (pts.length < 2 || !this.showTheoreticalBounds()) return [];

    const plotW = this.chartWidth - this.padding.left - this.padding.right;
    const plotH = this.chartHeight - this.padding.top - this.padding.bottom;
    const bottomY = this.chartHeight - this.padding.bottom;

    const nValues = this.samples().map(s => s.input_size);
    const nMax = Math.max(...nValues);
    const nMin = Math.min(...nValues);

    // Generate reference O(N) and O(N^2) paths scaled to fit chart window
    const oLinearPts = nValues.map((n, i) => {
      const normX = (n - nMin) / (nMax - nMin || 1);
      const normY = normX * 0.7; // Linear slope
      const x = this.padding.left + normX * plotW;
      const y = this.padding.top + (1 - normY) * plotH;
      return `${i === 0 ? 'M' : 'L'} ${Math.round(x)} ${Math.round(y)}`;
    }).join(' ');

    const oQuadraticPts = nValues.map((n, i) => {
      const normX = (n - nMin) / (nMax - nMin || 1);
      const normY = Math.pow(normX, 2) * 0.95; // Quadratic curve
      const x = this.padding.left + normX * plotW;
      const y = this.padding.top + (1 - normY) * plotH;
      return `${i === 0 ? 'M' : 'L'} ${Math.round(x)} ${Math.round(y)}`;
    }).join(' ');

    return [
      { name: 'O(N) Lineal', path: oLinearPts, class: 'curve-linear' },
      { name: 'O(N²) Cuadrática', path: oQuadraticPts, class: 'curve-quadratic' }
    ];
  });

  readonly isOptimal = computed<boolean>(() => {
    return this.report?.is_optimal ?? true;
  });

  onExecuteBenchmark(preset: 'small' | 'standard' | 'stress'): void {
    this.selectedPreset.set(preset);
    this.runBenchmark.emit(preset);
  }

  exportDataCsv(): void {
    const rep = this.report;
    if (!rep || rep.samples.length === 0) return;

    const rows = [
      ['N_Input_Size', 'Execution_Time_ms', 'Memory_Used_KB', 'Operations', 'Status'],
      ...rep.samples.map(s => [
        s.input_size,
        s.execution_time_ms,
        s.memory_used_kb,
        s.operations_count || 0,
        s.status
      ])
    ];

    const csvContent = rows.map(e => e.join(',')).join('\n');
    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `solv-benchmark-${this.submissionId || 'sub'}-${Date.now()}.csv`;
    link.click();
    URL.revokeObjectURL(url);
  }

  onClose(): void {
    this.close.emit();
  }
}
