import { Component, OnInit, computed, input, output, signal, inject } from '@angular/core';
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
import { AdminTemplatesService } from '@features/admin/services/admin-templates.service';
import { TemplateComponent } from '@features/admin/models/admin-templates.models';
import { TechLogoComponent } from '@features/admin/templates/components/tech-logo/tech-logo.component';

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
    FormFieldComponent,
    TechLogoComponent
  ],
  templateUrl: './template-request-modal.component.html',
  styleUrl: './template-request-modal.component.scss'
})
export class TemplateRequestModalComponent implements OnInit {
  private http = inject(HttpClient);
  private templatesService = inject(AdminTemplatesService);

  subjectId = input<string>('');
  subjectName = input<string>('');

  closed = output<void>();
  requested = output<void>();

  // Estado del wizard (2 pasos: 1 = Selección de Tecnologías y Perfil de Carga, 2 = Identidad y Justificación)
  currentStep = signal<1 | 2>(1);

  // Selección de Componentes del Entorno
  techComponents = signal<TemplateComponent[]>([]);
  selectedRuntimeId = signal<string | null>(null);
  selectedWebPlatformId = signal<string | null>(null);
  selectedDatabaseId = signal<string>('db-none');

  readonly backendRuntimes = computed(() =>
    this.techComponents().filter(i => i.layer === 'runtime' && i.sub_category !== 'web')
  );
  readonly webPlatforms = computed(() =>
    this.techComponents().filter(i => i.layer === 'runtime' && i.sub_category === 'web')
  );
  readonly databaseOptions = computed(() =>
    this.techComponents().filter(i => i.layer === 'database')
  );

  // Form state
  loadProfile = signal<LoadProfile>('standard');
  templateName = signal<string>('');
  dockerImage = signal<string>('');
  justification = signal<string>('');

  isSubmitting = signal<boolean>(false);
  formError = signal<string | null>(null);
  successMessage = signal<string | null>(null);

  ngOnInit(): void {
    this.templatesService.getTechCatalog().subscribe({
      next: (list) => {
        this.techComponents.set(list || []);
        this.syncFromCatalogSelection();
      },
      error: () => {
        this.techComponents.set([]);
        this.formError.set('No se pudo cargar el catálogo de tecnologías disponibles.');
      }
    });
  }

  selectRuntime(id: string): void {
    this.selectedRuntimeId.set(id);
    this.syncFromCatalogSelection();
  }

  selectWebPlatform(id: string | null): void {
    this.selectedWebPlatformId.set(this.selectedWebPlatformId() === id ? null : id);
    this.syncFromCatalogSelection();
  }

  selectDatabase(id: string): void {
    this.selectedDatabaseId.set(id);
    this.syncFromCatalogSelection();
  }

  private syncFromCatalogSelection(): void {
    const rtId = this.selectedRuntimeId();
    const rt = rtId
      ? this.techComponents().find(i => i.id === rtId || i.slug === rtId)
      : undefined;
    if (!rt) {
      this.dockerImage.set('');
      this.templateName.set('');
      return;
    }
    const webId = this.selectedWebPlatformId();
    const web = webId
      ? this.techComponents().find(i => i.id === webId || i.slug === webId)
      : undefined;
    const dbId = this.selectedDatabaseId();
    const db = this.techComponents().find(
      i => (i.id === dbId || i.slug === dbId) && i.service_engine !== 'none'
    );

    this.dockerImage.set(rt.base_image || '');
    const parts = [rt.name];
    if (web) parts.push(web.name);
    if (db) parts.push(db.name.split(' ')[0]);
    this.templateName.set(`Laboratorio: ${parts.join(' + ')}`);
  }

  setProfile(profile: LoadProfile): void {
    this.loadProfile.set(profile);
  }

  nextStep(): void {
    if (!this.selectedRuntimeId()) {
      this.formError.set('Seleccione el lenguaje principal que necesita para su materia.');
      return;
    }
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
      target_environment: 'IDE_PERSISTENTE',
      stack_config: {
        mode: 'visual',
        runtime_id: this.selectedRuntimeId() || undefined,
        web_platform_id: this.selectedWebPlatformId() || undefined,
        database_id: this.selectedDatabaseId()
      }
    };

    this.http.post('/api/v1/templates', payload).subscribe({
      next: () => {
        this.isSubmitting.set(false);
        this.successMessage.set('¡Solicitud enviada exitosamente a la cola de auditoría del Administrador!');
        setTimeout(() => {
          this.requested.emit();
          this.closed.emit();
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
