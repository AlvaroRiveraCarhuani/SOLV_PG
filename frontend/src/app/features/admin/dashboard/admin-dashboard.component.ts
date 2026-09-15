import { Component, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { AdminMetricsService } from '../services/admin-metrics.service';
import { HardwareKpiComponent } from './components/hardware-kpi/hardware-kpi.component';
import { CourseLoadComponent } from './components/course-load/course-load.component';
import { IncidentsPanelComponent } from './components/incidents-panel/incidents-panel.component';
import { CourseWorkspacesModalComponent } from './components/course-modal/course-workspaces-modal.component';
import { ContainerTableComponent } from './components/container-table/container-table.component';
import { CourseLoadSummary } from '@core/models/admin.model';
import { LucideRefreshCw, LucideMoon, LucideServer, LucideChevronDown, LucideChevronUp } from '@lucide/angular';

@Component({
  selector: 'solv-admin-dashboard',
  standalone: true,
  imports: [
    CommonModule, 
    HardwareKpiComponent, 
    CourseLoadComponent,
    IncidentsPanelComponent,
    CourseWorkspacesModalComponent,
    ContainerTableComponent, 
    LucideRefreshCw, 
    LucideMoon, 
    LucideServer,
    LucideChevronDown,
    LucideChevronUp
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

      <!-- 1. Grid KPI Hardware Real (Above the fold) -->
      @if (health()?.metrics) {
        <solv-hardware-kpi [metrics]="health()!.metrics" />
      }

      <!-- 2. Grilla Macro en 2 Columnas (2/3 y 1/3) -->
      <div class="grid-split-macro">
        <!-- Columna Izquierda (2/3): Carga por Materia -->
        <div class="col-course-load">
          @if (health()?.courses_load) {
            <solv-course-load 
              [courses]="health()!.courses_load" 
              (viewDetails)="onViewCourseDetails($event)" 
            />
          }
        </div>

        <!-- Columna Derecha (1/3): Incidencias Técnicas -->
        <div class="col-incidents">
          @if (health()?.incidents) {
            <solv-incidents-panel 
              [incidents]="health()!.incidents" 
              (restartWorkspace)="onRestartWorkspace($event)"
              (viewLogs)="onViewLogs($event)"
            />
          }
        </div>
      </div>

      <!-- 3. Sección Desplegable: Auditoría Global de Contenedores -->
      <div class="section-collapsible">
        <button class="btn-toggle-section" (click)="showAllContainers.set(!showAllContainers())">
          <span>Ver todas las instancias activas del clúster ({{ health()?.containers?.length || 0 }})</span>
          @if (showAllContainers()) {
            <svg lucideChevronUp class="toggle-icon"></svg>
          } @else {
            <svg lucideChevronDown class="toggle-icon"></svg>
          }
        </button>

        @if (showAllContainers() && health()?.containers) {
          <div class="collapsible-content">
            <solv-container-table 
              [containers]="health()!.containers" 
              (stopContainer)="stopContainer($event)" 
            />
          </div>
        }
      </div>

      <!-- 4. Modal Contextual: Detalle de Contenedores por Materia -->
      @if (selectedCourse()) {
        <solv-course-workspaces-modal 
          [course]="selectedCourse()!"
          [workspaces]="health()?.containers || []"
          (close)="selectedCourse.set(null)"
          (restartWorkspace)="onRestartWorkspace($event)"
          (pauseWorkspace)="onPauseWorkspace($event)"
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

    /* Grilla Macro en 2 Columnas */
    .grid-split-macro {
      display: grid;
      grid-template-columns: 2fr 1fr;
      gap: var(--space-4, 16px);
      align-items: stretch;

      @media (max-width: 1024px) {
        grid-template-columns: 1fr;
      }
    }

    .section-collapsible {
      display: flex;
      flex-direction: column;
      gap: 10px;
      margin-top: var(--space-2, 8px);
    }

    .btn-toggle-section {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 10px 16px;
      background-color: #FFFFFF;
      border: 1px solid var(--border-subtle, #E2E8F0);
      border-radius: var(--radius-md, 6px);
      font-size: 13px;
      font-weight: 600;
      color: var(--text-secondary, #475569);
      cursor: pointer;
      transition: all 150ms ease;

      &:hover {
        background-color: #F8FAFC;
        color: var(--text-primary, #0F172A);
      }

      .toggle-icon {
        width: 16px;
        height: 16px;
      }
    }

    .collapsible-content {
      animation: fadeIn 200ms ease;
    }

    @keyframes fadeIn {
      from { opacity: 0; transform: translateY(-4px); }
      to { opacity: 1; transform: translateY(0); }
    }
  `]
})
export class AdminDashboardComponent {
  private metricsService = inject(AdminMetricsService);

  health = this.metricsService.health;
  isLoading = this.metricsService.isLoading;

  selectedCourse = signal<CourseLoadSummary | null>(null);
  showAllContainers = signal<boolean>(false);

  refresh(): void {
    this.metricsService.fetchMetrics();
  }

  hibernateAll(): void {
    this.metricsService.hibernateAll();
  }

  stopContainer(id: string): void {
    this.metricsService.stopContainer(id);
  }

  onViewCourseDetails(course: CourseLoadSummary): void {
    this.selectedCourse.set(course);
  }

  onRestartWorkspace(workspaceId: string): void {
    this.metricsService.restartWorkspace(workspaceId);
  }

  onPauseWorkspace(workspaceId: string): void {
    this.metricsService.pauseWorkspace(workspaceId);
  }

  onViewLogs(workspaceId: string): void {
    // Abre registro de auditoría o log del contenedor
    console.info('Consultar logs para workspace:', workspaceId);
  }
}
