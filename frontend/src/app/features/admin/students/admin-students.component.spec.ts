import { describe, it, expect, beforeEach, vi } from 'vitest';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { AdminStudentsComponent } from './admin-students.component';
import { AdminStudentsService, AdminStudentItem, AcademicPeriodOption, SubjectOption } from '../services/admin-students.service';

interface MockAdminStudentsService {
  getAcademicPeriods: ReturnType<typeof vi.fn>;
  getSubjects: ReturnType<typeof vi.fn>;
  getStudents: ReturnType<typeof vi.fn>;
  getStudentCourses: ReturnType<typeof vi.fn>;
  createStudent: ReturnType<typeof vi.fn>;
  resetOOMStrikes: ReturnType<typeof vi.fn>;
  toggleStudentStatus: ReturnType<typeof vi.fn>;
}

describe('AdminStudentsComponent Unit Tests', () => {
  let component: AdminStudentsComponent;
  let fixture: ComponentFixture<AdminStudentsComponent>;
  let mockStudentsService: MockAdminStudentsService;

  const mockPeriods: AcademicPeriodOption[] = [
    { id: 'p1', code: '2026-1', name: 'Primer Semestre 2026', is_active: true },
    { id: 'p2', code: '2025-2', name: 'Segundo Semestre 2025', is_active: false }
  ];

  const mockSubjects: SubjectOption[] = [
    { id: 's1', code: 'SIS-101', name: 'Programación I' },
    { id: 's2', code: 'SIS-102', name: 'Sistemas Operativos' }
  ];

  const mockStudents: AdminStudentItem[] = [
    {
      id: 'std-1',
      first_name: 'Ana',
      last_name: 'Gómez',
      email: 'ana.gomez@uab.edu.bo',
      role: 'student',
      status: 'active',
      academic_status: 'enrolled',
      enrolled_courses_count: 2,
      active_environments_count: 1,
      oom_strike_count: 0
    },
    {
      id: 'std-2',
      first_name: 'Carlos',
      last_name: 'Pérez',
      email: 'carlos.perez@uab.edu.bo',
      role: 'student',
      status: 'active',
      academic_status: 'enrolled',
      enrolled_courses_count: 1,
      active_environments_count: 0,
      oom_strike_count: 2
    },
    {
      id: 'std-3',
      first_name: 'Beatriz',
      last_name: 'López',
      email: 'beatriz.lopez@uab.edu.bo',
      role: 'student',
      status: 'suspended',
      academic_status: 'inactive',
      enrolled_courses_count: 0,
      active_environments_count: 0,
      oom_strike_count: 3
    },
    {
      id: 'std-4',
      first_name: 'David',
      last_name: 'Roca',
      email: 'david.roca@uab.edu.bo',
      role: 'student',
      status: 'active',
      academic_status: 'enrolled',
      enrolled_courses_count: 3,
      active_environments_count: 2,
      oom_strike_count: 0
    }
  ];

  beforeEach(async () => {
    mockStudentsService = {
      getAcademicPeriods: vi.fn().mockReturnValue(of(mockPeriods)),
      getSubjects: vi.fn().mockReturnValue(of(mockSubjects)),
      getStudents: vi.fn().mockReturnValue(of(mockStudents)),
      getStudentCourses: vi.fn().mockReturnValue(of([])),
      createStudent: vi.fn(),
      resetOOMStrikes: vi.fn(),
      toggleStudentStatus: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [AdminStudentsComponent],
      providers: [
        { provide: AdminStudentsService, useValue: mockStudentsService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(AdminStudentsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('debe crearse e inicializar periodos y estudiantes', () => {
    expect(component).toBeTruthy();
    expect(component.students().length).toBe(4);
    expect(component.academicPeriods().length).toBe(2);
    expect(component.selectedPeriod()).toBe('p1');
  });

  it('debe calcular correctamente los KPIs operativos', () => {
    expect(component.totalStudents()).toBe(4);
    // enrolledStudentsCount: std-1, std-2, std-4 (3 estudiantes con enrolled_courses_count > 0)
    expect(component.enrolledStudentsCount()).toBe(3);
    // activeEnvironmentsTotal: 1 (std-1) + 0 (std-2) + 0 (std-3) + 2 (std-4) = 3
    expect(component.activeEnvironmentsTotal()).toBe(3);
    // strikesStudentsCount: std-2 (2 strikes, < 3)
    expect(component.strikesStudentsCount()).toBe(1);
    // blockedStudentsCount: std-3 (>= 3 strikes)
    expect(component.blockedStudentsCount()).toBe(1);
  });

  it('debe aplicar y limpiar el filtro de estado desde la línea de resumen', () => {
    component.currentPage.set(2);
    component.applyStatusFilter('strikes');
    expect(component.selectedStatus()).toBe('strikes');
    expect(component.currentPage()).toBe(1);

    component.applyStatusFilter('strikes');
    expect(component.selectedStatus()).toBe('all');
  });

  it('debe filtrar estudiantes al escribir en el buscador y reiniciar la página', () => {
    component.currentPage.set(2);
    component.onSearchChange('gomez');
    fixture.detectChanges();

    expect(component.searchTerm()).toBe('gomez');
    expect(component.currentPage()).toBe(1);
    expect(component.filteredStudents().length).toBe(1);
    expect(component.filteredStudents()[0].first_name).toBe('Ana');
  });

  it('debe filtrar por estado (running, strikes, blocked)', () => {
    component.onStatusSelected({ id: 'running', label: 'Entornos activos', value: 'running' });
    fixture.detectChanges();
    expect(component.filteredStudents().length).toBe(2); // std-1 y std-4

    component.onStatusSelected({ id: 'strikes', label: 'Con strikes', value: 'strikes' });
    fixture.detectChanges();
    expect(component.filteredStudents().length).toBe(2); // std-2 (2) y std-3 (3)

    component.onStatusSelected({ id: 'blocked', label: 'Bloqueados OOM', value: 'blocked' });
    fixture.detectChanges();
    expect(component.filteredStudents().length).toBe(1); // std-3
  });

  it('debe calcular los límites de paginación from y to', () => {
    expect(component.paginationFrom()).toBe(1);
    expect(component.paginationTo()).toBe(4);

    component.onSearchChange('inexistente');
    fixture.detectChanges();
    expect(component.paginationFrom()).toBe(0);
    expect(component.paginationTo()).toBe(0);
  });

  it('debe actualizar la página actual mediante goToPage', () => {
    component.goToPage(1);
    expect(component.currentPage()).toBe(1);
  });

  it('debe ordenar estudiantes al invocar onSortChange', () => {
    component.onSortChange({ column: 'student', direction: 'asc' });
    fixture.detectChanges();
    expect(component.filteredStudents()[0].first_name).toBe('Ana');
    expect(component.filteredStudents()[3].first_name).toBe('David');

    component.onSortChange({ column: 'student', direction: 'desc' });
    fixture.detectChanges();
    expect(component.filteredStudents()[0].first_name).toBe('David');
    expect(component.filteredStudents()[3].first_name).toBe('Ana');

    component.onSortChange({ column: 'strikes', direction: 'desc' });
    fixture.detectChanges();
    expect(component.filteredStudents()[0].first_name).toBe('Beatriz');
  });

  it('debe configurar pestañas de estado con badgeVariant neutral', () => {
    const tabs = component.statusTabs();
    expect(tabs.length).toBe(6);
    expect(tabs.every(t => t.badgeVariant === 'neutral')).toBe(true);
    expect(tabs.map(t => t.id)).toEqual(['all', 'active', 'running', 'ram_alert', 'blocked', 'suspended']);
  });

  it('debe filtrar por pestaña de alerta RAM y estado suspendido', () => {
    component.onStatusTabChange('ram_alert');
    fixture.detectChanges();
    // std-2 tiene 2 strikes (en riesgo/alerta RAM)
    expect(component.filteredStudents().length).toBe(1);
    expect(component.filteredStudents()[0].id).toBe('std-2');

    component.onStatusTabChange('suspended');
    fixture.detectChanges();
    // std-3 está suspendido
    expect(component.filteredStudents().length).toBe(1);
    expect(component.filteredStudents()[0].id).toBe('std-3');
  });

  it('debe abrir y cerrar el drawer de detalle de estudiante', () => {
    component.openStudentDrawer(mockStudents[0]);
    expect(component.selectedStudentForDrawer()).toEqual(mockStudents[0]);
    expect(mockStudentsService.getStudentCourses).toHaveBeenCalledWith('std-1');

    component.closeDrawer();
    expect(component.selectedStudentForDrawer()).toBeNull();
  });

  it('debe limpiar filtros y restablecer a all con clearAllFilters', () => {
    component.selectedPeriod.set('p2');
    component.selectedSubject.set('s1');
    component.clearAllFilters();

    expect(component.selectedPeriod()).toBe('all');
    expect(component.selectedSubject()).toBe('all');
    expect(component.currentPage()).toBe(1);
  });

  it('debe calcular el badge de periodo activo en la cabecera', () => {
    component.selectedPeriod.set('p1');
    expect(component.activePeriodBadge()).toContain('2026-1 · Vigente');
  });
});
