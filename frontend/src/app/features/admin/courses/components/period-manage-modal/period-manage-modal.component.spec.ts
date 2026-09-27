import '@angular/compiler';
import '@angular/localize/init';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { of, throwError } from 'rxjs';
import { PeriodManageModalComponent } from './period-manage-modal.component';
import { AdminCoursesService, AcademicPeriod } from '../../../services/admin-courses.service';

const makePeriod = (overrides: Partial<AcademicPeriod> = {}): AcademicPeriod => ({
  id: 'p1',
  tenant_id: 't1',
  name: 'Gestión I/2026',
  code: 'G1-26',
  is_active: false,
  start_date: '2026-02-01',
  end_date: '2030-12-31', // future so not expired
  ...overrides
});

describe('PeriodManageModalComponent', () => {
  let component: PeriodManageModalComponent;
  let mockCoursesService: any;

  beforeEach(async () => {
    mockCoursesService = {
      createPeriod: vi.fn().mockReturnValue(of(makePeriod({ id: 'p-new' }))),
      updatePeriod: vi.fn().mockReturnValue(of(makePeriod({ is_active: true }))),
      deletePeriod: vi.fn().mockReturnValue(of(void 0))
    };

    await TestBed.configureTestingModule({
      imports: [PeriodManageModalComponent],
      providers: [
        { provide: AdminCoursesService, useValue: mockCoursesService }
      ]
    }).compileComponents();

    const fixture = TestBed.createComponent(PeriodManageModalComponent);
    component = fixture.componentInstance;
    component.periods = [makePeriod()];
    fixture.detectChanges();
  });

  it('debe crearse correctamente', () => {
    expect(component).toBeTruthy();
  });

  it('submitNewPeriod con campos vacíos establece formError y no llama al servicio', () => {
    component.submitNewPeriod();
    expect(component.formError()).toBe('Todos los campos son obligatorios.');
    expect(mockCoursesService.createPeriod).not.toHaveBeenCalled();
  });

  it('submitNewPeriod con fecha fin anterior a inicio establece error de rango', () => {
    component.name.set('Gestión Mal');
    component.code.set('GM-26');
    component.startDate.set('2026-08-01');
    component.endDate.set('2026-07-01'); // anterior a startDate
    component.submitNewPeriod();
    expect(component.formError()).toContain('fecha de fin');
  });

  it('submitNewPeriod con datos válidos llama al servicio y emite periodCreated', () => {
    let emitted = false;
    component.periodCreated.subscribe(() => { emitted = true; });

    component.name.set('Gestión I/2026');
    component.code.set('g1-26');
    component.startDate.set('2026-02-01');
    component.endDate.set('2026-06-30');
    component.submitNewPeriod();

    expect(mockCoursesService.createPeriod).toHaveBeenCalledWith(
      expect.objectContaining({ code: 'G1-26' }) // código en mayúsculas
    );
    expect(emitted).toBe(true);
  });

  it('submitNewPeriod ante error HTTP propaga el mensaje del servidor', () => {
    mockCoursesService.createPeriod.mockReturnValue(
      throwError(() => ({ error: { error: 'El código ya existe' } }))
    );
    component.name.set('Test');
    component.code.set('TST');
    component.startDate.set('2026-01-01');
    component.endDate.set('2026-12-31');
    component.submitNewPeriod();
    expect(component.formError()).toBe('El código ya existe');
  });

  it('formatDate convierte ISO a formato dd/mm/yyyy', () => {
    expect(component.formatDate('2026-09-15T00:00:00Z')).toBe('15/09/2026');
    expect(component.formatDate(undefined)).toBe('');
  });

  it('isPeriodExpired devuelve true para fecha de fin pasada', () => {
    const expired = makePeriod({ end_date: '2020-01-01' });
    expect(component.isPeriodExpired(expired)).toBe(true);
  });

  it('isPeriodExpired devuelve false para fecha de fin futura', () => {
    const valid = makePeriod({ end_date: '2099-12-31' });
    expect(component.isPeriodExpired(valid)).toBe(false);
  });

  it('setActive no llama al servicio si el periodo ya está activo', () => {
    component.setActive(makePeriod({ is_active: true }));
    expect(mockCoursesService.updatePeriod).not.toHaveBeenCalled();
  });

  it('setActive bloquea activación de periodo expirado con mensaje de error', () => {
    const expired = makePeriod({ end_date: '2020-01-01' });
    component.setActive(expired);
    expect(component.formError()).toContain('expiró');
    expect(mockCoursesService.updatePeriod).not.toHaveBeenCalled();
  });

  it('requestDeletePeriod bloquea la eliminación del periodo activo', () => {
    const active = makePeriod({ is_active: true });
    component.requestDeletePeriod(active);
    expect(component.formError()).toContain('activo');
    expect(component.periodToDelete()).toBeNull();
  });

  it('requestDeletePeriod establece periodToDelete para periodo inactivo', () => {
    const inactive = makePeriod({ is_active: false });
    component.requestDeletePeriod(inactive);
    expect(component.periodToDelete()).toEqual(inactive);
  });

  it('cancelDeletePeriod limpia periodToDelete', () => {
    component.periodToDelete.set(makePeriod());
    component.cancelDeletePeriod();
    expect(component.periodToDelete()).toBeNull();
  });
});
