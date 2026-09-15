import { Component, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { AdminMetricsService } from '../services/admin-metrics.service';
import { HardwareKpiComponent } from './components/hardware-kpi/hardware-kpi.component';
import { CourseLoadComponent } from './components/course-load/course-load.component';
import { IncidentsPanelComponent } from './components/incidents-panel/incidents-panel.component';
import { CourseWorkspacesModalComponent } from './components/course-modal/course-workspaces-modal.component';
import { ContainerTableComponent } from './components/container-table/container-table.component';
import { CourseLoadSummary } from '@core/models/admin.model';
import { 
  LucideRefreshCw, 
  LucideMoon, 
  LucideServer, 
  LucideChevronDown, 
  LucideChevronUp,
  LucideInfo,
  LucideCheckCircle,
  LucideX
} from '@lucide/angular';

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
    LucideChevronUp,
    LucideInfo,
    LucideCheckCircle,
    LucideX
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
            [disabled]="isLoading() || refreshCooldown() > 0"
            title="Recargar Métricas">
            <svg lucideRefreshCw class="btn-icon" [class.spin]="isLoading()"></svg>
            <span>{{ refreshCooldown() > 0 ? 'Actualizar (' + refreshCooldown() + 's)' : 'Actualizar' }}</span>
          </button>

          <button 
            class="btn-action-hibernate" 
            (click)="hibernateAll()" 
            [disabled]="isHibernating() || hibernateCooldown() > 0"
            title="Hibernar contenedores inactivos">
            <svg lucideMoon class="btn-icon"></svg>
            <span>{{ isHibernating() ? 'Hibernando...' : (hibernateCooldown() > 0 ? 'Hibernar (' + hibernateCooldown() + 's)' : 'Hibernar Todo') }}</span>
          </button>
        </div>
      </header>

      <!-- Feedback Alert Banner -->
      @if (feedbackMessage()) {
        <div class="feedback-banner" [class]="feedbackMessage()!.type">
          <div class="feedback-content">
            @if (feedbackMessage()!.type === 'info') {
              <svg lucideInfo class="feedback-icon"></svg>
            } @else if (feedbackMessage()!.type === 'success') {
              <svg lucideCheckCircle class="feedback-icon"></svg>
            }
            <span>{{ feedbackMessage()!.text }}</span>
          </div>
          <button class="btn-close-feedback" (click)="feedbackMessage.set(null)" title="Cerrar aviso">
            <svg lucideX class="close-icon"></svg>
          </button>
        </div>
      }

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
      margin-bottom: var(--space-2, 8px);

      @media (max-width: 768px) {
        flex-direction: column;
        align-items: flex-start;
        gap: var(--space-3, 12px);
      }
    }

    .title-row {
      display: flex;
      align-items: center;
      gap: var(--space-3, 12px);
      margin-bottom: 4px;
    }

    .page-title {
      font-size: var(--font-size-xl, 20px);
      font-weight: 700;
      color: var(--text-primary, #0F172A);
      margin: 0;
      letter-spacing: -0.01em;
    }

    .status-pill {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 3px 8px;
      border-radius: var(--radius-full, 9999px);
      font-size: 11px;
      font-weight: 600;

      &.status-healthy {
        background-color: #DCFCE7;
        color: #15803D;

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
      font-size: var(--font-size-xs, 12px);
      color: var(--text-secondary, #64748B);
    }

    .host-info {
      display: flex;
      align-items: center;
      gap: 4px;
      font-family: var(--font-mono, monospace);
    }

    .meta-icon {
      width: 14px;
      height: 14px;
      color: var(--text-muted, #94A3B8);
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
      display: flex;
      align-items: center;
      gap: var(--space-2, 8px);
      padding: 8px 14px;
      background-color: var(--bg-surface, #FFFFFF);
      border: 1px solid var(--border-subtle, #CBD5E1);
      border-radius: var(--radius-md, 6px);
      font-size: var(--font-size-xs, 12px);
      font-weight: 600;
      color: var(--text-primary, #0F172A);
      cursor: pointer;
      transition: all 150ms ease;

      &:hover:not(:disabled) {
        background-color: var(--bg-canvas, #F1F5F9);
        border-color: #94A3B8;
      }

      &:disabled {
        opacity: 0.6;
        cursor: not-allowed;
      }

      .btn-icon {
        width: 14px;
        height: 14px;
        color: var(--text-secondary, #64748B);

        &.spin {
          animation: spin 1s linear infinite;
        }
      }
    }

    .btn-action-hibernate {
      display: flex;
      align-items: center;
      gap: var(--space-2, 8px);
      padding: 8px 14px;
      background-color: #F8FAFC;
      border: 1px solid var(--border-subtle, #CBD5E1);
      border-radius: var(--radius-md, 6px);
      font-size: var(--font-size-xs, 12px);
      font-weight: 600;
      color: #1E293B;
      cursor: pointer;
      transition: all 150ms ease;

      &:hover:not(:disabled) {
        background-color: #EDE9FE;
        border-color: #C4B5FD;
        color: #6D28D9;
      }

      &:disabled {
        opacity: 0.6;
        cursor: not-allowed;
      }

      .btn-icon {
        width: 14px;
        height: 14px;
      }
    }

    @keyframes spin {
      from { transform: rotate(0deg); }
      to { transform: rotate(360deg); }
    }

    .feedback-banner {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 10px 16px;
      border-radius: var(--radius-md, 6px);
      font-size: 13px;
      font-weight: 500;
      animation: bannerSlideDown 200ms ease;

      &.info {
        background-color: #EFF6FF;
        border: 1px solid #BFDBFE;
        color: #1E40AF;
      }

      &.success {
        background-color: #F0FDF4;
        border: 1px solid #BBF7D0;
        color: #166534;
      }

      &.warning {
        background-color: #FEFCE8;
        border: 1px solid #FEF08A;
        color: #854D0E;
      }

      .feedback-content {
        display: flex;
        align-items: center;
        gap: var(--space-2, 8px);
      }

      .feedback-icon {
        width: 16px;
        height: 16px;
        flex-shrink: 0;
      }

      .btn-close-feedback {
        background: transparent;
        border: none;
        cursor: pointer;
        padding: 2px;
        display: flex;
        align-items: center;
        opacity: 0.7;
        transition: opacity 150ms ease;

        &:hover {
          opacity: 1;
        }

        .close-icon {
          width: 14px;
          height: 14px;
        }
      }
    }

    @keyframes bannerSlideDown {
      from { opacity: 0; transform: translateY(-6px); }
      to { opacity: 1; transform: translateY(0); }
    }

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

  // Cooldowns and Feedback Signals
  refreshCooldown = signal<number>(0);
  hibernateCooldown = signal<number>(0);
  isHibernating = signal<boolean>(false);
  feedbackMessage = signal<{ type: 'info' | 'success' | 'warning', text: string } | null>(null);

  private refreshTimer?: any;
  private hibernateTimer?: any;
  private feedbackTimer?: any;

  refresh(): void {
    if (this.isLoading() || this.refreshCooldown() > 0) return;

    this.metricsService.fetchMetrics();
    this.refreshCooldown.set(5);

    if (this.refreshTimer) clearInterval(this.refreshTimer);
    this.refreshTimer = setInterval(() => {
      const current = this.refreshCooldown();
      if (current <= 1) {
        this.refreshCooldown.set(0);
        clearInterval(this.refreshTimer);
      } else {
        this.refreshCooldown.set(current - 1);
      }
    }, 1000);
  }

  hibernateAll(): void {
    if (this.isHibernating() || this.hibernateCooldown() > 0) return;

    const active = this.health()?.metrics?.containers_active ?? 0;

    if (active === 0) {
      this.showFeedback('info', 'No hay laboratorios activos para hibernar en este momento. Todos los contenedores se encuentran detenidos o en reposo.');
      this.startHibernateCooldown(3);
      return;
    }

    this.isHibernating.set(true);
    this.metricsService.hibernateAll().subscribe({
      next: () => {
        this.isHibernating.set(false);
        this.showFeedback('success', `Se enviaron órdenes de hibernación al clúster. Se suspendieron ${active} contenedor(es) activo(s).`);
        this.startHibernateCooldown(5);
      },
      error: () => {
        this.isHibernating.set(false);
        this.showFeedback('warning', 'Ocurrió un error al intentar hibernar los contenedores. Verifique los logs del clúster.');
        this.startHibernateCooldown(5);
      }
    });
  }

  private startHibernateCooldown(seconds: number): void {
    this.hibernateCooldown.set(seconds);
    if (this.hibernateTimer) clearInterval(this.hibernateTimer);
    this.hibernateTimer = setInterval(() => {
      const current = this.hibernateCooldown();
      if (current <= 1) {
        this.hibernateCooldown.set(0);
        clearInterval(this.hibernateTimer);
      } else {
        this.hibernateCooldown.set(current - 1);
      }
    }, 1000);
  }

  private showFeedback(type: 'info' | 'success' | 'warning', text: string): void {
    this.feedbackMessage.set({ type, text });
    if (this.feedbackTimer) clearTimeout(this.feedbackTimer);
    this.feedbackTimer = setTimeout(() => {
      this.feedbackMessage.set(null);
    }, 6000);
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
    console.info('Consultar logs para workspace:', workspaceId);
  }
}
