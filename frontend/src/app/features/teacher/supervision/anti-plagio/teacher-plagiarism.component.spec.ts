import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter } from '@angular/router';
import { of } from 'rxjs';

import { TeacherPlagiarismComponent } from './teacher-plagiarism.component';
import { TeacherCourseService } from '../../services/teacher-course.service';
import { PlagiarismReport } from '../../models/teacher.models';

describe('TeacherPlagiarismComponent', () => {
  let component: TeacherPlagiarismComponent;
  let fixture: ComponentFixture<TeacherPlagiarismComponent>;

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
        ranges: [
          {
            source_a: { file_path: 'src/tree.py', start_line: 10, end_line: 24 },
            source_b: { file_path: 'src/tree.py', start_line: 12, end_line: 26 },
            token_count: 140,
            exact_match: true
          }
        ]
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
        ranges: [
          {
            source_a: { start_line: 3, end_line: 9 },
            source_b: { start_line: 4, end_line: 10 },
            token_count: 80,
            exact_match: true
          }
        ]
      }
    ]
  };

  beforeEach(async () => {
    const mockCourseService = {
      analyzePlagiarism: vi.fn().mockReturnValue(of(mockReport))
    };

    await TestBed.configureTestingModule({
      imports: [TeacherPlagiarismComponent],
      providers: [
        provideRouter([]),
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: {
              paramMap: convertToParamMap({ courseId: 'course-1', exerciseId: 'lab-1' })
            }
          }
        },
        { provide: TeacherCourseService, useValue: mockCourseService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(TeacherPlagiarismComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('escanea la práctica que llega por la ruta, sin elegir materia', () => {
    expect(component.courseId()).toBe('course-1');
    expect(component.exerciseId()).toBe('lab-1');
    expect(component.report()).toEqual(mockReport);
    expect(component.selectedMatch()).toEqual(mockReport.matches[0]);
  });

  it('calcula los indicadores del reporte', () => {
    const stats = component.kpiStats();
    expect(stats.totalPairs).toBe(2);
    expect(stats.criticalCount).toBe(1);
    expect(stats.warningCount).toBe(1);
    expect(stats.uniqueStudents).toBe(4);
  });

  it('filtra los pares por nivel de riesgo', () => {
    component.riskFilter.set('critical');
    expect(component.filteredMatches().length).toBe(1);
    expect(component.filteredMatches()[0].student_name_a).toBe('Estudiante Alfa');

    component.riskFilter.set('warning');
    expect(component.filteredMatches().length).toBe(1);
    expect(component.filteredMatches()[0].student_name_a).toBe('Estudiante Gamma');
  });

  it('filtra los pares por nombre', () => {
    component.searchTerm.set('Delta');
    expect(component.filteredMatches().length).toBe(1);
    expect(component.filteredMatches()[0].student_name_b).toBe('Estudiante Delta');
  });

  it('cambia el par seleccionado', () => {
    component.selectMatch(mockReport.matches[1]);
    expect(component.selectedMatch()).toEqual(mockReport.matches[1]);
  });

  it('muestra los tramos compartidos con archivo y líneas', () => {
    const ranges = component.selectedMatch()?.ranges ?? [];
    expect(ranges.length).toBe(1);
    expect(ranges[0].source_a.file_path).toBe('src/tree.py');
    expect(ranges[0].source_a.start_line).toBe(10);
  });
});
