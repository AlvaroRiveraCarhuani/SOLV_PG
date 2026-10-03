import { Component, OnInit, OnDestroy, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { CdkDropList, CdkDrag, CdkDragHandle, CdkDragDrop, moveItemInArray } from '@angular/cdk/drag-drop';
import { 
  LucideAlertTriangle, 
  LucideClock, 
  LucideCheckCircle, 
  LucideSliders,
  LucideGripVertical,
  LucideRotateCcw,
  LucideEye,
  LucideEyeOff
} from '@lucide/angular';
import { TeacherDashboardService } from '../services/teacher-dashboard.service';
import { TeacherLiveService } from '../services/teacher-live.service';
import { DashboardLayoutService, WidgetLayoutItem, WidgetColSpan } from '@core/services/dashboard-layout.service';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { LiveWorkspaceSession } from '../models/teacher.models';
import { ShadowTerminalModalComponent } from './shadow-terminal-modal/shadow-terminal-modal.component';
import { DashboardKpiWidgetComponent } from './widgets/dashboard-kpi-widget/dashboard-kpi-widget.component';
import { DashboardCoursesWidgetComponent } from './widgets/dashboard-courses-widget/dashboard-courses-widget.component';
import { DashboardAttentionWidgetComponent } from './widgets/dashboard-attention-widget/dashboard-attention-widget.component';
import { DashboardTelemetryWidgetComponent } from './widgets/dashboard-telemetry-widget/dashboard-telemetry-widget.component';

export const TEACHER_DASHBOARD_STORAGE_KEY = 'solv_teacher_dashboard_bento_layout';

export const TEACHER_DEFAULT_WIDGETS: WidgetLayoutItem[] = [
  { id: 'kpis', title: 'Métricas Generales', colSpan: 12, visible: true },
  { id: 'courses', title: 'Mis Materias Asignadas', colSpan: 8, visible: true },
  { id: 'attention', title: 'Atención Requerida', colSpan: 4, visible: true },
  { id: 'live_sessions', title: 'Sesiones en Vivo y Telemetría', colSpan: 12, visible: true }
];

@Component({
  selector: 'teacher-dashboard',
  standalone: true,
  imports: [
    CommonModule, 
    FormsModule,
    CdkDropList,
    CdkDrag,
    CdkDragHandle,
    LucideAlertTriangle, 
    LucideClock, 
    LucideCheckCircle, 
    LucideSliders,
    LucideGripVertical,
    LucideRotateCcw,
    LucideEye,
    LucideEyeOff,
    ComboboxComponent,
    ShadowTerminalModalComponent,
    DashboardKpiWidgetComponent,
    DashboardCoursesWidgetComponent,
    DashboardAttentionWidgetComponent,
    DashboardTelemetryWidgetComponent
  ],
  templateUrl: './teacher-dashboard.component.html',
  styleUrl: './teacher-dashboard.component.scss'
})
export class TeacherDashboardComponent implements OnInit, OnDestroy {
  private dashboardService = inject(TeacherDashboardService);
  private liveService = inject(TeacherLiveService);
  private layoutService = inject(DashboardLayoutService);

  courses = this.dashboardService.courses;
  attention = this.dashboardService.attention;
  periods = this.dashboardService.periods;
  isLoading = this.dashboardService.isLoading;
  liveSessions = this.liveService.liveSessions;
  isPollingActive = this.liveService.isPollingActive;
  pollingRateMs = this.liveService.pollingIntervalMs;

  selectedPeriodId = signal<string>('');
  selectedLiveSession = signal<LiveWorkspaceSession | null>(null);

  periodOptions = computed<ComboboxOption[]>(() => {
    const list = this.periods();
    if (list.length === 0) {
      return [{ id: '', label: 'Semestre Actual (Activo)', value: '' }];
    }
    return list.map(p => ({
      id: p.id,
      label: `${p.name} (${p.code})${p.is_active ? ' - Activo' : ''}`,
      value: p.id
    }));
  });

  // Modo Bento Grid & Personalización
  isCustomizing = signal<boolean>(false);
  widgets = signal<WidgetLayoutItem[]>(TEACHER_DEFAULT_WIDGETS);

  criticalCount = computed(() => this.attention()?.critical?.length ?? 0);
  warningCount = computed(() => this.attention()?.warning?.length ?? 0);
  standardCount = computed(() => this.attention()?.standard?.length ?? 0);

  totalStudentsCount = computed(() => {
    return this.courses().reduce((sum, c) => sum + (c.students_count || 0), 0);
  });

  activeNowCount = computed(() => {
    return this.courses().reduce((sum, c) => sum + (c.active_now || 0), 0);
  });

  totalPendingReviews = computed(() => {
    return this.courses().reduce((sum, c) => sum + (c.pending_review || 0), 0);
  });

  totalAtRisk = computed(() => {
    return this.courses().reduce((sum, c) => sum + (c.at_risk || 0), 0);
  });

  visibleWidgets = computed(() => {
    if (this.isCustomizing()) {
      return this.widgets(); // Mostrar todos en modo edición para poder reactivarlos
    }
    return this.widgets().filter(w => w.visible);
  });

  greetingMessage = computed(() => {
    const critical = this.criticalCount();
    if (critical > 0) {
      return {
        type: 'critical',
        text: `Atención: ${critical} contenedor(es) excedieron el límite de memoria en tus cursos.`
      };
    }
    const standard = this.standardCount();
    if (standard > 0) {
      return {
        type: 'warning',
        text: `Tenés ${standard} entrega(s) pendiente(s) de revisión.`
      };
    }
    return {
      type: 'info',
      text: `Panel docente activo · ${this.courses().length} materia(s) asignadas en el semestre en curso.`
    };
  });

  ngOnInit(): void {
    this.loadLayout();

    this.dashboardService.getAcademicPeriods().subscribe({
      next: (periodsList) => {
        const active = periodsList.find(p => p.is_active) || periodsList[0];
        if (active) {
          this.selectedPeriodId.set(active.id);
          this.loadData(active.id);
        } else {
          this.loadData();
        }
      },
      error: () => this.loadData()
    });

    this.liveService.startPolling(3000);
  }

  ngOnDestroy(): void {
    this.liveService.pausePolling();
  }

  loadLayout(): void {
    const loaded = this.layoutService.loadLayout(TEACHER_DASHBOARD_STORAGE_KEY, TEACHER_DEFAULT_WIDGETS);
    this.widgets.set(loaded);
  }

  toggleCustomize(): void {
    this.isCustomizing.update(v => !v);
  }

  toggleCustomizing(): void {
    this.toggleCustomize();
  }

  setWidgetColSpan(widgetId: string, span: WidgetColSpan): void {
    const updated = this.widgets().map(w => {
      if (w.id === widgetId) {
        return { ...w, colSpan: span };
      }
      return w;
    });
    this.widgets.set(updated);
    this.layoutService.saveLayout(TEACHER_DASHBOARD_STORAGE_KEY, updated);
  }

  toggleWidgetVisibility(widgetId: string): void {
    const updated = this.widgets().map(w => {
      if (w.id === widgetId) {
        return { ...w, visible: !w.visible };
      }
      return w;
    });
    this.widgets.set(updated);
    this.layoutService.saveLayout(TEACHER_DASHBOARD_STORAGE_KEY, updated);
  }

  resetLayout(): void {
    this.widgets.set([...TEACHER_DEFAULT_WIDGETS]);
    this.layoutService.resetLayout(TEACHER_DASHBOARD_STORAGE_KEY, TEACHER_DEFAULT_WIDGETS);
  }

  onDropWidget(event: CdkDragDrop<WidgetLayoutItem[]>): void {
    const currentList = [...this.widgets()];
    moveItemInArray(currentList, event.previousIndex, event.currentIndex);
    this.widgets.set(currentList);
    this.layoutService.saveLayout(TEACHER_DASHBOARD_STORAGE_KEY, currentList);
  }

  loadData(periodId?: string): void {
    this.dashboardService.loadDashboardData(periodId).subscribe();
  }

  onPeriodChange(periodId: string): void {
    this.selectedPeriodId.set(periodId);
    this.loadData(periodId);
  }

  setPollingRate(rateMs: number): void {
    this.liveService.setPollingInterval(rateMs);
  }

  refreshLiveSessions(): void {
    this.liveService.loadLiveSessions().subscribe();
  }

  openTerminal(session: LiveWorkspaceSession): void {
    this.selectedLiveSession.set(session);
  }

  closeTerminal(): void {
    this.selectedLiveSession.set(null);
  }
}
