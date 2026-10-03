import { Component, OnInit, inject, computed, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { CdkDropList, CdkDrag, CdkDragHandle, CdkDragDrop, moveItemInArray } from '@angular/cdk/drag-drop';
import { AuthService } from '@core/services/auth.service';
import { StudentService, StudentSubjectItem, DueAssignment } from '@core/services/student.service';
import { DashboardLayoutService, WidgetLayoutItem, WidgetColSpan } from '@core/services/dashboard-layout.service';
import { CourseColorService, CourseThemeStyle } from '@core/services/course-color.service';
import { StatusBadgeComponent } from '@shared/components/status-badge/status-badge.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { DismissibleDirective } from '@shared/directives/dismissible.directive';
import { DateTextPipe, formatSolvDate } from '@shared/pipes/date-text.pipe';
import { 
  LucidePlay, 
  LucideRotateCw, 
  LucidePlus, 
  LucideClock, 
  LucideBookOpen,
  LucideSliders,
  LucideGripVertical,
  LucideRotateCcw,
  LucideEye,
  LucideEyeOff,
  LucidePalette
} from '@lucide/angular';
import { CourseColorPickerComponent } from '@shared/components/course-color-picker/course-color-picker.component';

export const STUDENT_DASHBOARD_STORAGE_KEY = 'solv_student_dashboard_bento_layout';

export const STUDENT_DEFAULT_WIDGETS: WidgetLayoutItem[] = [
  { id: 'labs', title: 'Mis Laboratorios Activos', colSpan: 8, visible: true },
  { id: 'agenda', title: 'Para Hoy • Próximas Entregas', colSpan: 4, visible: true },
  { id: 'progress', title: 'Mi Progreso Académico', colSpan: 6, visible: true },
  { id: 'recent', title: 'Accesos Recientes', colSpan: 6, visible: true }
];

@Component({
  selector: 'student-dashboard',
  standalone: true,
  imports: [
    CommonModule, 
    RouterModule, 
    FormsModule,
    CdkDropList,
    CdkDrag,
    CdkDragHandle,
    CourseColorPickerComponent,
    DismissibleDirective,
    StatusBadgeComponent,
    MachineDataDirective,
    DateTextPipe,
    LucidePlay, 
    LucideRotateCw, 
    LucidePlus, 
    LucideClock, 
    LucideBookOpen,
    LucideSliders,
    LucideGripVertical,
    LucideRotateCcw,
    LucideEye,
    LucideEyeOff,
    LucidePalette
  ],
  templateUrl: './student-dashboard.component.html',
  styleUrl: './student-dashboard.component.scss'
})
export class StudentDashboardComponent implements OnInit {
  authService = inject(AuthService);
  studentService = inject(StudentService);
  layoutService = inject(DashboardLayoutService);
  courseColorService = inject(CourseColorService);

  isActionLoading = signal<boolean>(false);
  isCustomizing = signal<boolean>(false);
  widgets = signal<WidgetLayoutItem[]>(STUDENT_DEFAULT_WIDGETS);
  activeColorPickerCourseId = signal<string | null>(null);

  studentFirstName = computed(() => {
    return this.authService.currentUser()?.first_name || 'Estudiante';
  });

  timeGreeting = computed(() => {
    const hour = new Date().getHours();
    if (hour < 12) return 'Buenos días';
    if (hour < 19) return 'Buenas tardes';
    return 'Buenas noches';
  });

  subjects = computed(() => {
    return this.studentService.dashboardData()?.subjects || [];
  });

  dueAssignments = computed(() => {
    return this.studentService.dueAssignments();
  });

  dueCount = computed(() => {
    return this.dueAssignments().length;
  });

  visibleWidgets = computed(() => {
    if (this.isCustomizing()) {
      return this.widgets();
    }
    return this.widgets().filter(w => w.visible);
  });

  ngOnInit(): void {
    this.studentService.loadDashboard();
    this.loadLayout();
  }

  loadLayout(): void {
    const loaded = this.layoutService.loadLayout(STUDENT_DASHBOARD_STORAGE_KEY, STUDENT_DEFAULT_WIDGETS);
    this.widgets.set(loaded);
  }

  toggleCustomize(): void {
    this.isCustomizing.update(v => !v);
  }

  setWidgetColSpan(widgetId: string, colSpan: WidgetColSpan): void {
    const updated = this.widgets().map(w => {
      if (w.id === widgetId) {
        return { ...w, colSpan };
      }
      return w;
    });
    this.widgets.set(updated);
    this.layoutService.saveLayout(STUDENT_DASHBOARD_STORAGE_KEY, updated);
  }

  toggleWidgetVisibility(widgetId: string): void {
    const updated = this.widgets().map(w => {
      if (w.id === widgetId) {
        return { ...w, visible: !w.visible };
      }
      return w;
    });
    this.widgets.set(updated);
    this.layoutService.saveLayout(STUDENT_DASHBOARD_STORAGE_KEY, updated);
  }

  resetLayout(): void {
    const reset = this.layoutService.resetLayout(STUDENT_DASHBOARD_STORAGE_KEY, STUDENT_DEFAULT_WIDGETS);
    this.widgets.set(reset);
  }

  onDropWidget(event: CdkDragDrop<WidgetLayoutItem[]>): void {
    const updated = [...this.widgets()];
    moveItemInArray(updated, event.previousIndex, event.currentIndex);
    this.widgets.set(updated);
    this.layoutService.saveLayout(STUDENT_DASHBOARD_STORAGE_KEY, updated);
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

  formatDate(dateStr: string): string {
    if (!dateStr || dateStr.trim() === '') return 'Próximamente';
    return formatSolvDate(dateStr, 'datetime') ?? dateStr;
  }

  async openIDE(item: StudentSubjectItem): Promise<void> {
    if (item.active_workspace?.access_url) {
      window.open(item.active_workspace.access_url, '_blank');
    }
  }

  async resumeIDE(item: StudentSubjectItem): Promise<void> {
    this.isActionLoading.set(true);
    try {
      await this.studentService.startWorkspace(item.subject.id);
    } finally {
      this.isActionLoading.set(false);
    }
  }

  async startNewWorkspace(item: StudentSubjectItem): Promise<void> {
    this.isActionLoading.set(true);
    try {
      await this.studentService.startWorkspace(item.subject.id);
    } finally {
      this.isActionLoading.set(false);
    }
  }
}
