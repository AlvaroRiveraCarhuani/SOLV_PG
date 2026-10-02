import { Component, OnInit, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import {
  LucidePlus,
  LucideSearch,
  LucideCode,
  LucideDatabase,
  LucideTerminal,
  LucideSparkles,
  LucideEdit,
  LucideExternalLink,
  LucideClock,
  LucideLayers,
  LucideUsers
} from '@lucide/angular';
import { TeacherDashboardService } from '../services/teacher-dashboard.service';
import { TeacherCourseService } from '../services/teacher-course.service';
import { TeacherLabStats, TeacherCourseSummary } from '../models/teacher.models';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';
import { ExerciseEditorModalComponent } from '../courses/exercise-editor/exercise-editor-modal.component';
import { FuzzingModalComponent } from '../evaluations/fuzzing-modal/fuzzing-modal.component';
import { forkJoin, of } from 'rxjs';
import { catchError } from 'rxjs/operators';

export interface EnrichedLabItem extends TeacherLabStats {
  subject_id: string;
  subject_name: string;
  subject_code: string;
}

@Component({
  selector: 'teacher-exercises',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    RouterLink,
    LucidePlus,
    LucideSearch,
    LucideCode,
    LucideDatabase,
    LucideTerminal,
    LucideSparkles,
    LucideEdit,
    LucideExternalLink,
    LucideClock,
    LucideLayers,
    LucideUsers,
    MachineDataDirective,
    DateTextPipe,
    SkeletonLoaderComponent,
    ExerciseEditorModalComponent,
    FuzzingModalComponent
  ],
  templateUrl: './teacher-exercises.component.html',
  styleUrl: './teacher-exercises.component.scss'
})
export class TeacherExercisesComponent implements OnInit {
  private dashboardService = inject(TeacherDashboardService);
  private courseService = inject(TeacherCourseService);

  courses = this.dashboardService.courses;
  isLoading = signal<boolean>(false);
  allExercises = signal<EnrichedLabItem[]>([]);

  // Filtros reactivos
  courseFilter = signal<string>('all');
  modalityFilter = signal<string>('all');
  statusFilter = signal<string>('all');
  searchTerm = signal<string>('');

  // Modales
  isEditorOpen = signal<boolean>(false);
  exerciseToEdit = signal<TeacherLabStats | null>(null);
  editorSubjectId = signal<string>('');
  editorSubjectName = signal<string>('');

  isFuzzingOpen = signal<boolean>(false);
  exerciseForFuzzing = signal<EnrichedLabItem | null>(null);

  // Computeds
  filteredExercises = computed(() => {
    let items = this.allExercises();
    const course = this.courseFilter();
    const modality = this.modalityFilter();
    const status = this.statusFilter();
    const query = this.searchTerm().trim().toLowerCase();

    if (course !== 'all') {
      items = items.filter(e => e.subject_id === course);
    }

    if (modality !== 'all') {
      items = items.filter(e => {
        const type = (e.type || 'ALGORITMO').toUpperCase();
        return type === modality;
      });
    }

    if (status !== 'all') {
      items = items.filter(e => e.status === status);
    }

    if (query) {
      items = items.filter(e =>
        e.title.toLowerCase().includes(query) ||
        e.subject_name.toLowerCase().includes(query) ||
        (e.language && e.language.toLowerCase().includes(query))
      );
    }

    return items;
  });

  kpiStats = computed(() => {
    const list = this.allExercises();
    const total = list.length;
    const published = list.filter(e => e.status === 'published').length;
    const drafts = list.filter(e => e.status === 'draft').length;
    const totalSubmissions = list.reduce((acc, curr) => acc + (curr.submissions_count || 0), 0);

    return { total, published, drafts, totalSubmissions };
  });

  ngOnInit(): void {
    this.loadData();
  }

  loadData(): void {
    this.isLoading.set(true);
    this.dashboardService.loadDashboardData().subscribe({
      next: data => {
        const coursesList = data.courses || [];
        if (coursesList.length === 0) {
          this.allExercises.set([]);
          this.isLoading.set(false);
          return;
        }

        const requests = coursesList.map(c =>
          this.courseService.getCourseLabs(c.id).pipe(
            catchError(() => of([] as TeacherLabStats[]))
          )
        );

        forkJoin(requests).subscribe({
          next: responses => {
            const aggregated: EnrichedLabItem[] = [];
            responses.forEach((labs, index) => {
              const course = coursesList[index];
              labs.forEach(lab => {
                aggregated.push({
                  ...lab,
                  subject_id: course.id,
                  subject_name: course.name,
                  subject_code: course.code
                });
              });
            });
            this.allExercises.set(aggregated);
            this.isLoading.set(false);
          },
          error: () => this.isLoading.set(false)
        });
      },
      error: () => this.isLoading.set(false)
    });
  }

  openCreateModal(preselectedCourseId?: string): void {
    const coursesList = this.courses();
    const course = coursesList.find(c => c.id === preselectedCourseId) || coursesList[0];
    if (course) {
      this.editorSubjectId.set(course.id);
      this.editorSubjectName.set(`${course.code} — ${course.name}`);
    } else {
      this.editorSubjectId.set('');
      this.editorSubjectName.set('');
    }
    this.exerciseToEdit.set(null);
    this.isEditorOpen.set(true);
  }

  openEditModal(lab: EnrichedLabItem): void {
    this.editorSubjectId.set(lab.subject_id);
    this.editorSubjectName.set(`${lab.subject_code} — ${lab.subject_name}`);
    this.exerciseToEdit.set(lab);
    this.isEditorOpen.set(true);
  }

  closeEditorModal(): void {
    this.isEditorOpen.set(false);
    this.exerciseToEdit.set(null);
  }

  onExerciseSaved(): void {
    this.closeEditorModal();
    this.loadData();
  }

  openFuzzingModal(lab: EnrichedLabItem): void {
    this.exerciseForFuzzing.set(lab);
    this.isFuzzingOpen.set(true);
  }

  closeFuzzingModal(): void {
    this.isFuzzingOpen.set(false);
    this.exerciseForFuzzing.set(null);
  }

  onFuzzingApplied(): void {
    this.closeFuzzingModal();
    this.loadData();
  }
}
