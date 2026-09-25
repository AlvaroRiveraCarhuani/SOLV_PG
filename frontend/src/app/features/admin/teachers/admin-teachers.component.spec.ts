import { describe, it, expect, beforeEach, vi } from 'vitest';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { of } from 'rxjs';
import { AdminTeachersComponent } from './admin-teachers.component';
import { AdminTeachersService } from '../services/admin-teachers.service';
import { TeacherItem } from '@core/models/admin.model';

describe('AdminTeachersComponent Unit Tests', () => {
  let component: AdminTeachersComponent;
  let fixture: ComponentFixture<AdminTeachersComponent>;
  let mockTeachersService: any;

  const mockTeachers: TeacherItem[] = [
    {
      id: 't1',
      full_name: 'Roberto Gómez',
      email: 'roberto.gomez@uab.edu.bo',
      status: 'active',
      role_type: 'titular',
      origin: 'manual',
      active_courses: 2,
      invited_at: '2026-02-15'
    },
    {
      id: 't2',
      full_name: 'Laura Fernández',
      email: 'laura.fernandez@uab.edu.bo',
      status: 'active',
      role_type: 'titular',
      origin: 'manual',
      active_courses: 0,
      invited_at: '2026-03-01'
    },
    {
      id: 't3',
      full_name: 'Marcos Vargas',
      email: 'marcos.vargas@uab.edu.bo',
      status: 'pending',
      role_type: 'auxiliar',
      origin: 'gclassroom',
      active_courses: 0,
      invited_at: '2026-03-20'
    },
    {
      id: 't4',
      full_name: 'Elena Ramos',
      email: 'elena.ramos@uab.edu.bo',
      status: 'expired',
      role_type: 'titular',
      origin: 'manual',
      active_courses: 0,
      invited_at: '2026-01-10'
    }
  ];

  beforeEach(async () => {
    const teachersSignal = signal<TeacherItem[]>(mockTeachers);
    const isLoadingSignal = signal<boolean>(false);

    mockTeachersService = {
      teachers: teachersSignal,
      isLoading: isLoadingSignal,
      fetchTeachers: vi.fn(),
      getTeacherCourses: vi.fn().mockReturnValue(of([])),
      getAvailableCourses: vi.fn().mockReturnValue(of([])),
      inviteTeacher: vi.fn(),
      reassignCourse: vi.fn(),
      deleteTeacher: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [AdminTeachersComponent],
      providers: [
        { provide: AdminTeachersService, useValue: mockTeachersService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(AdminTeachersComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('debe crearse e inicializar la lista de docentes', () => {
    expect(component).toBeTruthy();
    expect(component.teachers().length).toBe(4);
    expect(mockTeachersService.fetchTeachers).toHaveBeenCalled();
  });

  it('debe calcular correctamente los KPIs de docentes', () => {
    const counts = component.counts();
    expect(counts.all).toBe(4);
    expect(counts.active).toBe(2);
    // Con materias asignadas: active (2) - noCourses (1) = 1
    expect(counts.active - counts.noCourses).toBe(1);
    // Sin materias: laura (active con active_courses: 0)
    expect(counts.noCourses).toBe(1);
    expect(counts.pending).toBe(1);
    expect(counts.expired).toBe(1);
  });

  it('debe computar las pestañas de estado con los contadores correctos', () => {
    const tabs = component.teacherStatusTabs();
    expect(tabs.length).toBe(5);
    expect(tabs[0]).toEqual({ id: 'all', label: 'Todos', count: 4 });
    expect(tabs[1]).toEqual({ id: 'active', label: 'Activos', count: 2, badgeVariant: 'active' });
    expect(tabs[2]).toEqual({ id: 'no_courses', label: 'Sin materias', count: 1, badgeVariant: 'warning' });
    expect(tabs[3]).toEqual({ id: 'pending', label: 'Pendientes (72h)', count: 1, badgeVariant: 'warning' });
    expect(tabs[4]).toEqual({ id: 'expired', label: 'Expirados', count: 1, badgeVariant: 'neutral' });
  });

  it('debe filtrar docentes al tipear en el buscador y reiniciar la página a 1', () => {
    component.currentPage.set(2);
    component.onSearchChange('fernandez');
    fixture.detectChanges();

    expect(component.searchTerm()).toBe('fernandez');
    expect(component.currentPage()).toBe(1);
    expect(component.filteredTeachersList().length).toBe(1);
    expect(component.filteredTeachersList()[0].full_name).toBe('Laura Fernández');
  });

  it('debe cambiar de filtro de estado mediante onStatusChange', () => {
    component.onStatusChange('pending');
    fixture.detectChanges();

    expect(component.statusFilter()).toBe('pending');
    expect(component.currentPage()).toBe(1);
    expect(component.filteredTeachersList().length).toBe(1);
    expect(component.filteredTeachersList()[0].id).toBe('t3');
  });

  it('debe calcular el rango de paginación from y to correctamente', () => {
    expect(component.paginationDisplay()).toEqual({ from: 1, to: 4 });

    component.onSearchChange('inexistente');
    fixture.detectChanges();
    expect(component.paginationDisplay()).toEqual({ from: 0, to: 0 });
  });

  it('debe actualizar la página actual mediante goToPage', () => {
    component.goToPage(1);
    expect(component.currentPage()).toBe(1);
  });
});
