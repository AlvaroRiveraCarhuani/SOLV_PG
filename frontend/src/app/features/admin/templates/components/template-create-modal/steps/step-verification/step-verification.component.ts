import { Component, input, output, signal, computed, ViewChild } from '@angular/core';
import { CommonModule } from '@angular/common';
import { 
  LucideHelpCircle, 
  LucideAlertTriangle, 
  LucideAlertCircle,
  LucideRotateCw, 
  LucideTerminal,
  LucideChevronDown,
  LucideChevronUp,
  LucideCopy,
  LucideDownload,
  LucideX
} from '@lucide/angular';
import { 
  TargetEnvironment, 
  ServiceRequirement 
} from '../../../../../services/admin-templates.service';
import { 
  EnvTestButtonComponent 
} from '../../../env-test-button/env-test-button.component';
import { 
  EnvTestJob 
} from '../../../../../services/env-test-job.service';

export interface SuboptimalWarning {
  type: 'warning' | 'info';
  message: string;
}

@Component({
  selector: 'solv-step-verification',
  standalone: true,
  imports: [
    CommonModule, 
    LucideHelpCircle, 
    LucideAlertTriangle, 
    LucideAlertCircle,
    LucideRotateCw, 
    LucideTerminal,
    LucideChevronDown,
    LucideChevronUp,
    LucideCopy,
    LucideDownload,
    LucideX,
    EnvTestButtonComponent
  ],
  template: `
    <div class="verification-step-container">
      <div class="step-header-with-help mb-3">
        <h4 class="step-section-title" id="step-title-verification" tabindex="-1" i18n="@@PU-21-HEADING">
          Verificación y prueba de arranque
        </h4>
        <button 
          type="button" 
          class="btn-step-help" 
          (click)="helpRequested.emit()" 
          title="Ayuda contextual del paso" 
          aria-label="Ayuda contextual del paso"
          i18n-aria-label="@@AY-02"
        >
          <svg lucideHelpCircle class="w-4 h-4"></svg>
        </button>
      </div>

      <!-- Alerta de prueba obsoleta si cambiaron imagen o RAM -->
      @if (isEnvTestStale()) {
        <div class="stale-test-alert mb-3" role="alert">
          <div class="stale-alert-content">
            <svg lucideAlertTriangle class="w-5 h-5 text-warning shrink-0"></svg>
            <div class="stale-alert-text">
              <strong class="stale-title">Prueba obsoleta</strong>
              <p class="stale-desc">
                Prueba obsoleta: la configuración cambió tras la última verificación. Vuelva a ejecutar la prueba para validar la plantilla antes de publicar.
              </p>
            </div>
          </div>
          <button 
            type="button" 
            class="btn-retry-stale" 
            (click)="handleRetry()"
            title="Reintentar prueba de entorno"
          >
            <svg lucideRotateCw class="w-3.5 h-3.5 mr-1"></svg>
            <span>Re-ejecutar prueba</span>
          </button>
        </div>
      }

      <div class="env-test-card">
        <div class="test-card-header">
          <h5 class="test-card-title">Prueba de Integridad del Entorno</h5>
        </div>

        <solv-env-test-button
          #envTestButton
          [image]="dockerImage()"
          [tools]="toolsList()"
          [targetEnvironment]="targetEnvironment()"
          [entrypoint]="entrypoint()"
          [timeoutMS]="timeoutMS()"
          [sampleInput]="sampleInput()"
          [isLocal]="isLocalImage()"
          [imageSizeMB]="imageSizeMB()"
          (testCompleted)="onTestCompleted($event)"
        ></solv-env-test-button>

        <!-- Bloque de ayuda contextual debajo del botón de prueba -->
        <div class="test-help-block mt-3">
          <p class="test-card-desc">
            {{ targetEnvironment() === 'JUEZ_EFIMERO' ? 'Ejecución del comando en sandbox efímero aislado sin red midiendo tiempo y veredicto.' : 'Comprobación de arranque y presencia de binarios requeridos.' }}
          </p>
        </div>

        <!-- Panel expandible de logs de prueba de arranque -->
        @if (activeEnvTestJob()) {
          <div class="logs-toggle-row mt-3">
            <button 
              type="button" 
              class="btn-toggle-logs"
              (click)="showLogs.set(!showLogs())"
              [attr.aria-expanded]="showLogs()"
            >
              <svg lucideTerminal class="w-3.5 h-3.5 mr-1 text-primary"></svg>
              <span>{{ showLogs() ? 'Ocultar logs de ejecución' : 'Ver logs de ejecución' }}</span>
              @if (showLogs()) {
                <svg lucideChevronUp class="w-3.5 h-3.5 ml-1"></svg>
              } @else {
                <svg lucideChevronDown class="w-3.5 h-3.5 ml-1"></svg>
              }
            </button>

            <button 
              type="button" 
              class="btn-retry-action"
              (click)="handleRetry()"
              title="Disparar nueva ejecución de prueba"
            >
              <svg lucideRotateCw class="w-3.5 h-3.5 mr-1"></svg>
              <span>Reintentar</span>
            </button>
          </div>

          @if (showLogs()) {
            <div class="logs-panel-box animate-fade mt-2">
              <div class="logs-panel-header">
                <span class="logs-panel-title">Salida de la prueba de arranque (vista previa)</span>
                <button 
                  type="button" 
                  class="btn-link-action" 
                  (click)="openFullLogsModal()"
                  title="Abrir modal con registro completo"
                >
                  <svg lucideTerminal class="w-3.5 h-3.5 mr-1"></svg>
                  <span>Ver log completo</span>
                </button>
              </div>
              <pre class="logs-terminal font-mono"><code>{{ previewOutput() }}</code></pre>
            </div>
          }

          <!-- Diagnóstico de Fallo Estructurado (Hecho - Causa - Próxima Acción) -->
          @if (activeEnvTestJob()?.status === 'failed') {
            <div class="diagnostic-failure-box mt-3 animate-fade" role="alert">
              <div class="diagnostic-header">
                <svg lucideAlertCircle class="w-4 h-4 text-danger mr-1"></svg>
                <strong class="diagnostic-title">Diagnóstico de la Verificación</strong>
              </div>
              <div class="diagnostic-body">
                <div class="diagnostic-row">
                  <span class="diagnostic-label">Hecho:</span>
                  <span class="diagnostic-val">La prueba de integridad del entorno falló durante la ejecución.</span>
                </div>
                <div class="diagnostic-row">
                  <span class="diagnostic-label">Causa:</span>
                  <span class="diagnostic-val">{{ activeEnvTestJob()?.error_message || 'Uno o más binarios requeridos no fueron detectados o el contenedor terminó con código de error.' }}</span>
                </div>
                <div class="diagnostic-row">
                  <span class="diagnostic-label">Próxima acción:</span>
                  <span class="diagnostic-val">Revise las herramientas declaradas en el Paso 3 o corrija los parámetros en el Paso 4 antes de reintentar.</span>
                </div>
              </div>
            </div>
          }
        }
      </div>

      <!-- Resumen Técnico de la Plantilla con Advertencias Subóptimas y Badges -->
      <div class="verification-summary-card mt-4">
        <div class="summary-header">
          <span class="font-semibold text-sm">Resumen Técnico de la Plantilla</span>
        </div>
        <div class="summary-grid">
          <div class="summary-item">
            <span class="summary-label">Propósito:</span>
            <span class="summary-val font-semibold">
              {{ targetEnvironment() === 'JUEZ_EFIMERO' ? 'Juez Virtual (CLI Sandbox)' : 'Laboratorio Interactivo (IDE)' }}
            </span>
          </div>
          <div class="summary-item">
            <span class="summary-label">Nombre:</span>
            <span class="summary-val">{{ name() || 'Sin definir' }}</span>
          </div>
          <div class="summary-item">
            <span class="summary-label">Imagen:</span>
            <span class="summary-val font-mono">{{ dockerImage() || 'Sin definir' }}</span>
          </div>
          <div class="summary-item">
            <span class="summary-label">Memoria:</span>
            <span class="summary-val font-mono">{{ baseRamMB() }} MB</span>
          </div>
          @if (targetEnvironment() === 'JUEZ_EFIMERO') {
            <div class="summary-item">
              <span class="summary-label">Comando:</span>
              <span class="summary-val font-mono">{{ entrypoint() || 'Pendiente' }}</span>
            </div>
            <div class="summary-item">
              <span class="summary-label">Timeout:</span>
              <span class="summary-val font-mono">{{ timeoutMS() }} ms</span>
            </div>
          } @else {
            <div class="summary-item">
              <span class="summary-label">Servicios:</span>
              <span class="summary-val">
                {{ selectedServices().length > 0 ? serviceNames().join(', ') : 'Ninguno' }}
              </span>
            </div>
          }
        </div>

        <!-- Badges de advertencia en el resumen técnico -->
        @if (summaryWarningBadges().length > 0) {
          <div class="summary-badges-row">
            @for (badge of summaryWarningBadges(); track badge) {
              <span class="badge-tag-warning">{{ badge }}</span>
            }
          </div>
        }

        <!-- Advertencias de Configuración Subóptima -->
        @if (suboptimalWarnings().length > 0) {
          <div class="suboptimal-warnings-box mt-3">
            <div class="warnings-header">
              <svg lucideAlertTriangle class="w-4 h-4 text-warning mr-1"></svg>
              <span class="warnings-title">Observaciones de configuración:</span>
            </div>
            <ul class="warnings-list">
              @for (warn of suboptimalWarnings(); track warn.message) {
                <li class="warning-item" [class.item-warning]="warn.type === 'warning'" [class.item-info]="warn.type === 'info'">
                  {{ warn.message }}
                </li>
              }
            </ul>
          </div>
        }
      </div>

      <!-- Modal Visor de Logs Completo con Copiar y Descargar -->
      @if (isFullLogsModalOpen()) {
        <div class="full-logs-modal-backdrop" (click)="closeFullLogsModal()">
          <div class="full-logs-modal-card" (click)="$event.stopPropagation()">
            <div class="logs-modal-header">
              <div class="logs-modal-title-wrap">
                <svg lucideTerminal class="w-4 h-4 text-primary mr-2"></svg>
                <h5 class="logs-modal-title" i18n="@@PU-22">Registro Completo de Ejecución (Prueba de Arranque)</h5>
              </div>
              <button type="button" class="btn-icon-close" (click)="closeFullLogsModal()" aria-label="Cerrar visor de logs">
                <svg lucideX class="w-4 h-4"></svg>
              </button>
            </div>
            <div class="logs-modal-body">
              <pre class="full-logs-content font-mono"><code>{{ formattedSmokeTestOutput() }}</code></pre>
            </div>
            <div class="logs-modal-footer">
              <div class="footer-left-actions">
                <button type="button" class="btn btn-outline-secondary btn-sm" (click)="copyLogs()">
                  <svg lucideCopy class="w-3.5 h-3.5 mr-1"></svg>
                  <span>{{ copySuccess() ? '¡Copiado!' : 'Copiar log' }}</span>
                </button>
                <button type="button" class="btn btn-outline-secondary btn-sm" (click)="downloadLogs()">
                  <svg lucideDownload class="w-3.5 h-3.5 mr-1"></svg>
                  <span>Descargar log (.txt)</span>
                </button>
              </div>
              <button type="button" class="btn btn-secondary btn-sm" (click)="closeFullLogsModal()">
                Cerrar
              </button>
            </div>
          </div>
        </div>
      }
    </div>
  `,
  styleUrls: ['./step-verification.component.scss']
})
export class SolvStepVerificationComponent {
  @ViewChild('envTestButton') envTestButton?: EnvTestButtonComponent;

  dockerImage = input<string>('');
  toolsList = input<string[]>([]);
  targetEnvironment = input<TargetEnvironment>('IDE_PERSISTENTE');
  entrypoint = input<string>('');
  timeoutMS = input<number>(5000);
  sampleInput = input<string>('');
  name = input<string>('');
  baseRamMB = input<number>(1024);
  selectedServices = input<ServiceRequirement[]>([]);
  isLocalImage = input<boolean>(false);
  imageSizeMB = input<number>(0);
  activeEnvTestJob = input<EnvTestJob | null>(null);
  isEnvTestStale = input<boolean>(false);
  smokeTestOutput = input<string>('');
  isRamExceedingHost = input<boolean>(false);
  maxAllowedRamMB = input<number>(0);

  testCompleted = output<EnvTestJob>();
  retryTest = output<void>();
  helpRequested = output<void>();
  advance = output<void>();

  showLogs = signal<boolean>(false);
  isFullLogsModalOpen = signal<boolean>(false);
  copySuccess = signal<boolean>(false);

  handleRetry(): void {
    if (this.envTestButton) {
      this.envTestButton.startTest();
    }
    this.retryTest.emit();
  }

  openFullLogsModal(): void {
    this.isFullLogsModalOpen.set(true);
  }

  closeFullLogsModal(): void {
    this.isFullLogsModalOpen.set(false);
    this.copySuccess.set(false);
  }

  copyLogs(): void {
    navigator.clipboard.writeText(this.formattedSmokeTestOutput()).then(() => {
      this.copySuccess.set(true);
      setTimeout(() => this.copySuccess.set(false), 2000);
    });
  }

  downloadLogs(): void {
    const text = this.formattedSmokeTestOutput();
    const blob = new Blob([text], { type: 'text/plain;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    const jobId = this.activeEnvTestJob()?.id || 'smoke-test';
    a.download = `smoke-test-${jobId}.txt`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  }

  serviceNames = computed<string[]>(() => {
    return this.selectedServices().map(s => s.engine);
  });

  formattedSmokeTestOutput = computed<string>(() => {
    const explicit = this.smokeTestOutput();
    if (explicit && explicit.trim()) {
      return explicit;
    }

    const job = this.activeEnvTestJob();
    if (!job) {
      return 'No se ha ejecutado ninguna prueba todavía.';
    }

    const lines: string[] = [];
    lines.push(`=== SOLV PRUEBA DE ARRANQUE ===`);
    lines.push(`ID: ${job.id}`);
    lines.push(`Imagen: ${job.image}`);
    lines.push(`Entorno: ${job.target_environment || this.targetEnvironment()}`);
    lines.push(`Estado: ${job.status.toUpperCase()}`);

    if (job.result) {
      lines.push(`Duración: ${job.result.duration_ms} ms`);
      lines.push(`Exit Code: ${job.result.exit_code}`);
      lines.push(`--- Herramientas ---`);
      for (const t of job.result.tools) {
        lines.push(`  [${t.present ? 'PRESENTE' : 'FALTANTE'}] ${t.name} -> ${t.version || t.path || 'no disponible'}`);
      }
    }

    if (job.error_message) {
      lines.push(`--- Error ---`);
      lines.push(`Mensaje: ${job.error_message}`);
    }

    return lines.join('\n');
  });

  previewOutput = computed<string>(() => {
    const text = this.formattedSmokeTestOutput();
    const lines = text.split('\n');
    if (lines.length <= 5) return text;
    return '...\n' + lines.slice(-5).join('\n');
  });

  summaryWarningBadges = computed<string[]>(() => {
    const badges: string[] = [];
    if (this.isRamExceedingHost()) {
      badges.push('RAM sobre capacidad');
    }
    if (this.targetEnvironment() === 'IDE_PERSISTENTE' && this.toolsList().length === 0) {
      badges.push('Bajo piso de toolchain');
    }
    if (this.isEnvTestStale()) {
      badges.push('Prueba obsoleta');
    }
    const img = this.dockerImage().toLowerCase().trim();
    if (img && img.includes('/') && !img.startsWith('solv/') && !img.startsWith('library/')) {
      badges.push('Mantenedor no oficial');
    }
    if (/python:(?:2\.|3\.[0-7]\b)|node:(?:1[0-4]\b)|ubuntu:(?:1[46]\.04)/.test(img)) {
      badges.push('Versión en fin de vida');
    }
    return badges;
  });

  suboptimalWarnings = computed<SuboptimalWarning[]>(() => {
    const list: SuboptimalWarning[] = [];
    const env = this.targetEnvironment();
    const ram = this.baseRamMB();
    const tools = this.toolsList();

    if (env === 'IDE_PERSISTENTE') {
      if (ram < 512) {
        list.push({
          type: 'warning',
          message: 'Asignación de RAM ajustada (256 MB): OpenVSCode Server y procesos concurrentes podrían requerir mayor margen de memoria.'
        });
      }
      if (tools.length === 0) {
        list.push({
          type: 'info',
          message: 'No se declararon herramientas binarias requeridas para verificación.'
        });
      }
    } else {
      if (ram < 128) {
        list.push({
          type: 'warning',
          message: 'Memoria de Juez muy baja (<128 MB): pruebas de compilación complejas podrían fallar por OOM.'
        });
      }
      if (this.timeoutMS() > 10000) {
        list.push({
          type: 'warning',
          message: 'Tiempo de timeout elevado (>10 s): bucles no terminados podrían demorar la liberación del sandbox.'
        });
      }
    }

    return list;
  });

  onTestCompleted(job: EnvTestJob): void {
    this.testCompleted.emit(job);
  }
}
