import { ComponentFixture, TestBed } from '@angular/core/testing';
import { Router, provideRouter } from '@angular/router';
import { StudentWorkspacesComponent } from './student-workspaces.component';
import { StudentDashboardData, StudentService } from '@core/services/student.service';
import { WorkspaceInstance } from '@core/models/workspace.model';

const RUNNING_WORKSPACE = {
  id: 'w1',
  status: 'running',
  last_activity_at: '2026-10-24T10:30:00',
  access_url: 'https://lab.example/w1'
} as unknown as WorkspaceInstance;

const HIBERNATED_WORKSPACE = {
  id: 'w2',
  status: 'hibernated',
  last_activity_at: '2026-10-23T09:00:00'
} as unknown as WorkspaceInstance;

const DASHBOARD: StudentDashboardData = {
  student_id: 'u1',
  tenant_id: 't1',
  subjects: [
    {
      subject: { id: 's1', code: 'SIS-211', name: 'Estructuras de Datos' },
      total_exercises: 4,
      completed_exercises: 3,
      progress_percent: 75,
      active_workspace: RUNNING_WORKSPACE
    },
    {
      subject: { id: 's2', code: 'SIS-312', name: 'Sistemas Operativos' },
      total_exercises: 2,
      completed_exercises: 0,
      progress_percent: 0,
      active_workspace: HIBERNATED_WORKSPACE
    },
    {
      subject: { id: 's3', code: 'MAT-101', name: 'Cálculo I' },
      total_exercises: 5,
      completed_exercises: 0,
      progress_percent: 0
    }
  ],
  recent_submissions: []
};

describe('StudentWorkspacesComponent', () => {
  let component: StudentWorkspacesComponent;
  let fixture: ComponentFixture<StudentWorkspacesComponent>;
  let router: Router;
  let startWorkspace: ReturnType<typeof vi.fn>;
  let pauseWorkspace: ReturnType<typeof vi.fn>;

  function cards(): HTMLElement[] {
    return Array.from((fixture.nativeElement as HTMLElement).querySelectorAll<HTMLElement>('.lab-card'));
  }

  function cardBySubject(name: string): HTMLElement {
    const card = cards().find(c => c.textContent?.includes(name));
    if (!card) {
      throw new Error(`No card found for subject ${name}`);
    }
    return card;
  }

  async function setup(): Promise<void> {
    localStorage.clear();
    TestBed.resetTestingModule();

    startWorkspace = vi.fn().mockResolvedValue({ id: 'w3', status: 'running' });
    pauseWorkspace = vi.fn().mockResolvedValue(undefined);

    const mockStudentService = {
      dashboardData: () => DASHBOARD,
      dueAssignments: () => [],
      loading: () => false,
      loadDashboard: () => Promise.resolve(),
      startWorkspace,
      pauseWorkspace
    };

    await TestBed.configureTestingModule({
      imports: [StudentWorkspacesComponent],
      providers: [
        provideRouter([]),
        { provide: StudentService, useValue: mockStudentService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(StudentWorkspacesComponent);
    component = fixture.componentInstance;
    router = TestBed.inject(Router);
    vi.spyOn(router, 'navigate').mockResolvedValue(true);
    fixture.detectChanges();
  }

  afterEach(() => {
    localStorage.clear();
  });

  it('muestra el aviso y una tarjeta por materia', async () => {
    await setup();

    const text = (fixture.nativeElement as HTMLElement).textContent ?? '';
    expect(text).toContain('Tus archivos se conservan automáticamente entre sesiones.');
    expect(cards().length).toBe(3);
    expect(text).toContain('Estructuras de Datos');
    expect(text).toContain('SIS-211');
    expect(text).toContain('Sistemas Operativos');
    expect(text).toContain('Cálculo I');
  });

  it('filtra por nombre y por código desde el buscador', async () => {
    await setup();

    component.searchQuery.set('cálculo');
    fixture.detectChanges();
    expect(cards().length).toBe(1);
    expect(cards()[0].textContent).toContain('Cálculo I');

    component.searchQuery.set('SIS-');
    fixture.detectChanges();
    expect(cards().length).toBe(2);

    component.searchQuery.set('');
    fixture.detectChanges();
    expect(cards().length).toBe(3);
  });

  it('para un entorno activo muestra entrada al laboratorio, última actividad y pausa', async () => {
    await setup();

    const card = cardBySubject('Estructuras de Datos');
    expect(card.querySelector('a[href="/student/workspaces/s1"]')?.textContent).toContain('Entrar al laboratorio');
    expect(card.textContent).toContain('Última actividad:');
    expect(card.querySelector('time')?.getAttribute('datetime')).toBe('2026-10-24T10:30:00');

    const pauseButton = Array.from(card.querySelectorAll<HTMLButtonElement>('button')).find(btn => btn.textContent?.includes('Pausar entorno'));
    expect(pauseButton).toBeTruthy();
    pauseButton?.dispatchEvent(new Event('click'));
    fixture.detectChanges();
    await fixture.whenStable();
    expect(pauseWorkspace).toHaveBeenCalledWith('w1');
  });

  it('reanuda o inicia un entorno inactivo y navega al laboratorio', async () => {
    await setup();

    const hibernated = cardBySubject('Sistemas Operativos');
    expect(hibernated.querySelector('button')?.textContent).toContain('Reanudar laboratorio');

    const inactive = cardBySubject('Cálculo I');
    expect(inactive.querySelector('button')?.textContent).toContain('Iniciar laboratorio');
    expect(inactive.querySelector('a[href="/student/courses/s3"]')?.textContent).toContain('Ver ejercicios de la materia');

    inactive.querySelector<HTMLButtonElement>('button')?.dispatchEvent(new Event('click'));
    fixture.detectChanges();
    await fixture.whenStable();

    expect(startWorkspace).toHaveBeenCalledWith('s3');
    expect(router.navigate).toHaveBeenCalledWith(['/student/workspaces', 's3']);
  });
});
