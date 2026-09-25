import { describe, it, expect, beforeEach, vi } from 'vitest';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { of } from 'rxjs';
import { AdminCoursesComponent } from './admin-courses.component';
import { 
  AdminCoursesService, 
  AdminCourseItem, 
  AcademicPeriod, 
  DockerTemplateItem 
} from '../services/admin-courses.service';
import { TeacherItem } from '@core/models/admin.model';

describe('AdminCoursesComponent Unit Tests', () => {
  let component: AdminCoursesComponent;
  let fixture: ComponentFixture<AdminCoursesComponent>;
  let mockCoursesService: any;

  const mockPeriods: AcademicPeriod[] = [
    { id: 'p1', tenant_id: 't-1', name: 'Primer Semestre 2026', code: '2026-1', start_date: '2026-02-01', end_date: '2026-06-30', is_active: true },
    { id: 'p2', tenant_id: 't-1', name: 'Segundo Semestre 2025', code: '2025-2', start_date: '2025-08-01', end_date: '2025-12-15', is_active: false }
  ];

  const mockCourses: AdminCourseItem[] = [
    {
      id: 'c1',
      name: 'Estructuras de Datos',
      code: 'SIS-201',
      teacher_id: 'teach-1',
      teacher_name: 'Dr. Roberto Gómez',
      academic_period_id: 'p1',
      is_archived: false,
      active_students: 18,
      hibernated_students: 2,
      ram_used_mb: 2048,
      template_name: 'Debian Python'
    },
    {
      id: 'c2',
      name: 'Arquitectura de Software',
      code: 'SIS-301',
      teacher_id: undefined,
      teacher_name: 'Sin asignar',
      academic_period_id: 'p1',
      is_archived: false,
      active_students: 0,
      hibernated_students: 0,
      ram_used_mb: 0,
      template_name: 'Ubuntu Go'
    },
    {
      id: 'c3',
      name: 'Redes de Computadoras I',
      code: 'TEL-101',
      teacher_id: 'teach-2',
      teacher_name: 'Ing. Laura Fernández',
      academic_period_id: 'p1',
      is_archived: true,
      active_students: 0,
      hibernated_students: 0,
      ram_used_mb: 0,
      template_name: 'Alpine Network'
    }
  ];

  beforeEach(async () => {
    const periodsSignal = signal<AcademicPeriod[]>(mockPeriods);
    const activePeriodSignal = signal<AcademicPeriod | null>(mockPeriods[0]);
    const coursesSignal = signal<AdminCourseItem[]>(mockCourses);
    const teachersSignal = signal<TeacherItem[]>([]);
    const templatesSignal = signal<DockerTemplateItem[]>([]);
    const isLoadingSignal = signal<boolean>(false);
    const errorSignal = signal<string | null>(null);

    mockCoursesService = {
      periods: periodsSignal,
      activePeriod: activePeriodSignal,
      courses: coursesSignal,
      teachers: teachersSignal,
      templates: templatesSignal,
      isLoading: isLoadingSignal,
      error: errorSignal,
      loadAll: vi.fn(),
      setActivePeriod: vi.fn((period: AcademicPeriod | null) => activePeriodSignal.set(period)),
      createPeriod: vi.fn().mockReturnValue(of({})),
      activatePeriod: vi.fn().mockReturnValue(of({})),
      createCourse: vi.fn().mockReturnValue(of({})),
      reassignCourse: vi.fn().mockReturnValue(of({})),
      toggleArchive: vi.fn().mockReturnValue(of({}))
    };

    await TestBed.configureTestingModule({
      imports: [AdminCoursesComponent],
      providers: [
        { provide: AdminCoursesService, useValue: mockCoursesService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(AdminCoursesComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('debe crearse e invocar la carga de datos del servicio', () => {
    expect(component).toBeTruthy();
    expect(mockCoursesService.loadAll).toHaveBeenCalled();
  });

  it('debe calcular correctamente los KPIs del periodo', () => {
    const kpis = component.periodKpis();
    expect(kpis.totalCourses).toBe(3);
    expect(kpis.unassignedCourses).toBe(1); // c2
    expect(kpis.archivedCourses).toBe(1); // c3
  });

  it('debe generar las pestañas de estado con los contadores correctos', () => {
    const tabs = component.courseStatusTabs();
    expect(tabs.length).toBe(4);
    expect(tabs[0]).toEqual({ id: 'all', label: 'Todos', count: 3 });
    expect(tabs[1]).toEqual({ id: 'active', label: 'Activos', count: 2, badgeVariant: 'active' });
    expect(tabs[2]).toEqual({ id: 'unassigned', label: 'Sin Docente', count: 1, badgeVariant: 'warning' });
    expect(tabs[3]).toEqual({ id: 'archived', label: 'Archivados', count: 1, badgeVariant: 'neutral' });
  });

  it('debe filtrar cursos por término de búsqueda y reiniciar la página', () => {
    component.currentPage.set(2);
    component.onSearchChange('estructuras');
    fixture.detectChanges();

    expect(component.searchQuery()).toBe('estructuras');
    expect(component.currentPage()).toBe(1);
    expect(component.filteredCourses().length).toBe(1);
    expect(component.filteredCourses()[0].code).toBe('SIS-201');
  });

  it('debe filtrar cursos por estado mediante onStatusFilterChange', () => {
    component.onStatusFilterChange('unassigned');
    fixture.detectChanges();

    expect(component.statusFilter()).toBe('unassigned');
    expect(component.filteredCourses().length).toBe(1);
    expect(component.filteredCourses()[0].id).toBe('c2');

    component.onStatusFilterChange('archived');
    fixture.detectChanges();
    expect(component.filteredCourses().length).toBe(1);
    expect(component.filteredCourses()[0].id).toBe('c3');
  });

  it('debe calcular los límites de paginación from, to y total', () => {
    const display = component.paginationDisplay();
    expect(display).toEqual({ from: 1, to: 3, total: 3 });

    component.onSearchChange('sin-coincidencia-xyz');
    fixture.detectChanges();
    expect(component.paginationDisplay()).toEqual({ from: 0, to: 0, total: 0 });
  });

  it('debe actualizar la página actual mediante goToPage', () => {
    component.goToPage(1);
    expect(component.currentPage()).toBe(1);
  });
});
