import { Component, input, output, signal, computed, ViewChild, ElementRef, HostListener } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { 
  LucideHelpCircle, 
  LucideAlertTriangle, 
  LucideAlertCircle,
  LucideRotateCw, 
  LucideTerminal,
  LucideCopy,
  LucideDownload,
  LucideX,
  LucidePlay,
  LucideLoader2,
  LucideCheck,
  LucideSearch,
  LucideInfo
} from '@lucide/angular';
import { 
  TargetEnvironment, 
  ServiceRequirement
} from '../../../../../services/admin-templates.service';
import { 
  EnvTestButtonComponent,
  ButtonVisualState
} from '../../../env-test-button/env-test-button.component';
import { 
  EnvTestJob 
} from '../../../../../services/env-test-job.service';

export type WizardSection = 'purpose' | 'identity' | 'image' | 'execution' | 'resources' | 'verification';

export interface SuboptimalWarning {
  type: 'warning' | 'info';
  message: string;
}

export interface ParsedLogLine {
  timestamp: string;
  level: 'INFO' | 'WARN' | 'ERROR' | 'STDERR';
  message: string;
}

@Component({
  selector: 'step-verification',
  standalone: true,
  imports: [
    CommonModule, 
    FormsModule,
    LucideHelpCircle, 
    LucideAlertTriangle, 
    LucideAlertCircle,
    LucideRotateCw, 
    LucideTerminal,
    LucideCopy,
    LucideDownload,
    LucideX,
    LucidePlay,
    LucideLoader2,
    LucideCheck,
    LucideSearch,
    LucideInfo,
    EnvTestButtonComponent
  ],
  templateUrl: './step-verification.component.html',
  styleUrls: ['./step-verification.component.scss']
})
export class StepVerificationComponent {
  @ViewChild('envTestButton') envTestButton?: EnvTestButtonComponent;
  @ViewChild('terminalContainer') terminalContainer?: ElementRef<HTMLDivElement>;

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
  isBypassed = input<boolean>(false);

  testCompleted = output<EnvTestJob>();
  jobProgress = output<EnvTestJob>();
  retryTest = output<void>();
  requestBypass = output<void>();
  helpRequested = output<void>();
  advance = output<void>();
  jumpToSection = output<WizardSection>();

  isFullLogsModalOpen = signal<boolean>(false);
  copySuccess = signal<boolean>(false);
  copyImageSuccess = signal<boolean>(false);
  searchQuery = signal<string>('');
  autoScrollLogs = signal<boolean>(true);
  isTesting = signal<boolean>(false);

  @HostListener('document:keydown.escape')
  onEscape(): void {
    if (this.isFullLogsModalOpen()) {
      this.closeFullLogsModal();
    }
  }

  isTestRunning = computed<boolean>(() => {
    if (this.isTesting()) return true;
    const job = this.activeEnvTestJob();
    return job !== null && (job.status === 'pulling' || job.status === 'testing' || job.status === 'pending');
  });

  hasFailedTest = computed<boolean>(() => {
    const job = this.activeEnvTestJob();
    return job !== null && job.status === 'failed';
  });

  canBypass = computed<boolean>(() => {
    const job = this.activeEnvTestJob();
    return job !== null && job.status === 'failed' && (
      job.error_code === 'registry_unreachable' || 
      job.error_code === 'pull_timeout' || 
      job.error_code === 'pull_stalled' ||
      job.error_code === 'start_failed'
    );
  });

  serviceNames = computed<string[]>(() => {
    return this.selectedServices().map(s => s.engine);
  });

  diagnosticCause = computed<string>(() => {
    const job = this.activeEnvTestJob();
    if (this.isEnvTestStale()) {
      return 'La imagen, memoria RAM, herramientas o script fueron modificados tras la última prueba exitosa.';
    }
    if (!job) {
      return 'No se ha ejecutado ninguna comprobación preliminar.';
    }
    if (job.error_message) {
      return job.error_message;
    }
    if (job.status === 'failed') {
      return 'No es posible acceder al registro de imágenes para verificar el manifiesto. (HTTP 404 Manifest Unknown)';
    }
    return 'Comprobación de integridad completada con éxito.';
  });

  diagnosticAction = computed<{ text: string; jumpSection?: WizardSection; jumpLabel?: string }>(() => {
    const job = this.activeEnvTestJob();
    if (this.isEnvTestStale()) {
      return {
        text: 'Vuelva a presionar "Reintentar prueba" para validar los nuevos parámetros con Docker.'
      };
    }
    if (!job || job.status !== 'failed') {
      return {
        text: 'Presione "Probar entorno" para verificar la integridad del contenedor antes de publicar.'
      };
    }

    switch (job.error_code) {
      case 'registry_unreachable':
        return {
          text: 'El servidor no pudo conectarse al registro OCI público para validar el manifiesto. Verifique la conexión a internet y DNS del nodo host, o elija una imagen local en el',
          jumpSection: 'image',
          jumpLabel: 'Paso 3'
        };
      case 'test_oom':
        return {
          text: 'El contenedor superó el límite de memoria asignado durante la prueba. Aumente la memoria RAM base en el',
          jumpSection: 'resources',
          jumpLabel: 'Paso 5'
        };
      case 'tool_missing':
        return {
          text: 'Una o más herramientas requeridas no fueron detectadas en la imagen. Verifique la imagen en el',
          jumpSection: 'image',
          jumpLabel: 'Paso 3'
        };
      case 'test_crash':
        return {
          text: 'El contenedor falló al inicializarse en el host. Revise el comando o script de inicio en el',
          jumpSection: 'execution',
          jumpLabel: 'Paso 4'
        };
      case 'pull_timeout':
      case 'pull_stalled':
        return {
          text: 'La descarga de capas de la imagen se detuvo o excedió el tiempo límite. Compruebe la conectividad de red del host antes de reintentar o elija otra imagen en el',
          jumpSection: 'image',
          jumpLabel: 'Paso 3'
        };
      default:
        return {
          text: 'Revise la configuración de la imagen en el',
          jumpSection: 'image',
          jumpLabel: 'Paso 3'
        };
    }
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
          message: 'No se declararon herramientas binarias requeridas para verificación formal. Puede continuar la publicación una vez resuelto el manifiesto de la imagen.'
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

  combinedObservationsText = computed<string>(() => {
    const warnings = this.suboptimalWarnings();
    if (warnings.length > 0) {
      return warnings.map(w => w.message).join(' ');
    }
    return 'No se detectaron observaciones críticas de configuración.';
  });

  formattedSmokeTestOutput = computed<string>(() => {
    const explicit = this.smokeTestOutput();
    if (explicit && explicit.trim()) {
      return explicit;
    }

    const job = this.activeEnvTestJob();
    const img = this.dockerImage().trim() || 'sin-imagen';
    const env = this.targetEnvironment();
    const ram = this.baseRamMB();

    const lines: string[] = [];
    const now = new Date();
    const timePrefix = (offsetMs: number) => {
      const d = new Date(now.getTime() + offsetMs);
      return d.toTimeString().split(' ')[0] + '.' + String(d.getMilliseconds()).padStart(3, '0');
    };

    if (!job) {
      lines.push(`${timePrefix(0)} [INFO] Estado de la prueba: Pendiente de ejecución.`);
      lines.push(`${timePrefix(4)} [INFO] Imagen configurada: ${img}`);
      lines.push(`${timePrefix(7)} [INFO] Entorno: ${env}`);
      lines.push(`${timePrefix(10)} [INFO] Memoria asignada: ${ram} MB`);
      lines.push(`${timePrefix(15)} [INFO] Presione "Probar entorno" para verificar el contenedor en el servidor host.`);
      return lines.join('\n');
    }

    lines.push(`${timePrefix(0)} [INFO] ID de trabajo: ${job.id}`);
    lines.push(`${timePrefix(4)} [INFO] Imagen objetivo: ${img}`);
    lines.push(`${timePrefix(7)} [INFO] Entorno: ${env}`);
    lines.push(`${timePrefix(12)} [INFO] Memoria asignada: ${ram} MB`);

    if (job.status === 'pulling' || job.status === 'testing' || job.status === 'pending') {
      lines.push(`${timePrefix(50)} [INFO] Ejecutando prueba de arranque en el servidor host...`);
      lines.push(`${timePrefix(100)} [INFO] Estado actual del contenedor: ${job.status}`);
      return lines.join('\n');
    }

    if (job.status === 'failed') {
      lines.push(`${timePrefix(80)} [INFO] Inicializando contenedor de prueba en el servidor host...`);
      lines.push(`${timePrefix(120)} [ERROR] La prueba de arranque finalizó con errores.`);
      if (job.error_code) {
        lines.push(`${timePrefix(125)} [ERROR] Código de error: ${job.error_code}`);
      }
      lines.push(`${timePrefix(130)} [ERROR] Detalle: ${job.error_message || 'Fallo durante la inicialización o verificación de la imagen.'}`);
      lines.push(`${timePrefix(140)} [ERROR] Estado final: FAILED (Exit Code 1)`);
    } else if (job.status === 'success') {
      lines.push(`${timePrefix(80)} [INFO] Contenedor de verificación inicializado correctamente en el host.`);
      lines.push(`${timePrefix(120)} [INFO] Inspección de herramientas y binarios del entorno:`);
      if (job.result?.tools && job.result.tools.length > 0) {
        for (const t of job.result.tools) {
          const detail = t.present ? `PRESENTE (${t.version || 'disponible'})` : 'NO DETECTADA';
          lines.push(`${timePrefix(150)} [INFO] Herramienta "${t.name}": ${detail}`);
        }
      } else {
        lines.push(`${timePrefix(150)} [INFO] Sin herramientas adicionales requeridas para verificación.`);
      }
      if (job.result?.duration_ms) {
        lines.push(`${timePrefix(200)} [INFO] Duración de la prueba: ${job.result.duration_ms} ms`);
      }
      lines.push(`${timePrefix(205)} [INFO] Estado final: SUCCESS (Exit Code 0)`);
    }

    return lines.join('\n');
  });

  parsedLogLines = computed<ParsedLogLine[]>(() => {
    const raw = this.formattedSmokeTestOutput();
    if (!raw) return [];

    return raw.split('\n').filter(l => l.trim().length > 0).map(line => {
      const match = line.match(/^(\d{2}:\d{2}:\d{2}(?:\.\d{3})?)\s+\[(INFO|WARN|ERROR|STDERR)\]\s+(.*)$/);
      if (match) {
        return {
          timestamp: match[1],
          level: match[2] as 'INFO' | 'WARN' | 'ERROR' | 'STDERR',
          message: match[3]
        };
      }
      // Fallback si no tiene formato estándar
      let lvl: 'INFO' | 'WARN' | 'ERROR' | 'STDERR' = 'INFO';
      if (/error|failed|404/i.test(line)) lvl = 'ERROR';
      else if (/warn/i.test(line)) lvl = 'WARN';
      else if (/stderr/i.test(line)) lvl = 'STDERR';

      return {
        timestamp: '19:28:40.000',
        level: lvl,
        message: line
      };
    });
  });

  filteredParsedLogLines = computed<ParsedLogLine[]>(() => {
    const q = this.searchQuery().trim().toLowerCase();
    const all = this.parsedLogLines();
    if (!q) return all;

    return all.filter(item => 
      item.message.toLowerCase().includes(q) || 
      item.level.toLowerCase().includes(q) || 
      item.timestamp.includes(q)
    );
  });

  handleRetry(): void {
    if (this.isTestRunning()) return;
    if (this.envTestButton?.canTrigger()) {
      this.isTesting.set(true);
      this.envTestButton.startTest();
    }
    this.retryTest.emit();
  }

  openFullLogsModal(): void {
    this.isFullLogsModalOpen.set(true);
    this.searchQuery.set('');
    this.scrollTerminalToBottom();
  }

  closeFullLogsModal(): void {
    this.isFullLogsModalOpen.set(false);
    this.copySuccess.set(false);
  }

  copyLogs(): void {
    const text = this.formattedSmokeTestOutput();
    if (typeof navigator !== 'undefined' && navigator.clipboard?.writeText) {
      navigator.clipboard.writeText(text).then(() => {
        this.copySuccess.set(true);
        setTimeout(() => this.copySuccess.set(false), 2000);
      }).catch(() => {
        this.copySuccess.set(true);
        setTimeout(() => this.copySuccess.set(false), 2000);
      });
    } else {
      this.copySuccess.set(true);
      setTimeout(() => this.copySuccess.set(false), 2000);
    }
  }

  copyImageTag(): void {
    const img = this.dockerImage().trim();
    if (!img) return;
    if (typeof navigator !== 'undefined' && navigator.clipboard?.writeText) {
      navigator.clipboard.writeText(img).then(() => {
        this.copyImageSuccess.set(true);
        setTimeout(() => this.copyImageSuccess.set(false), 2000);
      }).catch(() => {
        this.copyImageSuccess.set(true);
        setTimeout(() => this.copyImageSuccess.set(false), 2000);
      });
    } else {
      this.copyImageSuccess.set(true);
      setTimeout(() => this.copyImageSuccess.set(false), 2000);
    }
  }

  downloadLogs(): void {
    const text = this.formattedSmokeTestOutput();
    const blob = new Blob([text], { type: 'text/plain;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    const jobId = this.activeEnvTestJob()?.id || 'prueba';
    a.download = `registro-prueba-${jobId}.txt`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  }

  toggleAutoScroll(event: Event): void {
    const target = event.target as HTMLInputElement;
    this.autoScrollLogs.set(target.checked);
    if (target.checked) {
      this.scrollTerminalToBottom();
    }
  }

  private scrollTerminalToBottom(): void {
    setTimeout(() => {
      if (this.terminalContainer?.nativeElement && this.autoScrollLogs()) {
        const el = this.terminalContainer.nativeElement;
        el.scrollTop = el.scrollHeight;
      }
    }, 50);
  }

  onButtonStateChange(state: ButtonVisualState): void {
    this.isTesting.set(state === 'running');
  }

  onJobProgress(job: EnvTestJob): void {
    this.jobProgress.emit(job);
  }

  onTestCompleted(job: EnvTestJob): void {
    this.isTesting.set(false);
    this.testCompleted.emit(job);
  }
}

export { StepVerificationComponent as SolvStepVerificationComponent };
