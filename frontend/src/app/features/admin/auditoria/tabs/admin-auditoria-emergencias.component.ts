import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpErrorResponse } from '@angular/common/http';
import { LucideSiren, LucideTriangleAlert } from '@lucide/angular';
import { ModalShellComponent, ModalIntent } from '@shared/components/modal-shell/modal-shell.component';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import { AdminMetricsService } from '@features/admin/services/admin-metrics.service';
import {
  AdminAuditoriaService,
  EMERGENCY_ACTIONS,
  EmergencyActionId,
  EmergencyActionResult
} from '../admin-auditoria.service';
import { AuditLog, humanizeConstant } from '@core/models/audit-log.model';

interface EmergencyActionDef {
  id: EmergencyActionId;
  label: string;
  phrase: string;
  impact: 'destructiva' | 'operativa';
  description: string;
}

interface EmergencyHistoryRow {
  log: AuditLog;
  label: string;
}

/**
 * Pestaña Emergencias (ADR-032): centro de control con las 5 acciones del
 * catálogo clasificadas por impacto, doble confirmación por frase tipada,
 * motivo obligatorio y trazabilidad de ejecuciones previas.
 */
@Component({
  selector: 'admin-auditoria-emergencias',
  standalone: true,
  imports: [CommonModule, FormsModule, LucideSiren, LucideTriangleAlert, ModalShellComponent, FormFieldComponent],
  templateUrl: './admin-auditoria-emergencias.component.html',
  styleUrl: './admin-auditoria-emergencias.component.scss'
})
export class AdminAuditoriaEmergenciasComponent implements OnInit {
  private readonly auditoriaService = inject(AdminAuditoriaService);
  private readonly metricsService = inject(AdminMetricsService);

  readonly actions: EmergencyActionDef[] = [...EMERGENCY_ACTIONS];

  readonly isExecuting = this.auditoriaService.isExecuting;
  readonly executingAction = signal<EmergencyActionId | null>(null);

  readonly toast = signal<{ type: 'success' | 'error'; message: string } | null>(null);
  private toastTimer?: ReturnType<typeof setTimeout>;

  readonly ramPercent = computed<number>(() => Math.round(this.metricsService.health()?.metrics.ram_percent ?? 0));
  readonly runningContainers = computed<number>(
    () => this.metricsService.health()?.metrics.containers_active ?? 0
  );
  readonly ramCritical = computed<boolean>(() => this.ramPercent() >= 90);

  readonly confirmTarget = signal<EmergencyActionDef | null>(null);
  readonly confirmPhrase = signal('');
  readonly confirmReason = signal('');
  readonly confirmError = signal<string | null>(null);
  readonly confirmIntent = computed<ModalIntent>(() =>
    this.confirmTarget()?.impact === 'destructiva' ? 'error' : 'warning'
  );
  readonly confirmDisabled = computed<boolean>(() => {
    const target = this.confirmTarget();
    return (
      !target ||
      this.confirmPhrase() !== target.phrase ||
      this.confirmReason().trim().length < 10 ||
      this.isExecuting()
    );
  });

  readonly history = signal<EmergencyHistoryRow[]>([]);
  readonly historyLoading = signal(false);

  ngOnInit(): void {
    this.loadHistory();
  }

  openConfirmation(action: EmergencyActionDef): void {
    this.confirmTarget.set(action);
    this.confirmPhrase.set('');
    this.confirmReason.set('');
    this.confirmError.set(null);
  }

  closeConfirmation(): void {
    if (this.isExecuting()) {
      return;
    }
    this.confirmTarget.set(null);
  }

  executeConfirmed(): void {
    const target = this.confirmTarget();
    if (!target || this.confirmDisabled()) {
      return;
    }

    this.executingAction.set(target.id);
    const reason = this.confirmReason().trim();

    this.auditoriaService.executeEmergencyAction(target.id, reason).subscribe({
      next: (result: EmergencyActionResult) => {
        this.executingAction.set(null);
        this.confirmTarget.set(null);
        this.showToast(result.message, 'success');
        this.loadHistory();
      },
      error: (err: HttpErrorResponse) => {
        this.executingAction.set(null);
        const backend = (err.error as { error?: string; message?: string }) ?? {};
        if (err.status === 422 && backend.error === 'invalid_confirmation_phrase') {
          this.confirmError.set('La frase de confirmación no coincide exactamente.');
        } else if (err.status === 503 && backend.error === 'executor_unavailable') {
          this.confirmError.set(backend.message ?? 'El executor de esta acción no está disponible en esta instancia.');
        } else if (err.status === 403) {
          this.confirmError.set('Solo el rol de administrador puede ejecutar acciones de emergencia.');
        } else {
          this.confirmError.set(backend.message ?? 'No se pudo ejecutar la acción. Intenta nuevamente.');
        }
      }
    });
  }

  private loadHistory(): void {
    this.historyLoading.set(true);
    // El filtro de acción del backend es igualdad exacta: se consultan los
    // 5 eventos EMERGENCY_* del catálogo y se fusionan por fecha desc.
    const events = [
      'EMERGENCY_TERMINATE_ALL',
      'EMERGENCY_HIBERNATE_ALL',
      'EMERGENCY_KILL_ZOMBIES',
      'EMERGENCY_PRUNE_DOCKER',
      'EMERGENCY_RESET_POOLS'
    ];
    let pending = events.length;
    const merged: EmergencyHistoryRow[] = [];
    if (pending === 0) {
      this.historyLoading.set(false);
      return;
    }
    for (const event of events) {
      this.auditoriaService.listAuditLogs(1, 10, { action: event }).subscribe({
        next: (resp) => {
          for (const log of resp.data) {
            merged.push({ log, label: humanizeConstant(log.action) });
          }
          pending--;
          if (pending === 0) {
            merged.sort((a, b) => b.log.created_at.localeCompare(a.log.created_at));
            this.history.set(merged.slice(0, 10));
            this.historyLoading.set(false);
          }
        },
        error: () => {
          pending--;
          if (pending === 0) {
            this.history.set(merged.slice(0, 10));
            this.historyLoading.set(false);
          }
        }
      });
    }
  }

  private showToast(message: string, type: 'success' | 'error'): void {
    this.toast.set({ message, type });
    if (this.toastTimer) {
      clearTimeout(this.toastTimer);
    }
    this.toastTimer = setTimeout(() => this.toast.set(null), 5000);
  }
}
