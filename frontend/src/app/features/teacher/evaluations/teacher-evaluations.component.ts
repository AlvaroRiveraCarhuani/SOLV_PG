import { Component, OnInit, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { 
  LucideDownload, 
  LucideShieldAlert, 
  LucideListChecks, 
  LucideAward, 
  LucideSearch, 
  LucideExternalLink,
  LucideCheckCircle
} from '@lucide/angular';
import { TeacherDashboardService } from '../services/teacher-dashboard.service';
import { TeacherCourseService } from '../services/teacher-course.service';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';
import { PlagiarismModalComponent } from './plagiarism-modal/plagiarism-modal.component';
import { TeacherLabStats, SubmissionQueueItem } from '../models/teacher.models';

export interface EvaluationRow {
  student_id: string;
  student_name: string;
  student_code: string;
  scores: Record<string, number | null>;
  average: number;
  status: 'aprobado' | 'riesgo' | 'reprobado';
}

export interface EnrichedQueueItem extends SubmissionQueueItem {
  course_id?: string;
  course_name?: string;
}

@Component({
  selector: 'teacher-evaluations',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    RouterLink,
    LucideDownload,
    LucideShieldAlert,
    LucideListChecks,
    LucideAward,
    LucideSearch,
    LucideExternalLink,
    LucideCheckCircle,
    MachineDataDirective,
    DateTextPipe,
    SkeletonLoaderComponent,
    PlagiarismModalComponent
  ],
  templateUrl: './teacher-evaluations.component.html',
  styleUrl: './teacher-evaluations.component.scss'
})
export class TeacherEvaluationsComponent implements OnInit {
  private dashboardService = inject(TeacherDashboardService);
  private courseService = inject(TeacherCourseService);

  courses = this.dashboardService.courses;
  activeTab = signal<'queue' | 'grades'>('queue');
  isLoading = signal<boolean>(false);
  isPlagiarismModalOpen = signal<boolean>(false);

  // Filtros Cola Global
  courseFilter = signal<string>('all');
  verdictFilter = signal<string>('all');
  statusFilter = signal<'all' | 'pending' | 'graded'>('all');
  searchTerm = signal<string>('');

  // Datos Cola
  queueList = signal<EnrichedQueueItem[]>([]);

  // Datos Planilla de Calificaciones (Actas)
  selectedCourseId = signal<string>('');
  selectedLabId = signal<string>('all');
  labsList = signal<TeacherLabStats[]>([]);
  evaluationsList = signal<EvaluationRow[]>([]);

  // Computeds Cola Global
  filteredQueue = computed(() => {
    let items = this.queueList();
    const course = this.courseFilter();
    const verdict = this.verdictFilter();
    const status = this.statusFilter();
    const query = this.searchTerm().trim().toLowerCase();

    if (course !== 'all') {
      items = items.filter(i => i.course_id === course);
    }
    if (verdict !== 'all') {
      items = items.filter(i => i.verdict === verdict);
    }
    if (status === 'pending') {
      items = items.filter(i => i.score === undefined || i.score === null);
    } else if (status === 'graded') {
      items = items.filter(i => i.score !== undefined && i.score !== null);
    }
    if (query) {
      items = items.filter(i => 
        i.student_name.toLowerCase().includes(query) ||
        i.exercise_title.toLowerCase().includes(query) ||
        (i.course_name && i.course_name.toLowerCase().includes(query))
      );
    }
    return items;
  });

  kpiStats = computed(() => {
    const queue = this.queueList();
    const total = queue.length;
    const pending = queue.filter(q => q.score === undefined || q.score === null).length;
    const graded = total - pending;
    const atRisk = queue.filter(q => q.verdict === 'WA' || q.verdict === 'TLE' || q.verdict === 'AST_BLOCKED').length;
    return { total, pending, graded, atRisk };
  });

  // Computeds Planilla de Calificaciones
  filteredLabs = computed(() => {
    const selected = this.selectedLabId();
    const all = this.labsList();
    if (selected === 'all') return all;
    return all.filter(l => l.id === selected);
  });

  labStats = computed(() => {
    const selected = this.selectedLabId();
    const rows = this.evaluationsList();
    if (rows.length === 0) {
      return { average: 0, passRate: 0, totalStudents: 0, pendingCount: 0 };
    }

    if (selected === 'all') {
      const avg = Math.round(rows.reduce((acc, r) => acc + r.average, 0) / rows.length);
      const passed = rows.filter(r => r.status === 'aprobado').length;
      const passRate = Math.round((passed / rows.length) * 100);
      return { average: avg, passRate, totalStudents: rows.length, pendingCount: rows.filter(r => r.status === 'riesgo' || r.status === 'reprobado').length };
    }

    const scores = rows.map(r => r.scores[selected]).filter((s): s is number => s !== undefined && s !== null);
    const avg = scores.length > 0 ? Math.round(scores.reduce((a, b) => a + b, 0) / scores.length) : 0;
    const passed = scores.filter(s => s >= 70).length;
    const passRate = rows.length > 0 ? Math.round((passed / rows.length) * 100) : 0;
    const pendingCount = rows.length - scores.length;
    return { average: avg, passRate, totalStudents: rows.length, pendingCount };
  });

  ngOnInit(): void {
    this.loadAllData();
  }

  loadAllData(): void {
    this.isLoading.set(true);

    this.dashboardService.loadDashboardData().subscribe({
      next: (data) => {
        if (data.courses.length > 0) {
          const firstId = data.courses[0].id;
          this.selectedCourseId.set(firstId);
          this.loadCourseEvaluations(firstId);
        }

        // Cargar entregas de todos los cursos para la cola global
        this.loadGlobalSubmissions(data.courses);
      },
      error: () => this.isLoading.set(false)
    });
  }

  loadGlobalSubmissions(courses: { id: string; name: string }[]): void {
    if (courses.length === 0) {
      this.isLoading.set(false);
      return;
    }

    let loadedCount = 0;
    const allItems: EnrichedQueueItem[] = [];

    for (const course of courses) {
      this.courseService.getCourseSubmissions(course.id).subscribe({
        next: (subs) => {
          const mapped = subs.map(s => ({
            ...s,
            course_id: course.id,
            course_name: course.name
          }));
          allItems.push(...mapped);
          loadedCount++;
          if (loadedCount === courses.length) {
            this.queueList.set(allItems);
            this.isLoading.set(false);
          }
        },
        error: () => {
          loadedCount++;
          if (loadedCount === courses.length) {
            this.queueList.set(allItems);
            this.isLoading.set(false);
          }
        }
      });
    }
  }

  onCourseChange(courseId: string): void {
    this.selectedCourseId.set(courseId);
    this.loadCourseEvaluations(courseId);
  }

  loadCourseEvaluations(courseId: string): void {
    if (!courseId) return;

    this.courseService.getCourseLabs(courseId).subscribe({
      next: (labs) => {
        this.labsList.set(labs);
        
        this.courseService.getCourseSubmissions(courseId).subscribe({
          next: (submissions) => {
            const studentMap = new Map<string, EvaluationRow>();
            
            for (const sub of submissions) {
              if (!studentMap.has(sub.student_id)) {
                studentMap.set(sub.student_id, {
                  student_id: sub.student_id,
                  student_name: sub.student_name,
                  student_code: sub.student_email ? sub.student_email.split('@')[0] : 'N/A',
                  scores: {},
                  average: 0,
                  status: 'aprobado'
                });
              }
              const row = studentMap.get(sub.student_id)!;
              const numericScore = sub.score ?? (sub.verdict === 'AC' ? 100 : 40);
              row.scores[sub.exercise_id] = numericScore;
            }

            const rows: EvaluationRow[] = Array.from(studentMap.values()).map(r => {
              const scoresArr = Object.values(r.scores).filter((s): s is number => s !== null);
              const total = scoresArr.reduce((acc, s) => acc + s, 0);
              const avg = scoresArr.length > 0 ? Math.round(total / scoresArr.length) : 0;
              r.average = avg;
              if (avg >= 70) {
                r.status = 'aprobado';
              } else if (avg >= 51) {
                r.status = 'riesgo';
              } else {
                r.status = 'reprobado';
              }
              return r;
            });

            this.evaluationsList.set(rows);
          }
        });
      }
    });
  }

  openPlagiarismModal(): void {
    this.isPlagiarismModalOpen.set(true);
  }

  closePlagiarismModal(): void {
    this.isPlagiarismModalOpen.set(false);
  }

  exportGradesCSV(): void {
    const courseId = this.selectedCourseId();
    if (!courseId) return;

    this.courseService.exportGradesCsv(courseId).subscribe({
      next: (blob) => {
        const link = document.createElement('a');
        const url = URL.createObjectURL(blob);
        link.setAttribute('href', url);
        link.setAttribute('download', `acta_calificaciones_${courseId}.csv`);
        link.click();
      },
      error: () => {
        const rows = this.evaluationsList();
        const headers = ['Estudiante', 'Código', 'Promedio', 'Estado'];
        const csvContent = [
          headers.join(','),
          ...rows.map(r => [`"${r.student_name}"`, r.student_code, r.average, r.status].join(','))
        ].join('\n');
        const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
        const link = document.createElement('a');
        link.href = URL.createObjectURL(blob);
        link.download = `acta_calificaciones_${courseId}.csv`;
        link.click();
      }
    });
  }
}
