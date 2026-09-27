import { Component, inject, signal, computed, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import {
  AdminConfigServidorService,
  ServerPolicies,
  BackupExecutionItem
} from '../admin-config-servidor.service';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
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

@Component({
  selector: 'admin-config-servidor',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ModalShellComponent,
    FormFieldComponent,
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
  readonly isTogglingMaintenance = signal(false);

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
    this.maintenanceModalOpen.set(true);
  }

  closeMaintenanceModal(): void {
    this.maintenanceModalOpen.set(false);
  }

  confirmMaintenance(): void {
    if (!this.maintenanceConfirmValid() || this.isTogglingMaintenance()) return;
    this.isTogglingMaintenance.set(true);

    // Capturar la intención ANTES de la llamada: el estado se actualiza en el
    // tap del servicio y un mensaje evaluado después quedaría invertido.
    const wasActive = this.maintenance()?.maintenance_mode ?? false;

    const call$ = wasActive
      ? this.servidorService.disableMaintenance()
      : this.servidorService.enableMaintenance(this.maintenanceUntil(), this.maintenanceReason().trim());

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
    if (this.isSavingBackup()) return;
    this.isSavingBackup.set(true);

    this.servidorService.updateBackupConfig({
      local_frequency_hours: this.backupFrequency(),
      local_retention_days: this.backupRetention()
    }).subscribe({
      next: () => {
        this.isSavingBackup.set(false);
        this.showToast('Estrategia de respaldos actualizada.');
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
        this.showToast('Respaldo generado y verificado con checksum SHA-256.');
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
        this.showToast(
          res.valid
            ? `Integridad verificada: ${exec.file_name} OK.`
            : `CORRUPTO: el checksum de ${exec.file_name} no coincide.`,
          res.valid ? 'success' : 'error'
        );
      },
      error: (err) => this.showToast(this.servidorService.resolveError(err), 'error')
    });
  }

  formatBytes(bytes: number): string {
    if (bytes >= 1024 * 1024 * 1024) {
      return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
    }
    return `${(bytes / (1024 * 1024)).toFixed(0)} MB`;
  }

  formatDate(iso: string): string {
    return new Date(iso).toLocaleString('es-BO', {
      day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit'
    });
  }

  trackByBackup(index: number, exec: BackupExecutionItem): string {
    return exec.id;
  }
}
