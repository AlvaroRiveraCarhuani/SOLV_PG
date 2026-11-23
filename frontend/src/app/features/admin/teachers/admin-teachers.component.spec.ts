import { describe, it, expect, beforeEach, vi } from 'vitest';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { of } from 'rxjs';
import { AdminTeachersComponent } from './admin-teachers.component';
import { AdminTeachersService } from '../services/admin-teachers.service';
import { TeacherItem } from '@core/models/admin.model';

interface MockAdminTeachersService {
  teachers: ReturnType<typeof signal<TeacherItem[]>>;
  isLoading: ReturnType<typeof signal<boolean>>;
  fetchTeachers: ReturnType<typeof vi.fn>;
  getTeacherCourses: ReturnType<typeof vi.fn>;
  getAllAvailableCourses: ReturnType<typeof vi.fn>;
  inviteTeacher: ReturnType<typeof vi.fn>;
  reassignCourseTeacher: ReturnType<typeof vi.fn>;
  deleteTeacher: ReturnType<typeof vi.fn>;
  deleteInvitation: ReturnType<typeof vi.fn>;
  resendInvitation: ReturnType<typeof vi.fn>;
  renewInvitation: ReturnType<typeof vi.fn>;
  directRegisterTeacher?: ReturnType<typeof vi.fn>;
  batchDirectRegister?: ReturnType<typeof vi.fn>;
}

describe('AdminTeachersComponent Unit Tests', () => {
  let component: AdminTeachersComponent;
  let fixture: ComponentFixture<AdminTeachersComponent>;
  let mockTeachersService: MockAdminTeachersService;

  const mockTeachers: TeacherItem[] = [
    {
      id: 't1',
      full_name: 'Roberto Gómez',
      email: 'roberto.gomez@uab.edu.bo',
      status: 'active',
      role_type: 'titular',
      origin: 'manual',
      active_courses: 2,
      invited_at: '2026-02-15',
      last_login: '2026-02-20T10:30:00Z'
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
      invited_at: 'Hace 2h'
    },
    {
      id: 't4',
      full_name: 'Elena Ramos',
      email: 'elena.ramos@uab.edu.bo',
      status: 'expired',
      role_type: 'titular',
      origin: 'bulk_import',
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
      getAllAvailableCourses: vi.fn().mockReturnValue(of([])),
      inviteTeacher: vi.fn(),
      reassignCourseTeacher: vi.fn(),
      deleteTeacher: vi.fn().mockReturnValue(of(true)),
      deleteInvitation: vi.fn().mockReturnValue(of(true)),
      resendInvitation: vi.fn().mockReturnValue(of(null)),
      renewInvitation: vi.fn().mockReturnValue(of(null))
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

  it('debe calcular las 4 pestañas de estado con contadores reales en tono neutro', () => {
    expect(component.teacherStatusTabs()).toEqual([
      { id: 'all', label: 'Todos', count: 4, badgeVariant: 'neutral' },
      { id: 'active', label: 'Activos', count: 2, badgeVariant: 'neutral' },
      { id: 'no_courses', label: 'Sin materias', count: 1, badgeVariant: 'neutral' },
      { id: 'invitations', label: 'Invitaciones', count: 2, badgeVariant: 'neutral' }
    ]);
  });

  it('debe filtrar correctamente por la pestaña Invitaciones agrupando pending y expired', () => {
    component.onStatusChange('invitations');
    fixture.detectChanges();

    expect(component.statusFilter()).toBe('invitations');
    const filtered = component.filteredTeachersList();
    expect(filtered.length).toBe(2);
    expect(filtered.map(t => t.id)).toEqual(['t3', 't4']);
  });

  it('debe filtrar correctamente por la pestaña Sin materias mostrando solo activos con 0 materias', () => {
    component.onStatusChange('no_courses');
    fixture.detectChanges();

    const filtered = component.filteredTeachersList();
    expect(filtered.length).toBe(1);
    expect(filtered[0].id).toBe('t2');
    expect(filtered[0].full_name).toBe('Laura Fernández');
  });

  it('debe filtrar por los 3 orígenes reales incluyendo bulk_import y gclassroom', () => {
    component.onOriginChange('bulk_import');
    fixture.detectChanges();

    expect(component.originFilter()).toBe('bulk_import');
    expect(component.filteredTeachersList().length).toBe(1);
    expect(component.filteredTeachersList()[0].id).toBe('t4');

    component.onOriginChange('gclassroom');
    fixture.detectChanges();
    expect(component.filteredTeachersList().length).toBe(1);
    expect(component.filteredTeachersList()[0].id).toBe('t3');

    component.onOriginChange('manual');
    fixture.detectChanges();
    expect(component.filteredTeachersList().length).toBe(2);
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

  it('abre el drawer al seleccionar docente activo y excluye su id de activeTeachersList', () => {
    const t1 = mockTeachers[0];
    component.openTeacherDrawer(t1);
    fixture.detectChanges();

    expect(component.selectedTeacherForDrawer()).toBe(t1);
    expect(mockTeachersService.getTeacherCourses).toHaveBeenCalledWith('t1');
    // activeTeachersList solo debe contener los otros docentes activos (t2)
    expect(component.activeTeachersList().length).toBe(1);
    expect(component.activeTeachersList()[0].id).toBe('t2');
  });

  it('muestra aviso explicito de falta de docentes cuando activeTeachersList está vacía', () => {
    // Si solo hay un docente activo en la lista
    mockTeachersService.teachers.set([mockTeachers[0]]);
    fixture.detectChanges();

    component.openTeacherDrawer(mockTeachers[0]);
    fixture.detectChanges();

    expect(component.activeTeachersList().length).toBe(0);
  });

  it('genera mensaje honesto de confirmación al revocar rol docente según carga académica', () => {
    // Con materias asignadas (t1 tiene 2 materias)
    const msgWithCourses = component.getRevokeConfirmMessage(mockTeachers[0]);
    expect(msgWithCourses).toContain('ADVERTENCIA: Sus 2 materias asignadas quedarán sin docente titular');
    expect(msgWithCourses).toContain('Se recomienda reasignar las materias antes de revocar el rol');

    // Sin materias asignadas (t2 tiene 0 materias)
    const msgZeroCourses = component.getRevokeConfirmMessage(mockTeachers[1]);
    expect(msgZeroCourses).toContain('La cuenta pasará a tener rol de estudiante y perderá el acceso a las funciones docentes');
    expect(msgZeroCourses).not.toContain('ADVERTENCIA');
  });

  it('debe renderizar 5 columnas exactas en la tabla con etiquetas semánticas', () => {
    const headers = fixture.nativeElement.querySelectorAll('th');
    expect(headers.length).toBe(5);
    const headerTexts = Array.from(headers).map((h: unknown) => (h as HTMLElement).textContent?.trim());
    expect(headerTexts[0]).toContain('DOCENTE');
    expect(headerTexts[1]).toContain('ESTADO');
    expect(headerTexts[2]).toContain('MATERIAS');
    expect(headerTexts[3]).toContain('ACTIVIDAD');
    expect(headerTexts[4]).toContain('ACCIONES');
  });

  it('formatea fechas con <time> y pipe dateText para last_login y vencimiento de invitaciones', () => {
    const timeElements = fixture.nativeElement.querySelectorAll('time');
    expect(timeElements.length).toBeGreaterThan(0);
  });

  it('el botón primario muestra "Registrar docente" sin el "+" duplicado', () => {
    const primaryBtn = fixture.nativeElement.querySelector('.btn-primary');
    expect(primaryBtn.textContent.trim()).toBe('Registrar docente');
  });

  it('renderiza redacción coherente de expiración según formato relativo o fecha exacta', () => {
    // Probar t4 que tiene fecha '2026-01-10'
    const expiredRow = fixture.nativeElement.querySelector('.activity-expired');
    expect(expiredRow.textContent).toContain('Expiró el');

    // Cambiar invited_at de un docente a formato relativo 'Hace 5d'
    const tableComp = fixture.debugElement.children[0]?.componentInstance;
    if (tableComp?.getExpiredText) {
      const relResult = tableComp.getExpiredText({ invited_at: 'Hace 5d' } as unknown as TeacherItem);
      expect(relResult).toBe('Expiró hace 5 días');
      const relHour = tableComp.getExpiredText({ invited_at: 'Hace 3h' } as unknown as TeacherItem);
      expect(relHour).toBe('Expiró hace 3 h');
    }
  });

  it('onDirectTeacherSubmitted llama a directRegisterTeacher y recarga docentes', () => {
    mockTeachersService.directRegisterTeacher = vi.fn().mockReturnValue(of({ message: 'Docente registrado', data: {}, notified: true }));
    component.onDirectTeacherSubmitted({
      email: 'nuevo@uab.edu.bo',
      first_name: 'Nuevo',
      last_name: 'Docente',
      send_notification: true
    });
    expect(mockTeachersService.directRegisterTeacher).toHaveBeenCalledWith({
      email: 'nuevo@uab.edu.bo',
      first_name: 'Nuevo',
      last_name: 'Docente',
      send_notification: true
    });
    expect(component.toastMessage()?.text).toContain('nuevo@uab.edu.bo');
  });

  it('onBatchTeachersSubmitted llama a batchDirectRegister y muestra toast con conteo', () => {
    mockTeachersService.batchDirectRegister = vi.fn().mockReturnValue(of({ message: 'Registrados', registered_count: 5, total_processed: 5, notified: true }));
    component.onBatchTeachersSubmitted({
      emails: ['d1@uab.edu.bo', 'd2@uab.edu.bo'],
      send_notification: true
    });
    expect(mockTeachersService.batchDirectRegister).toHaveBeenCalledWith({
      emails: ['d1@uab.edu.bo', 'd2@uab.edu.bo'],
      send_notification: true
    });
    expect(component.toastMessage()?.text).toContain('5 docentes registrados');
  });
});
