import { Component, input, output, signal, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { 
  LucideServer, 
  LucideLayers, 
  LucideCpu, 
  LucideZap, 
  LucideCheck, 
  LucideAlertCircle, 
  LucideChevronRight, 
  LucideChevronLeft,
  LucideInfo
} from '@lucide/angular';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';

export type LoadProfile = 'light' | 'standard' | 'intensive';

@Component({
  selector: 'template-request-modal',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideServer,
    LucideLayers,
    LucideCpu,
    LucideZap,
    LucideCheck,
    LucideAlertCircle,
    LucideChevronRight,
    LucideChevronLeft,
    LucideInfo,
    ModalShellComponent,
    FormFieldComponent
  ],
  templateUrl: './template-request-modal.component.html',
  styleUrl: './template-request-modal.component.scss'
})
export class TemplateRequestModalComponent {
  private http = inject(HttpClient);

  subjectId = input<string>('');
  subjectName = input<string>('');

  close = output<void>();
  requested = output<void>();

  // Wizard state (2 steps: 1 = Perfil Pedagógico, 2 = Imagen y Justificación)
  currentStep = signal<1 | 2>(1);

  // Form state
  loadProfile = signal<LoadProfile>('standard');
  templateName = signal<string>('');
  dockerImage = signal<string>('');
  justification = signal<string>('');

  isSubmitting = signal<boolean>(false);
  formError = signal<string | null>(null);
  successMessage = signal<string | null>(null);

  setProfile(profile: LoadProfile): void {
    this.loadProfile.set(profile);
  }

  nextStep(): void {
    this.formError.set(null);
    this.currentStep.set(2);
  }

  prevStep(): void {
    this.formError.set(null);
    this.currentStep.set(1);
  }

  submitRequest(): void {
    this.formError.set(null);

    const name = this.templateName().trim();
    const image = this.dockerImage().trim();
    const reason = this.justification().trim();

    if (!name) {
      this.formError.set('El nombre descriptivo de la plantilla es obligatorio.');
      return;
    }
    if (!image) {
      this.formError.set('La imagen Docker Hub (o repositorio) es obligatoria.');
      return;
    }
    if (reason.length < 10) {
      this.formError.set('Por favor proporciona una justificación académica de al menos 10 caracteres.');
      return;
    }

    this.isSubmitting.set(true);

    const profileRAMMap: Record<LoadProfile, number> = {
      light: 512,
      standard: 1024,
      intensive: 2048
    };

    const payload = {
      name: name,
      docker_image: image,
      base_ram_mb: profileRAMMap[this.loadProfile()],
      description: `[Solicitud Docente] ${reason} (Materia: ${this.subjectName() || this.subjectId()})`,
      target_environment: 'IDE_PERSISTENTE'
    };

    this.http.post('/api/v1/templates', payload).subscribe({
      next: () => {
        this.isSubmitting.set(false);
        this.successMessage.set('¡Solicitud enviada exitosamente a la cola de auditoría del Administrador!');
        setTimeout(() => {
          this.requested.emit();
          this.close.emit();
        }, 1200);
      },
      error: (err) => {
        this.isSubmitting.set(false);
        // If template endpoint returned error, provide clear guidance
        this.formError.set(err.error?.message || err.error?.error || 'No se pudo registrar la solicitud. Verifica el nombre de la imagen.');
      }
    });
  }
}
