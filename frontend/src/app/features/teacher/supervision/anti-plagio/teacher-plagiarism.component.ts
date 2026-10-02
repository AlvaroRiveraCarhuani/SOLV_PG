import { Component, OnInit, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import {
  LucideRefreshCw,
  LucideExternalLink,
  LucideCheckCircle2,
  LucideCode,
  LucideGitCompare,
  LucideInfo
} from '@lucide/angular';
import { TeacherDashboardService } from '../../services/teacher-dashboard.service';
import { TeacherCourseService } from '../../services/teacher-course.service';
import { PlagiarismReport, PlagiarismMatch, TeacherLabStats, TeacherCourseSummary } from '../../models/teacher.models';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';

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
    SkeletonLoaderComponent
  ],
  templateUrl: './teacher-plagiarism.component.html',
  styleUrl: './teacher-plagiarism.component.scss'
})
export class TeacherPlagiarismComponent implements OnInit {
  private dashboardService = inject(TeacherDashboardService);
  private courseService = inject(TeacherCourseService);

  courses = this.dashboardService.courses;
  selectedCourseId = signal<string>('');
  selectedExerciseId = signal<string>('all');
  labsList = signal<TeacherLabStats[]>([]);

  report = signal<PlagiarismReport | null>(null);
  selectedMatch = signal<PlagiarismMatch | null>(null);
  isLoading = signal<boolean>(false);
  isScanning = signal<boolean>(false);

  // Filtros
  riskFilter = signal<'all' | 'critical' | 'warning' | 'info'>('all');
  searchTerm = signal<string>('');

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
    this.dashboardService.loadDashboardData().subscribe({
      next: data => {
        const coursesList = data.courses || [];
        if (coursesList.length > 0) {
          const firstCourse = coursesList[0];
          this.selectedCourseId.set(firstCourse.id);
          this.loadLabsAndScan(firstCourse.id);
        }
      }
    });
  }

  onCourseChange(courseId: string): void {
    this.selectedCourseId.set(courseId);
    this.selectedExerciseId.set('all');
    this.selectedMatch.set(null);
    this.loadLabsAndScan(courseId);
  }

  onExerciseChange(exerciseId: string): void {
    this.selectedExerciseId.set(exerciseId);
    this.selectedMatch.set(null);
    this.runScan();
  }

  loadLabsAndScan(courseId: string): void {
    if (!courseId) return;
    this.courseService.getCourseLabs(courseId).subscribe({
      next: labs => {
        this.labsList.set(labs);
        this.runScan();
      },
      error: () => this.runScan()
    });
  }

  runScan(): void {
    const courseId = this.selectedCourseId();
    if (!courseId) return;

    this.isLoading.set(true);
    this.isScanning.set(true);
    const exerciseId = this.selectedExerciseId() === 'all' ? undefined : this.selectedExerciseId();

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
