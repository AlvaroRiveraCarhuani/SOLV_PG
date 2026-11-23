import { ChangeDetectionStrategy, Component, OnInit, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { StudentService, DueAssignment } from '@core/services/student.service';
import { CourseColorService, CourseThemeStyle } from '@core/services/course-color.service';
import { SearchBarComponent } from '@shared/components/search-bar/search-bar.component';
import { StatusTabsComponent, StatusTabItem } from '@shared/components/status-tabs/status-tabs.component';
import { FilterPopoverComponent } from '@shared/components/filter-popover/filter-popover.component';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';
import { ViewSwitcherComponent, ViewMode } from '@shared/components/view-switcher/view-switcher.component';
import { CourseColorPickerComponent } from '@shared/components/course-color-picker/course-color-picker.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { DismissibleDirective } from '@shared/directives/dismissible.directive';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import {
  LucideArrowLeft,
  LucideBookOpen,
  LucideCheck,
  LucideHistory,
  LucidePalette,
  LucideX
} from '@lucide/angular';
import { StudentAulaService } from '../../services/student-aula.service';

export type CourseProgressFilter = 'all' | 'unstarted' | 'in_progress' | 'completed';
export type CourseActivityFilter = 'all' | 'with_due' | 'with_live';
type FilterMode = 'all' | 'in_progress' | 'completed';

@Component({
  selector: 'student-courses',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    CommonModule,
    RouterLink,
    SearchBarComponent,
    StatusTabsComponent,
    FilterPopoverComponent,
    SkeletonLoaderComponent,
    CourseColorPickerComponent,
    MachineDataDirective,
    ViewSwitcherComponent,
    DismissibleDirective,
    DateTextPipe,
    LucideArrowLeft,
    LucideBookOpen,
    LucideCheck,
    LucideHistory,
    LucidePalette,
    LucideX
  ],
  templateUrl: './student-courses.component.html',
  styleUrl: './student-courses.component.scss'
})
export class StudentCoursesComponent implements OnInit {
  readonly studentService = inject(StudentService);
  readonly courseColorService = inject(CourseColorService);
  private readonly aulasService = inject(StudentAulaService);

  searchQuery = signal('');
  filterMode = signal<FilterMode>('all');
  progressFilter = signal<CourseProgressFilter>('all');
  activityFilter = signal<CourseActivityFilter>('all');
  viewMode = signal<ViewMode>('cards');
  activeColorPickerCourseId = signal<string | null>(null);

  subjects = computed(() => this.studentService.dashboardData()?.subjects ?? []);
  dueAssignments = computed(() => this.studentService.dueAssignments());
  isLoading = computed(() => this.studentService.loading());
  activeAulas = computed(() => this.aulasService.aulas());

  statusTabs = computed<StatusTabItem[]>(() => {
    const all = this.subjects();
    const completedCount = all.filter(
      item => (item.progress_percent ?? 0) === 100 && (item.total_exercises ?? 0) > 0
    ).length;
    return [
      { id: 'all', label: 'Todas', count: all.length, badgeVariant: 'neutral' },
      { id: 'in_progress', label: 'En curso', count: all.length - completedCount, badgeVariant: 'neutral' },
      { id: 'completed', label: 'Completadas', count: completedCount, badgeVariant: 'neutral' }
    ];
  });

  readonly activeFilterCount = computed(() => {
    let count = 0;
    if (this.progressFilter() !== 'all') count++;
    if (this.activityFilter() !== 'all') count++;
    return count;
  });

  readonly hasActiveFilters = computed(() =>
    this.searchQuery().trim() !== '' ||
    this.filterMode() !== 'all' ||
    this.progressFilter() !== 'all' ||
    this.activityFilter() !== 'all'
  );

  readonly selectedProgressLabel = computed(() => {
    switch (this.progressFilter()) {
      case 'unstarted': return 'Sin iniciar (0%)';
      case 'in_progress': return 'En progreso (1% - 99%)';
      case 'completed': return 'Completadas (100%)';
      default: return '';
    }
  });

  readonly selectedActivityLabel = computed(() => {
    switch (this.activityFilter()) {
      case 'with_due': return 'Con entregas próximas';
      case 'with_live': return 'Con clase en vivo';
      default: return '';
    }
  });

  filteredSubjects = computed(() => {
    const query = this.searchQuery().trim().toLowerCase();
    const tab = this.filterMode();
    const progress = this.progressFilter();
    const activity = this.activityFilter();

    let list = this.subjects().filter(item => {
      // 1. Filtro por pestaña de estado
      if (tab === 'in_progress' && (item.progress_percent ?? 0) === 100 && (item.total_exercises ?? 0) > 0) {
        return false;
      }
      if (tab === 'completed' && ((item.progress_percent ?? 0) < 100 || (item.total_exercises ?? 0) === 0)) {
        return false;
      }

      // 2. Filtro popover: Avance
      const pct = item.progress_percent ?? 0;
      const total = item.total_exercises ?? 0;
      if (progress === 'unstarted' && pct > 0) {
        return false;
      }
      if (progress === 'in_progress' && (pct === 0 || (pct === 100 && total > 0))) {
        return false;
      }
      if (progress === 'completed' && (pct < 100 || total === 0)) {
        return false;
      }

      // 3. Filtro popover: Actividad
      if (activity === 'with_due' && !this.nextDueFor(item.subject.id)) {
        return false;
      }
      if (activity === 'with_live' && !this.hasLiveAula(item.subject.id)) {
        return false;
      }

      // 4. Búsqueda por texto
      if (query) {
        return (
          item.subject.name.toLowerCase().includes(query) ||
          item.subject.code.toLowerCase().includes(query)
        );
      }

      return true;
    });

    return list;
  });

  setFilterMode(mode: string): void {
    this.filterMode.set(mode as FilterMode);
  }

  setProgressFilter(mode: CourseProgressFilter): void {
    this.progressFilter.set(mode);
  }

  setActivityFilter(mode: CourseActivityFilter): void {
    this.activityFilter.set(mode);
  }

  clearPopoverFilters(): void {
    this.progressFilter.set('all');
    this.activityFilter.set('all');
  }

  clearAllFilters(): void {
    this.searchQuery.set('');
    this.filterMode.set('all');
    this.progressFilter.set('all');
    this.activityFilter.set('all');
  }

  clearFilters(): void {
    this.clearAllFilters();
  }

  getCourseColor(courseId: string, courseCode?: string): string {
    return this.courseColorService.getCourseColor(courseId, courseCode);
  }

  getCourseThemeStyle(courseId: string, courseCode?: string): CourseThemeStyle {
    return this.courseColorService.getCourseThemeStyle(this.getCourseColor(courseId, courseCode));
  }

  toggleColorPicker(courseId: string, event?: Event): void {
    if (event) {
      event.stopPropagation();
    }
    this.activeColorPickerCourseId.update(current => current === courseId ? null : courseId);
  }

  onCourseColorChanged(courseId: string, color: string): void {
    this.courseColorService.setCourseColor(courseId, color);
  }

  nextDueFor(subjectId: string): DueAssignment | undefined {
    return this.dueAssignments().find(due => due.subject_id === subjectId);
  }

  hasLiveAula(subjectId: string): boolean {
    return this.activeAulas().some(aula => aula.subject_id === subjectId && aula.status === 'in_progress');
  }

  async ngOnInit(): Promise<void> {
    void this.aulasService.load();
    if (!this.studentService.dashboardData()) {
      await this.studentService.loadDashboard();
    }
  }
}
