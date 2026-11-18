import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, provideRouter } from '@angular/router';
import { of } from 'rxjs';
import { vi, describe, beforeEach, it, expect } from 'vitest';
import { PracticeDetailComponent } from './practice-detail.component';
import { TeacherCourseService } from '../../services/teacher-course.service';
import { PracticeEnvTestService } from '../../services/practice-env-test.service';
import { TeacherEnvironmentService } from '../../services/teacher-environment.service';
import { ContextHelpService } from '@shared/help/context-help.service';
import { ProductTourService } from '@shared/help/product-tour.service';
import { InteractiveTutorialService } from '@shared/help/interactive-tutorial.service';
import { ExerciseDetailDTO } from '../../models/teacher.models';

describe('PracticeDetailComponent', () => {
  let component: PracticeDetailComponent;
  let fixture: ComponentFixture<PracticeDetailComponent>;

  const mockExercise: ExerciseDetailDTO = {
    id: 'ex-123',
    title: 'Sistema de Gestión de Tareas',
    description: `
Descripción del proyecto con estructura de carpetas:

\`\`\`text
proyecto/
├── backend/
│   ├── src/
│   │   └── index.ts
│   └── package.json
└── README.md
\`\`\`
    `.trim(),
    environment_type: 'IDE_PERSISTENTE',
    status: 'published',
    template: {
      id: 'tmpl-1',
      name: 'Node.js + React',
      docker_image: 'solv/node-react:20'
    },
    config: {
      ide: {
        allow_resubmit: true
      },
      database: {
        engine: 'postgresql',
        init_script: 'CREATE TABLE tasks (id SERIAL);'
      }
    }
  };

  const mockCourseService = {
    getExercise: vi.fn().mockReturnValue(of(mockExercise)),
    getCourseLabs: vi.fn().mockReturnValue(of([])),
    unpublishExercise: vi.fn().mockReturnValue(of({}))
  };

  const mockEnvTestService = {
    latest: vi.fn().mockReturnValue(of(null)),
    start: vi.fn().mockReturnValue(of({ id: 'job-1', status: 'pending' })),
    cancelJob: vi.fn().mockReturnValue(of({})),
    pollJob: vi.fn().mockReturnValue(of({ id: 'job-1', status: 'success' })),
    listStudentDatabases: vi.fn().mockReturnValue(of([
      {
        student_id: 'std-1',
        student_name: 'Santiago Morales',
        student_email: 'santiago@uab.edu.bo',
        status: 'ready',
        updated_at: new Date().toISOString()
      }
    ])),
    resetStudentDatabase: vi.fn().mockReturnValue(of({ student_id: 'std-1' })),
    isTerminal: vi.fn().mockReturnValue(true)
  };

  const mockEnvironmentService = {
    openPracticeIDE: vi.fn().mockReturnValue(of({ access_url: 'https://ide.solv.dev' })),
    describePracticeIDEError: vi.fn().mockReturnValue('Error al abrir IDE')
  };

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [PracticeDetailComponent],
      providers: [
        provideRouter([]),
        { provide: TeacherCourseService, useValue: mockCourseService },
        { provide: PracticeEnvTestService, useValue: mockEnvTestService },
        { provide: TeacherEnvironmentService, useValue: mockEnvironmentService },
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: {
              paramMap: {
                get: (key: string) => (key === 'courseId' ? 'course-1' : 'ex-123')
              }
            }
          }
        }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(PracticeDetailComponent);
    component = fixture.componentInstance;
  });

  it('se inicializa y carga la práctica con sus secciones', () => {
    fixture.detectChanges();
    expect(component.exercise()?.id).toBe('ex-123');
    expect(component.isIde()).toBe(true);
    expect(component.hasDatabase()).toBe(true);
  });

  it('registra las guías contextuales en OnInit y las limpia en OnDestroy', () => {
    const help = TestBed.inject(ContextHelpService);
    const registerScreenSpy = vi.spyOn(help, 'registerScreen');
    const unregisterScreenSpy = vi.spyOn(help, 'unregisterScreen');
    const tourService = TestBed.inject(ProductTourService);
    const destroyTourSpy = vi.spyOn(tourService, 'destroy');

    fixture.detectChanges();
    expect(registerScreenSpy).toHaveBeenCalledWith('teacher-practice-detail');

    fixture.destroy();
    expect(unregisterScreenSpy).toHaveBeenCalledWith('teacher-practice-detail');
    expect(destroyTourSpy).toHaveBeenCalled();
  });

  it('permite alternar el colapso y despliegue de las secciones', () => {
    fixture.detectChanges();

    // Consigna arranca abierta
    expect(component.isDescriptionExpanded()).toBe(true);
    component.isDescriptionExpanded.set(false);
    expect(component.isDescriptionExpanded()).toBe(false);

    // Bases de datos arranca colapsada
    expect(component.isDatabasesExpanded()).toBe(false);
    component.isDatabasesExpanded.set(true);
    expect(component.isDatabasesExpanded()).toBe(true);
  });

  it('renderiza la estructura de carpetas en HTML enriquecido con íconos vectoriales', () => {
    fixture.detectChanges();
    const renderedHtml = fixture.nativeElement.innerHTML;

    // Debe contener el contenedor de árbol accesible
    expect(renderedHtml).toContain('class="tree"');
    expect(renderedHtml).toContain('role="group"');
    // Debe contener las filas con íconos
    expect(renderedHtml).toContain('tree-icon');
    expect(renderedHtml).toContain('tree-dir');
    expect(renderedHtml).toContain('tree-file');
    expect(renderedHtml).toContain('backend');
    expect(renderedHtml).toContain('README.md');
    // No debe contener glifos ASCII
    expect(renderedHtml).not.toContain('├──');
  });
});
