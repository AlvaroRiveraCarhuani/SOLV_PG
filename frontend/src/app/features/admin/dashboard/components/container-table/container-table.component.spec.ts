import '@angular/compiler';
import '@angular/localize/init';
import { describe, it, expect, beforeEach } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { ContainerTableComponent } from './container-table.component';
import { DockerContainerSummary } from '@core/models/admin.model';

describe('ContainerTableComponent', () => {
  let component: ContainerTableComponent;

  const mockContainers: DockerContainerSummary[] = [
    {
      id: 'c-1',
      student_name: 'Zoe Morales',
      student_email: 'zoe@uab.edu.bo',
      course_name: 'Base de Datos II',
      image_tag: 'postgres-lab:1.0',
      memory_used_mb: 200,
      memory_limit_mb: 512,
      ttl_remaining_seconds: 1200,
      status: 'running',
      started_at: '2026-10-17T10:00:00Z'
    },
    {
      id: 'c-2',
      student_name: 'Ana Pérez',
      student_email: 'ana@uab.edu.bo',
      course_name: 'Sistemas Operativos',
      image_tag: 'linux-kernel:2.0',
      memory_used_mb: 150,
      memory_limit_mb: 512,
      ttl_remaining_seconds: 300,
      status: 'running',
      started_at: '2026-10-17T11:00:00Z'
    },
    {
      id: 'c-3',
      student_name: 'Bernardo Cruz',
      student_email: 'bernardo@uab.edu.bo',
      course_name: 'Base de Datos II',
      image_tag: 'postgres-lab:1.0',
      memory_used_mb: 180,
      memory_limit_mb: 512,
      ttl_remaining_seconds: 0,
      status: 'hibernated',
      started_at: '2026-10-17T09:00:00Z'
    }
  ];

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ContainerTableComponent]
    }).compileComponents();

    const fixture = TestBed.createComponent(ContainerTableComponent);
    fixture.componentRef.setInput('containers', mockContainers);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('debe crearse correctamente e inicializar opciones de cursos', () => {
    expect(component).toBeTruthy();
    expect(component.courseFilterOptions().length).toBe(3); // 'Todos los cursos' + 2 cursos únicos
    expect(component.selectedCourseLabel()).toBe('Todos los cursos');
  });

  it('ordena jerárquicamente por curso ASC y luego estudiante ASC', () => {
    const list = component.filteredContainers();
    expect(list.length).toBe(3);
    // Base de Datos II viene antes que Sistemas Operativos
    expect(list[0].course_name).toBe('Base de Datos II');
    expect(list[0].student_name).toBe('Bernardo Cruz');
    expect(list[1].course_name).toBe('Base de Datos II');
    expect(list[1].student_name).toBe('Zoe Morales');
    expect(list[2].course_name).toBe('Sistemas Operativos');
    expect(list[2].student_name).toBe('Ana Pérez');
  });

  it('filtra por término de búsqueda en varios campos', () => {
    component.onSearchChange('linux');
    expect(component.filteredContainers().length).toBe(1);
    expect(component.filteredContainers()[0].student_name).toBe('Ana Pérez');

    component.onSearchChange('zoe@uab');
    expect(component.filteredContainers().length).toBe(1);
    expect(component.filteredContainers()[0].student_name).toBe('Zoe Morales');
  });

  it('filtra por curso seleccionado mediante combobox', () => {
    component.onCourseFilterSelected({ id: 'Sistemas Operativos', label: 'Sistemas Operativos', value: 'Sistemas Operativos' });
    expect(component.filteredContainers().length).toBe(1);
    expect(component.filteredContainers()[0].course_name).toBe('Sistemas Operativos');
  });

  it('formatea TTL en minutos y segundos o expirado', () => {
    expect(component.formatTTL(0)).toBe('Expirado');
    expect(component.formatTTL(-10)).toBe('Expirado');
    expect(component.formatTTL(65)).toBe('1m 05s');
    expect(component.formatTTL(3600)).toBe('60m 00s');
  });
});
