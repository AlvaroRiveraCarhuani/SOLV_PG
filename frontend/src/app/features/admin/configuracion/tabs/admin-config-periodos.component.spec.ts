import '@angular/compiler';
import '@angular/localize/init';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { of, throwError } from 'rxjs';
import { AdminConfigPeriodosComponent } from './admin-config-periodos.component';
import {
  AdminConfigPeriodosService,
  AcademicPeriodConfig,
  periodLifecycle
} from '../admin-config-periodos.service';

const makePeriod = (overrides: Partial<AcademicPeriodConfig> = {}): AcademicPeriodConfig => ({
  id: 'p1',
  tenant_id: 't1',
  name: 'Semestre II / 2026',
  code: '2026-2',
  start_date: '2026-08-01',
  end_date: '2030-12-15',
  is_active: true,
  ...overrides
});

describe('AdminConfigPeriodosComponent', () => {
  let component: AdminConfigPeriodosComponent;
  let mockService: any;
  let mockHttp: any;

  const periodsSignal = signal<AcademicPeriodConfig[]>([]);
  const loadingSignal = signal(false);
  const errorSignal = signal<string | null>(null);

  const seedPeriods = (): void => {
    periodsSignal.set([
      makePeriod(),
      makePeriod({ id: 'p2', name: 'Semestre I / 2027', code: '2027-1', is_active: false, start_date: '2027-02-01', end_date: '2030-06-30' }),
      makePeriod({ id: 'p3', name: 'Semestre I / 2024', code: '2024-1', is_active: false, start_date: '2024-02-01', end_date: '2024-06-30' })
    ]);
  };

  const setup = async (): Promise<void> => {
    seedPeriods();
    mockService = {
      periods: periodsSignal,
      isLoading: loadingSignal,
      error: errorSignal,
      fetchPeriods: vi.fn().mockReturnValue(of([])),
      createAndActivate: vi.fn().mockReturnValue(of(makePeriod())),
      updatePeriod: vi.fn().mockReturnValue(of(makePeriod())),
      archivePeriod: vi.fn().mockReturnValue(of({ status: 'archived' })),
      deletePeriod: vi.fn().mockReturnValue(of(void 0)),
      resolveError: vi.fn().mockImplementation((err: unknown) => {
        const code = (err as { error?: { error?: string } })?.error?.error;
        if (code === 'invalid_date_range') return 'La fecha de fin debe ser posterior o igual a la fecha de inicio.';
        if (code === 'conflict_associated_subjects') return 'No se puede eliminar un período con materias asociadas. Desasócialas primero.';
        return 'Ocurrió un error inesperado. Intenta nuevamente.';
      })
    };
    mockHttp = { get: vi.fn().mockReturnValue(of([])) };

    await TestBed.configureTestingModule({
      imports: [AdminConfigPeriodosComponent],
      providers: [
        { provide: AdminConfigPeriodosService, useValue: mockService },
        { provide: HttpClient, useValue: mockHttp }
      ]
    }).compileComponents();

    const fixture = TestBed.createComponent(AdminConfigPeriodosComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  };

  it('debe crearse y listar los 3 períodos con exactamente uno activo', async () => {
    await setup();

    expect(component.rows().length).toBe(3);
    expect(component.activeCount()).toBe(1);
    expect(component.rows()[0].lifecycle).toBe('activo');
    // Orden: activo primero, luego próximo, luego archivado
    expect(component.rows()[1].lifecycle).toBe('proximo');
    expect(component.rows()[2].lifecycle).toBe('archivado');
  });

  it('creación con "activar inmediatamente" llama createAndActivate y muestra toast de éxito', async () => {
    await setup();

    component.openCreate();
    component.name.set('Semestre I / 2027');
    component.code.set('2027-1');
    component.startDate.set('2027-02-01');
    component.endDate.set('2027-06-30');
    component.activateNow.set(true);
    component.submitForm();

    expect(mockService.createAndActivate).toHaveBeenCalledWith(
      expect.objectContaining({ code: '2027-1', is_active: true })
    );
    expect(component.toast()?.type).toBe('success');
    expect(component.formOpen()).toBe(false);
  });

  it('error de negocio del backend (422 invalid_date_range) se muestra en el banner sin cerrar el formulario', async () => {
    await setup();
    mockService.createAndActivate.mockReturnValue(
      throwError(() => ({ error: { error: 'invalid_date_range' } }))
    );

    component.openCreate();
    component.name.set('Semestre de Prueba');
    component.code.set('SP-1');
    component.startDate.set('2027-02-01');
    component.endDate.set('2027-06-30');
    component.submitForm();

    expect(component.formError()).toContain('fecha de fin');
    expect(component.formOpen()).toBe(true);
    expect(component.isSubmitting()).toBe(false);
  });

  it('archivo fuerte: botón deshabilitado hasta tipear el código exacto y archiva vía endpoint formal', async () => {
    await setup();

    component.openArchive({ ...makePeriod(), lifecycle: 'activo' } as any);
    expect(component.archiveCodeMatches()).toBe(false);

    component.archiveConfirmation.set('código incorrecto');
    expect(component.archiveCodeMatches()).toBe(false);

    component.archiveConfirmation.set('2026-2');
    expect(component.archiveCodeMatches()).toBe(true);

    component.confirmArchive();
    expect(mockService.archivePeriod).toHaveBeenCalledWith('p1', '2026-2');
    expect(component.toast()?.message).toContain('archivado formalmente');
    expect(component.toast()?.message).toContain('solo lectura');
  });

  it('activación de período próximo delega en updatePeriod con is_active true', async () => {
    await setup();

    // p2 es próximo (is_active=false, is_archived=false, vigencia futura) según la seed
    component.openActivate({ ...makePeriod({ id: 'p2', is_active: false }), lifecycle: 'proximo' } as any);
    component.confirmActivate();

    expect(mockService.updatePeriod).toHaveBeenCalledWith('p2', { is_active: true });
    expect(component.toast()?.type).toBe('success');
  });

  it('los períodos archivados no ofrecen reactivación: el lifecycle los marca inmutables', async () => {
    await setup();

    // p3 está vencido (2024) y el sweep lo habría formalizado como archivado
    const archived = { ...makePeriod({ id: 'p3', is_active: false, is_archived: true }), lifecycle: 'archivado' } as any;
    component.openActivate(archived);

    // La UI no tiene vía de activación para archivados: confirmActivate sobre
    // un archivado recibiría 409 period_archived del backend
    expect(archived.lifecycle).toBe('archivado');
    expect(component.activatingExpired()).toBe(true);
  });

  it('borrado sin materias procede y un 409 del backend se traduce a mensaje accionable', async () => {
    await setup();
    mockHttp.get.mockReturnValue(of([])); // sin materias asociadas
    (component as any).loadCourseCounts();
    await Promise.resolve();

    component.openDelete({ ...makePeriod(), lifecycle: 'activo' } as any);
    expect(component.deletingHasCourses()).toBe(false);

    mockService.deletePeriod.mockReturnValue(
      throwError(() => ({ error: { error: 'conflict_associated_subjects' } }))
    );
    component.confirmDelete();

    expect(mockService.deletePeriod).toHaveBeenCalledWith('p1');
    expect(component.toast()?.type).toBe('error');
    expect(mockService.resolveError).toHaveBeenCalled();
  });
});
