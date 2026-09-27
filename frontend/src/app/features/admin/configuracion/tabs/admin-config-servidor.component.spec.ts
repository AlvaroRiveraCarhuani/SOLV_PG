import '@angular/compiler';
import '@angular/localize/init';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { of, throwError } from 'rxjs';
import { AdminConfigServidorComponent } from './admin-config-servidor.component';
import {
  AdminConfigServidorService,
  ServerPolicies,
  MaintenanceStatus,
  BackupExecutionItem
} from '../admin-config-servidor.service';

const makePolicies = (overrides: Partial<ServerPolicies> = {}): ServerPolicies => ({
  ram_limit_mb: 512,
  inactivity_minutes: 15,
  max_containers: 40,
  ...overrides
});

const makeExec = (overrides: Partial<BackupExecutionItem> = {}): BackupExecutionItem => ({
  id: 'b1',
  file_name: 'solv_snap_20260926.tar.gz',
  file_size_bytes: 1.8 * 1024 * 1024 * 1024,
  sha256_checksum: 'abc123',
  storage_tier: 'local',
  status: 'success',
  started_at: '2026-09-26T03:00:00Z',
  ...overrides
});

describe('AdminConfigServidorComponent', () => {
  let component: AdminConfigServidorComponent;
  let mockService: any;

  const policiesSignal = signal<ServerPolicies | null>(null);
  const maintenanceSignal = signal<MaintenanceStatus | null>(null);
  const backupsSignal = signal<BackupExecutionItem[]>([]);

  const setup = async (): Promise<void> => {
    policiesSignal.set(makePolicies());
    maintenanceSignal.set({ maintenance_mode: false });
    backupsSignal.set([makeExec()]);

    mockService = {
      policies: policiesSignal,
      maintenance: maintenanceSignal,
      backupConfig: signal<any>({ local_frequency_hours: 6, local_retention_days: 7 }),
      backups: backupsSignal,
      isLoading: signal(false),
      error: signal<string | null>(null),
      loadAll: vi.fn(),
      updatePolicies: vi.fn().mockImplementation((dto: Partial<ServerPolicies>) => {
        policiesSignal.set(makePolicies(dto));
        return of(makePolicies(dto));
      }),
      enableMaintenance: vi.fn().mockImplementation((until: string, reason: string) => {
        maintenanceSignal.set({ maintenance_mode: true, maintenance_until: until, maintenance_reason: reason });
        return of({});
      }),
      disableMaintenance: vi.fn().mockImplementation(() => {
        maintenanceSignal.set({ maintenance_mode: false });
        return of({});
      }),
      updateBackupConfig: vi.fn().mockReturnValue(of({ local_frequency_hours: 6, local_retention_days: 7 })),
      triggerBackup: vi.fn().mockImplementation(() => {
        backupsSignal.update((prev) => [makeExec({ id: 'b-new' }), ...prev]);
        return of(makeExec({ id: 'b-new' }));
      }),
      verifyBackup: vi.fn().mockReturnValue(of({ valid: true })),
      resolveError: vi.fn().mockReturnValue('Error de prueba')
    };

    await TestBed.configureTestingModule({
      imports: [AdminConfigServidorComponent],
      providers: [{ provide: AdminConfigServidorService, useValue: mockService }]
    }).compileComponents();

    const fixture = TestBed.createComponent(AdminConfigServidorComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  };

  it('carga el estado real: QoS vigente, mantenimiento inactivo, historial con 1 snapshot', async () => {
    await setup();

    expect(component.selectedRam()).toBe(512);
    expect(component.selectedInactivity()).toBe(15);
    expect(component.selectedMaxContainers()).toBe(40);
    expect(component.maintenance()?.maintenance_mode).toBe(false);
    expect(component.backups().length).toBe(1);
    expect(mockService.loadAll).toHaveBeenCalled();
  });

  it('guardado de políticas: valores del catálogo se envían y confirman efecto en workspaces nuevos', async () => {
    await setup();

    component.selectedRam.set(1024);
    component.selectedInactivity.set(10);
    component.selectedMaxContainers.set(60);
    expect(component.policiesDirty()).toBe(true);

    component.savePolicies();

    expect(mockService.updatePolicies).toHaveBeenCalledWith({
      ram_limit_mb: 1024,
      inactivity_minutes: 10,
      max_containers: 60
    });
    expect(component.toast()?.message).toContain('próximo ciclo');
  });

  it('estrategia de respaldos: frecuencia y retención se guardan', async () => {
    await setup();

    component.backupFrequency.set(12);
    component.backupRetention.set(14);
    component.saveBackupStrategy();

    expect(mockService.updateBackupConfig).toHaveBeenCalledWith(
      expect.objectContaining({ local_frequency_hours: 12, local_retention_days: 14 })
    );
    expect(component.toast()?.type).toBe('success');
  });

  it('respaldo manual se agrega al inicio del historial', async () => {
    await setup();

    component.triggerBackup();

    expect(mockService.triggerBackup).toHaveBeenCalled();
    expect(component.backups()[0].id).toBe('b-new');
  });

  it('verificación de integridad: checksum OK genera toast de éxito', async () => {
    await setup();

    component.verifyBackup(makeExec());

    expect(mockService.verifyBackup).toHaveBeenCalledWith('b1');
    expect(component.toast()?.message).toContain('OK');
  });

  it('activación de mantenimiento exige motivo >= 10 caracteres y ventana opcional', async () => {
    await setup();

    component.openMaintenanceModal();
    expect(component.maintenanceConfirmValid()).toBe(false);

    component.maintenanceReason.set('Corto');
    expect(component.maintenanceConfirmValid()).toBe(false);

    component.maintenanceReason.set('Actualización programada del orquestador');
    expect(component.maintenanceConfirmValid()).toBe(true);

    component.confirmMaintenance();
    expect(mockService.enableMaintenance).toHaveBeenCalledWith('', 'Actualización programada del orquestador');
    expect(component.maintenance()?.maintenance_mode).toBe(true);
    expect(component.toast()?.message).toContain('bypass');
  });

  it('desactivación de mantenimiento no exige motivo', async () => {
    await setup();
    maintenanceSignal.set({ maintenance_mode: true, maintenance_reason: 'Ventana previa' });

    component.openMaintenanceModal();
    expect(component.maintenanceConfirmValid()).toBe(true);

    component.confirmMaintenance();
    expect(mockService.disableMaintenance).toHaveBeenCalled();
    expect(component.maintenance()?.maintenance_mode).toBe(false);
  });
});
