import { Component, input, output, computed, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { 
  LucideAlertTriangle, 
  LucideXCircle, 
  LucideCheckCircle2, 
  LucideX,
  LucideSend
} from '@lucide/angular';
import { EnvTestJob } from '../../../services/env-test-job.service';

export interface PreflightCheckItem {
  id: string;
  type: 'pass' | 'warning' | 'error';
  message: string;
  detail?: string;
}

@Component({
  selector: 'publish-dialog',
  standalone: true,
  imports: [
    CommonModule,
    LucideAlertTriangle,
    LucideXCircle,
    LucideCheckCircle2,
    LucideX,
    LucideSend
  ],
  templateUrl: './publish-dialog.component.html',
  styleUrls: ['./publish-dialog.component.scss']
})
export class PublishDialogComponent {
  // Inputs
  templateName = input<string>('');
  dockerImage = input<string>('');
  baseRamMB = input<number>(512);
  tools = input<string[]>([]);
  setupScript = input<string>('');
  services = input<string[]>([]);
  envTestJob = input<EnvTestJob | null>(null);

  // Estado del visor de script
  showScriptModal = signal<boolean>(false);

  // Outputs nativos
  confirmed = output<void>();
  closed = output<void>();

  toggleScriptModal(): void {
    this.showScriptModal.update(v => !v);
  }

  // Pre-flight checks reactivos
  checks = computed<PreflightCheckItem[]>(() => {
    const list: PreflightCheckItem[] = [];
    const rawImage = this.dockerImage().trim();
    const job = this.envTestJob();

    // 1. Imagen y tags
    if (!rawImage) {
      list.push({
        id: 'img-empty',
        type: 'error',
        message: 'No se ha ingresado una imagen Docker.',
        detail: 'Es obligatorio especificar una referencia de imagen OCI válida.'
      });
    } else if (rawImage.endsWith(':latest')) {
      list.push({
        id: 'img-latest',
        type: 'error',
        message: 'El tag :latest está prohibido.',
        detail: 'Por reproducibilidad académica y gobernanza debe especificar una versión fija.'
      });
    } else {
      list.push({
        id: 'img-ok',
        type: 'pass',
        message: 'Formato de imagen OCI válido.',
        detail: rawImage
      });
    }

    // 2. Estado de la prueba de entorno
    if (!job) {
      list.push({
        id: 'env-not-tested',
        type: 'warning',
        message: 'No se ha ejecutado la prueba de entorno.',
        detail: 'Recomendamos probar la imagen antes de publicarla para evitar incidencias en clase.'
      });
    } else if (job.status === 'success') {
      if (job.digest_unverified) {
        list.push({
          id: 'env-digest-unverified',
          type: 'warning',
          message: 'Verificación local sin validación de digest remoto.',
          detail: 'El registro público no estuvo accesible durante la prueba. Se usó la copia local.'
        });
      } else {
        list.push({
          id: 'env-verified',
          type: 'pass',
          message: 'Entorno verificado correctamente.',
          detail: `${job.result?.tools?.length || 0} herramientas confirmadas en el contenedor.`
        });
      }
    } else if (job.status === 'failed') {
      list.push({
        id: 'env-failed',
        type: 'error',
        message: 'La última prueba de entorno falló.',
        detail: job.error_message || 'Herramientas requeridas ausentes o error de ejecución.'
      });
    }

    // 3. Recursos de RAM
    if (this.baseRamMB() > 2048) {
      list.push({
        id: 'ram-high',
        type: 'warning',
        message: `Asignación de memoria alta: ${this.baseRamMB()} MB.`,
        detail: 'Podría limitar la concurrencia simultánea de alumnos en aulas compartidas.'
      });
    } else {
      list.push({
        id: 'ram-ok',
        type: 'pass',
        message: `Asignación de memoria adecuada: ${this.baseRamMB()} MB.`
      });
    }

    return list;
  });

  hasBlockingErrors = computed<boolean>(() => {
    return this.checks().some(c => c.type === 'error');
  });

  onConfirm(): void {
    if (!this.hasBlockingErrors()) {
      this.confirmed.emit();
    }
  }

  onClose(): void {
    this.closed.emit();
  }
}
