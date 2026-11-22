import { Component, OnInit, OnDestroy, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import {
  LucidePlus,
  LucideSearch,
  LucideSparkles,
  LucideEdit,
  LucideExternalLink,
  LucideLayers,
  LucideBookOpen
} from '@lucide/angular';
import { TeacherDashboardService } from '../services/teacher-dashboard.service';
import { TeacherCourseService } from '../services/teacher-course.service';
import { ContextHelpService } from '@shared/help/context-help.service';
import { ProductTourService } from '@shared/help/product-tour.service';
import { InteractiveTutorialService } from '@shared/help/interactive-tutorial.service';
import {
  buildTeacherExercisesTour,
  buildTeacherExercisesTutorial,
  TeacherExercisesTutorialContext
} from './teacher-exercises-help.definitions';
import { TeacherExerciseBankItem, TeacherLabStats } from '../models/teacher.models';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';
import { EnvironmentLabelPipe } from '@shared/pipes/environment-label.pipe';
import { environmentTypeOptions } from '@shared/constants/environment-types.constants';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { FuzzingModalComponent } from '../evaluations/fuzzing-modal/fuzzing-modal.component';
import { TechLogoComponent, TechIconKey } from '@features/admin/templates/components/tech-logo/tech-logo.component';
import { inferTemplateTechnologies } from '@features/admin/templates/template-tech-detector.utils';

export type EnrichedLabItem = TeacherExerciseBankItem;

export interface LabTechChip {
  name: string;
  iconKey: TechIconKey;
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
    LucideSparkles,
    LucideEdit,
    LucideExternalLink,
    LucideLayers,
    LucideBookOpen,
    MachineDataDirective,
    SkeletonLoaderComponent,
    ComboboxComponent,
    FuzzingModalComponent,
    EnvironmentLabelPipe,
    TechLogoComponent
  ],
  templateUrl: './teacher-exercises.component.html',
  styleUrl: './teacher-exercises.component.scss'
})
export class TeacherExercisesComponent implements OnInit, OnDestroy {
  private readonly dashboardService = inject(TeacherDashboardService);
  private readonly courseService = inject(TeacherCourseService);
  private readonly router = inject(Router);
  private readonly help = inject(ContextHelpService);
  private readonly tour = inject(ProductTourService);
  private readonly tutorial = inject(InteractiveTutorialService);

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

  // Duplicado a otra materia
  duplicatingLab = signal<EnrichedLabItem | null>(null);
  duplicateTargetId = signal<string>('');
  isDuplicating = signal<boolean>(false);
  duplicateError = signal<string>('');
  readonly isDuplicateCompleted = signal<boolean>(false);

  readonly tutorialContext = computed<TeacherExercisesTutorialContext>(() => ({
    hasCourses: () => this.hasCourses(),
    coursesCount: () => this.courses().length,
    totalExercisesCount: () => this.allExercises().length,
    isDuplicateModalOpen: () => this.duplicatingLab() !== null,
    hasSelectedTargetCourse: () => this.duplicateTargetId().trim().length > 0,
    isDuplicationFinished: () => this.isDuplicateCompleted()
  }));

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

    return { total, published, drafts };
  });

  // Materias a las que se puede copiar el laboratorio en curso (todas menos la de origen).
  duplicateTargetOptions = computed<ComboboxOption[]>(() => {
    const lab = this.duplicatingLab();
    return this.courses()
      .filter(c => c.id !== lab?.subject_id)
      .map(c => ({ id: c.id, label: `${c.code} — ${c.name}`, value: c.id }));
  });

  duplicateTargetLabel = computed(() => {
    const match = this.courses().find(c => c.id === this.duplicateTargetId());
    return match ? `${match.code} — ${match.name}` : 'Selecciona una materia...';
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
    ...environmentTypeOptions({ include: ['JUEZ_EFIMERO', 'IDE_PERSISTENTE'] })
  ]);

  selectedModalityLabel = computed(() => {
    const sel = this.modalityFilter();
    return this.modalityComboboxOptions().find(o => o.value === sel)?.label ?? 'Todos';
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
    this.help.registerScreen('teacher-exercises');
    this.help.registerGuides('teacher-exercises', {
      tour: {
        label: 'Recorrido por el Catálogo de Laboratorios',
        hint: 'Explicación de modalidades, filtros, casos límite y duplicación',
        run: () => this.tour.start(buildTeacherExercisesTour())
      },
      tutorial: {
        label: 'Cómo reutilizar un laboratorio en otra materia',
        hint: 'Guía paso a paso para duplicar una práctica limpia a otra materia',
        run: () => this.tutorial.start(buildTeacherExercisesTutorial(this.tutorialContext()))
      }
    });
  }

  ngOnDestroy(): void {
    this.help.unregisterGuides('teacher-exercises');
    this.help.unregisterScreen('teacher-exercises');
    this.tutorial.exit();
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

  techChipsFor(lab: EnrichedLabItem): LabTechChip[] {
    const inferred = inferTemplateTechnologies({
      name: lab.title,
      description: lab.language || ''
    });
    if (inferred.length > 0) {
      return inferred.map(t => ({ name: t.name, iconKey: t.iconType }));
    }
    if (lab.language) {
      return [{ name: lab.language, iconKey: 'docker' as TechIconKey }];
    }
    return [];
  }



  openCreateModal(preselectedCourseId?: string): void {
    if (!this.hasCourses()) {
      return;
    }
    const coursesList = this.courses();
    const course = coursesList.find(c => c.id === preselectedCourseId) || coursesList[0];
    const courseId = course ? course.id : 'all';
    this.router.navigate(['/teacher/cursos', courseId, 'ejercicios', 'nuevo']);
  }

  openEditModal(lab: EnrichedLabItem): void {
    this.router.navigate(['/teacher/cursos', lab.subject_id, 'ejercicios', lab.id, 'editar']);
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

  openDuplicateModal(lab: EnrichedLabItem): void {
    this.duplicatingLab.set(lab);
    this.duplicateTargetId.set('');
    this.duplicateError.set('');
    this.isDuplicateCompleted.set(false);
  }

  closeDuplicateModal(): void {
    if (this.isDuplicating()) {
      return;
    }
    this.duplicatingLab.set(null);
    this.duplicateError.set('');
  }

  onDuplicateTargetSelected(opt: ComboboxOption): void {
    this.duplicateTargetId.set(opt.value);
  }

  confirmDuplicate(): void {
    const lab = this.duplicatingLab();
    const targetId = this.duplicateTargetId();
    if (!lab || !targetId || this.isDuplicating()) {
      return;
    }
    this.isDuplicating.set(true);
    this.duplicateError.set('');
    this.courseService.duplicateExercise(lab.id, targetId).subscribe({
      next: () => {
        this.isDuplicating.set(false);
        this.duplicatingLab.set(null);
        this.isDuplicateCompleted.set(true);
        this.loadData();
      },
      error: () => {
        this.isDuplicating.set(false);
        this.duplicateError.set('No se pudo duplicar el laboratorio. Intente de nuevo.');
      }
    });
  }
}
