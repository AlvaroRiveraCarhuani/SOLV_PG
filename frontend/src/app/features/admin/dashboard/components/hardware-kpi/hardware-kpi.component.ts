import { Component, input, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { HostHardwareMetrics } from '@core/models/admin.model';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { LucideCpu, LucideDatabase, LucideHardDrive, LucideBoxes } from '@lucide/angular';

@Component({
  selector: 'hardware-kpi',
  standalone: true,
  imports: [CommonModule, MachineDataDirective, LucideCpu, LucideDatabase, LucideHardDrive, LucideBoxes],
  template: `
    <div class="kpi-strip">
      <div class="kpi-cell">
        <div class="kpi-label-row">
          <span class="kpi-label">MEMORIA RAM (HOST)</span>
          <svg lucideDatabase class="kpi-icon"></svg>
        </div>
        <div class="kpi-value-row">
          <span class="kpi-value" machineData>{{ ramUsedGB() }} / {{ ramTotalGB() }} GB</span>
          <span
            class="kpi-percent"
            [class.is-warning]="ramPercentComputed() >= 75 && ramPercentComputed() <= 85"
            [class.is-danger]="ramPercentComputed() > 85">
            <span machineData>{{ ramPercentComputed() }}%</span>
          </span>
        </div>
        <div class="meter-track">
          <div
            class="meter-fill"
            [style.width.%]="ramPercentComputed()"
            [class.bar-warning]="ramPercentComputed() >= 75 && ramPercentComputed() <= 85"
            [class.bar-danger]="ramPercentComputed() > 85"></div>
          <span class="meter-threshold" aria-hidden="true"></span>
        </div>
        <div class="kpi-footer">
          <span>Umbral QoS: <span machineData>{{ metrics().host_safety_ceiling_pct ?? 85 }}%</span></span>
          <span machineData>{{ ramFreeGB() }} GB libres</span>
        </div>
      </div>

      <div class="kpi-cell">
        <div class="kpi-label-row">
          <span class="kpi-label">PROCESAMIENTO CPU</span>
          <svg lucideCpu class="kpi-icon"></svg>
        </div>
        <div class="kpi-value-row">
          <span class="kpi-value"><span machineData>{{ metrics().cpu_cores }}</span> vCPU</span>
          <span
            class="kpi-percent"
            [class.is-warning]="metrics().cpu_percent >= 75 && metrics().cpu_percent <= 85"
            [class.is-danger]="metrics().cpu_percent > 85">
            <span machineData>{{ metrics().cpu_percent }}%</span>
          </span>
        </div>
        <div class="meter-track">
          <div
            class="meter-fill"
            [style.width.%]="metrics().cpu_percent"
            [class.bar-warning]="metrics().cpu_percent >= 75 && metrics().cpu_percent <= 85"
            [class.bar-danger]="metrics().cpu_percent > 85"></div>
        </div>
        <div class="kpi-footer">
          <span>Carga media</span>
          <span machineData>{{ metrics().cpu_cores }} vCPU</span>
        </div>
      </div>

      <div class="kpi-cell">
        <div class="kpi-label-row">
          <span class="kpi-label">ALMACENAMIENTO NVMe</span>
          <svg lucideHardDrive class="kpi-icon"></svg>
        </div>
        <div class="kpi-value-row">
          <span class="kpi-value" machineData>{{ diskUsedGB() }} / {{ diskTotalGB() }} GB</span>
          <span class="kpi-percent is-neutral"><span machineData>{{ metrics().disk_percent }}%</span></span>
        </div>
        <div class="meter-track">
          <div class="meter-fill" [style.width.%]="metrics().disk_percent"></div>
        </div>
        <div class="kpi-footer">
          <span machineData>{{ metrics().disk_path || '—' }}</span>
          <span machineData>{{ diskTotalGB() }} GB total</span>
        </div>
      </div>

      <div class="kpi-cell">
        <div class="kpi-label-row">
          <span class="kpi-label">CONCURRENCIA</span>
          <svg lucideBoxes class="kpi-icon"></svg>
        </div>
        <div class="kpi-value-row">
          <span class="kpi-value"><span machineData>{{ metrics().containers_active }} / {{ concurrencyMax() }}</span></span>
          <span class="kpi-percent is-neutral"><span machineData>{{ concurrencyPercent() }}%</span></span>
        </div>
        <div class="meter-track">
          <div class="meter-fill" [style.width.%]="concurrencyPercent()"></div>
        </div>
        <div class="kpi-footer">
          <span>{{ capacityBasisLabel() }}</span>
          <span><span machineData>{{ metrics().containers_hibernated }}</span> hibernados</span>
        </div>
      </div>
    </div>
  `,
  styleUrl: './hardware-kpi.component.scss',
})
export class HardwareKpiComponent {
  metrics = input.required<HostHardwareMetrics>();

  ramUsedGB = computed(() => (this.metrics().ram_used_bytes / (1024 * 1024 * 1024)).toFixed(1));
  ramTotalGB = computed(() => (this.metrics().ram_total_bytes / (1024 * 1024 * 1024)).toFixed(1));
  ramFreeGB = computed(() => {
    const free = Math.max(0, this.metrics().ram_total_bytes - this.metrics().ram_used_bytes);
    return (free / (1024 * 1024 * 1024)).toFixed(1);
  });
  ramPercentComputed = computed(() => {
    if (this.metrics().ram_total_bytes > 0) {
      return Math.min(100, Math.round((this.metrics().ram_used_bytes / this.metrics().ram_total_bytes) * 100));
    }
    return this.metrics().ram_percent || 0;
  });
  diskUsedGB = computed(() => (this.metrics().disk_used_bytes / (1024 * 1024 * 1024)).toFixed(0));
  diskTotalGB = computed(() => (this.metrics().disk_total_bytes / (1024 * 1024 * 1024)).toFixed(0));
  concurrencyMax = computed(() => this.metrics().estimated_max_containers ?? this.metrics().containers_max ?? 40);
  // El denominador de la tarjeta es la capacidad real calculada por el backend
  // (min(politica, RAM disponible)), y el subtitulo explica cual limite mando.
  capacityBasisLabel = computed(() => {
    switch (this.metrics().capacity_basis) {
      case 'mem_available':
        return 'Limitado por RAM disponible del host';
      case 'policy':
        return 'Techo de política (max_containers)';
      default:
        return 'Capacidad según políticas del tenant';
    }
  });
  concurrencyPercent = computed(() => {
    const max = this.concurrencyMax();
    return max > 0 ? Math.round((this.metrics().containers_active / max) * 100) : 0;
  });
}
