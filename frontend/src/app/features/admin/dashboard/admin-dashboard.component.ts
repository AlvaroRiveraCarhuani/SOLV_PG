import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { AdminMetricsService } from '../services/admin-metrics.service';
import { HardwareKpiComponent } from './components/hardware-kpi/hardware-kpi.component';
import { ContainerTableComponent } from './components/container-table/container-table.component';
import { LucideRefreshCw, LucideMoon, LucideServer } from '@lucide/angular';

@Component({
  selector: 'solv-admin-dashboard',
  standalone: true,
  imports: [
    CommonModule, 
    HardwareKpiComponent, 
    ContainerTableComponent, 
    LucideRefreshCw, 
    LucideMoon, 
    LucideServer
  ],
  template: `
    <div class="dashboard-page">
      <!-- Header Superior de Operaciones -->
      <header class="page-header">
        <div class="header-left">
          <div class="title-row">
            <h1 class="page-title">Salud del Servidor y Recursos</h1>
            <span class="status-pill status-healthy">
              <span class="pill-dot"></span>
              {{ health()?.status === 'degraded' ? 'Rendimiento Degradado' : 'Operativo y Estable' }}
            </span>
          </div>
          <div class="meta-row">
            <span class="host-info">
              <svg lucideServer class="meta-icon"></svg>
              {{ health()?.docker_version }}
            </span>
            <span class="meta-separator">&bull;</span>
            <span class="uptime-text">Uptime: 14 días 5 horas</span>
          </div>
        </div>

        <div class="header-actions">
          <button 
            class="btn-secondary" 
            (click)="refresh()" 
            [disabled]="isLoading()"
            title="Recargar Métricas">
            <svg lucideRefreshCw class="btn-icon" [class.spin]="isLoading()"></svg>
            <span>Actualizar</span>
          </button>

          <button 
            class="btn-action-hibernate" 
            (click)="hibernateAll()" 
            [disabled]="isLoading()"
            title="Hibernar contenedores inactivos">
            <svg lucideMoon class="btn-icon"></svg>
            <span>Hibernar Todo</span>
          </button>
        </div>
      </header>

      <!-- Grid KPI Hardware -->
      @if (health()?.metrics) {
        <solv-hardware-kpi [metrics]="health()!.metrics" />
      }

      <!-- Tabla de Monitoreo de Contenedores -->
      @if (health()?.containers) {
        <solv-container-table 
          [containers]="health()!.containers" 
          (stopContainer)="stopContainer($event)" 
        />
      }
    </div>
  `,
  styles: [`
    .dashboard-page {
      display: flex;
      flex-direction: column;
      gap: var(--space-4, 16px);
      max-width: 1440px;
      margin: 0 auto;
    }

    .page-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      flex-wrap: wrap;
      gap: var(--space-4, 16px);
      padding-bottom: var(--space-2, 8px);
    }

    .header-left {
      display: flex;
      flex-direction: column;
      gap: 4px;
    }

    .title-row {
      display: flex;
      align-items: center;
      gap: var(--space-3, 12px);
    }

    .page-title {
      font-size: 20px;
      font-weight: 700;
      color: var(--text-primary, #0F172A);
      margin: 0;
      letter-spacing: -0.01em;
    }

    .status-pill {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 3px 10px;
      border-radius: var(--radius-full, 9999px);
      font-size: 11px;
      font-weight: 600;

      &.status-healthy {
        background-color: #DCFCE7;
        color: #15803D;
        border: 1px solid #BBF7D0;

        .pill-dot {
          background-color: #16A34A;
        }
      }

      .pill-dot {
        width: 6px;
        height: 6px;
        border-radius: 50%;
      }
    }

    .meta-row {
      display: flex;
      align-items: center;
      gap: var(--space-2, 8px);
      font-size: 12px;
      color: var(--text-muted, #64748B);
    }

    .host-info {
      display: inline-flex;
      align-items: center;
      gap: 5px;
      font-family: 'JetBrains Mono', monospace;
      font-size: 11px;
    }

    .meta-icon {
      width: 13px;
      height: 13px;
    }

    .meta-separator {
      color: var(--border-subtle, #CBD5E1);
    }

    .header-actions {
      display: flex;
      align-items: center;
      gap: var(--space-2-5, 10px);
    }

    .btn-secondary {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 7px 14px;
      background-color: var(--bg-surface, #FFFFFF);
      border: 1px solid var(--border-subtle, #CBD5E1);
      border-radius: var(--radius-md, 6px);
      color: var(--text-primary, #1E293B);
      font-size: 13px;
      font-weight: 500;
      cursor: pointer;
      transition: all 150ms ease;

      &:hover:not(:disabled) {
        background-color: var(--bg-canvas, #F8FAFC);
        border-color: #94A3B8;
      }

      &:disabled {
        opacity: 0.6;
        cursor: not-allowed;
      }
    }

    .btn-action-hibernate {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 7px 14px;
      background-color: #F1F5F9;
      border: 1px solid #CBD5E1;
      border-radius: var(--radius-md, 6px);
      color: #334155;
      font-size: 13px;
      font-weight: 600;
      cursor: pointer;
      transition: all 150ms ease;

      &:hover:not(:disabled) {
        background-color: #E2E8F0;
        color: #0F172A;
      }

      &:disabled {
        opacity: 0.6;
        cursor: not-allowed;
      }
    }

    .btn-icon {
      width: 14px;
      height: 14px;

      &.spin {
        animation: spin 1s linear infinite;
      }
    }

    @keyframes spin {
      100% {
        transform: rotate(360deg);
      }
    }
  `]
})
export class AdminDashboardComponent {
  private metricsService = inject(AdminMetricsService);

  health = this.metricsService.health;
  isLoading = this.metricsService.isLoading;

  refresh(): void {
    this.metricsService.fetchMetrics();
  }

  hibernateAll(): void {
    this.metricsService.hibernateAll();
  }

  stopContainer(id: string): void {
    this.metricsService.stopContainer(id);
  }
}
