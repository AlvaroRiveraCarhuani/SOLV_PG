import { ChangeDetectionStrategy, Component, OnInit, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { StudentService, DueAssignment } from '@core/services/student.service';
import { CourseColorService, CourseThemeStyle } from '@core/services/course-color.service';
import { SearchBarComponent } from '@shared/components/search-bar/search-bar.component';
import { StatusTabsComponent, StatusTabItem } from '@shared/components/status-tabs/status-tabs.component';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';
import { ViewSwitcherComponent, ViewMode } from '@shared/components/view-switcher/view-switcher.component';
import { CourseColorPickerComponent } from '@shared/components/course-color-picker/course-color-picker.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { DismissibleDirective } from '@shared/directives/dismissible.directive';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import {
  LucideArrowLeft,
  LucideBookOpen,
  LucideHistory,
  LucidePalette
} from '@lucide/angular';
import { StudentAulaService } from '../../services/student-aula.service';

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
    SkeletonLoaderComponent,
    CourseColorPickerComponent,
    MachineDataDirective,
    ViewSwitcherComponent,
    DismissibleDirective,
    DateTextPipe,
    LucideArrowLeft,
    LucideBookOpen,
    LucideHistory,
    LucidePalette
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
      { id: 'all', label: 'Todas', count: all.length },
      { id: 'in_progress', label: 'En curso', count: all.length - completedCount },
      { id: 'completed', label: 'Completadas', count: completedCount }
    ];
  });

  filteredSubjects = computed(() => {
    const query = this.searchQuery().trim().toLowerCase();
    let list = this.subjects().filter(item => {
      switch (this.filterMode()) {
        case 'in_progress':
          return (item.progress_percent ?? 0) < 100;
        case 'completed':
          return (item.progress_percent ?? 0) === 100 && (item.total_exercises ?? 0) > 0;
        default:
          return true;
      }
    });

    if (query) {
      list = list.filter(item =>
        item.subject.name.toLowerCase().includes(query) ||
        item.subject.code.toLowerCase().includes(query)
      );
    }

    return list;
  });

  setFilterMode(mode: string): void {
    this.filterMode.set(mode as FilterMode);
  }

  clearFilters(): void {
    this.searchQuery.set('');
    this.filterMode.set('all');
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
