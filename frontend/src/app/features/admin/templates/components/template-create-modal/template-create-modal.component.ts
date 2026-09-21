import { 
  Component, 
  EventEmitter, 
  Output, 
  signal, 
  computed, 
  HostListener, 
  inject, 
  OnInit, 
  OnDestroy 
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Subject, Subscription } from 'rxjs';
import { debounceTime, distinctUntilChanged } from 'rxjs/operators';
import { 
  CreateOfficialTemplateDTO, 
  AdminTemplatesService, 
  AvailableSatelliteService, 
  ServiceRequirement,
  LocalImageItem,
  ImageVerificationResult
} from '../../../services/admin-templates.service';
import { 
  EnvTestButtonComponent 
} from '../env-test-button/env-test-button.component';
import { 
  PublishDialogComponent 
} from '../publish-dialog/publish-dialog.component';
import { 
  EnvTestJob 
} from '../../../services/env-test-job.service';
import { 
  LucideX, 
  LucideLayers, 
  LucideDatabase,
  LucideInfo,
  LucideRefreshCw,
  LucideAlertCircle,
  LucideHelpCircle,
  LucideSparkles,
  LucideSave,
  LucideSend,
  LucideRotateCw
} from '@lucide/angular';

export interface ImageSuggestion {
  repoTag: string;
  isLocal: boolean;
  sizeMB?: number;
  description?: string;
  isOfficial: boolean;
}

export interface TemplateRecipe {
  id: string;
  title: string;
  description: string;
  image: string;
  baseRamMB: number;
  tools: string;
}

export type WizardSection = 'identity' | 'environment' | 'resources';

@Component({
  selector: 'solv-template-create-modal',
  standalone: true,
  imports: [
    CommonModule, 
    FormsModule, 
    EnvTestButtonComponent,
    PublishDialogComponent,
    LucideX, 
    LucideLayers, 
    LucideDatabase,
    LucideInfo,
    LucideRefreshCw,
    LucideAlertCircle,
    LucideHelpCircle,
    LucideSparkles,
    LucideSave,
    LucideSend,
    LucideRotateCw
  ],
  templateUrl: './template-create-modal.component.html',
  styleUrls: ['./template-create-modal.component.scss']
})
export class TemplateCreateModalComponent implements OnInit, OnDestroy {
  private templatesService = inject(AdminTemplatesService);

  @Output() created = new EventEmitter<CreateOfficialTemplateDTO>();
  @Output() closed = new EventEmitter<void>();

  // Navegación en 3 secciones no-lineales (P-04, DA-05)
  activeSection = signal<WizardSection>('identity');

  // Modo de creación de 3 puertas (ST-10, ST-11, ST-12, ST-13)
  creationMode = signal<'blank' | 'recipe' | 'duplicate'>('blank');
  toastMessage = signal<string | null>(null);

  name = signal<string>('');
  dockerImage = signal<string>('');
  baseRamMB = signal<number>(512);
  setupScript = signal<string>('');
  description = signal<string>('');
  toolsDeclared = signal<string>('');
  isSubmitting = signal<boolean>(false);

  // Job de prueba de entorno y diálogo de publicación
  activeEnvTestJob = signal<EnvTestJob | null>(null);
  showPublishDialog = signal<boolean>(false);

  // Autosave y Telemetría (UX-13, UX-14)
  hasDraftToResume = signal<boolean>(false);
  private modalOpenTime = Date.now();
  private stepPath = signal<string[]>(['identity']);
  private recipeUsed = signal<string | null>(null);
  private envTestRan = signal<boolean>(false);

  // Verificación y Caché de Imagen
  verificationState = signal<'idle' | 'checking' | 'verified' | 'error'>('idle');
  verificationResult = signal<ImageVerificationResult | null>(null);
  verificationError = signal<string | null>(null);

  // Typeahead y Sugerencias de Imágenes
  localImages = signal<LocalImageItem[]>([]);
  isDropdownOpen = signal<boolean>(false);
  isHelpDrawerOpen = signal<boolean>(false);
  copiedCommand = signal<boolean>(false);

  // Recetas predefinidas de inicio rápido
  recipes: TemplateRecipe[] = [
    {
      id: 'python-ds',
      title: 'Python 3.12 Data Science',
      description: 'Entorno Debian con Python 3.12 y pip optimizado para ciencias.',
      image: 'python:3.12-slim-bookworm',
      baseRamMB: 1024,
      tools: 'python3, pip'
    },
    {
      id: 'node-lts',
      title: 'Node.js 20 LTS',
      description: 'JavaScript & TypeScript para desarrollo web moderno backend/fullstack.',
      image: 'node:20-bookworm-slim',
      baseRamMB: 768,
      tools: 'node, npm'
    },
    {
      id: 'gcc-cpp',
      title: 'C/C++ GCC 13',
      description: 'Herramientas de compilación para algoritmia y sistemas operativos.',
      image: 'gcc:13.2-bookworm',
      baseRamMB: 512,
      tools: 'gcc, g++, make'
    },
    {
      id: 'go-sdk',
      title: 'Go 1.22 Standard',
      description: 'Compilador oficial y herramientas para concurrencia y backend de alto rendimiento.',
      image: 'golang:1.22-bookworm',
      baseRamMB: 1024,
      tools: 'go, git'
    },
    {
      id: 'java-openjdk',
      title: 'Java 21 LTS OpenJDK',
      description: 'Base ultraligera Alpine con Eclipse Temurin para POO y estructuras.',
      image: 'eclipse-temurin:21-alpine',
      baseRamMB: 1024,
      tools: 'java, javac'
    }
  ];

  dockerHubSearchUrl = computed(() => {
    const raw = this.dockerImage().trim();
    if (!raw) return 'https://hub.docker.com/search';
    const repo = raw.split(':')[0];
    return 'https://hub.docker.com/search?q=' + encodeURIComponent(repo);
  });

  private dockerRegex = /^[a-z0-9]+(?:[._-][a-z0-9]+)*(?:\/[a-z0-9]+(?:[._-][a-z0-9]+)*)*:[a-zA-Z0-9_.-]+$/;

  private imageDebounce$ = new Subject<string>();
  private sub = new Subscription();

  private curatedOfficialImages: ImageSuggestion[] = [
    { repoTag: 'python:3.12-slim-bookworm', isLocal: false, isOfficial: true, description: 'Python 3.12 oficial ligero Debian' },
    { repoTag: 'python:3.11-slim', isLocal: false, isOfficial: true, description: 'Python 3.11 versión estable' },
    { repoTag: 'node:20-bookworm-slim', isLocal: false, isOfficial: true, description: 'Node.js LTS 20 Debian Bookworm' },
    { repoTag: 'node:20-alpine', isLocal: false, isOfficial: true, description: 'Node.js LTS 20 ultraligero Alpine' },
    { repoTag: 'gcc:13.2-bookworm', isLocal: false, isOfficial: true, description: 'Entorno de compilación C/C++ GCC' },
    { repoTag: 'golang:1.22-bookworm', isLocal: false, isOfficial: true, description: 'Go SDK 1.22 oficial' },
    { repoTag: 'eclipse-temurin:21-alpine', isLocal: false, isOfficial: true, description: 'Java OpenJDK 21 LTS Alpine' },
    { repoTag: 'postgres:16-alpine', isLocal: false, isOfficial: true, description: 'PostgreSQL 16 base ligera' }
  ];

  availableServices = signal<AvailableSatelliteService[]>(
    this.templatesService.getAvailableSatelliteServices()
  );
  selectedServices = signal<ServiceRequirement[]>([]);

  // Array reactivo de herramientas declaradas
  toolsList = computed<string[]>(() => {
    return this.toolsDeclared()
      .split(',')
      .map(t => t.trim())
      .filter(t => t.length > 0);
  });

  selectedServiceEngines = computed<string[]>(() => {
    return this.selectedServices().map(s => s.engine);
  });

  // Chips de validez por sección
  identityStatus = computed<'complete' | 'pending'>(() => {
    return this.name().trim().length >= 3 ? 'complete' : 'pending';
  });

  environmentStatus = computed<'complete' | 'pending' | 'warning'>(() => {
    const raw = this.dockerImage().trim();
    if (!raw || this.isInvalidFormat() || this.isLatestImage()) {
      return 'pending';
    }
    const job = this.activeEnvTestJob();
    if (job?.status === 'success') {
      return 'complete';
    }
    if (job?.status === 'failed') {
      return 'warning';
    }
    return 'pending';
  });

  resourcesStatus = computed<'complete' | 'pending'>(() => {
    return this.baseRamMB() >= 256 ? 'complete' : 'pending';
  });

  ngOnInit(): void {
    this.fetchLocalImages();

    this.sub.add(
      this.imageDebounce$.pipe(
        debounceTime(600),
        distinctUntilChanged()
      ).subscribe(image => {
        if (this.canVerify(image)) {
          this.triggerVerification(image, false);
        } else {
          this.verificationState.set('idle');
          this.verificationResult.set(null);
          this.verificationError.set(null);
        }
      })
    );

    this.checkDraft();
  }

  ngOnDestroy(): void {
    this.sub.unsubscribe();
  }

  // Telemetría de eventos (UX-14)
  private emitTelemetry(event: string, meta: Record<string, any> = {}): void {
    const payload = {
      event,
      timestamp: new Date().toISOString(),
      step_path: this.stepPath(),
      recipe_used: this.recipeUsed(),
      test_env_run: this.envTestRan(),
      tiempo_hasta_publicar: Math.round((Date.now() - this.modalOpenTime) / 1000),
      ...meta
    };
    // Registro estructurado en consola y almacenamiento local de telemetría para auditoría
    console.debug('[SOLV Telemetry]', payload);
  }

  // Gestión de Autosave y Reanudación (UX-13)
  private checkDraft(): void {
    try {
      const saved = localStorage.getItem('solv_template_draft');
      if (saved) {
        const parsed = JSON.parse(saved);
        if (parsed.name || parsed.dockerImage) {
          this.hasDraftToResume.set(true);
        }
      }
    } catch (_) {}
  }

  resumeDraft(): void {
    try {
      const saved = localStorage.getItem('solv_template_draft');
      if (saved) {
        const d = JSON.parse(saved);
        if (d.name) this.name.set(d.name);
        if (d.dockerImage) this.dockerImage.set(d.dockerImage);
        if (d.baseRamMB) this.baseRamMB.set(d.baseRamMB);
        if (d.setupScript) this.setupScript.set(d.setupScript);
        if (d.description) this.description.set(d.description);
        if (d.toolsDeclared) this.toolsDeclared.set(d.toolsDeclared);
        if (d.selectedServices) this.selectedServices.set(d.selectedServices);
        if (d.dockerImage) this.triggerVerification(d.dockerImage, false);
      }
    } catch (_) {}
    this.hasDraftToResume.set(false);
  }

  discardDraft(): void {
    try {
      localStorage.removeItem('solv_template_draft');
    } catch (_) {}
    this.hasDraftToResume.set(false);
  }

  private saveDraftToStorage(): void {
    try {
      const draft = {
        name: this.name(),
        dockerImage: this.dockerImage(),
        baseRamMB: this.baseRamMB(),
        setupScript: this.setupScript(),
        description: this.description(),
        toolsDeclared: this.toolsDeclared(),
        selectedServices: this.selectedServices(),
        updatedAt: new Date().toISOString()
      };
      localStorage.setItem('solv_template_draft', JSON.stringify(draft));
    } catch (_) {}
  }

  setSection(section: WizardSection): void {
    this.activeSection.set(section);
    this.stepPath.update((path: string[]) => [...path, section]);
    this.saveDraftToStorage();
  }

  showToast(msg: string): void {
    this.toastMessage.set(msg);
    setTimeout(() => {
      if (this.toastMessage() === msg) {
        this.toastMessage.set(null);
      }
    }, 3500);
  }

  selectCreationMode(mode: 'blank' | 'recipe' | 'duplicate'): void {
    this.creationMode.set(mode);
    if (mode === 'recipe') {
      if (this.recipes.length > 0) {
        this.applyRecipe(this.recipes[0]);
      }
    } else if (mode === 'blank') {
      this.name.set('');
      this.dockerImage.set('');
      this.toolsDeclared.set('');
      this.description.set('');
      this.activeSection.set('identity');
    }
  }

  saveAsInstitutionalRecipe(): void {
    const msg = $localize`:@@ST-16:Solicitud de receta enviada a aprobación.`;
    this.showToast(msg);
    this.emitTelemetry('save_recipe_request', { name: this.name() });
  }

  saveDraftManually(): void {
    this.saveDraftToStorage();
    const msg = $localize`:@@ST-08:Borrador guardado.`;
    this.showToast(msg);
  }

  applyRecipe(recipe: TemplateRecipe): void {
    this.recipeUsed.set(recipe.id);
    if (!this.name().trim()) {
      this.name.set(recipe.title);
    }
    this.dockerImage.set(recipe.image);
    this.baseRamMB.set(recipe.baseRamMB);
    this.toolsDeclared.set(recipe.tools);
    if (!this.description().trim()) {
      this.description.set(recipe.description);
    }
    this.triggerVerification(recipe.image, false);
    this.activeSection.set('environment');
    this.stepPath.update((path: string[]) => [...path, 'recipe:' + recipe.id, 'environment']);
    this.saveDraftToStorage();

    const msg = $localize`:@@ST-14:Receta ${recipe.title}:name: aplicada. Edite lo que necesite.`;
    this.showToast(msg);
  }

  onEnvTestCompleted(job: EnvTestJob): void {
    this.activeEnvTestJob.set(job);
    this.envTestRan.set(true);
    this.emitTelemetry('test_env_run', { status: job.status, duration_ms: job.result?.duration_ms });
  }

  openPublishDialog(): void {
    if (!this.canSaveDraft()) return;
    this.showPublishDialog.set(true);
  }

  closePublishDialog(): void {
    this.showPublishDialog.set(false);
  }

  onPublishConfirmed(): void {
    this.showPublishDialog.set(false);
    this.confirmCreate();
  }

  @HostListener('document:keydown.escape')
  onEscape(): void {
    if (this.showPublishDialog()) {
      this.showPublishDialog.set(false);
      return;
    }
    if (this.isDropdownOpen()) {
      this.isDropdownOpen.set(false);
      return;
    }
    this.closeModal();
  }

  @HostListener('document:click', ['$event'])
  onDocumentClick(event: MouseEvent): void {
    const target = event.target as HTMLElement;
    if (!target.closest('.typeahead-container')) {
      this.isDropdownOpen.set(false);
    }
  }

  fetchLocalImages(): void {
    this.templatesService.getLocalImages().subscribe({
      next: (images) => {
        this.localImages.set(images);
      },
      error: () => {}
    });
  }

  onImageInputChange(value: string): void {
    this.dockerImage.set(value);
    this.isDropdownOpen.set(true);
    this.imageDebounce$.next(value.trim());
  }

  imageSuggestions = computed<ImageSuggestion[]>(() => {
    const query = this.dockerImage().trim().toLowerCase();
    const local = this.localImages();
    const suggestions: ImageSuggestion[] = [];
    const seen = new Set<string>();

    for (const img of local) {
      if (!img.has_latest_tag && !seen.has(img.repo_tag)) {
        if (!query || img.repo_tag.toLowerCase().includes(query)) {
          suggestions.push({
            repoTag: img.repo_tag,
            isLocal: true,
            sizeMB: img.size_mb,
            isOfficial: img.is_official,
            description: 'En el servidor (despliegue inmediato)'
          });
          seen.add(img.repo_tag);
        }
      }
    }

    for (const cur of this.curatedOfficialImages) {
      if (!seen.has(cur.repoTag)) {
        if (!query || cur.repoTag.toLowerCase().includes(query)) {
          const isActuallyLocal = local.some(l => l.repo_tag === cur.repoTag);
          suggestions.push({
            ...cur,
            isLocal: isActuallyLocal
          });
          seen.add(cur.repoTag);
        }
      }
    }

    return suggestions.slice(0, 8);
  });

  selectSuggestion(suggestion: ImageSuggestion): void {
    this.dockerImage.set(suggestion.repoTag);
    this.isDropdownOpen.set(false);
    this.triggerVerification(suggestion.repoTag, false);
  }

  isLatestImage = computed(() => {
    const img = this.dockerImage().trim().toLowerCase();
    if (!img) return false;
    return img.endsWith(':latest');
  });

  isInvalidFormat = computed(() => {
    const img = this.dockerImage().trim();
    if (!img) return false;
    return !this.dockerRegex.test(img);
  });

  isRamTooLow = computed(() => {
    return this.baseRamMB() < 256;
  });

  private canVerify(image: string): boolean {
    const trimmed = image.trim();
    return trimmed.length > 0 && this.dockerRegex.test(trimmed) && !trimmed.toLowerCase().endsWith(':latest');
  }

  triggerVerification(imageToVerify?: string, force = false): void {
    const img = (imageToVerify || this.dockerImage()).trim();
    if (!this.canVerify(img)) return;

    this.verificationState.set('checking');
    this.verificationError.set(null);

    this.templatesService.verifyImage(img, force).subscribe({
      next: (result) => {
        this.verificationResult.set(result);
        if (result.exists) {
          this.verificationState.set('verified');
        } else {
          this.verificationState.set('error');
          this.verificationError.set(result.error_message || 'Imagen no encontrada en el registro OCI');
        }
      },
      error: (err) => {
        this.verificationState.set('error');
        const msg = err?.error?.message || err?.message || 'Error de conexión con el servicio de verificación';
        this.verificationError.set(msg);
      }
    });
  }

  reverify(): void {
    this.triggerVerification(this.dockerImage(), true);
  }

  copyBuildxCommand(cmd: string): void {
    if (!cmd) return;
    navigator.clipboard.writeText(cmd).then(() => {
      this.copiedCommand.set(true);
      setTimeout(() => this.copiedCommand.set(false), 2000);
    });
  }

  resourceProfilePreview = computed(() => {
    const ram = this.baseRamMB() || 256;
    const editorBase = 210;
    const usable = Math.max(0, ram - editorBase);
    const minReservation = Math.max(128, Math.floor(ram * 0.5));
    const highWarning = Math.floor(ram * 0.8);
    const estimatedCapacity = Math.floor((32 * 1024 * 0.85) / ram);
    return {
      ram,
      editorBase,
      usable,
      minReservation,
      highWarning,
      estimatedCapacity
    };
  });

  isStorageBlocked = computed(() => {
    const res = this.verificationResult();
    return res?.storage_status === 'CRITICAL_BLOCKED';
  });

  isArchIncompatible = computed(() => {
    const res = this.verificationResult();
    return res !== null && res.exists && !res.architecture_compatible;
  });

  // Habilitado para guardar borrador (UX-13): nombre e imagen sintácticamente válidos sin :latest
  canSaveDraft = computed(() => {
    const hasName = this.name().trim().length > 0;
    const hasImage = this.dockerImage().trim().length > 0;
    const formatValid = !this.isInvalidFormat() && !this.isLatestImage();
    const ramValid = !this.isRamTooLow();
    return hasName && hasImage && formatValid && ramValid && !this.isSubmitting();
  });

  isValid = computed(() => {
    const draftOk = this.canSaveDraft();
    const notBlocked = !this.isStorageBlocked() && !this.isArchIncompatible();
    const notChecking = this.verificationState() !== 'checking';
    return draftOk && notBlocked && notChecking;
  });

  setRam(mb: number): void {
    this.baseRamMB.set(mb);
  }

  onRamInput(event: Event): void {
    const input = event.target as HTMLInputElement;
    const parsed = parseInt(input.value, 10);
    if (!isNaN(parsed) && parsed > 0) {
      this.baseRamMB.set(parsed);
    }
  }

  toggleService(service: AvailableSatelliteService): void {
    const current = this.selectedServices();
    const exists = current.some(s => s.engine === service.engine);

    if (exists) {
      this.selectedServices.set(current.filter(s => s.engine !== service.engine));
    } else {
      let filtered = current;
      if (service.engine === 'postgres') {
        filtered = filtered.filter(s => s.engine !== 'mysql');
      } else if (service.engine === 'mysql') {
        filtered = filtered.filter(s => s.engine !== 'postgres');
      }

      this.selectedServices.set([
        ...filtered,
        {
          category: service.category,
          engine: service.engine,
          version: service.version
        }
      ]);
    }
  }

  isServiceSelected(engine: string): boolean {
    return this.selectedServices().some(s => s.engine === engine);
  }

  confirmCreate(): void {
    if (!this.canSaveDraft() || this.isSubmitting()) return;
    this.isSubmitting.set(true);

    try {
      localStorage.removeItem('solv_template_draft');
    } catch (_) {}
    this.emitTelemetry('template_saved', { name: this.name(), image: this.dockerImage() });

    this.created.emit({
      name: this.name().trim(),
      docker_image: this.dockerImage().trim(),
      base_ram_mb: this.baseRamMB(),
      setup_script: this.setupScript().trim(),
      description: this.description().trim(),
      tools_declared: this.toolsList(),
      services_config: {
        services: this.selectedServices()
      }
    });
  }

  closeModal(): void {
    if (this.isSubmitting()) return;
    if (this.name() || this.dockerImage()) {
      this.emitTelemetry('save_abandoned');
    }
    this.closed.emit();
  }
}
