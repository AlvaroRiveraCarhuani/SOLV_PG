import { Component, input, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { HostHardwareMetrics } from '@core/models/admin.model';
import { LucideCpu, LucideDatabase, LucideHardDrive, LucideBoxes } from '@lucide/angular';

@Component({
  selector: 'solv-hardware-kpi',
  standalone: true,
  imports: [CommonModule, LucideCpu, LucideDatabase, LucideHardDrive, LucideBoxes],
  template: `
    <div class="kpi-grid">
      <!-- 1. RAM KPI -->
      <div class="kpi-card" [class.danger]="metrics().ram_percent > 85" [class.warning]="metrics().ram_percent >= 70 && metrics().ram_percent <= 85">
        <div class="kpi-header">
          <span class="kpi-title">MEMORIA RAM</span>
          <svg lucideDatabase class="kpi-icon"></svg>
        </div>
        <div class="kpi-body">
          <div class="kpi-value-row">
            <span class="kpi-mono-val">{{ ramUsedGB() }} / {{ ramTotalGB() }} GB</span>
            <span class="kpi-badge" [class.badge-danger]="metrics().ram_percent > 85" [class.badge-warning]="metrics().ram_percent >= 70 && metrics().ram_percent <= 85">
              {{ metrics().ram_percent }}%
            </span>
          </div>
          <div class="meter-track">
            <div class="meter-bar" [style.width.%]="metrics().ram_percent" [class.bar-danger]="metrics().ram_percent > 85" [class.bar-warning]="metrics().ram_percent >= 70 && metrics().ram_percent <= 85"></div>
          </div>
        </div>
        <div class="kpi-footer">
          <span>Umbral de seguridad: 85%</span>
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
          <span>{{ metrics().containers_hibernated }} contenedores hibernados</span>
        </div>
      </div>
    </div>
  `,
  styles: [`
    .kpi-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
      gap: var(--space-4, 16px);
      margin-bottom: var(--space-6, 24px);
    }

    .kpi-card {
      background-color: var(--bg-surface, #FFFFFF);
      border: 1px solid var(--border-subtle, #E2E8F0);
      border-radius: var(--radius-lg, 8px);
      padding: var(--space-4, 16px);
      display: flex;
      flex-direction: column;
      gap: var(--space-3, 12px);

      &.warning {
        border-color: #FDE68A;
      }

      &.danger {
        border-color: #FECACA;
      }
    }

    .kpi-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
    }

    .kpi-title {
      font-size: 11px;
      font-weight: 700;
      color: var(--text-muted, #64748B);
      letter-spacing: 0.05em;
    }

    .kpi-icon {
      width: 16px;
      height: 16px;
      color: var(--text-muted, #94A3B8);
    }

    .kpi-body {
      display: flex;
      flex-direction: column;
      gap: var(--space-2, 8px);
    }

    .kpi-value-row {
      display: flex;
      justify-content: space-between;
      align-items: baseline;
    }

    .kpi-mono-val {
      font-family: 'JetBrains Mono', monospace;
      font-size: 18px;
      font-weight: 600;
      color: var(--text-primary, #0F172A);
    }

    .kpi-badge {
      font-family: 'JetBrains Mono', monospace;
      font-size: 12px;
      font-weight: 600;
      padding: 2px 6px;
      border-radius: var(--radius-sm, 4px);
      background-color: var(--bg-canvas, #F1F5F9);
      color: var(--text-secondary, #475569);

      &.badge-warning {
        background-color: #FEF3C7;
        color: #92400E;
      }

      &.badge-danger {
        background-color: #FEE2E2;
        color: #B91C1C;
      }

      &.badge-success {
        background-color: #DCFCE7;
        color: #15803D;
      }
    }

    .meter-track {
      height: 6px;
      width: 100%;
      background-color: var(--border-subtle, #E2E8F0);
      border-radius: 3px;
      overflow: hidden;
    }

    .meter-bar {
      height: 100%;
      background-color: var(--tenant-primary, #2563EB);
      border-radius: 3px;
      transition: width 300ms ease;

      &.bar-warning {
        background-color: #D97706;
      }

      &.bar-danger {
        background-color: #DC2626;
      }

      &.bar-primary {
        background-color: var(--tenant-primary, #2563EB);
      }
    }

    .kpi-footer {
      font-size: 11px;
      color: var(--text-muted, #94A3B8);
    }
  `]
})
export class HardwareKpiComponent {
  metrics = input.required<HostHardwareMetrics>();

  ramUsedGB = computed(() => (this.metrics().ram_used_bytes / (1024 * 1024 * 1024)).toFixed(1));
  ramTotalGB = computed(() => (this.metrics().ram_total_bytes / (1024 * 1024 * 1024)).toFixed(1));
  diskUsedGB = computed(() => (this.metrics().disk_used_bytes / (1024 * 1024 * 1024)).toFixed(0));
  diskTotalGB = computed(() => (this.metrics().disk_total_bytes / (1024 * 1024 * 1024)).toFixed(0));
  concurrencyPercent = computed(() => {
    const max = this.metrics().containers_max || 40;
    return Math.round((this.metrics().containers_active / max) * 100);
  });
}
