import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, Router, provideRouter } from '@angular/router';
import { StudentActiveLabComponent } from './student-active-lab.component';
import { StudentDashboardData, StudentService } from '@core/services/student.service';
import { WorkspaceInstance } from '@core/models/workspace.model';

const RUNNING_WORKSPACE = {
  id: 'w1',
  status: 'running',
  last_activity_at: '2026-10-24T10:30:00',
  access_url: 'https://lab.example/w1'
} as unknown as WorkspaceInstance;

const PENDING_WORKSPACE = {
  id: 'w2',
  status: 'pending'
} as unknown as WorkspaceInstance;

function dashboardFor(workspace: WorkspaceInstance | undefined): StudentDashboardData {
  return {
    student_id: 'u1',
    tenant_id: 't1',
    subjects: [
      {
        subject: { id: 's1', code: 'SIS-211', name: 'Estructuras de Datos' },
        total_exercises: 4,
        completed_exercises: 3,
        progress_percent: 75,
        active_workspace: workspace
      }
    ],
    recent_submissions: []
  };
}

describe('StudentActiveLabComponent', () => {
  let component: StudentActiveLabComponent;
  let fixture: ComponentFixture<StudentActiveLabComponent>;
  let router: Router;
  let startWorkspace: ReturnType<typeof vi.fn>;
  let pauseWorkspace: ReturnType<typeof vi.fn>;

  async function setup(workspace: WorkspaceInstance | undefined): Promise<void> {
    localStorage.clear();
    TestBed.resetTestingModule();

    startWorkspace = vi.fn().mockResolvedValue(RUNNING_WORKSPACE);
    pauseWorkspace = vi.fn().mockResolvedValue(undefined);

    const mockStudentService = {
      dashboardData: () => dashboardFor(workspace),
      dueAssignments: () => [],
      loading: () => false,
      loadDashboard: () => Promise.resolve(),
      startWorkspace,
      pauseWorkspace
    };

    const routeStub = {
      snapshot: {
        paramMap: {
          get: (key: string) => (key === 'subjectId' ? 's1' : null)
        }
      }
    };

    await TestBed.configureTestingModule({
      imports: [StudentActiveLabComponent],
      providers: [
        provideRouter([]),
        { provide: StudentService, useValue: mockStudentService },
        { provide: ActivatedRoute, useValue: routeStub }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(StudentActiveLabComponent);
    component = fixture.componentInstance;
    router = TestBed.inject(Router);
    vi.spyOn(router, 'navigate').mockResolvedValue(true);
    fixture.detectChanges();
  }

  afterEach(() => {
    localStorage.clear();
  });

  it('muestra la barra con nombre, código, estado y enlace externo', async () => {
    await setup(RUNNING_WORKSPACE);

    const text = (fixture.nativeElement as HTMLElement).textContent ?? '';
    expect(text).toContain('Estructuras de Datos');
    expect(text).toContain('SIS-211');
    expect(text).toContain('Activo');
    expect(text).toContain('Abrir en pestaña nueva');
    expect(text).toContain('Pausar y salir');

    const externalLink = (fixture.nativeElement as HTMLElement).querySelector<HTMLAnchorElement>('a[target="_blank"]');
    expect(externalLink?.getAttribute('href')).toBe('https://lab.example/w1');
  });

  it('renderiza el iframe seguro cuando el entorno está activo con URL', async () => {
    await setup(RUNNING_WORKSPACE);

    expect(component.safeLabUrl()).toBeTruthy();
    const iframe = fixture.nativeElement.querySelector('iframe.lab-iframe');
    expect(iframe).toBeTruthy();
    expect(iframe?.getAttribute('title')).toBe('Entorno de práctica');
  });

  it('muestra estado vacío con acción de inicio cuando no está activo', async () => {
    await setup(PENDING_WORKSPACE);

    expect(component.safeLabUrl()).toBeNull();
    expect(fixture.nativeElement.querySelector('iframe.lab-iframe')).toBeNull();

    const text = (fixture.nativeElement as HTMLElement).textContent ?? '';
    expect(text).toContain('Tu entorno de práctica no está activo');

    const startButton = Array.from((fixture.nativeElement as HTMLElement).querySelectorAll('button')).find(btn => btn.textContent?.includes('Iniciar entorno de práctica'));
    expect(startButton).toBeTruthy();
    startButton?.dispatchEvent(new Event('click'));
    fixture.detectChanges();
    await fixture.whenStable();
    expect(startWorkspace).toHaveBeenCalledWith('s1');
  });

  it('pausa el entorno y navega de vuelta a laboratorios', async () => {
    await setup(RUNNING_WORKSPACE);

    const pauseButton = Array.from((fixture.nativeElement as HTMLElement).querySelectorAll('button')).find(btn => btn.textContent?.includes('Pausar y salir'));
    expect(pauseButton).toBeTruthy();
    pauseButton?.dispatchEvent(new Event('click'));
    fixture.detectChanges();
    await fixture.whenStable();

    expect(pauseWorkspace).toHaveBeenCalledWith('w1');
    expect(router.navigate).toHaveBeenCalledWith(['/student/workspaces']);
  });
});
