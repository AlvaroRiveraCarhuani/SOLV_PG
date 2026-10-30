import { Component, OnInit, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import {
  LucidePlus,
  LucideSearch,
  LucideCode,
  LucideTerminal,
  LucideSparkles,
  LucideEdit,
  LucideExternalLink,
  LucideClock,
  LucideLayers,
  LucideUsers,
  LucideBookOpen
} from '@lucide/angular';
import { TeacherDashboardService } from '../services/teacher-dashboard.service';
import { TeacherCourseService } from '../services/teacher-course.service';
import { TeacherExerciseBankItem, TeacherLabStats } from '../models/teacher.models';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { FuzzingModalComponent } from '../evaluations/fuzzing-modal/fuzzing-modal.component';

export type EnrichedLabItem = TeacherExerciseBankItem;

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
    LucideTerminal,
    LucideSparkles,
    LucideEdit,
    LucideExternalLink,
    LucideClock,
    LucideLayers,
    LucideUsers,
    LucideBookOpen,
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
  hasCourses = computed(() => this.courses().length > 0);
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
        const env = (e.environment_type || 'JUEZ_EFIMERO').toUpperCase();
        return env === modality;
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
    { id: 'JUEZ_EFIMERO', label: 'Juez Virtual', value: 'JUEZ_EFIMERO' },
    { id: 'IDE_PERSISTENTE', label: 'IDE Persistente (Workspace)', value: 'IDE_PERSISTENTE' }
  ]);

  selectedModalityLabel = computed(() => {
    const sel = this.modalityFilter();
    if (sel === 'JUEZ_EFIMERO') return 'Juez Virtual';
    if (sel === 'IDE_PERSISTENTE') return 'IDE Persistente (Workspace)';
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
        // Read model del banco: agrega materias y semántica de revisión por entorno.
        this.courseService.getExerciseBank().subscribe({
          next: (bank) => {
            this.allExercises.set(bank);
            this.isLoading.set(false);
          },
          error: () => this.isLoading.set(false)
        });
      },
      error: () => this.isLoading.set(false)
    });
  }

  // Los casos de prueba I/O pertenecen al juez virtual; los laboratorios IDE
  // se corrigen con rúbrica en SpeedGrader y no ofrecen fuzzing.
  canFuzz(lab: EnrichedLabItem): boolean {
    return (lab.environment_type || 'JUEZ_EFIMERO').toUpperCase() === 'JUEZ_EFIMERO';
  }



  openCreateModal(preselectedCourseId?: string): void {
    if (!this.hasCourses()) {
      return;
    }
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
