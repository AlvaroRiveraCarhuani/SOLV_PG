import '@angular/compiler';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { of, throwError } from 'rxjs';
import { CourseReassignModalComponent } from './course-reassign-modal.component';
import { AdminCoursesService, AdminCourseItem } from '../../../services/admin-courses.service';
import { TeacherItem } from '@core/models/admin.model';

const mockCourse: AdminCourseItem = {
  id: 'course-1',
  code: 'SIS-123-A',
  name: 'Entorno de prueba',
  teacher_id: 'doc-1',
  teacher_name: 'Alvaro Rivera',
  academic_period_id: 'p1',
  is_archived: false,
  active_students: 10,
  hibernated_students: 2,
  ram_used_mb: 256
};

const mockTeachers: TeacherItem[] = [
  { id: 'doc-1', full_name: 'Alvaro Rivera', email: 'alvaro@uab.edu.bo', status: 'active', role_type: 'titular', origin: 'manual', active_courses: 2, invited_at: '2026-01-01' },
  { id: 'doc-2', full_name: 'Carlos Mendoza Ríos', email: 'carlos@uab.edu.bo', status: 'active', role_type: 'titular', origin: 'manual', active_courses: 1, invited_at: '2026-01-01' },
  { id: 'doc-3', full_name: 'Docente E2E', email: 'e2e@uab.edu.bo', status: 'active', role_type: 'titular', origin: 'manual', active_courses: 3, invited_at: '2026-01-01' }
];

interface MockAdminCoursesService {
  reassignCourseTeacher: ReturnType<typeof vi.fn>;
}

describe('CourseReassignModalComponent', () => {
  let component: CourseReassignModalComponent;
  let mockCoursesService: MockAdminCoursesService;

  beforeEach(async () => {
    mockCoursesService = {
      reassignCourseTeacher: vi.fn().mockReturnValue(of({}))
    };

    await TestBed.configureTestingModule({
      imports: [CourseReassignModalComponent],
      providers: [
        { provide: AdminCoursesService, useValue: mockCoursesService }
      ]
    }).compileComponents();

    const fixture = TestBed.createComponent(CourseReassignModalComponent);
    component = fixture.componentInstance;
    component.course = mockCourse;
    component.teachers = mockTeachers;
    component.ngOnInit();
    fixture.detectChanges();
  });

  it('debe crearse correctamente', () => {
    expect(component).toBeTruthy();
  });

  it('ngOnInit NO debe preseleccionar ningún docente automáticamente', () => {
    expect(component.selectedTeacherId()).toBe('');
    expect(component.selectedTeacherLabel()).toBe('');
  });

  it('teacherComboboxOptions debe excluir al docente actual de la materia', () => {
    const opts = component.teacherComboboxOptions();
    expect(opts.length).toBe(2);
    expect(opts.some(o => o.id === 'doc-1')).toBe(false);
    expect(opts.map(o => o.id)).toEqual(['doc-2', 'doc-3']);
  });

  it('submitReassign sin docente seleccionado establece formError y no llama al servicio', () => {
    component.reason.set('Cambio de cátedra');
    component.submitReassign();
    expect(component.formError()).toBe('Debe seleccionar un docente para la materia.');
    expect(mockCoursesService.reassignCourseTeacher).not.toHaveBeenCalled();
  });

  it('submitReassign con el docente actual establece formError y no llama al servicio', () => {
    component.selectedTeacherId.set('doc-1');
    component.reason.set('Cambio de cátedra');
    component.submitReassign();
    expect(component.formError()).toBe('Debe seleccionar un docente diferente al titular actual.');
    expect(mockCoursesService.reassignCourseTeacher).not.toHaveBeenCalled();
  });

  it('submitReassign sin motivo establece formError y no llama al servicio', () => {
    component.selectedTeacherId.set('doc-2');
    component.reason.set('   ');
    component.submitReassign();
    expect(component.formError()).toBe('Debe ingresar un motivo para registrar en la auditoría institucional.');
    expect(mockCoursesService.reassignCourseTeacher).not.toHaveBeenCalled();
  });

  it('submitReassign con datos válidos llama al servicio y emite reassignCompleted', () => {
    let completed = false;
    component.reassignCompleted.subscribe(() => { completed = true; });

    component.selectedTeacherId.set('doc-2');
    component.reason.set('Reorganización departamental por licencia');
    component.submitReassign();

    expect(mockCoursesService.reassignCourseTeacher).toHaveBeenCalledWith('course-1', {
      new_teacher_id: 'doc-2',
      reason: 'Reorganización departamental por licencia'
    });
    expect(completed).toBe(true);
    expect(component.isSubmitting()).toBe(false);
  });

  it('ante error HTTP al reasignar propaga el mensaje y limpia isSubmitting', () => {
    mockCoursesService.reassignCourseTeacher.mockReturnValue(
      throwError(() => ({ error: { error: 'Permiso denegado' } }))
    );

    component.selectedTeacherId.set('doc-2');
    component.reason.set('Motivo válido');
    component.submitReassign();

    expect(component.formError()).toBe('Permiso denegado');
    expect(component.isSubmitting()).toBe(false);
  });

  it('onTeacherSelected asigna el id y limpia formError', () => {
    component.formError.set('Error previo');
    component.onTeacherSelected({ id: 'doc-3', label: 'Docente E2E', value: 'doc-3' });
    expect(component.selectedTeacherId()).toBe('doc-3');
    expect(component.formError()).toBeNull();
  });

  it('onTeacherCleared limpia la selección del docente', () => {
    component.selectedTeacherId.set('doc-3');
    component.onTeacherCleared();
    expect(component.selectedTeacherId()).toBe('');
  });
});
