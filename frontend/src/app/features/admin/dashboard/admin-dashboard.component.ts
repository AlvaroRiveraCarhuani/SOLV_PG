import { Component, inject, signal, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { CdkDropList, CdkDrag, CdkDragHandle, CdkDragDrop, moveItemInArray } from '@angular/cdk/drag-drop';
import { AdminMetricsService } from '../services/admin-metrics.service';
import { HardwareKpiComponent } from './components/hardware-kpi/hardware-kpi.component';
import { CourseLoadComponent } from './components/course-load/course-load.component';
import { IncidentsPanelComponent } from './components/incidents-panel/incidents-panel.component';
import { CourseWorkspacesModalComponent } from './components/course-modal/course-workspaces-modal.component';
import { ContainerTableComponent } from './components/container-table/container-table.component';
import { LogViewerModalComponent } from './components/log-viewer-modal/log-viewer-modal.component';
import { LoadHistoryChartComponent } from './components/load-history-chart/load-history-chart.component';
import { CourseLoadSummary, TechnicalIncident } from '@core/models/admin.model';
import { 
  LucideRefreshCw, 
  LucideMoon, 
  LucideServer, 
  LucideChevronDown, 
  LucideChevronUp,
  LucideInfo,
  LucideCheckCircle,
  LucideX,
  LucideSliders,
  LucideGripVertical,
  LucideRotateCcw
} from '@lucide/angular';

export type DashboardSectionId = 'kpis' | 'chart' | 'split' | 'containers';

const STORAGE_LAYOUT_KEY = 'solv_admin_dashboard_layout';
const DEFAULT_BLOCK_ORDER: DashboardSectionId[] = ['kpis', 'chart', 'split', 'containers'];

@Component({
  selector: 'solv-admin-dashboard',
  standalone: true,
  imports: [
    CommonModule, 
    CdkDropList, 
    CdkDrag, 
    CdkDragHandle,
    HardwareKpiComponent, 
    CourseLoadComponent, 
    IncidentsPanelComponent,
    CourseWorkspacesModalComponent,
    ContainerTableComponent, 
    LogViewerModalComponent,
    LoadHistoryChartComponent,
    LucideRefreshCw, 
    LucideMoon, 
    LucideServer, 
    LucideChevronDown, 
    LucideChevronUp,
    LucideInfo,
    LucideCheckCircle,
    LucideX,
    LucideSliders,
    LucideGripVertical,
    LucideRotateCcw
  ],
  templateUrl: './admin-dashboard.component.html',
  styleUrl: './admin-dashboard.component.scss'
})
export class AdminDashboardComponent implements OnDestroy {
  private metricsService = inject(AdminMetricsService);

  health = this.metricsService.health;
  isLoading = this.metricsService.isLoading;

  selectedCourse = signal<CourseLoadSummary | null>(null);
  selectedIncidentForLogs = signal<TechnicalIncident | null>(null);
  showAllContainers = signal<boolean>(false);

  // Modo Personalización de la Vista (CDK Drag & Drop)
  isCustomizing = signal<boolean>(false);
  blockOrder = signal<DashboardSectionId[]>(DEFAULT_BLOCK_ORDER);

  // Cooldowns and Feedback Signals
  refreshCooldown = signal<number>(0);
  hibernateCooldown = signal<number>(0);
  isHibernating = signal<boolean>(false);
  autoRefresh = signal<boolean>(false);
  feedbackMessage = signal<{ type: 'info' | 'success' | 'warning', text: string } | null>(null);

  private refreshTimer?: ReturnType<typeof setInterval>;
  private hibernateTimer?: ReturnType<typeof setInterval>;
  private feedbackTimer?: ReturnType<typeof setTimeout>;
  private autoRefreshTimer?: ReturnType<typeof setInterval>;

  constructor() {
    this.loadLayoutPreference();
  }

  private loadLayoutPreference(): void {
    try {
      if (typeof window !== 'undefined' && window.localStorage) {
        const raw = localStorage.getItem(STORAGE_LAYOUT_KEY);
        if (raw) {
          const parsed = JSON.parse(raw);
          if (Array.isArray(parsed) && parsed.length === DEFAULT_BLOCK_ORDER.length) {
            this.blockOrder.set(parsed);
          }
        }
      }
    } catch {
      // Ignorar restricciones en entornos privados
    }
  }

  toggleCustomizing(): void {
    this.isCustomizing.update(v => !v);
  }

  onDropBlock(event: CdkDragDrop<DashboardSectionId[]>): void {
    const current = [...this.blockOrder()];
    moveItemInArray(current, event.previousIndex, event.currentIndex);
    this.blockOrder.set(current);
  }

  moveBlock(blockId: DashboardSectionId, delta: number): void {
    const current = [...this.blockOrder()];
    const idx = current.indexOf(blockId);
    if (idx < 0) return;
    const newIdx = idx + delta;
    if (newIdx < 0 || newIdx >= current.length) return;
    moveItemInArray(current, idx, newIdx);
    this.blockOrder.set(current);
  }

  saveLayout(): void {
    try {
      if (typeof window !== 'undefined' && window.localStorage) {
        localStorage.setItem(STORAGE_LAYOUT_KEY, JSON.stringify(this.blockOrder()));
      }
      this.isCustomizing.set(false);
      this.showFeedback('success', 'Distribución personalizada del dashboard guardada en este navegador.');
    } catch {
      this.isCustomizing.set(false);
    }
  }

  cancelCustomizing(): void {
    this.loadLayoutPreference();
    this.isCustomizing.set(false);
  }

  resetLayout(): void {
    try {
      if (typeof window !== 'undefined' && window.localStorage) {
        localStorage.removeItem(STORAGE_LAYOUT_KEY);
      }
      this.blockOrder.set(DEFAULT_BLOCK_ORDER);
      this.showFeedback('info', 'Se restauró el orden original del dashboard.');
    } catch {
      this.blockOrder.set(DEFAULT_BLOCK_ORDER);
    }
  }

  getBlockLabel(blockId: DashboardSectionId): string {
    switch (blockId) {
      case 'kpis': return '1. Métricas de Hardware (RAM, vCPU, NVMe, Concurrencia)';
      case 'chart': return '2. Telemetría de Carga Temporal (Últimos 60 minutos)';
      case 'split': return '3. Distribución por Materia & Panel de Incidencias';
      case 'containers': return '4. Auditoría Global de Instancias Activas';
    }
  }

  toggleAutoRefresh(): void {
    const nextState = !this.autoRefresh();
    this.autoRefresh.set(nextState);
    if (nextState) {
      this.startAutoRefresh();
      this.showFeedback('info', 'Actualización automática activada (intervalo de 10 segundos).');
    } else {
      this.stopAutoRefresh();
      this.showFeedback('info', 'Actualización automática desactivada.');
    }
  }

  private startAutoRefresh(): void {
    this.stopAutoRefresh();
    this.autoRefreshTimer = setInterval(() => {
      if (!this.isLoading() && this.refreshCooldown() === 0) {
        this.metricsService.fetchMetrics();
      }
    }, 10000);
  }

  private stopAutoRefresh(): void {
    if (this.autoRefreshTimer) {
      clearInterval(this.autoRefreshTimer);
      this.autoRefreshTimer = undefined;
    }
  }

  formatUptime(seconds?: number): string {
    if (!seconds || seconds <= 0) return 'Uptime: Calculando...';
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);

    if (days > 0) {
      return `Uptime: ${days} ${days === 1 ? 'día' : 'días'} ${hours} h`;
    }
    if (hours > 0) {
      return `Uptime: ${hours} h ${minutes} min`;
    }
    return `Uptime: ${minutes} min`;
  }

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
        this.refreshTimer = undefined;
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
        this.hibernateTimer = undefined;
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
      this.feedbackTimer = undefined;
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

  onResolveIncident(workspaceId: string): void {
    this.metricsService.resolveIncident(workspaceId).subscribe({
      next: () => {
        this.showFeedback('success', `Incidencia de la instancia ${workspaceId} resuelta y contadores OOM normalizados.`);
      },
      error: () => {
        this.showFeedback('warning', `No se pudo resolver la incidencia de la instancia ${workspaceId}.`);
      }
    });
  }

  onPauseWorkspace(workspaceId: string): void {
    this.metricsService.pauseWorkspace(workspaceId);
  }

  onViewLogs(workspaceId: string): void {
    const inc = this.health()?.incidents.find(i => i.workspace_id === workspaceId) || {
      id: 'inc-manual',
      type: 'oom_killed' as const,
      workspace_id: workspaceId,
      student_name: 'Estudiante',
      course_name: 'Laboratorio de Programación',
      description: 'Excedió cuota de memoria configurada (Exit code 137)',
      memory_limit_mb: 512,
      timestamp: 'Reciente'
    };
    this.selectedIncidentForLogs.set(inc);
  }

  ngOnDestroy(): void {
    this.stopAutoRefresh();
    if (this.refreshTimer) clearInterval(this.refreshTimer);
    if (this.hibernateTimer) clearInterval(this.hibernateTimer);
    if (this.feedbackTimer) clearTimeout(this.feedbackTimer);
  }
}
