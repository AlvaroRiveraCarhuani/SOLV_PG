import '@angular/compiler';
import '@angular/localize/init';
import { describe, it, expect, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { of } from 'rxjs';
import { AdminConfigServidorComponent } from './admin-config-servidor.component';
import {
  AdminConfigServidorService,
  ServerPolicies,
  MaintenanceStatus,
  BackupConfigItem,
  BackupExecutionItem,
  VerifyBackupResult,
  OrphanVolumePlan,
  KeptVolumeItem
} from '../admin-config-servidor.service';

const makePolicies = (overrides: Partial<ServerPolicies> = {}): ServerPolicies => ({
  host_safety_ceiling_pct: 85,
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
  last_verify_ok: null,
  last_verify_at: null,
  ...overrides
});

const makeVerifyResult = (overrides: Partial<VerifyBackupResult> = {}): VerifyBackupResult => ({
  execution_id: 'b1',
  file_name: 'solv_snap_20260926.tar.gz',
  database_checksum: 'abc123',
  computed_checksum: 'abc123',
  is_valid: true,
  message: 'Integridad criptográfica verificada: el archivo no está corrupto ni alterado',
  ...overrides
});

const makeBackupConfig = (overrides: Partial<BackupConfigItem> = {}): BackupConfigItem => ({
  id: 'cfg-1',
  tenant_id: 't1',
  local_frequency_hours: 6,
  local_retention_days: 7,
  remote_enabled: false,
  remote_provider: '',
  remote_bucket_name: '',
  remote_endpoint: '',
  remote_retention_days: 30,
  is_active: true,
  updated_at: '2026-09-26T00:00:00Z',
  ...overrides
});

interface MockAdminConfigServidorService {
  policies: ReturnType<typeof signal<ServerPolicies | null>>;
  maintenance: ReturnType<typeof signal<MaintenanceStatus | null>>;
  backupConfig: ReturnType<typeof signal<BackupConfigItem | null>>;
  backups: ReturnType<typeof signal<BackupExecutionItem[]>>;
  orphanPlan: ReturnType<typeof signal<OrphanVolumePlan | null>>;
  orphansLoading: ReturnType<typeof signal<boolean>>;
  orphansError: ReturnType<typeof signal<string | null>>;
  orphansCheckedAt: ReturnType<typeof signal<string | null>>;
  isLoading: ReturnType<typeof signal<boolean>>;
  error: ReturnType<typeof signal<string | null>>;
  loadAll: ReturnType<typeof vi.fn>;
  loadOrphanVolumes: ReturnType<typeof vi.fn>;
  pruneOrphanVolumes: ReturnType<typeof vi.fn>;
  updatePolicies: ReturnType<typeof vi.fn>;
  enableMaintenance: ReturnType<typeof vi.fn>;
  disableMaintenance: ReturnType<typeof vi.fn>;
  updateBackupConfig: ReturnType<typeof vi.fn>;
  triggerBackup: ReturnType<typeof vi.fn>;
  verifyBackup: ReturnType<typeof vi.fn>;
  resolveError: ReturnType<typeof vi.fn>;
}

const verifyEnvelope = (result: VerifyBackupResult) => ({ data: result });

const makeKept = (name: string, size_mb: number, reason: string): KeptVolumeItem => ({
  name,
  size_mb,
  reason
});

const makeOrphanPlan = (
  kept: KeptVolumeItem[],
  orphans: { name: string; size_mb: number }[] = []
): OrphanVolumePlan => ({
  orphans,
  kept,
  total_orphan_mb: orphans.reduce((sum, item) => sum + item.size_mb, 0),
  total_kept_mb: kept.reduce((sum, item) => sum + item.size_mb, 0)
});

describe('AdminConfigServidorComponent', () => {
  let component: AdminConfigServidorComponent;
  let mockService: MockAdminConfigServidorService;

  const policiesSignal = signal<ServerPolicies | null>(null);
  const maintenanceSignal = signal<MaintenanceStatus | null>(null);
  const backupsSignal = signal<BackupExecutionItem[]>([]);
  const orphanPlanSignal = signal<OrphanVolumePlan | null>(null);

  const setup = async (): Promise<void> => {
    policiesSignal.set(makePolicies());
    maintenanceSignal.set({ maintenance_mode: false });
    backupsSignal.set([makeExec()]);
    orphanPlanSignal.set(null);

    mockService = {
      policies: policiesSignal,
      maintenance: maintenanceSignal,
      backupConfig: signal<BackupConfigItem | null>(makeBackupConfig({ local_frequency_hours: 6, local_retention_days: 7 })),
      backups: backupsSignal,
      orphanPlan: orphanPlanSignal,
      orphansLoading: signal(false),
      orphansError: signal<string | null>(null),
      orphansCheckedAt: signal<string | null>(null),
      isLoading: signal(false),
      error: signal<string | null>(null),
      loadAll: vi.fn(),
      loadOrphanVolumes: vi.fn(),
      pruneOrphanVolumes: vi.fn().mockReturnValue(of({ data: { removed: 0, freed_mb: 0 } })),
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
      verifyBackup: vi.fn().mockReturnValue(of(verifyEnvelope(makeVerifyResult()))),
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

  it('carga el estado real: autogestión vigente, mantenimiento inactivo, historial con 1 snapshot', async () => {
    await setup();

    expect(component.selectedCeilingPct()).toBe(85);
    expect(component.selectedInactivity()).toBe(15);
    expect(component.maintenance()?.maintenance_mode).toBe(false);
    expect(component.backups().length).toBe(1);
    expect(mockService.loadAll).toHaveBeenCalled();
  });

  it('guardado de políticas: valores del catálogo se envían y confirman efecto en environments nuevos', async () => {
    await setup();

    component.selectedCeilingPct.set(90);
    component.selectedInactivity.set(30);
    expect(component.policiesDirty()).toBe(true);

    component.savePolicies();

    expect(mockService.updatePolicies).toHaveBeenCalledWith({
      host_safety_ceiling_pct: 90,
      inactivity_minutes: 30,
      max_containers: 40
    });
    expect(component.toast()?.message).toContain('próximo ciclo');
  });

  it('concurrencia: adopta el techo recomendado por el servidor y lo guarda', async () => {
    await setup();
    policiesSignal.set(
      makePolicies({
        max_containers: 40,
        recommendation: {
          recommended_max_containers: 213,
          host_total_mb: 131072,
          platform_reserve_mb: 2048,
          ram_per_lab_mb: 512,
          host_safety_ceiling_pct: 85,
          capacity_basis: 'host_ram'
        }
      })
    );

    expect(component.recommendedMaxContainers()).toBe(213);
    expect(component.hostTotalGB()).toBe(128);

    component.adoptRecommendedMaxContainers();
    expect(component.selectedMaxContainers()).toBe(213);
    expect(component.policiesDirty()).toBe(true);

    component.savePolicies();

    expect(mockService.updatePolicies).toHaveBeenCalledWith(
      expect.objectContaining({ max_containers: 213 })
    );
  });

  it('concurrencia: rechaza un valor fuera del rango de la política', async () => {
    await setup();

    component.selectedMaxContainers.set(0);
    expect(component.maxContainersError()).not.toBe('');
    expect(component.canSavePolicies()).toBe(false);
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

  it('verificación de integridad: checksum OK genera toast de éxito con envelope data.is_valid', async () => {
    await setup();

    component.verifyBackup(makeExec());

    expect(mockService.verifyBackup).toHaveBeenCalledWith('b1');
    expect(component.toast()?.message).toContain('OK');
  });

  it('verificación: archivo faltante muestra copy de missing (is_valid=false, checksum vacío)', async () => {
    await setup();

    const missingResult = makeVerifyResult({
      is_valid: false,
      computed_checksum: '',
      database_checksum: 'abc123',
      message: 'Archivo físico no encontrado en disco'
    });
    mockService.verifyBackup.mockReturnValue(of(verifyEnvelope(missingResult)));

    component.verifyBackup(makeExec());

    expect(component.toast()?.message).toContain('no disponible');
  });

  it('verificación: checksum mismatch muestra copy distinta de missing', async () => {
    await setup();

    const mismatchResult = makeVerifyResult({
      is_valid: false,
      computed_checksum: 'tampered123',
      database_checksum: 'abc123',
      message: 'ALERTA: El checksum del archivo físico difiere'
    });
    mockService.verifyBackup.mockReturnValue(of(verifyEnvelope(mismatchResult)));

    component.verifyBackup(makeExec());

    expect(component.toast()?.message).toContain('Checksum no coincide');
    expect(component.toast()?.message).toContain('corrupto');
  });

  it('formatBytes: 150 B renderiza KB, nunca 0 MB', () => {
    expect(component.formatBytes(150)).not.toBe('0 MB');
    expect(component.formatBytes(150)).toContain('B');
    expect(component.formatBytes(150)).not.toContain('MB');
  });

  it('formatBytes: 1024 B renderiza 1.0 KB', () => {
    expect(component.formatBytes(1024)).toBe('1.0 KB');
  });

  it('formatBytes: 0 B renderiza 0 B', () => {
    expect(component.formatBytes(0)).toBe('0 B');
  });

  it('trigger toast no contiene claim de verificación', async () => {
    await setup();

    component.triggerBackup();

    expect(component.toast()?.message).not.toContain('verificad');
    expect(component.toast()?.message).toContain('Integridad');
  });

  it('activación de mantenimiento exige motivo >= 10 caracteres y ventana opcional', async () => {
    await setup();

    component.openMaintenanceModal();
    expect(component.maintenanceConfirmValid()).toBe(false);

    component.maintenanceReason.set('Corto');
    expect(component.maintenanceConfirmValid()).toBe(false);

    component.maintenanceReason.set('Actualización programada del orquestador');
    expect(component.maintenanceConfirmValid()).toBe(true);

    component.confirmPhrase.set('MANTENIMIENTO');
    component.confirmMaintenance();
    expect(mockService.enableMaintenance).toHaveBeenCalledWith('', 'Actualización programada del orquestador', 'MANTENIMIENTO');
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

  it('respaldo inválido: error inline, Save bloqueado y sin toast de éxito', async () => {
    await setup();

    component.backupFrequency.set(0);
    expect(component.backupFrequencyError()).toContain('1');
    expect(component.canSaveBackup()).toBe(false);

    component.saveBackupStrategy();
    expect(mockService.updateBackupConfig).not.toHaveBeenCalled();
    expect(component.toast()).toBeNull();

    component.backupFrequency.set(6);
    component.backupRetention.set(366);
    expect(component.backupRetentionError()).toContain('365');
    expect(component.canSaveBackup()).toBe(false);
  });

  it('respaldo NaN o decimal se rechaza igual que fuera de rango', async () => {
    await setup();

    component.backupFrequency.set(NaN);
    expect(component.backupFrequencyError()).toBeTruthy();
    expect(component.canSaveBackup()).toBe(false);

    component.backupFrequency.set(6.5);
    expect(component.backupFrequencyError()).toBeTruthy();
    expect(component.canSaveBackup()).toBe(false);
  });

  it('bajar retención advierte cuántos respaldos expirarían', async () => {
    await setup();
    backupsSignal.set([
      makeExec({ id: 'old', started_at: '2026-01-01T00:00:00Z' }),
      makeExec({ id: 'new', started_at: new Date().toISOString() })
    ]);

    component.backupRetention.set(7);
    expect(component.purgeCount()).toBe(1);
    expect(component.canSaveBackup()).toBe(true);
  });

  it('frase distinta de MANTENIMIENTO bloquea la activación', async () => {
    await setup();

    component.openMaintenanceModal();
    component.maintenanceReason.set('Actualización programada del orquestador');
    component.confirmPhrase.set('mantenimiento');
    expect(component.confirmDisabled()).toBe(true);

    component.confirmMaintenance();
    expect(mockService.enableMaintenance).not.toHaveBeenCalled();

    component.confirmPhrase.set('MANTENIMIENTO');
    expect(component.confirmDisabled()).toBe(false);
  });

  it('vigencia vacía advierte permanencia indefinida', async () => {
    await setup();

    component.openMaintenanceModal();
    component.maintenanceUntil.set('');
    expect(component.vigenciaWarning()).toContain('manual');
  });

  it('volúmenes conservados: agrupa por motivo con conteo y subtotal', async () => {
    await setup();

    orphanPlanSignal.set(
      makeOrphanPlan([
        makeKept('solv_workspace_a', 10, 'entorno_registrado'),
        makeKept('pg_data', 4, 'fuera_del_espacio_de_la_plataforma'),
        makeKept('solv_workspace_b', 20, 'entorno_registrado')
      ])
    );

    const groups = component.keptVolumeGroups();
    expect(groups).toHaveLength(2);
    expect(groups[0].reason).toBe('entorno_registrado');
    expect(groups[0].label).toBe('Entornos registrados');
    expect(groups[0].count).toBe(2);
    expect(groups[0].totalMB).toBe(30);
    expect(groups[1].label).toBe('Infraestructura y volúmenes ajenos al laboratorio');
    expect(groups[1].count).toBe(1);
  });

  it('volúmenes conservados: dentro del grupo ordena por peso descendente', async () => {
    await setup();

    orphanPlanSignal.set(
      makeOrphanPlan([
        makeKept('chico', 1, 'entorno_registrado'),
        makeKept('grande', 80, 'entorno_registrado'),
        makeKept('mediano', 12, 'entorno_registrado')
      ])
    );

    expect(component.keptVolumeGroups()[0].items.map((item) => item.name)).toEqual([
      'grande',
      'mediano',
      'chico'
    ]);
  });

  it('volúmenes conservados: acorta hashes anónimos de 64 caracteres y deja intactos los nombres legibles', async () => {
    await setup();

    const hash = `040a9364${'a'.repeat(50)}c0d2f1`;
    expect(hash).toHaveLength(64);
    expect(component.shortVolumeName(hash)).toBe('040a9364…c0d2f1');
    expect(component.shortVolumeName('solv_workspace_ab12')).toBe('solv_workspace_ab12');
    expect(component.shortVolumeName('pg_data')).toBe('pg_data');
  });

  it('volúmenes conservados: tamaño de fila cae a < 1 MB cuando size_mb es 0', async () => {
    await setup();

    expect(component.formatKeptSize(0)).toBe('< 1 MB');
    expect(component.formatKeptSize(2.5)).toBe('2.5 MB');
  });

  it('lectura limpia: sin toast verde, solo la marca de revisado', async () => {
    await setup();
    mockService.loadOrphanVolumes.mockImplementation((onDone?: () => void) => onDone?.());
    orphanPlanSignal.set(makeOrphanPlan([makeKept('solv_workspace_a', 3, 'entorno_registrado')]));

    component.reloadOrphans();

    expect(component.toast()).toBeNull();
    expect(component.orphanNotice()).toBeNull();
  });

  it('lectura con huérfanos: aviso informativo con el espacio recuperable', async () => {
    await setup();
    mockService.loadOrphanVolumes.mockImplementation((onDone?: () => void) => onDone?.());
    orphanPlanSignal.set(
      makeOrphanPlan([], [
        { name: 'solv_workspace_huerfano', size_mb: 2048 },
        { name: 'solv_workspace_viejo', size_mb: 1024 }
      ])
    );

    component.reloadOrphans();

    expect(component.toast()).toBeNull();
    expect(component.orphanNotice()).toContain('2');
    expect(component.orphanNotice()).toContain('3.0 GB');
  });
});
