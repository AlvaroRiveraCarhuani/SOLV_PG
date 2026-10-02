import { ComponentFixture, TestBed } from '@angular/core/testing';
import { TeacherPlagiarismComponent } from './teacher-plagiarism.component';
import { TeacherDashboardService } from '../../services/teacher-dashboard.service';
import { TeacherCourseService } from '../../services/teacher-course.service';
import { RouterTestingModule } from '@angular/router/testing';
import { of } from 'rxjs';
import { signal } from '@angular/core';
import { TeacherCourseSummary, PlagiarismReport, TeacherLabStats } from '../../models/teacher.models';

describe('TeacherPlagiarismComponent', () => {
  let component: TeacherPlagiarismComponent;
  let fixture: ComponentFixture<TeacherPlagiarismComponent>;
  let mockDashboardService: Partial<TeacherDashboardService>;
  let mockCourseService: Partial<TeacherCourseService>;

  const mockCourses: TeacherCourseSummary[] = [
    {
      id: 'course-1',
      code: 'CS101',
      name: 'Estructuras de Datos',
      students_count: 30,
      active_now: 4,
      pending_review: 2,
      at_risk: 1
    }
  ];

  const mockLabs: TeacherLabStats[] = [
    {
      id: 'lab-1',
      title: 'Árboles Binarios',
      status: 'published',
      submissions_count: 20,
      students_count: 30,
      auto_graded: 15,
      pending_review: 5,
      at_risk: 2
    }
  ];

  const mockReport: PlagiarismReport = {
    subject_id: 'course-1',
    subject_name: 'CS101 — Estructuras de Datos',
    exercise_id: 'lab-1',
    exercise_title: 'Árboles Binarios',
    analyzed_at: '2026-03-01T12:00:00Z',
    total_submissions: 20,
    suspect_pairs_count: 2,
    matches: [
      {
        student_id_a: 'std-1',
        student_name_a: 'Estudiante Alfa',
        submission_id_a: 'sub-1',
        student_id_b: 'std-2',
        student_name_b: 'Estudiante Beta',
        submission_id_b: 'sub-2',
        exercise_id: 'lab-1',
        exercise_title: 'Árboles Binarios',
        similarity: 92,
        risk_level: 'critical',
        matching_tokens: 140,
        total_tokens_a: 150,
        total_tokens_b: 155,
        common_structures: ['BinarySearchTree.insert', 'BinarySearchTree.balance']
      },
      {
        student_id_a: 'std-3',
        student_name_a: 'Estudiante Gamma',
        submission_id_a: 'sub-3',
        student_id_b: 'std-4',
        student_name_b: 'Estudiante Delta',
        submission_id_b: 'sub-4',
        exercise_id: 'lab-1',
        exercise_title: 'Árboles Binarios',
        similarity: 65,
        risk_level: 'warning',
        matching_tokens: 80,
        total_tokens_a: 120,
        total_tokens_b: 130,
        common_structures: ['BinarySearchTree.inorder']
      }
    ]
  };

  beforeEach(async () => {
    mockDashboardService = {
      courses: signal<TeacherCourseSummary[]>(mockCourses),
      loadDashboardData: vi.fn().mockReturnValue(of({ courses: mockCourses, attention: { critical: [], warning: [], standard: [] } }))
    };

    mockCourseService = {
      getCourseLabs: vi.fn().mockReturnValue(of(mockLabs)),
      analyzePlagiarism: vi.fn().mockReturnValue(of(mockReport))
    };

    await TestBed.configureTestingModule({
      imports: [TeacherPlagiarismComponent, RouterTestingModule],
      providers: [
        { provide: TeacherDashboardService, useValue: mockDashboardService },
        { provide: TeacherCourseService, useValue: mockCourseService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(TeacherPlagiarismComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should initialize and scan report on course selection', () => {
    expect(component.selectedCourseId()).toBe('course-1');
    expect(component.report()).toEqual(mockReport);
    expect(component.selectedMatch()).toEqual(mockReport.matches[0]);
  });

  it('should calculate KPI statistics accurately', () => {
    const stats = component.kpiStats();
    expect(stats.totalPairs).toBe(2);
    expect(stats.criticalCount).toBe(1);
    expect(stats.warningCount).toBe(1);
    expect(stats.uniqueStudents).toBe(4);
  });

  it('should filter matches by risk level', () => {
    component.riskFilter.set('critical');
    expect(component.filteredMatches().length).toBe(1);
    expect(component.filteredMatches()[0].student_name_a).toBe('Estudiante Alfa');

    component.riskFilter.set('warning');
    expect(component.filteredMatches().length).toBe(1);
    expect(component.filteredMatches()[0].student_name_a).toBe('Estudiante Gamma');
  });

  it('should filter matches by search query', () => {
    component.searchTerm.set('Delta');
    expect(component.filteredMatches().length).toBe(1);
    expect(component.filteredMatches()[0].student_name_b).toBe('Estudiante Delta');
  });

  it('should update selected match on click', () => {
    component.selectMatch(mockReport.matches[1]);
    expect(component.selectedMatch()).toEqual(mockReport.matches[1]);
  });
});
