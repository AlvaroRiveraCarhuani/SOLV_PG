import { Component, input, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { HostHardwareMetrics } from '@core/models/admin.model';
import { LucideCpu, LucideDatabase, LucideHardDrive, LucideBoxes } from '@lucide/angular';

@Component({
  selector: 'hardware-kpi',
  standalone: true,
  imports: [CommonModule, LucideCpu, LucideDatabase, LucideHardDrive, LucideBoxes],
  template: `
    <div class="kpi-grid">
      <!-- 1. RAM KPI -->
      <div class="kpi-card" [class.danger]="ramPercentComputed() > 85" [class.warning]="ramPercentComputed() >= 70 && ramPercentComputed() <= 85">
        <div class="kpi-header">
          <span class="kpi-title">MEMORIA RAM (HOST)</span>
          <svg lucideDatabase class="kpi-icon"></svg>
        </div>
        <div class="kpi-body">
          <div class="kpi-value-row">
            <span class="kpi-mono-val">{{ ramUsedGB() }} / {{ ramTotalGB() }} GB</span>
            <span class="kpi-badge" [class.badge-danger]="ramPercentComputed() > 85" [class.badge-warning]="ramPercentComputed() >= 70 && ramPercentComputed() <= 85">
              {{ ramPercentComputed() }}%
            </span>
          </div>
          <div class="meter-track">
            <div class="meter-bar" [style.width.%]="ramPercentComputed()" [class.bar-danger]="ramPercentComputed() > 85" [class.bar-warning]="ramPercentComputed() >= 70 && ramPercentComputed() <= 85"></div>
          </div>
        </div>
        <div class="kpi-footer">
          <span>Umbral de seguridad: 85% &bull; Servidor Físico</span>
        </div>
      </div>

      <!-- 2. CPU KPI -->
      <div class="kpi-card" [class.danger]="metrics().cpu_percent > 85" [class.warning]="metrics().cpu_percent >= 70 && metrics().cpu_percent <= 85">
        <div class="kpi-header">
          <span class="kpi-title">PROCESAMIENTO CPU</span>
          <svg lucideCpu class="kpi-icon"></svg>
        </div>
        <div class="kpi-body">
          <div class="kpi-value-row">
            <span class="kpi-mono-val">{{ metrics().cpu_cores }} vCPUs</span>
            <span class="kpi-badge" [class.badge-danger]="metrics().cpu_percent > 85" [class.badge-warning]="metrics().cpu_percent >= 70 && metrics().cpu_percent <= 85">
              {{ metrics().cpu_percent }}%
            </span>
          </div>
          <div class="meter-track">
            <div class="meter-bar" [style.width.%]="metrics().cpu_percent" [class.bar-danger]="metrics().cpu_percent > 85" [class.bar-warning]="metrics().cpu_percent >= 70 && metrics().cpu_percent <= 85"></div>
          </div>
        </div>
        <div class="kpi-footer">
          <span>Carga media del host</span>
        </div>
      </div>

      <!-- 3. NVMe Storage KPI -->
      <div class="kpi-card">
        <div class="kpi-header">
          <span class="kpi-title">ALMACENAMIENTO NVMe</span>
          <svg lucideHardDrive class="kpi-icon"></svg>
        </div>
        <div class="kpi-body">
          <div class="kpi-value-row">
            <span class="kpi-mono-val">{{ diskUsedGB() }} / {{ diskTotalGB() }} GB</span>
            <span class="kpi-badge badge-neutral">
              {{ metrics().disk_percent }}%
            </span>
          </div>
          <div class="meter-track">
            <div class="meter-bar" [style.width.%]="metrics().disk_percent"></div>
          </div>
        </div>
        <div class="kpi-footer">
          <span>/var/lib/docker overlay2</span>
        </div>
      </div>

      <!-- 4. Contenedores KPI -->
      <div class="kpi-card">
        <div class="kpi-header">
          <span class="kpi-title">CONCURRENCIA</span>
          <svg lucideBoxes class="kpi-icon"></svg>
        </div>
        <div class="kpi-body">
          <div class="kpi-value-row">
            <span class="kpi-mono-val">{{ metrics().containers_active }} / {{ metrics().containers_max }} Activos</span>
            <span class="kpi-badge badge-success">
              {{ concurrencyPercent() }}%
            </span>
          </div>
          <div class="meter-track">
            <div class="meter-bar bar-primary" [style.width.%]="concurrencyPercent()"></div>
          </div>
        </div>
        <div class="kpi-footer">
          @if (metrics().containers_active === 0) {
            <span>0 labs en RAM &bull; {{ metrics().containers_hibernated }} hibernados</span>
          } @else {
            <span>{{ metrics().containers_hibernated }} contenedores hibernados</span>
          }
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
  ramPercentComputed = computed(() => {
    if (this.metrics().ram_total_bytes > 0) {
      return Math.min(100, Math.round((this.metrics().ram_used_bytes / this.metrics().ram_total_bytes) * 100));
    }
    return this.metrics().ram_percent || 0;
  });
  diskUsedGB = computed(() => (this.metrics().disk_used_bytes / (1024 * 1024 * 1024)).toFixed(0));
  diskTotalGB = computed(() => (this.metrics().disk_total_bytes / (1024 * 1024 * 1024)).toFixed(0));
  concurrencyPercent = computed(() => {
    const max = this.metrics().containers_max || 40;
    return Math.round((this.metrics().containers_active / max) * 100);
  });
}
