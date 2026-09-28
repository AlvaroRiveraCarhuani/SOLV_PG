import { Component, input, output, signal, inject, effect, HostListener } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { TechnicalIncident } from '@core/models/admin.model';
import { AdminMetricsService } from '../../../services/admin-metrics.service';
import { LucideX, LucideTerminal, LucideCopy, LucideCheck, LucideRefreshCw } from '@lucide/angular';

@Component({
  selector: 'log-viewer-modal',
  standalone: true,
  imports: [CommonModule, MachineDataDirective, LucideX, LucideTerminal, LucideCopy, LucideCheck, LucideRefreshCw],
  template: `
    <div class="modal-backdrop" (click)="close.emit()">
      <div class="modal-card" (click)="$event.stopPropagation()">
        <!-- Header -->
        <div class="modal-header">
          <div class="header-left">
            <svg lucideTerminal class="terminal-icon"></svg>
            <div class="title-group">
              <h3 class="modal-title">Registro de Ejecución (Logs del Contenedor)</h3>
              <span class="modal-subtitle">
                Instancia: <span machineData>{{ incident()?.workspace_id || 'WS-CONTAINER' }}</span> &bull; Salida en vivo de Docker daemon
              </span>
            </div>
          </div>
          <div class="header-actions">
            <button class="btn-action" (click)="loadLogs()" [disabled]="isLoading()" title="Recargar logs">
              <svg lucideRefreshCw class="action-icon" [class.spin]="isLoading()"></svg>
              <span>Refrescar</span>
            </button>
            <button class="btn-action" (click)="copyLogs()" title="Copiar logs al portapapeles">
              @if (copied()) {
                <svg lucideCheck class="action-icon success"></svg>
                <span>Copiado</span>
              } @else {
                <svg lucideCopy class="action-icon"></svg>
                <span>Copiar</span>
              }
            </button>
            <button class="btn-close" (click)="close.emit()" title="Cerrar ventana">
              <svg lucideX class="close-icon"></svg>
            </button>
          </div>
        </div>

        <!-- Terminal Body -->
        <div class="terminal-body">
          @if (isLoading()) {
            <div class="terminal-loading">
              <svg lucideRefreshCw class="loading-spinner spin"></svg>
              <span>Consultando registros en el motor Docker del host...</span>
            </div>
          } @else if (logsContent()) {
            <pre class="terminal-output">{{ logsContent() }}</pre>
          } @else {
            <div class="terminal-empty">
              <span>No hay registros disponibles en el buffer de salida de este contenedor.</span>
            </div>
          }
        </div>

        <!-- Footer -->
        <div class="modal-footer">
          <div class="footer-meta">
            <span class="badge-status">Estado: Error 137 (SIGKILL por OOM)</span>
            <span class="meta-tip">El recolector QoS o el kernel detuvieron el proceso por superar la cuota máxima permitida.</span>
          </div>
          <button class="btn-primary" (click)="close.emit()">Entendido</button>
        </div>
      </div>
    </div>
  `,
  styleUrl: './log-viewer-modal.component.scss',
})
export class LogViewerModalComponent {
  private metricsService = inject(AdminMetricsService);

  incident = input<TechnicalIncident | null>(null);
  close = output<void>();

  @HostListener('document:keydown.escape')
  handleEscape(): void {
    this.close.emit();
  }

  logsContent = signal<string>('');
  isLoading = signal<boolean>(false);
  copied = signal<boolean>(false);

  constructor() {
    effect(() => {
      const inc = this.incident();
      if (inc?.workspace_id) {
        this.loadLogs();
      }
    });
  }

  loadLogs(): void {
    const inc = this.incident();
    if (!inc?.workspace_id) return;

    this.isLoading.set(true);
    this.metricsService.getWorkspaceLogs(inc.workspace_id).subscribe({
      next: (res) => {
        this.isLoading.set(false);
        this.logsContent.set(res.logs || 'Sin registros en el búfer de Docker daemon.');
      },
      error: () => {
        this.isLoading.set(false);
        this.logsContent.set(`[${new Date().toISOString()}] [docker:daemon] No se pudieron recuperar los registros del contenedor.`);
      }
    });
  }

  copyLogs(): void {
    const text = this.logsContent();
    if (!text) return;

    navigator.clipboard.writeText(text).then(() => {
      this.copied.set(true);
      setTimeout(() => this.copied.set(false), 2500);
    });
  }
}
