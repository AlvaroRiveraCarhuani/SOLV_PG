import { Component, OnInit, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
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
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
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
    ComboboxComponent,
    FuzzingModalComponent
  ],
  templateUrl: './teacher-exercises.component.html',
  styleUrl: './teacher-exercises.component.scss'
})
export class TeacherExercisesComponent implements OnInit {
  private dashboardService = inject(TeacherDashboardService);
  private courseService = inject(TeacherCourseService);
  private router = inject(Router);

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

  courseComboboxOptions = computed<ComboboxOption[]>(() => {
    const opts: ComboboxOption[] = [
      { id: 'all', label: 'Todos', value: 'all' }
    ];
    this.courses().forEach(c => {
      opts.push({ id: c.id, label: `${c.code} — ${c.name}`, value: c.id });
    });
    return opts;
  });

  selectedCourseLabel = computed(() => {
    const sel = this.courseFilter();
    if (sel === 'all') return 'Todos';
    const match = this.courses().find(c => c.id === sel);
    return match ? `${match.code} — ${match.name}` : 'Todos';
  });

  modalityComboboxOptions = computed<ComboboxOption[]>(() => [
    { id: 'all', label: 'Todos', value: 'all' },
    { id: 'ALGORITMO', label: 'Juez Automático', value: 'ALGORITMO' },
    { id: 'WORKSPACE', label: 'Laboratorio VS Code', value: 'WORKSPACE' }
  ]);

  selectedModalityLabel = computed(() => {
    const sel = this.modalityFilter();
    if (sel === 'ALGORITMO') return 'Juez Automático';
    if (sel === 'WORKSPACE') return 'Laboratorio VS Code';
    return 'Todos';
  });

  statusComboboxOptions = computed<ComboboxOption[]>(() => [
    { id: 'all', label: 'Todos', value: 'all' },
    { id: 'published', label: 'Publicados', value: 'published' },
    { id: 'draft', label: 'Borradores', value: 'draft' }
  ]);

  selectedStatusLabel = computed(() => {
    const sel = this.statusFilter();
    if (sel === 'published') return 'Publicados';
    if (sel === 'draft') return 'Borradores';
    return 'Todos';
  });

  onCourseSelected(opt: ComboboxOption): void {
    this.courseFilter.set(opt.value);
  }

  onModalitySelected(opt: ComboboxOption): void {
    this.modalityFilter.set(opt.value);
  }

  onStatusSelected(opt: ComboboxOption): void {
    this.statusFilter.set(opt.value);
  }

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
    const courseId = course ? course.id : 'all';
    this.router.navigate(['/teacher/courses', courseId, 'exercises', 'new']);
  }

  openEditModal(lab: EnrichedLabItem): void {
    this.router.navigate(['/teacher/courses', lab.subject_id, 'exercises', lab.id, 'edit']);
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
