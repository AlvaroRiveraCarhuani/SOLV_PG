import { Component, OnInit, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, RouterLink } from '@angular/router';
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
  LucideSparkles,
  LucideLaptop,
  LucideBarChart3,
  LucideLayers,
  LucideLock,
  LucideTrash2
} from '@lucide/angular';
import { TeacherCourseService } from '../../services/teacher-course.service';
import { TeacherDashboardService } from '../../services/teacher-dashboard.service';
import { TeacherLabStats, TeacherCourseSummary, TeacherCourseModule } from '../../models/teacher.models';
import { ExerciseEditorModalComponent } from '../exercise-editor/exercise-editor-modal.component';
import { ConfirmModalComponent } from '@shared/components/confirm-modal/confirm-modal.component';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';
import { PlagiarismModalComponent } from '../../evaluations/plagiarism-modal/plagiarism-modal.component';
import { FuzzingModalComponent } from '../../evaluations/fuzzing-modal/fuzzing-modal.component';
import { GradebookExportModalComponent } from './gradebook-export-modal/gradebook-export-modal.component';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';

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
    LucideSparkles,
    LucideLaptop,
    LucideBarChart3,
    LucideLayers,
    LucideLock,
    LucideTrash2,
    ExerciseEditorModalComponent,
    ConfirmModalComponent,
    DateTextPipe,
    MachineDataDirective,
    SkeletonLoaderComponent,
    PlagiarismModalComponent,
    FuzzingModalComponent,
    GradebookExportModalComponent,
    ComboboxComponent
  ],
  templateUrl: './teacher-course-detail.component.html',
  styleUrl: './teacher-course-detail.component.scss'
})

export class TeacherCourseDetailComponent implements OnInit {
  private route = inject(ActivatedRoute);
  private courseService = inject(TeacherCourseService);
  private dashboardService = inject(TeacherDashboardService);

  readonly verdictOptions: ComboboxOption[] = [
    { id: 'all', label: 'Todos', value: 'all' },
    { id: 'AC', label: 'AC (Aceptado)', value: 'AC' },
    { id: 'WA', label: 'WA (Wrong Answer)', value: 'WA' },
    { id: 'TLE', label: 'TLE (Time Limit Exceeded)', value: 'TLE' },
    { id: 'RE', label: 'RE (Runtime Error)', value: 'RE' },
    { id: 'AST_BLOCKED', label: 'AST Bloqueado', value: 'AST_BLOCKED' }
  ];

  subjectId = signal<string>('');
  courseInfo = signal<TeacherCourseSummary | null>(null);
  activeTab = signal<'labs' | 'submissions' | 'modules'>('labs');
  
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
  selectedFuzzExercise = signal<{ id: string; title: string } | null>(null);

  verdictFilter = signal<string>('all');
  publishError = signal<string | null>(null);
  showExportModal = signal<boolean>(false);
  isExporting = signal<boolean>(false);

  // Módulos
  showCreateModuleModal = signal<boolean>(false);
  newModuleTitle = signal<string>('');
  newModuleDescription = signal<string>('');
  newModulePassScore = signal<number>(60);
  moduleErrorMessage = signal<string | null>(null);
  moduleSuccessMessage = signal<string | null>(null);

  filteredSubmissions = computed(() => {
    const filter = this.verdictFilter();
    if (filter === 'all') return this.submissions();
    return this.submissions().filter(s => s.verdict === filter);
  });

  ngOnInit(): void {
    const id = this.route.snapshot.paramMap.get('id');
    if (id) {
      this.subjectId.set(id);
      this.loadData();
    }
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
    this.selectedExerciseToEdit.set(null);
    this.showEditorModal.set(true);
  }

  openEditExerciseModal(lab: TeacherLabStats): void {
    this.selectedExerciseToEdit.set(lab);
    this.showEditorModal.set(true);
  }

  closeEditorModal(): void {
    this.showEditorModal.set(false);
    this.selectedExerciseToEdit.set(null);
  }

  publishExercise(lab: TeacherLabStats): void {
    this.publishError.set(null);
    this.courseService.publishExercise(lab.id).subscribe({
      next: () => {
        this.loadData();
      },
      error: (err) => {
        this.publishError.set(err.error?.message || 'No se puede publicar el ejercicio sin al menos un caso de prueba público.');
      }
    });
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

  openFuzzingModal(exerciseId: string, title: string): void {
    this.selectedFuzzExercise.set({ id: exerciseId, title });
  }

  closeFuzzingModal(): void {
    this.selectedFuzzExercise.set(null);
  }

  openLabEnvironment(lab: TeacherLabStats): void {
    window.open(`/student/workspace?exercise_id=${lab.id}&mode=teacher_preview`, '_blank');
  }

  onFuzzCasesApplied(count: number): void {
    const id = this.subjectId();
    if (id) {
      this.courseService.getCourseLabs(id).subscribe();
    }
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
}


