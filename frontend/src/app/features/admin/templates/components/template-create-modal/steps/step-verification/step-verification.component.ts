import { Component, input, output, signal, computed, ViewChild } from '@angular/core';
import { CommonModule } from '@angular/common';
import { 
  LucideHelpCircle
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
    EnvTestButtonComponent
  ],
  template: `
    <div class="verification-step-container">
      <div class="step-header-with-help mb-3">
        <h4 class="step-section-title" id="step-title-verification" tabindex="-1" i18n="@@PU-21-HEADING">
          Verificación y smoke test
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
                La imagen Docker o la asignación de memoria cambiaron tras la última ejecución. Vuelva a probar el entorno para asegurar la validez de la plantilla antes de publicar.
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
          <p class="test-card-desc">
            {{ targetEnvironment() === 'JUEZ_EFIMERO' ? 'Ejecución del comando en sandbox efímero aislado sin red midiendo tiempo y veredicto.' : 'Comprobación de arranque y presencia de binarios requeridos.' }}
          </p>
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
      </div>

      <!-- Resumen Técnico de la Plantilla con Advertencias Subóptimas -->
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
      </div>
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

  testCompleted = output<EnvTestJob>();
  retryTest = output<void>();
  helpRequested = output<void>();
  advance = output<void>();

  showLogs = signal<boolean>(false);

  handleRetry(): void {
    if (this.envTestButton) {
      this.envTestButton.startTest();
    }
    this.retryTest.emit();
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
    lines.push(`=== SOLV SMOKE TEST RUNNER ===`);
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
