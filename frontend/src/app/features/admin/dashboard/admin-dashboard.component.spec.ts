import '@angular/compiler';
import '@angular/localize/init';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { AdminDashboardComponent, DEFAULT_BLOCK_ORDER } from './admin-dashboard.component';
import { AdminMetricsService } from '../services/admin-metrics.service';
import { CourseLoadSummary, TechnicalIncident } from '@core/models/admin.model';

describe('AdminDashboardComponent', () => {
  let component: AdminDashboardComponent;
  let mockMetricsService: any;

  const healthSignal = signal<any>(null);
  const isLoadingSignal = signal<boolean>(false);

  beforeEach(async () => {
    mockMetricsService = {
      health: healthSignal,
      isLoading: isLoadingSignal,
      refresh: vi.fn(),
      hibernateAll: vi.fn(),
      stopContainer: vi.fn(),
      autoRefreshActive: vi.fn().mockReturnValue(false)
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
    component.selectedCourse.set({ id: 'c1' } as any);
    component.selectedCourse.set(null);
    expect(component.selectedCourse()).toBeNull();
  });

  it('selectedIncidentForLogs inicia en null', () => {
    expect(component.selectedIncidentForLogs()).toBeNull();
  });

  it('onViewLogs construye incidencia de respaldo con el workspace solicitado', () => {
    component.onViewLogs('ws-1');
    const incident = component.selectedIncidentForLogs();
    expect(incident).not.toBeNull();
    expect(incident?.workspace_id).toBe('ws-1');
  });

  it('onViewLogs localiza la incidencia existente cuando health la contiene', () => {
    const existing = { id: 'inc-1', workspace_id: 'ws-9', student_name: 'Ana' } as TechnicalIncident;
    healthSignal.set({ incidents: [existing] });
    component.onViewLogs('ws-9');
    expect(component.selectedIncidentForLogs()).toBe(existing);
    healthSignal.set(null);
  });

  it('el cierre del visor de logs limpia selectedIncidentForLogs', () => {
    component.selectedIncidentForLogs.set({ id: 'inc-1' } as TechnicalIncident);
    component.selectedIncidentForLogs.set(null);
    expect(component.selectedIncidentForLogs()).toBeNull();
  });

  it('showAllContainers inicia en false', () => {
    expect(component.showAllContainers()).toBe(false);
  });
});
