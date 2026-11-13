import '@angular/compiler';
import '@angular/localize/init';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { of, throwError } from 'rxjs';
import { AdminDashboardComponent, DEFAULT_BLOCK_ORDER } from './admin-dashboard.component';
import { AdminMetricsService } from '../services/admin-metrics.service';
import { CourseLoadSummary, HostSystemHealth, TechnicalIncident } from '@core/models/admin.model';

interface MockAdminMetricsService {
  health: ReturnType<typeof signal<HostSystemHealth | null>>;
  isLoading: ReturnType<typeof signal<boolean>>;
  refresh: ReturnType<typeof vi.fn>;
  hibernateAll: ReturnType<typeof vi.fn>;
  stopContainer: ReturnType<typeof vi.fn>;
  autoRefreshActive: ReturnType<typeof vi.fn>;
  pauseEnvironment: ReturnType<typeof vi.fn>;
  restartEnvironment: ReturnType<typeof vi.fn>;
}

const makeHealth = (overrides: Partial<HostSystemHealth> = {}): HostSystemHealth => ({
  status: 'operational',
  docker: { server_version: '27.0', storage_driver: 'overlay2', healthy: true },
  docker_version: 'Docker Engine v27.0 (overlay2)',
  health_reasons: [],
  uptime_seconds: 3600,
  metrics: {
    ram_used_bytes: 0,
    ram_total_bytes: 0,
    ram_percent: 50,
    cpu_cores: 4,
    cpu_percent: 10,
    disk_used_bytes: 0,
    disk_total_bytes: 0,
    disk_percent: 20,
    containers_active: 0,
    containers_hibernated: 0,
    containers_max: 40
  },
  containers: [],
  courses_load: [],
  incidents: [],
  ...overrides
});

describe('AdminDashboardComponent', () => {
  let component: AdminDashboardComponent;
  let mockMetricsService: MockAdminMetricsService;

  const healthSignal = signal<HostSystemHealth | null>(null);
  const isLoadingSignal = signal<boolean>(false);

  beforeEach(async () => {
    mockMetricsService = {
      health: healthSignal,
      isLoading: isLoadingSignal,
      refresh: vi.fn(),
      hibernateAll: vi.fn(),
      stopContainer: vi.fn(),
      autoRefreshActive: vi.fn().mockReturnValue(false),
      pauseEnvironment: vi.fn().mockReturnValue(of(undefined)),
      restartEnvironment: vi.fn().mockReturnValue(of(undefined))
    };

    await TestBed.configureTestingModule({
      imports: [AdminDashboardComponent],
      providers: [
        { provide: AdminMetricsService, useValue: mockMetricsService }
      ]
    }).compileComponents();

    const fixture = TestBed.createComponent(AdminDashboardComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('debe crearse correctamente', () => {
    expect(component).toBeTruthy();
  });

  it('blockOrder inicial sigue el orden por defecto', () => {
    expect(component.blockOrder()).toEqual(['kpis', 'chart', 'split', 'containers']);
  });

  it('isCustomizing inicia en false', () => {
    expect(component.isCustomizing()).toBe(false);
  });

  it('toggleCustomizing invierte isCustomizing', () => {
    component.toggleCustomizing();
    expect(component.isCustomizing()).toBe(true);
    component.toggleCustomizing();
    expect(component.isCustomizing()).toBe(false);
  });

  it('resetLayout restaura el orden por defecto', () => {
    component.blockOrder.set(['containers', 'kpis', 'chart', 'split']);
    component.resetLayout();
    expect(component.blockOrder()).toEqual(DEFAULT_BLOCK_ORDER);
  });

  it('selectedCourse inicia en null', () => {
    expect(component.selectedCourse()).toBeNull();
  });

  it('onPauseEnvironment incrementa el recargo del modal en éxito', () => {
    expect(component.wsReloadToken()).toBe(0);

    component.onPauseEnvironment('ws-1');

    expect(mockMetricsService.pauseEnvironment).toHaveBeenCalledWith('ws-1');
    expect(component.wsReloadToken()).toBe(1);
  });

  it('onPauseEnvironment no recarga el modal si la pausa falla', () => {
    mockMetricsService.pauseEnvironment.mockReturnValue(throwError(() => new Error('404')));

    component.onPauseEnvironment('ws-1');

    expect(component.wsReloadToken()).toBe(0);
  });

  it('onRestartEnvironment recarga el modal en éxito y avisa si falla', () => {
    component.onRestartEnvironment('ws-1');
    expect(mockMetricsService.restartEnvironment).toHaveBeenCalledWith('ws-1');
    expect(component.wsReloadToken()).toBe(1);

    mockMetricsService.restartEnvironment.mockReturnValue(throwError(() => new Error('404')));
    component.onRestartEnvironment('ws-1');
    expect(component.wsReloadToken()).toBe(1);
  });

  it('onViewCourseDetails establece selectedCourse', () => {
    const course = {
      id: 'c1',
      subject_id: 's1',
      subject_name: 'Redes I',
      course_name: 'Redes I',
      teacher_name: 'Docente',
      active_students: 10,
      hibernated_students: 2,
      ram_used_mb: 512
    } as CourseLoadSummary;
    component.onViewCourseDetails(course);
    expect(component.selectedCourse()).toBe(course);
  });

  it('el cierre del modal de curso limpia selectedCourse', () => {
    const dummyCourse: CourseLoadSummary = {
      id: 'c1',
      course_name: 'Redes I',
      teacher_name: 'Docente',
      active_students: 0,
      hibernated_students: 0,
      ram_used_mb: 0
    };
    component.selectedCourse.set(dummyCourse);
    component.selectedCourse.set(null);
    expect(component.selectedCourse()).toBeNull();
  });

  it('selectedIncidentForLogs inicia en null', () => {
    expect(component.selectedIncidentForLogs()).toBeNull();
  });

  it('onViewLogs construye incidencia de respaldo con el environment solicitado', () => {
    component.onViewLogs('ws-1');
    const incident = component.selectedIncidentForLogs();
    expect(incident).not.toBeNull();
    expect(incident?.environment_id).toBe('ws-1');
  });

  it('onViewLogs localiza la incidencia existente cuando health la contiene', () => {
    const existing: TechnicalIncident = {
      id: 'inc-1',
      type: 'oom_killed',
      environment_id: 'ws-9',
      student_name: 'Ana',
      course_name: 'Prog',
      description: 'OOM',
      timestamp: '2026-10-01'
    };
    healthSignal.set(makeHealth({ incidents: [existing] }));
    component.onViewLogs('ws-9');
    expect(component.selectedIncidentForLogs()).toBe(existing);
    healthSignal.set(null);
  });

  it('el cierre del visor de logs limpia selectedIncidentForLogs', () => {
    component.selectedIncidentForLogs.set({ id: 'inc-1' } as TechnicalIncident);
    component.selectedIncidentForLogs.set(null);
    expect(component.selectedIncidentForLogs()).toBeNull();
  });

  it('requestHibernateAll pide confirmación sin ejecutar la operación', () => {
    healthSignal.set(makeHealth({ metrics: { ...makeHealth().metrics, containers_active: 3 } }));

    component.requestHibernateAll();

    expect(component.showHibernateConfirm()).toBe(true);
    expect(mockMetricsService.hibernateAll).not.toHaveBeenCalled();

    component.cancelHibernateAll();

    expect(component.showHibernateConfirm()).toBe(false);
    expect(mockMetricsService.hibernateAll).not.toHaveBeenCalled();
    healthSignal.set(null);
  });

  it('confirmHibernateAll ejecuta la hibernación una sola vez', () => {
    healthSignal.set(makeHealth({ metrics: { ...makeHealth().metrics, containers_active: 2 } }));
    mockMetricsService.hibernateAll.mockReturnValue(of(undefined));

    component.requestHibernateAll();
    component.confirmHibernateAll();
    component.confirmHibernateAll();

    expect(component.showHibernateConfirm()).toBe(false);
    expect(mockMetricsService.hibernateAll).toHaveBeenCalledTimes(1);
    component.ngOnDestroy();
    healthSignal.set(null);
  });

  it('requestHibernateAll con cero activos no abre confirmación', () => {
    healthSignal.set(makeHealth());

    component.requestHibernateAll();

    expect(component.showHibernateConfirm()).toBe(false);
    expect(mockMetricsService.hibernateAll).not.toHaveBeenCalled();
    healthSignal.set(null);
  });

  it('expone los motivos de salud fuera del tooltip', () => {
    healthSignal.set(makeHealth({ status: 'degraded', health_reasons: ['oom_killed_environments'] }));

    expect(component.healthReasons()).toEqual(['Laboratorios detenidos por límite de memoria']);

    healthSignal.set(makeHealth());
    expect(component.healthReasons()).toEqual([]);
    healthSignal.set(null);
  });

  it('showAllContainers inicia en false', () => {
    expect(component.showAllContainers()).toBe(false);
  });
});
