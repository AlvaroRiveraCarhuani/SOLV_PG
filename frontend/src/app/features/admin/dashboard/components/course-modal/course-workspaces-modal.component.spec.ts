import '@angular/compiler';
import '@angular/localize/init';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { CourseWorkspacesModalComponent } from './course-workspaces-modal.component';
import { AdminMetricsService } from '../../../services/admin-metrics.service';
import { CourseLoadSummary, DockerContainerSummary } from '@core/models/admin.model';

describe('CourseWorkspacesModalComponent', () => {
  let component: CourseWorkspacesModalComponent;
  let mockMetricsService: any;

  const mockCourse: CourseLoadSummary = {
    id: 'course-123',
    course_name: 'Algoritmos Avanzados',
    teacher_name: 'Profesor Revisor',
    active_students: 2,
    hibernated_students: 1,
    ram_used_mb: 512
  };

  const mockWorkspaces: DockerContainerSummary[] = [
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
      getCourseWorkspaces: vi.fn().mockReturnValue(of(mockWorkspaces))
    };

    await TestBed.configureTestingModule({
      imports: [CourseWorkspacesModalComponent],
      providers: [
        { provide: AdminMetricsService, useValue: mockMetricsService }
      ]
    }).compileComponents();

    const fixture = TestBed.createComponent(CourseWorkspacesModalComponent);
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
    const percent = component.getMemoryPercent(mockWorkspaces[0]);
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
});
