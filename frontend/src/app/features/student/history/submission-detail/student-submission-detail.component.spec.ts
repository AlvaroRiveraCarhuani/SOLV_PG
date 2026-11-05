import { ComponentFixture, TestBed } from '@angular/core/testing';
import { By } from '@angular/platform-browser';
import { ActivatedRoute, provideRouter } from '@angular/router';
import { StudentSubmissionDetailComponent } from './student-submission-detail.component';
import { StudentService, StudentSubmissionHistoryItem } from '@core/services/student.service';
import { ReviewFileTreeComponent } from '@shared/components/review-file-tree/review-file-tree.component';
import { RubricDTO } from '@core/models/rubric.model';
import { SubmissionComment } from '@core/models/submission-comment.model';

const IDE_SUBMISSION: StudentSubmissionHistoryItem = {
  id: 'sub-ide-1',
  exercise_id: 'ex-ide-1',
  exercise_title: 'API Express con pruebas unitarias',
  subject_id: 'sub-1',
  subject_code: 'SIS-420',
  subject_name: 'Desarrollo Web Fullstack',
  environment_type: 'IDE_PERSISTENTE',
  code: '# === FILE: src/app.ts ===\nimport express from "express";\nconst app = express();\n# === FILE: package.json ===\n{"name":"api"}\n',
  verdict: 'AC',
  score: 95,
  execution_time_ms: 0,
  memory_used_mb: 0,
  manual_override: true,
  override_reason: 'Excelente estructura y modularización.',
  submitted_at: '2026-10-26T19:47:08'
};

const JUDGE_SUBMISSION: StudentSubmissionHistoryItem = {
  id: 'sub-judge-1',
  exercise_id: 'ex-judge-1',
  exercise_title: 'Recorrido de listas',
  subject_id: 'sub-2',
  subject_code: 'SIS-211',
  subject_name: 'Estructuras de Datos',
  environment_type: 'JUEZ_EFIMERO',
  language: 'python',
  code: 'def recorrer(lista):\n    return lista\n',
  verdict: 'AC',
  score: 100,
  execution_time_ms: 18,
  memory_used_mb: 32,
  submitted_at: '2026-10-20T14:05:00'
};

const MOCK_COMMENTS: SubmissionComment[] = [
  {
    id: 'comm-1',
    submission_id: 'sub-ide-1',
    author_id: 'teacher-1',
    author_name: 'Ing. Perez',
    file_path: 'src/app.ts',
    line_number: 2,
    comment: 'Buen uso de middleware de express.',
    created_at: '2026-10-27T10:00:00'
  }
];

const MOCK_RUBRIC: RubricDTO = {
  id: 'rub-1',
  exercise_id: 'ex-ide-1',
  criteria: [
    {
      name: 'Diseño y Tipado',
      description: 'Uso de interfaces claras.',
      weight: 40,
      levels: [
        { name: 'Excelente', score: 100, description: 'Tipado estricto' },
        { name: 'Regular', score: 50, description: 'Algunos any' }
      ]
    },
    {
      name: 'Calidad de Código',
      description: 'Estructura legible.',
      weight: 60,
      levels: [
        { name: 'Completo', score: 100, description: 'Código limpio' }
      ]
    }
  ]
};

describe('StudentSubmissionDetailComponent', () => {
  let component: StudentSubmissionDetailComponent;
  let fixture: ComponentFixture<StudentSubmissionDetailComponent>;

  async function setup(submissionIdParam = 'sub-ide-1'): Promise<void> {
    TestBed.resetTestingModule();

    const mockStudentService = {
      getSubmissionReview: (id: string) => {
        if (id === 'sub-ide-1') return Promise.resolve(IDE_SUBMISSION);
        if (id === 'sub-judge-1') return Promise.resolve(JUDGE_SUBMISSION);
        return Promise.resolve(null);
      },
      getSubmissionComments: (id: string) => {
        if (id === 'sub-ide-1') return Promise.resolve(MOCK_COMMENTS);
        return Promise.resolve([]);
      },
      getExerciseRubric: (exId: string) => {
        if (exId === 'ex-ide-1') return Promise.resolve(MOCK_RUBRIC);
        return Promise.resolve(null);
      }
    };

    await TestBed.configureTestingModule({
      imports: [StudentSubmissionDetailComponent],
      providers: [
        provideRouter([]),
        { provide: StudentService, useValue: mockStudentService },
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: {
              paramMap: {
                get: (key: string) => (key === 'submissionId' ? submissionIdParam : null)
              }
            }
          }
        }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(StudentSubmissionDetailComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    await component.loadPromise;
    fixture.detectChanges();
  }

  it('renderiza la vista completa para una entrega de proyecto IDE con rúbrica y árbol de archivos', async () => {
    await setup('sub-ide-1');

    expect(component.isLoading()).toBe(false);
    expect(component.isProject()).toBe(true);

    const el = fixture.nativeElement as HTMLElement;
    expect(el.textContent).toContain('API Express con pruebas unitarias');
    expect(el.textContent).toContain('Entorno de Desarrollo');
    expect(el.textContent).toContain('SIS-420');
    expect(el.textContent).toContain('Desarrollo Web Fullstack');

    // Tree exists
    const treeDebug = fixture.debugElement.query(By.directive(ReviewFileTreeComponent));
    expect(treeDebug).not.toBeNull();

    // Code & file header
    expect(el.querySelector('.file-path')?.textContent).toContain('src/app.ts');
    expect(el.textContent).toContain('import express from "express"');

    // Inline comment
    expect(el.textContent).toContain('Ing. Perez');
    expect(el.textContent).toContain('Buen uso de middleware de express.');

    // Rubric
    expect(el.textContent).toContain('Evaluación por Rúbrica');
    expect(el.textContent).toContain('Diseño y Tipado');
    expect(el.textContent).toContain('40%');
    expect(el.textContent).toContain('Calidad de Código');
    expect(el.textContent).toContain('60%');

    // Score & observation
    expect(el.textContent).toContain('95');
    expect(el.textContent).toContain('/ 100 pts');
    expect(el.textContent).toContain('Excelente estructura y modularización.');
  });

  it('renderiza la vista para una entrega de juez algorítmico sin árbol de archivos pero con métricas', async () => {
    await setup('sub-judge-1');

    expect(component.isLoading()).toBe(false);
    expect(component.isProject()).toBe(false);

    const el = fixture.nativeElement as HTMLElement;
    expect(el.textContent).toContain('Recorrido de listas');
    expect(el.textContent).toContain('Entorno de Evaluación');

    // No tree
    const treeDebug = fixture.debugElement.query(By.directive(ReviewFileTreeComponent));
    expect(treeDebug).toBeNull();

    // Metrics visible
    expect(el.textContent).toContain('18 ms');
    expect(el.textContent).toContain('32 MB');
  });

  it('muestra estado de error si el ID de entrega no existe', async () => {
    await setup('non-existent-sub');

    expect(component.loadError()).toBeTruthy();
    const el = fixture.nativeElement as HTMLElement;
    expect(el.querySelector('.detail-error-card')).not.toBeNull();
  });
});
