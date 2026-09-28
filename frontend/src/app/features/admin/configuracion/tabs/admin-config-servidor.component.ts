import { Component, inject, signal, computed, effect, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import {
  AdminConfigServidorService,
  ServerPolicies,
  BackupExecutionItem,
  VerifyBackupResult
} from '../admin-config-servidor.service';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import { PaginationBarComponent } from '@shared/components/pagination-bar/pagination-bar.component';
import { DateTextPipe, formatSolvDate } from '@shared/pipes/date-text.pipe';
import {
  LucideServer,
  LucideHardDrive,
  LucideWrench,
  LucideSave,
  LucideDatabaseBackup,
  LucideShieldCheck,
  LucideDownload,
  LucideRefreshCw,
  LucideCheckCircle2,
  LucideAlertCircle,
  LucideAlertTriangle,
  LucidePower
} from '@lucide/angular';

const RAM_OPTIONS = [256, 512, 1024];
const INACTIVITY_OPTIONS = [10, 15, 30];

// Pure fail-closed integer check: NaN, empty, decimals and out-of-range all
// produce an inline message; valid input returns ''.
export function validateBackupInt(raw: unknown, min: number, max: number, field: string): string {
  const value = Number(raw);
  if (!Number.isInteger(value) || value < min || value > max) {
    return `La ${field} debe ser un entero entre ${min} y ${max}.`;
  }
  return '';
}

@Component({
  selector: 'admin-config-servidor',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    DateTextPipe,
    ModalShellComponent,
    FormFieldComponent,
    PaginationBarComponent,
    LucideServer,
    LucideHardDrive,
    LucideWrench,
    LucideSave,
    LucideDatabaseBackup,
    LucideShieldCheck,
    LucideDownload,
    LucideRefreshCw,
    LucideCheckCircle2,
    LucideAlertCircle,
    LucideAlertTriangle,
    LucidePower
  ],
  templateUrl: './admin-config-servidor.component.html',
  styleUrls: ['./admin-config-servidor.component.scss']
})
export class AdminConfigServidorComponent implements OnInit {
  private readonly servidorService = inject(AdminConfigServidorService);

  readonly policies = this.servidorService.policies;
  readonly maintenance = this.servidorService.maintenance;
  readonly backupConfig = this.servidorService.backupConfig;
  readonly backups = this.servidorService.backups;
  readonly isLoading = this.servidorService.isLoading;
  readonly loadError = this.servidorService.error;

  readonly ramOptions = RAM_OPTIONS;
  readonly inactivityOptions = INACTIVITY_OPTIONS;

  // Form QoS local (radios del wireframe: solo valores del catálogo)
  readonly selectedRam = signal<number>(512);
  readonly selectedInactivity = signal<number>(15);
  readonly selectedMaxContainers = signal<number>(40);
  readonly isSavingPolicies = signal(false);

  readonly policiesDirty = computed(() => {
    const p = this.policies();
    if (!p) return false;
    return (
      this.selectedRam() !== p.ram_limit_mb ||
      this.selectedInactivity() !== p.inactivity_minutes ||
      this.selectedMaxContainers() !== p.max_containers
    );
  });

  readonly canSavePolicies = computed(() => this.policiesDirty() && !this.isSavingPolicies());

  // Respaldos: estrategia local (frecuencia/retención)
  readonly backupFrequency = signal(6);
  readonly backupRetention = signal(7);
  readonly isSavingBackup = signal(false);
  readonly isTriggering = signal(false);

  // Fail-closed: ngModel con type="number" entrega ""/NaN; Number() los
  // normaliza a 0/NaN para que caigan en la rama inválida.
  readonly backupFrequencyError = computed(() => validateBackupInt(this.backupFrequency(), 1, 168, 'frecuencia'));
  readonly backupRetentionError = computed(() => validateBackupInt(this.backupRetention(), 1, 365, 'retención'));

  readonly canSaveBackup = computed(
    () => !this.backupFrequencyError() && !this.backupRetentionError() && !this.isSavingBackup()
  );

  // Conteo client-side (sin endpoint nuevo): respaldos más viejos que la
  // retención ingresada expirarían al guardar.
  readonly purgeCount = computed(() => {
    const retention = Number(this.backupRetention());
    if (!Number.isInteger(retention) || retention < 1) return 0;
    const cutoff = Date.now() - retention * 86400000;
    return this.backups().filter((b) => new Date(b.started_at).getTime() < cutoff).length;
  });

  // ------------------------------------------------------------------
  // Paginación historial de respaldos (estilo Docentes)
  // ------------------------------------------------------------------
  readonly backupsCurrentPage = signal(1);
  readonly backupsPageSize = 10;

  readonly paginatedBackups = computed(() => {
    const start = (this.backupsCurrentPage() - 1) * this.backupsPageSize;
    return this.backups().slice(start, start + this.backupsPageSize);
  });

  readonly backupsTotalPages = computed(() => Math.max(1, Math.ceil(this.backups().length / this.backupsPageSize)));

  readonly backupsPaginationDisplay = computed(() => {
    const total = this.backups().length;
    if (total === 0) return { from: 0, to: 0 };
    const from = (this.backupsCurrentPage() - 1) * this.backupsPageSize + 1;
    const to = Math.min(this.backupsCurrentPage() * this.backupsPageSize, total);
    return { from, to };
  });

  goToBackupsPage(page: number): void {
    if (page >= 1 && page <= this.backupsTotalPages()) {
      this.backupsCurrentPage.set(page);
    }
  }

  // El servicio carga async; sin esta sincronización el form muestra los
  // defaults locales (512/15/40, 6/7) aunque el backend tenga otros valores:
  // falso "dirty" en QoS y riesgo de pisar la estrategia de respaldos.
  // Se sincroniza una sola vez por carga para no pisar ediciones en curso.
  private formSynced = false;

  constructor() {
    effect(() => {
      const p = this.policies();
      const cfg = this.backupConfig();
      if (this.formSynced) return;
      if (!p && !cfg) return;
      if (p) {
        this.selectedRam.set(p.ram_limit_mb);
        this.selectedInactivity.set(p.inactivity_minutes);
        this.selectedMaxContainers.set(p.max_containers);
      }
      if (cfg) {
        this.backupFrequency.set(cfg.local_frequency_hours);
        this.backupRetention.set(cfg.local_retention_days);
      }
      this.formSynced = true;
    });
  }

  // ------------------------------------------------------------------
  // Toast + modal de mantenimiento
  // ------------------------------------------------------------------
  readonly toast = signal<{ type: 'success' | 'error'; message: string } | null>(null);
  private toastTimer?: ReturnType<typeof setTimeout>;

  showToast(message: string, type: 'success' | 'error' = 'success'): void {
    this.toast.set({ message, type });
    if (this.toastTimer) clearTimeout(this.toastTimer);
    this.toastTimer = setTimeout(() => this.toast.set(null), 4000);
  }

  readonly maintenanceModalOpen = signal(false);
  readonly maintenanceUntil = signal('');
  readonly maintenanceReason = signal('');
  readonly confirmPhrase = signal('');
  readonly isTogglingMaintenance = signal(false);

  readonly maintenanceUntilError = computed(() => {
    const raw = this.maintenanceUntil().trim();
    if (raw === '') return '';
    if (new Date(raw).getTime() < Date.now()) return 'La vigencia debe ser futura.';
    return '';
  });

  // Vacía = indefinida: avisar que queda activo hasta baja manual.
  readonly vigenciaWarning = computed(() =>
    this.maintenanceUntil().trim() === ''
      ? 'Sin vigencia el mantenimiento permanece activo hasta desactivación manual.'
      : ''
  );

  // Activar exige frase exacta + motivo >= 10 + vigencia no vencida.
  readonly confirmDisabled = computed(() => {
    if (this.maintenance()?.maintenance_mode) return this.isTogglingMaintenance();
    return (
      this.confirmPhrase() !== 'MANTENIMIENTO' ||
      this.maintenanceReason().trim().length < 10 ||
      this.maintenanceUntilError() !== '' ||
      this.isTogglingMaintenance()
    );
  });

  readonly maintenanceConfirmValid = computed(() => {
    const status = this.maintenance();
    if (status?.maintenance_mode) return true; // apagar no exige motivo
    // Activar exige motivo >= 10 caracteres; la ventana es opcional
    return this.maintenanceReason().trim().length >= 10;
  });

  ngOnInit(): void {
    this.servidorService.loadAll();
  }

  // ------------------------------------------------------------------
  // QoS
  // ------------------------------------------------------------------
  syncFormFromPolicies(): void {
    const p = this.policies();
    if (!p) return;
    this.selectedRam.set(p.ram_limit_mb);
    this.selectedInactivity.set(p.inactivity_minutes);
    this.selectedMaxContainers.set(p.max_containers);
  }

  savePolicies(): void {
    if (!this.canSavePolicies()) return;
    this.isSavingPolicies.set(true);

    this.servidorService.updatePolicies({
      ram_limit_mb: this.selectedRam(),
      inactivity_minutes: this.selectedInactivity(),
      max_containers: this.selectedMaxContainers()
    }).subscribe({
      next: () => {
        this.isSavingPolicies.set(false);
        this.showToast('Políticas QoS guardadas. El worker aplicará la configuración en su próximo ciclo.');
      },
      error: (err) => {
        this.isSavingPolicies.set(false);
        this.showToast(this.servidorService.resolveError(err), 'error');
      }
    });
  }

  // ------------------------------------------------------------------
  // Mantenimiento
  // ------------------------------------------------------------------
  openMaintenanceModal(): void {
    this.maintenanceUntil.set('');
    this.maintenanceReason.set('');
    this.confirmPhrase.set('');
    this.maintenanceModalOpen.set(true);
  }

  closeMaintenanceModal(): void {
    this.maintenanceModalOpen.set(false);
  }

  confirmMaintenance(): void {
    if (this.confirmDisabled() || this.isTogglingMaintenance()) return;
    this.isTogglingMaintenance.set(true);

    // Capturar la intención ANTES de la llamada: el estado se actualiza en el
    // tap del servicio y un mensaje evaluado después quedaría invertido.
    const wasActive = this.maintenance()?.maintenance_mode ?? false;

    const call$ = wasActive
      ? this.servidorService.disableMaintenance()
      : this.servidorService.enableMaintenance(this.maintenanceUntil(), this.maintenanceReason().trim(), this.confirmPhrase());

    call$.subscribe({
      next: () => {
        this.isTogglingMaintenance.set(false);
        this.closeMaintenanceModal();
        this.showToast(
          wasActive
            ? 'Modo mantenimiento desactivado. La plataforma vuelve a aceptar sesiones.'
            : 'Modo mantenimiento activado. Los estudiantes y docentes verán la pantalla de mantenimiento; los administradores conservan bypass.'
        );
      },
      error: (err) => {
        this.isTogglingMaintenance.set(false);
        this.showToast(this.servidorService.resolveError(err), 'error');
      }
    });
  }

  // ------------------------------------------------------------------
  // Respaldos
  // ------------------------------------------------------------------
  saveBackupStrategy(): void {
    if (!this.canSaveBackup() || this.isSavingBackup()) return;
    this.isSavingBackup.set(true);

    this.servidorService.updateBackupConfig({
      local_frequency_hours: this.backupFrequency(),
      local_retention_days: this.backupRetention()
    }).subscribe({
      next: () => {
        this.isSavingBackup.set(false);
        this.showToast('Configuración de respaldos guardada.');
      },
      error: (err) => {
        this.isSavingBackup.set(false);
        this.showToast(this.servidorService.resolveError(err), 'error');
      }
    });
  }

  triggerBackup(): void {
    if (this.isTriggering()) return;
    this.isTriggering.set(true);

    this.servidorService.triggerBackup().subscribe({
      next: () => {
        this.isTriggering.set(false);
        this.showToast('Respaldo creado. Integridad sin verificar — usa Verificar para confirmar.');
      },
      error: (err) => {
        this.isTriggering.set(false);
        this.showToast(this.servidorService.resolveError(err), 'error');
      }
    });
  }

  async verifyBackup(exec: BackupExecutionItem): Promise<void> {
    this.servidorService.verifyBackup(exec.id).subscribe({
      next: (res) => {
        const data = res.data;
        if (data.is_valid) {
          this.showToast(`Integridad verificada: ${exec.file_name} OK.`, 'success');
        } else if (data.computed_checksum === '') {
          this.showToast(`Archivo no disponible en disco: ${exec.file_name}.`, 'error');
        } else {
          this.showToast(`Checksum no coincide: ${exec.file_name} podría estar corrupto.`, 'error');
        }
      },
      error: (err) => this.showToast(this.servidorService.resolveError(err), 'error')
    });
  }

  formatBytes(bytes: number): string {
    if (bytes <= 0) return '0 B';
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
  }

  formatDate(iso: string): string {
    return formatSolvDate(iso, 'daymonth') ?? iso;
  }

  trackByBackup(index: number, exec: BackupExecutionItem): string {
    return exec.id;
  }
}
