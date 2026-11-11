import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { ActivatedRoute, Router, convertToParamMap } from '@angular/router';
import { signal } from '@angular/core';
import { of } from 'rxjs';
import { ExerciseEditorComponent } from './exercise-editor.component';
import { ExerciseEditorStore } from './exercise-editor.store';
import { TeacherAiCopilotService } from '../services/teacher-ai-copilot.service';
import { TeacherDashboardService } from '../services/teacher-dashboard.service';
import { TeacherCourseService } from '../services/teacher-course.service';
import { TeacherCourseSummary } from '../models/teacher.models';

describe('ExerciseEditorComponent', () => {
  let fixture: ComponentFixture<ExerciseEditorComponent>;
  let store: ExerciseEditorStore;

  const mockCourses: TeacherCourseSummary[] = [
    {
      id: 'course-1',
      code: 'CS101',
      name: 'Programación I',
      students_count: 0,
      active_now: 0,
      pending_review: 0,
      at_risk: 0
    }
  ];

  function mount(params: Record<string, string>, query: Record<string, string>): void {
    TestBed.configureTestingModule({
      imports: [ExerciseEditorComponent],
      providers: [
        ExerciseEditorStore,
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: Router, useValue: { navigate: vi.fn() } },
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: {
              paramMap: convertToParamMap(params),
              queryParamMap: convertToParamMap(query)
            }
          }
        },
        { provide: TeacherAiCopilotService, useValue: { getStatus: () => of({ enabled: true }) } },
        { provide: TeacherDashboardService, useValue: { courses: signal<TeacherCourseSummary[]>(mockCourses) } },
        { provide: TeacherCourseService, useValue: {} }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(ExerciseEditorComponent);
    store = TestBed.inject(ExerciseEditorStore);
    fixture.detectChanges();
  }

  it('debe crearse correctamente', () => {
    mount({ courseId: 'course-1' }, {});
    expect(fixture.componentInstance).toBeTruthy();
  });

  it('preselecciona la plantilla y el tipo de entorno desde los query params al crear un ejercicio nuevo', () => {
    mount(
      { courseId: 'course-1' },
      { template_id: 'tpl-1', environment_type: 'IDE_PERSISTENTE' }
    );

    expect(store.metadata().template_id).toBe('tpl-1');
    expect(store.metadata().environment_type).toBe('IDE_PERSISTENTE');
    expect(store.metadata().modality).toBe('environment');
  });

  it('preserva el título y la plantilla previa al aplicar un borrador de IA', () => {
    mount({ courseId: 'course-1' }, {});
    store.updateMetadata({ title: 'Título previo del docente', template_id: 'tpl-fijada' });

    store.applyAIGeneratedDraft({
      title: 'Título generado por IA',
      description: 'Consigna generada',
      difficulty: 'medium',
      language: 'python',
      template_id: 'tpl-otra-de-ia',
      environment_type: 'IDE_PERSISTENTE',
      runner_verified_all: true,
      latency_ms: 120
    });

    expect(store.metadata().title).toBe('Título previo del docente');
    expect(store.metadata().template_id).toBe('tpl-fijada');
    expect(store.statement().description).toBe('Consigna generada');
  });

  it('no preselecciona plantilla cuando falta template_id', () => {
    mount({ courseId: 'course-1' }, {});

    expect(store.metadata().template_id).toBeNull();
  });

  it('muestra el nombre del curso resuelto en el encabezado', () => {
    mount({ courseId: 'course-1' }, {});

    expect(fixture.componentInstance.courseName()).toBe('Programación I');
  });
});
