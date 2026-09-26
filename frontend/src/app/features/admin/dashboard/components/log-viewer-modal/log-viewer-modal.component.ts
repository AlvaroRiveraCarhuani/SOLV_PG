import { Component, input, output, signal, inject, effect, HostListener } from '@angular/core';
import { CommonModule } from '@angular/common';
import { TechnicalIncident } from '@core/models/admin.model';
import { AdminMetricsService } from '../../../services/admin-metrics.service';
import { LucideX, LucideTerminal, LucideCopy, LucideCheck, LucideRefreshCw } from '@lucide/angular';

@Component({
  selector: 'log-viewer-modal',
  standalone: true,
  imports: [CommonModule, LucideX, LucideTerminal, LucideCopy, LucideCheck, LucideRefreshCw],
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
                Instancia: {{ incident()?.workspace_id || 'WS-CONTAINER' }} &bull; Salida en vivo de Docker daemon
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
  styles: [`
    .modal-backdrop {
      position: fixed;
      inset: 0;
      background-color: rgba(15, 23, 42, 0.4);
      backdrop-filter: blur(2px);
      z-index: 1000;
      display: flex;
      align-items: center;
      justify-content: center;
      padding: var(--space-4, 16px);
      animation: fadeIn 150ms ease;
    }

    .modal-card {
      background-color: var(--bg-surface, #FFFFFF);
      border: 1px solid var(--border-subtle, #E2E8F0);
      border-radius: var(--radius-lg, 8px);
      width: 100%;
      max-width: 780px;
      display: flex;
      flex-direction: column;
      box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04);
      overflow: hidden;
      animation: scaleUp 150ms ease;
    }

    .modal-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 14px 20px;
      border-bottom: 1px solid var(--border-subtle, #E2E8F0);
      background-color: var(--bg-surface, #FFFFFF);
    }

    .header-left {
      display: flex;
      align-items: center;
      gap: 12px;
    }

    .terminal-icon {
      width: 18px;
      height: 18px;
      color: var(--text-secondary, #64748B);
    }

    .title-group {
      display: flex;
      flex-direction: column;
      gap: 2px;
    }

    .modal-title {
      font-size: 14px;
      font-weight: 700;
      color: var(--text-primary, #0F172A);
      margin: 0;
    }

    .modal-subtitle {
      font-size: 11px;
      color: var(--text-secondary, #64748B);
      font-family: var(--font-mono, monospace);
    }

    .header-actions {
      display: flex;
      align-items: center;
      gap: 8px;
    }

    .btn-action {
      display: flex;
      align-items: center;
      gap: 6px;
      background: var(--bg-surface, #FFFFFF);
      border: 1px solid var(--border-subtle, #CBD5E1);
      color: var(--text-secondary, #64748B);
      border-radius: var(--radius-md, 6px);
      padding: 4px 10px;
      font-size: 11px;
      font-weight: 600;
      cursor: pointer;
      transition: all 150ms ease;

      &:hover:not(:disabled) {
        background-color: var(--bg-canvas, #F1F5F9);
        color: var(--text-primary, #0F172A);
      }

      &:disabled {
        opacity: 0.6;
        cursor: not-allowed;
      }

      .action-icon {
        width: 13px;
        height: 13px;

        &.success {
          color: var(--state-running, #16A34A);
        }

        &.spin {
          animation: spin 1s linear infinite;
        }
      }
    }

    .btn-close {
      background: transparent;
      border: none;
      color: var(--text-muted, #94A3B8);
      cursor: pointer;
      padding: 4px;
      display: flex;
      align-items: center;
      border-radius: var(--radius-sm, 4px);
      transition: all 150ms ease;

      &:hover {
        background-color: var(--bg-canvas, #F1F5F9);
        color: var(--text-primary, #0F172A);
      }

      .close-icon {
        width: 16px;
        height: 16px;
      }
    }

    .terminal-body {
      padding: 16px 20px;
      background-color: var(--bg-canvas, #F8FAFC);
      border-bottom: 1px solid var(--border-subtle, #E2E8F0);
      font-family: var(--font-mono, monospace);
      font-size: 12px;
      line-height: 1.6;
      min-height: 200px;
      max-height: 380px;
      overflow-y: auto;
    }

    .terminal-output {
      margin: 0;
      color: var(--text-primary, #0F172A);
      font-family: inherit;
      font-size: inherit;
      line-height: inherit;
      white-space: pre-wrap;
      word-break: break-word;
    }

    .terminal-loading, .terminal-empty {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 8px;
      padding: 40px 16px;
      color: var(--text-muted, #94A3B8);
      font-size: 12px;
    }

    .loading-spinner {
      width: 16px;
      height: 16px;
    }

    .modal-footer {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 12px 20px;
      background-color: var(--bg-surface, #FFFFFF);
    }

    .footer-meta {
      display: flex;
      align-items: center;
      gap: 12px;
    }

    .badge-status {
      font-size: 11px;
      background-color: #FEF2F2;
      border: 1px solid #FECACA;
      color: var(--verdict-wa, #DC2626);
      padding: 2px 8px;
      border-radius: var(--radius-sm, 4px);
      font-weight: 600;
    }

    .meta-tip {
      font-size: 11px;
      color: var(--text-secondary, #64748B);
    }

    .btn-primary {
      padding: 6px 16px;
      background-color: var(--tenant-primary, #2563EB);
      border: none;
      border-radius: var(--radius-md, 6px);
      color: #FFFFFF;
      font-size: 12px;
      font-weight: 600;
      cursor: pointer;
      transition: background-color 150ms ease;

      &:hover {
        background-color: #1D4ED8;
      }
    }

    @keyframes spin {
      from { transform: rotate(0deg); }
      to { transform: rotate(360deg); }
    }

    @keyframes fadeIn {
      from { opacity: 0; }
      to { opacity: 1; }
    }

    @keyframes scaleUp {
      from { opacity: 0; transform: scale(0.98); }
      to { opacity: 1; transform: scale(1); }
    }
  `]
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
