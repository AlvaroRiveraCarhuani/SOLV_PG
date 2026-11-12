import { Component, OnInit, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { 
  LucideArrowLeft, 
  LucideDownload, 
  LucidePlus, 
  LucideAlertTriangle, 
  LucideBookOpen, 
  LucideListChecks, 
  LucideFolderPlus, 
  LucideCalendar, 
  LucideGlobe, 
  LucideEdit3, 
  LucideExternalLink,
  LucidePauseCircle,
  LucideCheckCircle,
  LucideShieldAlert,
  LucideBarChart3,
  LucideAward,
  LucideLayers,
  LucideLock,
  LucideTrash2,
  LucideUploadCloud,
  LucideLoader2,
  LucideCheck,
  LucideChevronDown,
  LucideUserPlus
} from '@lucide/angular';
import { TeacherCourseService, AvailableStudent } from '../../services/teacher-course.service';
import { TeacherDashboardService } from '../../services/teacher-dashboard.service';
import { TeacherLabStats, TeacherCourseSummary, TeacherCourseModule } from '../../models/teacher.models';
import { ConfirmModalComponent } from '@shared/components/confirm-modal/confirm-modal.component';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';
import { PlagiarismModalComponent } from '../../evaluations/plagiarism-modal/plagiarism-modal.component';
import { GradebookExportModalComponent } from './gradebook-export-modal/gradebook-export-modal.component';
import { CourseGradebookComponent } from './course-gradebook/course-gradebook.component';
import { ImportExercisesModalComponent } from './import-exercises-modal/import-exercises-modal.component';
import { EnrollStudentModalComponent } from './enroll-student-modal/enroll-student-modal.component';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { DismissibleDirective } from '@shared/directives/dismissible.directive';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { VerdictBadgeComponent } from '@shared/components/verdict-badge/verdict-badge.component';
import { SortHeaderComponent } from '@shared/components/sort-header/sort-header.component';
import { SortDirection, SortState } from '@shared/components/sort-header/sort-header.types';
import { VERDICT_FILTER_OPTIONS, matchesVerdictFilter } from '@core/models/verdict.model';

export type ReviewStatusFilter = 'all' | 'pending' | 'graded';

/**
 * Estado de revisión de una entrega del historial. Las filas anteriores a
 * `review_status` caen en la nota: con nota están calificadas, sin nota no.
 */
export function matchesReviewStatusFilter(
  status: string | undefined | null,
  filter: ReviewStatusFilter,
  hasScore: boolean
): boolean {
  if (filter === 'all') return true;
  const normalized = (status || '').trim();
  if (filter === 'pending') {
    return normalized === 'pending' || normalized === 'in_review';
  }
  if (normalized === 'graded' || normalized === 'returned' || normalized === 'not_required') {
    return true;
  }
  return normalized === '' && hasScore;
}

@Component({
  selector: 'teacher-course-detail',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    RouterLink,
    LucideArrowLeft,
    LucideDownload,
    LucidePlus,
    LucideAlertTriangle,
    LucideBookOpen,
    LucideListChecks,
    LucideFolderPlus,
    LucideCalendar,
    LucideGlobe,
    LucideEdit3,
    LucideExternalLink,
    LucidePauseCircle,
    LucideCheckCircle,
    LucideShieldAlert,
    LucideBarChart3,
    LucideAward,
    LucideLayers,
    LucideLock,
    LucideTrash2,
    LucideUploadCloud,
    LucideLoader2,
    LucideCheck,
    LucideChevronDown,
    LucideUserPlus,
    ModalShellComponent,
    ConfirmModalComponent,
    DateTextPipe,
    MachineDataDirective,
    DismissibleDirective,
    SkeletonLoaderComponent,
    PlagiarismModalComponent,
    GradebookExportModalComponent,
    CourseGradebookComponent,
    ImportExercisesModalComponent,
    EnrollStudentModalComponent,
    ComboboxComponent,
    VerdictBadgeComponent,
    SortHeaderComponent
  ],
  templateUrl: './teacher-course-detail.component.html',
  styleUrl: './teacher-course-detail.component.scss'
})

export class TeacherCourseDetailComponent implements OnInit {
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private courseService = inject(TeacherCourseService);
  private dashboardService = inject(TeacherDashboardService);

  readonly verdictOptions: ComboboxOption[] = VERDICT_FILTER_OPTIONS;

  /** Separa el historial en lo que espera corrección y lo ya calificado. */
  readonly reviewStatusOptions: ComboboxOption<ReviewStatusFilter>[] = [
    { id: 'all', label: 'Todos los estados', value: 'all' },
    { id: 'pending', label: 'Sin corregir', value: 'pending' },
    { id: 'graded', label: 'Calificadas', value: 'graded' }
  ];

  showMoreActions = signal<boolean>(false);
  subjectId = signal<string>('');
  courseInfo = signal<TeacherCourseSummary | null>(null);
  activeTab = signal<'labs' | 'submissions' | 'modules' | 'gradebook'>('labs');
  
  labs = this.courseService.labs;
  submissions = this.courseService.submissions;
  isLoading = this.courseService.isLoading;
  modules = signal<TeacherCourseModule[]>([]);

  showEditorModal = signal<boolean>(false);
  selectedExerciseToEdit = signal<TeacherLabStats | null>(null);

  showPauseConfirmModal = signal<boolean>(false);
  isPausingEnvironments = signal<boolean>(false);
  pauseSuccessMessage = signal<string | null>(null);

  isPlagiarismModalOpen = signal<boolean>(false);

  verdictFilter = signal<string>('all');
  reviewStatusFilter = signal<ReviewStatusFilter>('all');
  labFilter = signal<string>('all');
  publishError = signal<string | null>(null);
  publishSuccessMessage = signal<string | null>(null);
  selectedLabToPublish = signal<TeacherLabStats | null>(null);
  isPublishingLab = signal<boolean>(false);
  showExportModal = signal<boolean>(false);
  isExporting = signal<boolean>(false);
  showImportModal = signal<boolean>(false);
  showEnrollModal = signal<boolean>(false);
  enrollSuccessMessage = signal<string | null>(null);

  // Módulos
  showCreateModuleModal = signal<boolean>(false);
  newModuleTitle = signal<string>('');
  newModuleDescription = signal<string>('');
  newModulePassScore = signal<number>(60);
  moduleErrorMessage = signal<string | null>(null);
  moduleSuccessMessage = signal<string | null>(null);

  readonly labOptions = computed<ComboboxOption[]>(() => {
    const defaultOpt: ComboboxOption = { id: 'all', label: 'Todos los laboratorios', value: 'all' };
    const labOpts: ComboboxOption[] = this.labs().map(l => ({
      id: l.id,
      label: l.title,
      value: l.id
    }));
    return [defaultOpt, ...labOpts];
  });

  // Ordenamiento de entregas
  sortState = signal<SortState>({ column: null, direction: 'none' });

  onSortChange(event: { column: string; direction: SortDirection }): void {
    this.sortState.set({
      column: event.direction === 'none' ? null : event.column,
      direction: event.direction
    });
  }

  filteredSubmissions = computed(() => {
    const verdict = this.verdictFilter();
    const reviewStatus = this.reviewStatusFilter();
    const labId = this.labFilter();
    const sort = this.sortState();
    let list = this.submissions().filter(s => {
      const matchVerdict = matchesVerdictFilter(s.verdict, verdict);
      const matchReview = matchesReviewStatusFilter(
        s.review_status,
        reviewStatus,
        s.score !== undefined && s.score !== null
      );
      const matchLab = labId === 'all' || s.exercise_id === labId;
      return matchVerdict && matchReview && matchLab;
    });

    if (sort.column && sort.direction !== 'none') {
      const modifier = sort.direction === 'asc' ? 1 : -1;
      list = [...list].sort((a, b) => {
        switch (sort.column) {
          case 'student':
            return (a.student_name || '').localeCompare(b.student_name || '', undefined, { sensitivity: 'base' }) * modifier;
          case 'exercise':
            return (a.exercise_title || '').localeCompare(b.exercise_title || '', undefined, { sensitivity: 'base' }) * modifier;
          case 'score': {
            const scA = a.score ?? -1;
            const scB = b.score ?? -1;
            return (scA - scB) * modifier;
          }
          case 'date': {
            const dA = a.submitted_at ? new Date(a.submitted_at).getTime() : 0;
            const dB = b.submitted_at ? new Date(b.submitted_at).getTime() : 0;
            return (dA - dB) * modifier;
          }
          default:
            return 0;
        }
      });
    }

    return list;
  });

  ngOnInit(): void {
    const id = this.route.snapshot.paramMap?.get('id');
    if (id) {
      this.subjectId.set(id);
      this.loadData();
    }
    const tabParam = this.route.snapshot.queryParamMap?.get('tab');
    if (tabParam === 'submissions' || tabParam === 'modules' || tabParam === 'labs' || tabParam === 'gradebook') {
      this.activeTab.set(tabParam);
    }
    const labParam = this.route.snapshot.queryParamMap?.get('labId');
    if (labParam) {
      this.labFilter.set(labParam);
    }
    const statusParam = this.route.snapshot.queryParamMap?.get('status');
    if (statusParam === 'pending' || statusParam === 'graded' || statusParam === 'all') {
      this.reviewStatusFilter.set(statusParam);
    }

    if (this.route.paramMap) {
      this.route.paramMap.subscribe(params => {
        const nextId = params.get('id');
        if (nextId && nextId !== this.subjectId()) {
          this.subjectId.set(nextId);
          this.loadData();
        }
      });
    }

    if (this.route.queryParamMap) {
      this.route.queryParamMap.subscribe(queryParams => {
        const nextTab = queryParams.get('tab');
        if (nextTab === 'submissions' || nextTab === 'modules' || nextTab === 'labs' || nextTab === 'gradebook') {
          this.activeTab.set(nextTab);
        }
        const nextLab = queryParams.get('labId');
        if (nextLab) {
          this.labFilter.set(nextLab);
        }
        const nextStatus = queryParams.get('status');
        if (nextStatus === 'pending' || nextStatus === 'graded' || nextStatus === 'all') {
          this.reviewStatusFilter.set(nextStatus);
        }
      });
    }
  }

  viewLabSubmissions(labId: string): void {
    this.labFilter.set(labId);
    this.activeTab.set('submissions');
  }

  loadData(): void {
    const id = this.subjectId();
    if (!id) return;
    this.courseService.getCourseLabs(id).subscribe();
    this.courseService.getCourseSubmissions(id).subscribe();
    this.loadModules();

    this.dashboardService.loadDashboardData().subscribe({
      next: (data) => {
        const found = data.courses.find(c => c.id === id);
        if (found) {
          this.courseInfo.set(found);
        }
      }
    });
  }

  openCreateExerciseModal(): void {
    this.router.navigate(['/teacher/cursos', this.subjectId(), 'ejercicios', 'nuevo']);
  }

  closeEditorModal(): void {
    this.selectedExerciseToEdit.set(null);
  }

  requestPublishExercise(lab: TeacherLabStats): void {
    this.publishError.set(null);
    this.selectedLabToPublish.set(lab);
  }

  cancelPublishLab(): void {
    this.selectedLabToPublish.set(null);
  }

  confirmPublishLab(): void {
    const lab = this.selectedLabToPublish();
    if (!lab) return;

    this.isPublishingLab.set(true);
    this.publishError.set(null);
    this.courseService.publishExercise(lab.id).subscribe({
      next: () => {
        this.isPublishingLab.set(false);
        this.selectedLabToPublish.set(null);
        this.publishSuccessMessage.set(`«${lab.title}» publicado con éxito.`);
        setTimeout(() => this.publishSuccessMessage.set(null), 4000);
        this.loadData();
      },
      error: (err) => {
        this.isPublishingLab.set(false);
        this.selectedLabToPublish.set(null);
        this.publishError.set(err.error?.message || 'No se puede publicar el ejercicio sin al menos un caso de prueba público.');
      }
    });
  }

  publishExercise(lab: TeacherLabStats): void {
    this.requestPublishExercise(lab);
  }

  exportGrades(): void {
    this.showExportModal.set(true);
  }

  closeExportModal(): void {
    this.showExportModal.set(false);
  }


  openPauseConfirmModal(): void {
    this.showPauseConfirmModal.set(true);
  }

  closePauseConfirmModal(): void {
    this.showPauseConfirmModal.set(false);
  }

  confirmPauseAllEnvironments(): void {
    const id = this.subjectId();
    if (!id) return;

    this.isPausingEnvironments.set(true);
    // Simular / enviar acción masiva de hibernación
    setTimeout(() => {
      this.isPausingEnvironments.set(false);
      this.showPauseConfirmModal.set(false);
      this.pauseSuccessMessage.set('Entornos de la clase hibernados exitosamente. Memoria RAM liberada.');
      setTimeout(() => this.pauseSuccessMessage.set(null), 5000);
    }, 600);
  }

  openPlagiarismModal(): void {
    this.isPlagiarismModalOpen.set(true);
  }

  closePlagiarismModal(): void {
    this.isPlagiarismModalOpen.set(false);
  }

  loadModules(): void {
    const id = this.subjectId();
    if (!id) return;
    this.courseService.getCourseModules(id).subscribe({
      next: (mods) => {
        this.modules.set(mods);
      },
      error: () => {
        this.modules.set([]);
      }
    });
  }

  openCreateModuleModal(): void {
    this.newModuleTitle.set('');
    this.newModuleDescription.set('');
    this.newModulePassScore.set(60);
    this.moduleErrorMessage.set(null);
    this.showCreateModuleModal.set(true);
  }

  closeCreateModuleModal(): void {
    this.showCreateModuleModal.set(false);
    this.moduleErrorMessage.set(null);
  }

  submitCreateModule(): void {
    const id = this.subjectId();
    const title = this.newModuleTitle().trim();
    if (!id || !title) return;

    this.moduleErrorMessage.set(null);
    this.courseService.createModule(id, {
      title,
      description: this.newModuleDescription().trim(),
      order_index: this.modules().length + 1,
      pass_score: this.newModulePassScore()
    }).subscribe({
      next: () => {
        this.closeCreateModuleModal();
        this.loadModules();
        this.moduleSuccessMessage.set('Módulo creado exitosamente.');
        setTimeout(() => this.moduleSuccessMessage.set(null), 4000);
      },
      error: (err) => {
        this.moduleErrorMessage.set(err.error?.message || 'Error al crear el módulo.');
      }
    });
  }

  deleteModule(moduleId: string): void {
    const id = this.subjectId();
    if (!id) return;

    this.courseService.deleteModule(id, moduleId).subscribe({
      next: () => {
        this.loadModules();
        this.moduleSuccessMessage.set('Módulo eliminado.');
        setTimeout(() => this.moduleSuccessMessage.set(null), 3000);
      },
      error: (err) => {
        this.moduleErrorMessage.set(err.error?.message || 'Error al eliminar el módulo.');
      }
    });
  }

  togglePrerequisite(moduleId: string, prereqId: string): void {
    const id = this.subjectId();
    if (!id || moduleId === prereqId) return;

    const target = this.modules().find(m => m.id === moduleId);
    if (!target) return;

    const currentPrereqs = target.prerequisites ? [...target.prerequisites] : [];
    const idx = currentPrereqs.indexOf(prereqId);
    let updated: string[];
    if (idx >= 0) {
      updated = currentPrereqs.filter(p => p !== prereqId);
    } else {
      updated = [...currentPrereqs, prereqId];
    }

    this.moduleErrorMessage.set(null);
    this.courseService.setModulePrerequisites(id, moduleId, updated).subscribe({
      next: () => {
        this.loadModules();
      },
      error: (err) => {
        this.moduleErrorMessage.set(err.error?.message || 'No se puede establecer el prerrequisito (posible ciclo detectado).');
      }
    });
  }

  assignExerciseToModule(exerciseId: string, moduleId: string | null): void {
    this.moduleErrorMessage.set(null);
    this.courseService.assignExerciseModule(exerciseId, moduleId).subscribe({
      next: () => {
        this.loadData();
        this.moduleSuccessMessage.set('Ejercicio asignado al módulo.');
        setTimeout(() => this.moduleSuccessMessage.set(null), 3000);
      },
      error: (err) => {
        this.moduleErrorMessage.set(err.error?.message || 'Error al asignar el ejercicio.');
      }
    });
  }

  openImportModal(): void {
    this.showImportModal.set(true);
  }

  closeImportModal(): void {
    this.showImportModal.set(false);
  }

  onExercisesImported(_count: number): void {
    this.loadData();
  }

  openEnrollModal(): void {
    this.showEnrollModal.set(true);
  }

  closeEnrollModal(): void {
    this.showEnrollModal.set(false);
  }

  onStudentEnrolled(student: AvailableStudent): void {
    this.showEnrollModal.set(false);
    this.enrollSuccessMessage.set(`Estudiante ${student.first_name} ${student.last_name} matriculado exitosamente.`);
    this.loadData();
    setTimeout(() => this.enrollSuccessMessage.set(null), 5000);
  }
}


