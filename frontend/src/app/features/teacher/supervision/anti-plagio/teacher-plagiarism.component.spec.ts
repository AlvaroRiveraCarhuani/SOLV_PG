import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter } from '@angular/router';
import { of } from 'rxjs';

import { TeacherPlagiarismComponent } from './teacher-plagiarism.component';
import { TeacherCourseService } from '../../services/teacher-course.service';
import { PlagiarismPair, PlagiarismReport } from '../../models/teacher.models';

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

  const mockPair: PlagiarismPair = {
    exercise: {
      id: 'lab-1',
      title: 'Árboles Binarios',
      subject_id: 'course-1',
      subject_name: 'CS101 — Estructuras de Datos',
      environment_type: 'IDE_PERSISTENTE',
      expected_complexity: 'O(log n)',
      base_files: [
        { path: 'src/node.py', language: 'python', content: 'class Node:\n    pass\n', start_line: 1, line_count: 2 }
      ],
      base_files_available: true
    },
    a: {
      submission_id: 'sub-1',
      student_id: 'std-1',
      student_name: 'Estudiante Alfa',
      student_email: 'alfa@solv.test',
      verdict: 'AC',
      language: 'python',
      complexity: 'O(log n)',
      execution_time_ms: 120,
      memory_used_mb: 18,
      files_available: true,
      files: [
        {
          path: 'src/tree.py',
          language: 'python',
          content: 'def insert(node, value):\n    if node is None:\n        return value\n    return node\n',
          start_line: 1,
          line_count: 4
        }
      ],
      metrics: {
        total_time_ms: 65000,
        total_keystrokes: 42,
        paste_events_count: 1,
        paste_percentage: 12.5,
        events_available: true
      }
    },
    b: {
      submission_id: 'sub-2',
      student_id: 'std-2',
      student_name: 'Estudiante Beta',
      student_email: 'beta@solv.test',
      verdict: 'WA',
      language: 'python',
      complexity: 'O(n)',
      execution_time_ms: 180,
      memory_used_mb: 22,
      files_available: true,
      files: [
        {
          path: 'src/tree.py',
          language: 'python',
          content: 'def insert(node, value):\n    if node is None:\n        return value\n    return node\n',
          start_line: 1,
          line_count: 4
        }
      ],
      metrics: {
        total_time_ms: 48000,
        total_keystrokes: 30,
        paste_events_count: 0,
        paste_percentage: 0,
        events_available: true
      }
    },
    similarity: 92,
    risk_level: 'critical',
    ranges: [
      {
        source_a: { file_path: 'src/tree.py', start_line: 1, end_line: 4 },
        source_b: { file_path: 'src/tree.py', start_line: 1, end_line: 4 },
        token_count: 140,
        exact_match: true
      }
    ]
  };

  beforeEach(async () => {
    const mockCourseService = {
      analyzePlagiarism: vi.fn().mockReturnValue(of(mockReport)),
      getPlagiarismPair: vi.fn().mockReturnValue(of(mockPair))
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
    const ranges = component.pairRanges();
    expect(ranges.length).toBe(1);
    expect(ranges[0].source_a.file_path).toBe('src/tree.py');
    expect(ranges[0].source_a.start_line).toBe(1);
  });

  it('carga el par y arma el árbol de archivos afectados con la base descartada', () => {
    expect(component.pair()).toEqual(mockPair);
    expect(component.isIde()).toBe(true);
    expect(component.affectedFiles().map(file => file.path)).toEqual(['src/tree.py']);
    expect(component.baseFiles().map(file => file.path)).toEqual(['src/node.py']);
  });

  it('resalta en cada lado solo el tramo del archivo elegido', () => {
    component.selectPath('src/tree.py');
    expect(component.highlightsA()).toEqual([{ start: 1, end: 4 }]);
    expect(component.highlightsB()).toEqual([{ start: 1, end: 4 }]);
    expect(component.codeA()).toContain('def insert');
  });

  it('resume la cadencia de escritura de cada entrega', () => {
    expect(component.formatDuration(65000)).toBe('1 min 5 s');
    expect(component.formatPercent(12.5)).toBe('12.5%');
    expect(component.pair()?.a.metrics.paste_events_count).toBe(1);
  });

  it('traduce los veredictos del juez a texto para el docente', () => {
    expect(component.verdictText('AC')).toBe('Todos los tests pasan');
    expect(component.verdictText('AST_BLOCKED')).toBe('Bloqueado por análisis');
    expect(component.verdictClass('WA')).toBe('verdict-error');
  });
});
