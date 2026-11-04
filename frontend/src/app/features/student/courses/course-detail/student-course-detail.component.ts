import { Component, OnInit, computed, inject, input, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink, ActivatedRoute } from '@angular/router';
import { LucideArrowLeft, LucideRefreshCw } from '@lucide/angular';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { StudentService } from '@core/services/student.service';
import { RecommendationsPanelComponent } from './recommendations-panel/recommendations-panel.component';
import { CurricularMapComponent } from './curricular-map/curricular-map.component';

/**
 * Ficha de una materia para el alumno: su mapa curricular, las prácticas que
 * puede resolver y el refuerzo opcional. El encabezado resume el avance de la
 * materia; las prácticas y el mapa se cargan aparte, así que un fallo de esos
 * pedidos no borra la identidad de la materia.
 */
@Component({
  selector: 'student-course-detail',
  standalone: true,
  imports: [
    CommonModule,
    RouterLink,
    LucideArrowLeft,
    LucideRefreshCw,
    MachineDataDirective,
    RecommendationsPanelComponent,
    CurricularMapComponent
  ],
  templateUrl: './student-course-detail.component.html',
  styleUrl: './student-course-detail.component.scss'
})
export class StudentCourseDetailComponent implements OnInit {
  private studentService = inject(StudentService);
  private route = inject(ActivatedRoute);

  courseId = input<string>('');
  effectiveCourseId = signal<string>('');

  /** Cambia para que el mapa vuelva a leer; el encabezado es el único que manda. */
  refreshTick = signal(0);

  subjectItem = computed(() => {
    const id = this.effectiveCourseId();
    const subs = this.studentService.dashboardData()?.subjects || [];
    return subs.find((s) => s.subject.id === id);
  });

  subjectName = computed(() => this.subjectItem()?.subject.name || 'Detalle del Curso');
  subjectCode = computed(() => this.subjectItem()?.subject.code || '');

  totalExercises = computed(() => this.subjectItem()?.total_exercises ?? 0);
  completedExercises = computed(() => this.subjectItem()?.completed_exercises ?? 0);
  progressPercent = computed(() => this.subjectItem()?.progress_percent ?? 0);

  ngOnInit(): void {
    const fromInput = this.courseId();
    const fromRoute = this.route.snapshot.paramMap.get('courseId') || this.route.snapshot.paramMap.get('id') || '';
    const resolved = fromInput || fromRoute;
    this.effectiveCourseId.set(resolved);

    if (!this.studentService.dashboardData()) {
      this.studentService.loadDashboard();
    }
  }

  refresh(): void {
    this.refreshTick.update((tick) => tick + 1);
  }
}
