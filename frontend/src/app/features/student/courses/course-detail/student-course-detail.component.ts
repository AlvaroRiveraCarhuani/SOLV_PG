import { Component, OnInit, computed, inject, input, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink, ActivatedRoute } from '@angular/router';
import {
  LucideArrowLeft,
  LucideCode
} from '@lucide/angular';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { StudentService, StudentSubjectItem } from '@core/services/student.service';
import { RecommendationsPanelComponent } from './recommendations-panel/recommendations-panel.component';
import { CurricularMapComponent } from './curricular-map/curricular-map.component';

@Component({
  selector: 'student-course-detail',
  standalone: true,
  imports: [
    CommonModule,
    RouterLink,
    LucideArrowLeft,
    LucideCode,
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

  subjectItem = computed(() => {
    const id = this.effectiveCourseId();
    const subs = this.studentService.dashboardData()?.subjects || [];
    return subs.find((s) => s.subject.id === id);
  });

  subjectName = computed(() => this.subjectItem()?.subject.name || 'Detalle del Curso');
  subjectCode = computed(() => this.subjectItem()?.subject.code || '');

  ngOnInit(): void {
    const fromInput = this.courseId();
    const fromRoute = this.route.snapshot.paramMap.get('courseId') || this.route.snapshot.paramMap.get('id') || '';
    const resolved = fromInput || fromRoute;
    this.effectiveCourseId.set(resolved);

    if (!this.studentService.dashboardData()) {
      this.studentService.loadDashboard();
    }
  }
}
