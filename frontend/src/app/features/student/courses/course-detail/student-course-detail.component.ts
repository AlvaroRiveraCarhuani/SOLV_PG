import { Component, OnInit, computed, inject, input, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router, RouterLink, ActivatedRoute } from '@angular/router';
import { LucideArrowLeft } from '@lucide/angular';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { StudentService } from '@core/services/student.service';
import { StudentAulaService } from '../../services/student-aula.service';
import { CurricularMapComponent } from './curricular-map/curricular-map.component';

@Component({
  selector: 'student-course-detail',
  standalone: true,
  imports: [
    CommonModule,
    RouterLink,
    LucideArrowLeft,
    MachineDataDirective,
    CurricularMapComponent
  ],
  templateUrl: './student-course-detail.component.html',
  styleUrl: './student-course-detail.component.scss'
})
export class StudentCourseDetailComponent implements OnInit {
  private readonly studentService = inject(StudentService);
  private readonly aulasService = inject(StudentAulaService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);

  courseId = input<string>('');
  effectiveCourseId = signal<string>('');

  readonly subjects = computed(() => this.studentService.dashboardData()?.subjects || []);

  readonly subjectItem = computed(() => {
    const id = this.effectiveCourseId();
    return this.subjects().find((s) => s.subject.id === id);
  });

  readonly subjectName = computed(() => this.subjectItem()?.subject.name || 'Detalle del Curso');
  readonly subjectCode = computed(() => this.subjectItem()?.subject.code || '');

  readonly totalExercises = computed(() => this.subjectItem()?.total_exercises ?? 0);
  readonly completedExercises = computed(() => this.subjectItem()?.completed_exercises ?? 0);
  readonly pendingReviewExercises = computed(() => this.subjectItem()?.pending_review_exercises ?? 0);
  readonly progressPercent = computed(() => this.subjectItem()?.progress_percent ?? 0);

  readonly hasLiveAula = computed(() => {
    const id = this.effectiveCourseId();
    return this.aulasService.aulas().some(aula => aula.subject_id === id && aula.status === 'open');
  });

  /** Próximo paso en esta materia */
  readonly nextStep = computed(() => {
    const id = this.effectiveCourseId();
    const dues = this.studentService.dueAssignments().filter(d => d.subject_id === id);
    if (dues.length > 0) {
      const firstDue = dues[0];
      const isExam = firstDue.type === 'exam' || firstDue.title.toLowerCase().includes('examen');
      return {
        title: firstDue.title,
        exerciseId: firstDue.exercise_id,
        isExam,
        label: isExam ? 'Ir al examen →' : 'Continuar práctica →'
      };
    }
    return null;
  });

  readonly inProgressExercises = computed(() => {
    // Estimación para desglose: total menos completadas si hay avance parcial
    const completed = this.completedExercises();
    const total = this.totalExercises();
    if (completed === total) return 0;
    return Math.min(1, total - completed);
  });

  readonly notStartedExercises = computed(() => {
    const total = this.totalExercises();
    const completed = this.completedExercises();
    const inProg = this.inProgressExercises();
    return Math.max(0, total - completed - inProg);
  });

  onSwitchSubject(event: Event): void {
    const target = event.target as HTMLSelectElement;
    if (target?.value && target.value !== this.effectiveCourseId()) {
      this.effectiveCourseId.set(target.value);
      void this.router.navigate(['/student/courses', target.value]);
    }
  }

  ngOnInit(): void {
    void this.aulasService.load();
    const snapshotId = this.route.snapshot?.paramMap?.get('courseId') || this.route.snapshot?.paramMap?.get('id') || '';
    if (this.courseId() || snapshotId) {
      this.effectiveCourseId.set(this.courseId() || snapshotId);
    }
    if (this.route.paramMap) {
      this.route.paramMap.subscribe(params => {
        const fromRoute = params.get('courseId') || params.get('id') || '';
        const resolved = this.courseId() || fromRoute;
        if (resolved) this.effectiveCourseId.set(resolved);
      });
    }

    if (!this.studentService.dashboardData()) {
      this.studentService.loadDashboard();
    }
  }
}
