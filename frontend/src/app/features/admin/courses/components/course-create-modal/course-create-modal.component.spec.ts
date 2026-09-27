import '@angular/compiler';
import '@angular/localize/init';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { of, throwError } from 'rxjs';
import { CourseCreateModalComponent } from './course-create-modal.component';
import { AdminCoursesService, AcademicPeriod, DockerTemplateItem } from '../../../services/admin-courses.service';
import { TeacherItem } from '@core/models/admin.model';

const mockPeriods: AcademicPeriod[] = [
  { id: 'p1', tenant_id: 't1', name: 'Gestión I/2026', code: 'G1-26', is_active: false, start_date: '2026-02-01', end_date: '2026-06-30' },
  { id: 'p2', tenant_id: 't1', name: 'Gestión II/2026', code: 'G2-26', is_active: true, start_date: '2026-07-01', end_date: '2026-12-15' }
];

const mockTeachers: TeacherItem[] = [
  { id: 't1', full_name: 'Ana Gutierrez', email: 'ana@uab.edu.bo', status: 'active', role_type: 'titular', origin: 'manual', active_courses: 1, invited_at: '2026-01-01' }
];

const mockTemplates: DockerTemplateItem[] = [
  { id: 'tpl1', name: 'Python 3.12', display_name: 'Python 3.12 Slim', image: 'python:3.12-slim', base_ram_mb: 512 }
];

describe('CourseCreateModalComponent', () => {
  let component: CourseCreateModalComponent;
  let mockCoursesService: any;

  beforeEach(async () => {
    mockCoursesService = {
      createCourse: vi.fn().mockReturnValue(of({}))
    };

    await TestBed.configureTestingModule({
      imports: [CourseCreateModalComponent],
      providers: [
        { provide: AdminCoursesService, useValue: mockCoursesService }
      ]
    }).compileComponents();

    const fixture = TestBed.createComponent(CourseCreateModalComponent);
    component = fixture.componentInstance;
    component.periods = mockPeriods;
    component.teachers = mockTeachers;
    component.templates = mockTemplates;
    component.ngOnInit();
    fixture.detectChanges();
  });

  it('debe crearse correctamente', () => {
    expect(component).toBeTruthy();
  });

  it('ngOnInit pre-selecciona el periodo activo cuando no hay defaultPeriodId', () => {
    expect(component.selectedPeriodId()).toBe('p2');
  });

  it('ngOnInit pre-selecciona el primer template disponible', () => {
    expect(component.selectedTemplateId()).toBe('tpl1');
  });

  it('defaultPeriodId tiene precedencia sobre el periodo activo', async () => {
    const fixture = TestBed.createComponent(CourseCreateModalComponent);
    const c = fixture.componentInstance;
    c.periods = mockPeriods;
    c.teachers = mockTeachers;
    c.templates = mockTemplates;
    c.defaultPeriodId = 'p1';
    c.ngOnInit();
    expect(c.selectedPeriodId()).toBe('p1');
  });

  it('submitCourse con nombre vacío establece formError y no llama al servicio', () => {
    component.name.set('');
    component.code.set('INFO-101');
    component.submitCourse();
    expect(component.formError()).toBe('El nombre y el código de la materia son obligatorios.');
    expect(mockCoursesService.createCourse).not.toHaveBeenCalled();
  });

  it('submitCourse con código vacío establece formError y no llama al servicio', () => {
    component.name.set('Programación I');
    component.code.set('');
    component.submitCourse();
    expect(component.formError()).not.toBeNull();
    expect(mockCoursesService.createCourse).not.toHaveBeenCalled();
  });

  it('submitCourse envía payload con código en mayúsculas', () => {
    component.name.set('Programación I');
    component.code.set('info-101');
    component.selectedTeacherId.set('t1');
    component.selectedPeriodId.set('p2');
    component.selectedTemplateId.set('tpl1');
    component.submitCourse();

    expect(mockCoursesService.createCourse).toHaveBeenCalledWith(
      expect.objectContaining({
        name: 'Programación I',
        code: 'INFO-101',
        teacher_id: 't1',
        academic_period_id: 'p2',
        template_id: 'tpl1'
      })
    );
  });

  it('al crear exitosamente emite courseCreated y limpia isSubmitting', () => {
    let created = false;
    component.courseCreated.subscribe(() => { created = true; });
    component.name.set('Redes I');
    component.code.set('NET-101');
    component.submitCourse();
    expect(created).toBe(true);
    expect(component.isSubmitting()).toBe(false);
  });

  it('ante error HTTP propaga el mensaje y limpia isSubmitting', () => {
    mockCoursesService.createCourse.mockReturnValue(
      throwError(() => ({ error: { error: 'Código duplicado' } }))
    );
    component.name.set('Redes I');
    component.code.set('NET-101');
    component.submitCourse();
    expect(component.formError()).toBe('Código duplicado');
    expect(component.isSubmitting()).toBe(false);
  });

  it('periodComboboxOptions incluye opción vacía al inicio', () => {
    const opts = component.periodComboboxOptions();
    expect(opts[0].id).toBe('');
    expect(opts.length).toBe(mockPeriods.length + 1);
  });

  it('selectedPeriodLabel refleja el periodo seleccionado', () => {
    component.selectedPeriodId.set('p2');
    expect(component.selectedPeriodLabel()).toContain('Gestión II/2026');
  });
});
