import { 
  Component, 
  input, 
  output, 
  signal, 
  computed, 
  inject, 
  OnDestroy,
  effect,
  untracked 
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { Subscription } from 'rxjs';
import { 
  EnvTestJobService, 
  EnvTestJob, 
  ToolResult 
} from '../../../services/env-test-job.service';
import { 
  LucidePlay, 
  LucideLoader2, 
  LucideCheckCircle2, 
  LucideAlertTriangle, 
  LucideXCircle, 
  LucideX,
  LucideChevronDown,
  LucideChevronUp
} from '@lucide/angular';

export type ButtonVisualState = 'idle' | 'running' | 'ok' | 'missing' | 'error';

export const ENV_TEST_ERROR_MESSAGES: Record<string, string> = {
  pull_stalled: $localize`:@@TE-75:Descarga detenida: sin datos de red por más de 60 s`,
  pull_timeout: $localize`:@@TE-77:Tiempo total de prueba excedido (15 min)`,
  registry_unreachable: $localize`:@@TE-76:No se pudo conectar al registro OCI público`,
  test_oom: $localize`:@@TE-71:Límite de memoria superado durante la prueba (OOM)`,
  test_crash: $localize`:@@TE-72:Fallo al ejecutar el contenedor de prueba`,
  internal: $localize`:@@TE-70:Error interno en la prueba de entorno`
};

export function getEnvTestErrorMessage(code: string): string {
  return ENV_TEST_ERROR_MESSAGES[code] || ENV_TEST_ERROR_MESSAGES['internal'];
}

@Component({
  selector: 'env-test-button',
  standalone: true,
  imports: [
    CommonModule,
    MachineDataDirective,
    LucidePlay,
    LucideLoader2,
    LucideCheckCircle2,
    LucideAlertTriangle,
    LucideXCircle,
    LucideX,
    LucideChevronDown,
    LucideChevronUp
  ],
  templateUrl: './env-test-button.component.html',
  styleUrls: ['./env-test-button.component.scss']
})
export class EnvTestButtonComponent implements OnDestroy {
  private envTestService = inject(EnvTestJobService);

  // Inputs con signals nativos de Angular 22
  image = input<string>('');
  tools = input<string[]>([]);
  isLocal = input<boolean>(false);
  imageSizeMB = input<number>(0);
  hasDigestMismatch = input<boolean>(false);
  targetEnvironment = input<string>('IDE_PERSISTENTE');
  entrypoint = input<string>('');
  timeoutMS = input<number>(5000);
  sampleInput = input<string>('');

  // Outputs con signals nativos de Angular 22
  testCompleted = output<EnvTestJob>();
  stateChange = output<ButtonVisualState>();

  // Estado interno reactivo
  visualState = signal<ButtonVisualState>('idle');
  activeJob = signal<EnvTestJob | null>(null);
  isExpanded = signal<boolean>(false);
  errorMessage = signal<string>('');
  errorCode = signal<string>('');

  private pollSub: Subscription | null = null;

  constructor() {
    // Si cambia la imagen o las tools, reiniciamos el estado a idle si no está corriendo
    effect(() => {
      this.image();
      this.tools();
      this.targetEnvironment();
      this.entrypoint();

      untracked(() => {
        if (this.visualState() !== 'running') {
          this.visualState.set('idle');
          this.activeJob.set(null);
          this.isExpanded.set(false);
        }
      });
    });
  }

  // Textos y etiquetas dinámicas para el estado IDLE
  idleLabel = computed(() => {
    const rawImage = this.image().trim();
    if (!rawImage || rawImage.endsWith(':latest')) {
      return 'Probar entorno';
    }
    if (this.targetEnvironment() === 'JUEZ_EFIMERO') {
      return 'Probar ejecución de juez (sandbox CLI)';
    }
    if (this.hasDigestMismatch()) {
      return 'Actualizar imagen y probar';
    }
    if (this.isLocal()) {
      return 'Probar entorno (local, < 2s)';
    }
    const size = this.imageSizeMB();
    if (size > 0) {
      return `Descargar y probar (~${size} MB)`;
    }
    return 'Descargar y probar entorno';
  });

  isLatestImage = computed(() => {
    const raw = this.image().trim().toLowerCase();
    return raw.endsWith(':latest');
  });

  canTrigger = computed(() => {
    const img = this.image().trim();
    if (!img || this.isLatestImage()) return false;
    if (this.targetEnvironment() === 'JUEZ_EFIMERO' && !this.entrypoint().trim()) return false;
    return this.visualState() !== 'running';
  });

  disabledReason = computed(() => {
    const missing: string[] = [];
    const rawImage = this.image().trim();
    if (!rawImage) {
      missing.push('imagen');
    } else if (this.isLatestImage()) {
      missing.push('etiqueta de imagen válida (no :latest)');
    }
    if (this.targetEnvironment() === 'JUEZ_EFIMERO' && !this.entrypoint().trim()) {
      missing.push('comando de ejecución');
    }
    if (missing.length > 0) {
      return `Falta configurar para habilitar la prueba: ${missing.join(', ')}.`;
    }
    return $localize`:@@TE-02:Ingrese y verifique una imagen válida para habilitar la prueba del entorno.`;
  });

  startTest(): void {
    if (!this.canTrigger()) return;

    this.visualState.set('running');
    this.stateChange.emit('running');
    this.errorMessage.set('');
    this.errorCode.set('');
    this.isExpanded.set(false);

    const req = {
      image: this.image().trim(),
      tools: this.tools(),
      target_environment: this.targetEnvironment(),
      entrypoint: this.entrypoint(),
      timeout_ms: this.timeoutMS(),
      sample_input: this.sampleInput()
    };

    this.envTestService.startJob(req).subscribe({
      next: (job) => {
        this.activeJob.set(job);
        this.pollJob(job.id);
      },
      error: (err) => {
        this.visualState.set('error');
        this.stateChange.emit('error');
        this.errorMessage.set(err.error?.message || 'Error al solicitar la prueba de entorno');
        this.errorCode.set(err.error?.error || 'start_failed');
      }
    });
  }

  cancelMessage = signal<string>('');

  cancelTest(): void {
    const job = this.activeJob();
    if (!job || this.visualState() !== 'running') return;

    this.cancelMessage.set($localize`:@@TE-35:Prueba cancelada. Las capas ya descargadas quedan en caché.`);

    this.envTestService.cancelJob(job.id).subscribe({
      next: () => {
        this.stopPolling();
        this.visualState.set('idle');
        this.stateChange.emit('idle');
        this.activeJob.set(null);
      },
      error: () => {
        this.stopPolling();
        this.visualState.set('idle');
        this.stateChange.emit('idle');
      }
    });
  }

  toggleDetails(): void {
    this.isExpanded.update(v => !v);
  }

  private pollJob(id: string): void {
    this.stopPolling();
    this.pollSub = this.envTestService.pollJob(id, 1500).subscribe({
      next: (job) => {
        this.activeJob.set(job);
        if (job.status === 'success') {
          this.visualState.set('ok');
          this.stateChange.emit('ok');
          this.testCompleted.emit(job);
          this.stopPolling();
        } else if (job.status === 'failed') {
          // Si falló por herramientas faltantes mostramos missing, sino error
          const hasMissingTools = job.result?.tools?.some(t => !t.present);
          if (hasMissingTools) {
            this.visualState.set('missing');
            this.stateChange.emit('missing');
          } else {
            this.visualState.set('error');
            this.stateChange.emit('error');
          }
          this.errorMessage.set(job.error_message || 'La prueba de entorno no fue superada');
          this.errorCode.set(job.error_code || 'failed');
          this.testCompleted.emit(job);
          this.stopPolling();
        } else if (job.status === 'canceled') {
          this.visualState.set('idle');
          this.stateChange.emit('idle');
          this.stopPolling();
        }
      },
      error: (err) => {
        this.visualState.set('error');
        this.stateChange.emit('error');
        this.errorMessage.set(err.error?.message || 'Fallo de comunicación durante la prueba');
        this.stopPolling();
      }
    });
  }

  private stopPolling(): void {
    if (this.pollSub) {
      this.pollSub.unsubscribe();
      this.pollSub = null;
    }
  }

  ngOnDestroy(): void {
    this.stopPolling();
  }
}
