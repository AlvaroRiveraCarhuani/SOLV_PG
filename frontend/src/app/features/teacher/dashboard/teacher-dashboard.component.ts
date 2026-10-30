import { Component, OnInit, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { CdkDropList, CdkDrag, CdkDragHandle, CdkDragDrop, moveItemInArray } from '@angular/cdk/drag-drop';
import { 
  LucideAlertTriangle, 
  LucideClock, 
  LucideCheckCircle, 
  LucideSliders,
  LucideGripVertical,
  LucideRotateCcw,
  LucideEye,
  LucideEyeOff,
  LucideArrowRight
} from '@lucide/angular';
import { TeacherDashboardService } from '../services/teacher-dashboard.service';
import { DashboardLayoutService, WidgetLayoutItem, WidgetColSpan } from '@core/services/dashboard-layout.service';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { DashboardCoursesWidgetComponent } from './widgets/dashboard-courses-widget/dashboard-courses-widget.component';
import { DashboardAttentionWidgetComponent } from './widgets/dashboard-attention-widget/dashboard-attention-widget.component';

export const TEACHER_DASHBOARD_STORAGE_KEY = 'solv_teacher_dashboard_bento_layout_v2';

export const TEACHER_DEFAULT_WIDGETS: WidgetLayoutItem[] = [
  { id: 'courses', title: 'Mis Materias Asignadas', colSpan: 8, visible: true },
  { id: 'attention', title: 'Tareas Pendientes', colSpan: 4, visible: true }
];

@Component({
  selector: 'teacher-dashboard',
  standalone: true,
  imports: [
    CommonModule, 
    FormsModule,
    RouterLink,
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
    LucideArrowRight,
    ComboboxComponent,
    DateTextPipe,
    DashboardCoursesWidgetComponent,
    DashboardAttentionWidgetComponent
  ],
  templateUrl: './teacher-dashboard.component.html',
  styleUrl: './teacher-dashboard.component.scss'
})
export class TeacherDashboardComponent implements OnInit {
  private dashboardService = inject(TeacherDashboardService);
  private layoutService = inject(DashboardLayoutService);

  courses = this.dashboardService.courses;
  attention = this.dashboardService.attention;
  periods = this.dashboardService.periods;
  isLoading = this.dashboardService.isLoading;

  selectedPeriodId = signal<string>('');
  today = new Date();

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

  visibleWidgets = computed(() => {
    if (this.isCustomizing()) {
      return this.widgets();
    }
    return this.widgets().filter(w => w.visible);
  });

  greetingMessage = computed(() => {
    const critical = this.criticalCount();
    if (critical > 0) {
      return {
        type: 'critical',
        text: `Atención: ${critical} estudiante(s) presentaron bloqueos de ejecución en sus prácticas.`
      };
    }
    const standard = this.standardCount();
    if (standard > 0) {
      return {
        type: 'warning',
        text: `Tenés ${standard} entrega(s) pendiente(s) de calificación en tus materias.`
      };
    }
    return {
      type: 'info',
      text: `Todo al día · ${this.courses().length} materia(s) asignadas en el período activo.`
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
  }

  loadLayout(): void {
    const loaded = this.layoutService.loadLayout(TEACHER_DASHBOARD_STORAGE_KEY, TEACHER_DEFAULT_WIDGETS);
    this.widgets.set(loaded);
  }

  toggleCustomizing(): void {
    this.isCustomizing.update(v => !v);
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
}
