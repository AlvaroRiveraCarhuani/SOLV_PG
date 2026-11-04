import '@angular/compiler';
import '@angular/localize/init';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { CourseEnvironmentsModalComponent } from './course-environments-modal.component';
import { AdminMetricsService } from '../../../services/admin-metrics.service';
import { CourseLoadSummary, DockerContainerSummary } from '@core/models/admin.model';

interface MockAdminMetricsService {
  getCourseEnvironments: ReturnType<typeof vi.fn>;
}

describe('CourseEnvironmentsModalComponent', () => {
  let component: CourseEnvironmentsModalComponent;
  let fixture: ComponentFixture<CourseEnvironmentsModalComponent>;
  let mockMetricsService: MockAdminMetricsService;

  const mockCourse: CourseLoadSummary = {
    id: 'course-123',
    course_name: 'Algoritmos Avanzados',
    teacher_name: 'Profesor Revisor',
    active_students: 2,
    hibernated_students: 1,
    ram_used_mb: 512
  };

  const mockEnvironments: DockerContainerSummary[] = [
    {
      id: 'ws-abc-123',
      student_name: 'Carlos Mendoza',
      student_email: 'carlos@uab.edu.bo',
      course_name: 'Algoritmos Avanzados',
      image_tag: 'solv-cpp:v1',
      memory_used_mb: 256,
      memory_limit_mb: 512,
      ttl_remaining_seconds: 3600,
      status: 'running',
      started_at: '2026-10-17T12:00:00Z'
    }
  ];

  beforeEach(async () => {
    mockMetricsService = {
      getCourseEnvironments: vi.fn().mockReturnValue(of(mockEnvironments))
    };

    await TestBed.configureTestingModule({
      imports: [CourseEnvironmentsModalComponent],
      providers: [
        { provide: AdminMetricsService, useValue: mockMetricsService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(CourseEnvironmentsModalComponent);
    fixture.componentRef.setInput('course', mockCourse);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('debe crearse correctamente e inicializar las opciones de estado', () => {
    expect(component).toBeTruthy();
    expect(component.statusOptions.length).toBe(4);
    expect(component.selectedStatusLabel()).toBe('Todos los estados');
  });

  it('calcula correctamente el porcentaje de memoria', () => {
    const percent = component.getMemoryPercent(mockEnvironments[0]);
    expect(percent).toBe(50);
  });

  it('obtiene las iniciales del estudiante adecuadamente', () => {
    expect(component.getStudentInitials('Carlos Mendoza')).toBe('CM');
    expect(component.getStudentInitials('Roberto')).toBe('RO');
    expect(component.getStudentInitials('')).toBe('ES');
  });

  it('permite filtrar por estado con onStatusSelected', () => {
    component.onStatusSelected({ id: 'running', label: 'En ejecución', value: 'running' });
    expect(component.statusFilter()).toBe('running');
    expect(component.selectedStatusLabel()).toBe('En ejecución');
  });

  it('hasRealMetrics respeta metrics_available del backend', () => {
    // Sin flag y con consumo > 0 (fuente antigua): se muestra la barra.
    expect(component.hasRealMetrics(mockEnvironments[0])).toBe(true);
    // El backend marca sin telemetría: no se dibuja barra inventada.
    expect(
      component.hasRealMetrics({ ...mockEnvironments[0], metrics_available: false, memory_used_mb: 0 })
    ).toBe(false);
  });

  it('renderiza «Sin métricas» en la fila sin telemetría real', () => {
    component.environments.set([
      { ...mockEnvironments[0], metrics_available: false, memory_used_mb: 0 }
    ]);
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    expect(el.textContent).toContain('Sin métricas');
    expect(el.querySelector('.mem-gauge')).toBeNull();
  });

  it('muestra la barra de memoria cuando hay métricas reales', () => {
    component.environments.set([
      { ...mockEnvironments[0], metrics_available: true, memory_used_mb: 85, memory_limit_mb: 256 }
    ]);
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    expect(el.querySelector('.mem-gauge')).toBeTruthy();
    expect(el.textContent).not.toContain('Sin métricas');
  });

  it('recarga la lista cuando el dashboard incrementa reloadToken', () => {
    const callsBefore = mockMetricsService.getCourseEnvironments.mock.calls.length;

    fixture.componentRef.setInput('reloadToken', 1);
    TestBed.flushEffects();

    expect(mockMetricsService.getCourseEnvironments.mock.calls.length).toBe(callsBefore + 1);
  });
});
