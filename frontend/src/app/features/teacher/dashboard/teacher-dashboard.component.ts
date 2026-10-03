import { Component, OnInit, inject, signal, computed, effect } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { CdkDropList, CdkDrag, CdkDragHandle, CdkDragDrop, moveItemInArray } from '@angular/cdk/drag-drop';
import { 
  LucideBookOpen, 
  LucideArrowRight, 
  LucideCheckCircle2, 
  LucideAlertTriangle, 
  LucideClock, 
  LucideCheckCircle, 
  LucideFlame, 
  LucideShieldAlert,
  LucideTerminal,
  LucideSliders,
  LucideGripVertical,
  LucideRotateCcw,
  LucideEye,
  LucideEyeOff,
  LucidePalette
} from '@lucide/angular';
import { TeacherDashboardService } from '../services/teacher-dashboard.service';
import { TeacherLiveService } from '../services/teacher-live.service';
import { DashboardLayoutService, WidgetLayoutItem, WidgetColSpan } from '@core/services/dashboard-layout.service';
import { CourseColorService, CourseThemeStyle } from '@core/services/course-color.service';
import { CourseColorPickerComponent } from '@shared/components/course-color-picker/course-color-picker.component';
import { DismissibleDirective } from '@shared/directives/dismissible.directive';
import { LiveWorkspaceSession } from '../models/teacher.models';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { ViewSwitcherComponent, ViewMode } from '@shared/components/view-switcher/view-switcher.component';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';
import { ShadowTerminalModalComponent } from './shadow-terminal-modal/shadow-terminal-modal.component';

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
    RouterLink, 
    FormsModule,
    CdkDropList,
    CdkDrag,
    CdkDragHandle,
    CourseColorPickerComponent,
    DismissibleDirective,
    LucideBookOpen, 
    LucideArrowRight, 
    LucideCheckCircle2, 
    LucideAlertTriangle, 
    LucideClock, 
    LucideCheckCircle, 
    LucideFlame, 
    LucideShieldAlert,
    LucideTerminal,
    LucideSliders,
    LucideGripVertical,
    LucideRotateCcw,
    LucideEye,
    LucideEyeOff,
    LucidePalette,
    DateTextPipe, 
    MachineDataDirective,
    ViewSwitcherComponent,
    SkeletonLoaderComponent,
    ShadowTerminalModalComponent
  ],
  templateUrl: './teacher-dashboard.component.html',
  styleUrl: './teacher-dashboard.component.scss'
})
export class TeacherDashboardComponent implements OnInit {
  private dashboardService = inject(TeacherDashboardService);
  private liveService = inject(TeacherLiveService);
  private layoutService = inject(DashboardLayoutService);
  private courseColorService = inject(CourseColorService);

  courses = this.dashboardService.courses;
  attention = this.dashboardService.attention;
  periods = this.dashboardService.periods;
  isLoading = this.dashboardService.isLoading;
  liveSessions = this.liveService.liveSessions;

  viewMode = signal<ViewMode>((localStorage.getItem('solv_teacher_view_mode') as ViewMode) || 'cards');
  selectedPeriodId = signal<string>('');
  selectedLiveSession = signal<LiveWorkspaceSession | null>(null);
  activeColorPickerCourseId = signal<string | null>(null);

  // Modo Bento Grid & Personalización
  isCustomizing = signal<boolean>(false);
  widgets = signal<WidgetLayoutItem[]>(TEACHER_DEFAULT_WIDGETS);

  constructor() {
    effect(() => {
      const mode = this.viewMode();
      try {
        localStorage.setItem('solv_teacher_view_mode', mode);
      } catch {}
    });
  }

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

    this.liveService.loadLiveSessions().subscribe();
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
    const reset = this.layoutService.resetLayout(TEACHER_DASHBOARD_STORAGE_KEY, TEACHER_DEFAULT_WIDGETS);
    this.widgets.set(reset);
  }

  onDropWidget(event: CdkDragDrop<WidgetLayoutItem[]>): void {
    const updated = [...this.widgets()];
    moveItemInArray(updated, event.previousIndex, event.currentIndex);
    this.widgets.set(updated);
    this.layoutService.saveLayout(TEACHER_DASHBOARD_STORAGE_KEY, updated);
  }

  loadData(periodId?: string): void {
    this.dashboardService.loadDashboardData(periodId).subscribe();
  }

  onPeriodChange(periodId: string): void {
    this.selectedPeriodId.set(periodId);
    this.loadData(periodId);
  }

  getCourseColor(courseId: string, courseCode?: string): string {
    return this.courseColorService.getCourseColor(courseId, courseCode);
  }

  getCourseThemeStyle(courseId: string, courseCode?: string): CourseThemeStyle {
    const color = this.getCourseColor(courseId, courseCode);
    return this.courseColorService.getCourseThemeStyle(color);
  }

  toggleCourseColorPicker(courseId: string, event?: Event): void {
    if (event) {
      event.stopPropagation();
    }
    this.activeColorPickerCourseId.update(current => current === courseId ? null : courseId);
  }

  onCourseColorChanged(courseId: string, color: string): void {
    this.courseColorService.setCourseColor(courseId, color);
  }

  openTerminal(session: LiveWorkspaceSession): void {
    this.selectedLiveSession.set(session);
  }

  closeTerminal(): void {
    this.selectedLiveSession.set(null);
  }
}
