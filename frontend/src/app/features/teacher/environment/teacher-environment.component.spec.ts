import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter } from '@angular/router';
import { WritableSignal, signal } from '@angular/core';
import { of } from 'rxjs';
import { WorkspaceInstance } from '@core/models/workspace.model';
import { TeacherCourseSummary } from '../models/teacher.models';
import { TeacherDashboardService } from '../services/teacher-dashboard.service';
import { TeacherEnvironmentService } from '../services/teacher-environment.service';
import { TeacherEnvironmentComponent } from './teacher-environment.component';

describe('TeacherEnvironmentComponent', () => {
  let fixture: ComponentFixture<TeacherEnvironmentComponent>;
  let component: TeacherEnvironmentComponent;

  const courses: TeacherCourseSummary[] = [
    {
      id: 'subj-1',
      name: 'Programación I',
      code: 'PROG-1',
      students_count: 24,
      active_now: 0,
      pending_review: 0,
      at_risk: 0
    }
  ];

  const running: WorkspaceInstance = {
    id: 'ws-1',
    student_id: 'teacher-1',
    subject_id: 'subj-1',
    status: 'running',
    access_url: 'http://localhost:3999',
    memory_limit_mb: 512,
    created_at: '2026-10-31T09:00:00Z'
  };

  const hibernated: WorkspaceInstance = { ...running, status: 'hibernated' };

  let environmentSignal: WritableSignal<WorkspaceInstance | null>;
  let errorSignal: WritableSignal<string | null>;
  let mockService: {
    environment: WritableSignal<WorkspaceInstance | null>;
    isLoading: WritableSignal<boolean>;
    isStarting: WritableSignal<boolean>;
    error: WritableSignal<string | null>;
    load: ReturnType<typeof vi.fn>;
    open: ReturnType<typeof vi.fn>;
    pause: ReturnType<typeof vi.fn>;
  };

  async function setup(current: WorkspaceInstance | null, subjectId: string | null = null): Promise<void> {
    environmentSignal = signal<WorkspaceInstance | null>(current);
    errorSignal = signal<string | null>(null);

    mockService = {
      environment: environmentSignal,
      isLoading: signal(false),
      isStarting: signal(false),
      error: errorSignal,
      load: vi.fn().mockReturnValue(of(current)),
      open: vi.fn().mockReturnValue(of(running)),
      pause: vi.fn().mockReturnValue(of(hibernated))
    };

    await TestBed.configureTestingModule({
      imports: [TeacherEnvironmentComponent],
      providers: [
        provideRouter([]),
        { provide: TeacherEnvironmentService, useValue: mockService },
        {
          provide: TeacherDashboardService,
          useValue: {
            courses: signal(courses),
            getCourses: () => of(courses)
          }
        },
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: { queryParamMap: convertToParamMap(subjectId ? { subject_id: subjectId } : {}) }
          }
        }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(TeacherEnvironmentComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  }

  it('lee el entorno de la materia que trae la URL y muestra el editor cuando corre', async () => {
    await setup(running, 'subj-1');

    expect(mockService.load).toHaveBeenCalledWith('subj-1');
    expect(component.subject()?.code).toBe('PROG-1');
    expect(component.isRunning()).toBe(true);
    expect(fixture.nativeElement.querySelector('iframe')).not.toBeNull();
  });

  it('sin materia elegida pide una y no ofrece abrir nada', async () => {
    await setup(null);

    const text = (fixture.nativeElement as HTMLElement).textContent ?? '';
    expect(text).toContain('Elegí una materia');
    expect(fixture.nativeElement.querySelector('.open-button')).toBeNull();
    expect(mockService.load).not.toHaveBeenCalled();
  });

  it('abre el entorno de la materia elegida', async () => {
    await setup(null);
    component.onSubjectChange('subj-1');

    expect(mockService.load).toHaveBeenCalledWith('subj-1');

    component.open();

    expect(mockService.open).toHaveBeenCalledWith('subj-1');
  });

  it('con el entorno en pausa el texto de la acción es reanudar', async () => {
    await setup(hibernated, 'subj-1');

    expect(component.openLabel()).toBe('Reanudar entorno');
    const text = (fixture.nativeElement as HTMLElement).textContent ?? '';
    expect(text).toContain('El entorno está en pausa');
  });

  it('pausa el entorno con su identificador', async () => {
    await setup(running);

    component.pause();

    expect(mockService.pause).toHaveBeenCalledWith('ws-1');
  });

  it('muestra el motivo cuando el servicio falla', async () => {
    await setup(null);
    errorSignal.set('El cupo institucional está lleno.');
    fixture.detectChanges();

    const alert = fixture.nativeElement.querySelector('.error-panel');
    expect(alert?.textContent).toContain('El cupo institucional está lleno.');
  });
});
