import { Component, OnInit, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import {
  LucideRefreshCw,
  LucideExternalLink,
  LucideCheckCircle2,
  LucideCode,
  LucideGitCompare,
  LucideInfo
} from '@lucide/angular';
import { TeacherCourseService } from '../../services/teacher-course.service';
import { PlagiarismReport, PlagiarismMatch } from '../../models/teacher.models';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';

/**
 * Auditoría de similitud de una práctica. La pantalla no elige materia: el
 * contexto viene de la ficha de la práctica y el motor ya restó el código base
 * antes de comparar. Sin práctica no hay auditoría global: se vuelve a la cola
 * de revisión.
 */
@Component({
  selector: 'teacher-plagiarism',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    RouterLink,
    LucideRefreshCw,
    LucideExternalLink,
    LucideCheckCircle2,
    LucideCode,
    LucideGitCompare,
    LucideInfo,
    MachineDataDirective,
    SkeletonLoaderComponent,
    ComboboxComponent
  ],
  templateUrl: './teacher-plagiarism.component.html',
  styleUrl: './teacher-plagiarism.component.scss'
})
export class TeacherPlagiarismComponent implements OnInit {
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private courseService = inject(TeacherCourseService);

  readonly courseId = signal<string>('');
  readonly exerciseId = signal<string>('');

  report = signal<PlagiarismReport | null>(null);
  selectedMatch = signal<PlagiarismMatch | null>(null);
  isLoading = signal<boolean>(false);
  isScanning = signal<boolean>(false);

  // Filtros
  riskFilter = signal<'all' | 'critical' | 'warning' | 'info'>('all');
  searchTerm = signal<string>('');

  readonly riskOptions: ComboboxOption<'all' | 'critical' | 'warning' | 'info'>[] = [
    { id: 'all', label: 'Todos', value: 'all' },
    { id: 'critical', label: 'Crítico (>= 80%)', value: 'critical' },
    { id: 'warning', label: 'Advertencia (50-79%)', value: 'warning' },
    { id: 'info', label: 'Informativo (< 50%)', value: 'info' }
  ];

  readonly practiceTitle = computed(() => this.report()?.exercise_title ?? '');

  // Computeds
  filteredMatches = computed(() => {
    const rep = this.report();
    if (!rep || !rep.matches) return [];

    let list = rep.matches;
    const risk = this.riskFilter();
    const query = this.searchTerm().trim().toLowerCase();

    if (risk !== 'all') {
      list = list.filter(m => m.risk_level === risk);
    }

    if (query) {
      list = list.filter(m =>
        m.student_name_a.toLowerCase().includes(query) ||
        m.student_name_b.toLowerCase().includes(query) ||
        m.exercise_title.toLowerCase().includes(query)
      );
    }

    return list;
  });

  kpiStats = computed(() => {
    const rep = this.report();
    if (!rep || !rep.matches) {
      return { totalPairs: 0, criticalCount: 0, warningCount: 0, uniqueStudents: 0 };
    }

    const matches = rep.matches;
    const criticalCount = matches.filter(m => m.risk_level === 'critical').length;
    const warningCount = matches.filter(m => m.risk_level === 'warning').length;

    const studentIds = new Set<string>();
    matches.forEach(m => {
      if (m.student_id_a) studentIds.add(m.student_id_a);
      if (m.student_id_b) studentIds.add(m.student_id_b);
    });

    return {
      totalPairs: matches.length,
      criticalCount,
      warningCount,
      uniqueStudents: studentIds.size
    };
  });

  ngOnInit(): void {
    const courseId = this.route.snapshot.paramMap.get('courseId') ?? '';
    const exerciseId = this.route.snapshot.paramMap.get('exerciseId') ?? '';

    if (!courseId || !exerciseId) {
      void this.router.navigate(['/teacher/por-calificar'], { replaceUrl: true });
      return;
    }

    this.courseId.set(courseId);
    this.exerciseId.set(exerciseId);
    this.runScan();
  }

  runScan(): void {
    const courseId = this.courseId();
    const exerciseId = this.exerciseId();
    if (!courseId || !exerciseId) return;

    this.isLoading.set(true);
    this.isScanning.set(true);

    this.courseService.analyzePlagiarism(courseId, exerciseId).subscribe({
      next: report => {
        this.report.set(report);
        if (report.matches && report.matches.length > 0) {
          this.selectedMatch.set(report.matches[0]);
        } else {
          this.selectedMatch.set(null);
        }
        this.isLoading.set(false);
        this.isScanning.set(false);
      },
      error: () => {
        this.isLoading.set(false);
        this.isScanning.set(false);
      }
    });
  }

  selectMatch(match: PlagiarismMatch): void {
    this.selectedMatch.set(match);
  }
}
