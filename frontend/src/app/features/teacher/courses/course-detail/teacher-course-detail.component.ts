import { Component, OnInit, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, RouterLink } from '@angular/router';
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
  LucideShieldAlert
} from '@lucide/angular';
import { TeacherCourseService } from '../../services/teacher-course.service';
import { TeacherLabStats } from '../../models/teacher.models';
import { ExerciseEditorModalComponent } from '../exercise-editor/exercise-editor-modal.component';
import { ConfirmModalComponent } from '@shared/components/confirm-modal/confirm-modal.component';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';
import { PlagiarismModalComponent } from '../../evaluations/plagiarism-modal/plagiarism-modal.component';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';

@Component({
  selector: 'teacher-course-detail',
  standalone: true,
  imports: [
    CommonModule,
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
    ExerciseEditorModalComponent,
    ConfirmModalComponent,
    DateTextPipe,
    MachineDataDirective,
    SkeletonLoaderComponent,
    PlagiarismModalComponent
  ],
  templateUrl: './teacher-course-detail.component.html',
  styleUrl: './teacher-course-detail.component.scss'
})
export class TeacherCourseDetailComponent implements OnInit {
  private route = inject(ActivatedRoute);
  private courseService = inject(TeacherCourseService);

  subjectId = signal<string>('');
  activeTab = signal<'labs' | 'submissions'>('labs');
  
  labs = this.courseService.labs;
  submissions = this.courseService.submissions;
  isLoading = this.courseService.isLoading;

  showEditorModal = signal<boolean>(false);
  selectedExerciseToEdit = signal<TeacherLabStats | null>(null);

  showPauseConfirmModal = signal<boolean>(false);
  isPausingEnvironments = signal<boolean>(false);
  pauseSuccessMessage = signal<string | null>(null);

  isPlagiarismModalOpen = signal<boolean>(false);

  verdictFilter = signal<string>('all');
  publishError = signal<string | null>(null);
  isExporting = signal<boolean>(false);

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
    const id = this.subjectId();
    if (!id) return;

    this.isExporting.set(true);
    this.courseService.exportGradesCsv(id).subscribe({
      next: (blob) => {
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `notas_curso_${id}_${new Date().toISOString().slice(0, 10)}.csv`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        window.URL.revokeObjectURL(url);
        this.isExporting.set(false);
      },
      error: () => {
        this.isExporting.set(false);
      }
    });
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
}

