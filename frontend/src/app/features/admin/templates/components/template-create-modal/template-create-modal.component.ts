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
import { Router, RouterModule } from '@angular/router';
import { Subject, Subscription } from 'rxjs';
import { debounceTime, distinctUntilChanged } from 'rxjs/operators';
import { 
  CreateOfficialTemplateDTO, 
  AdminTemplatesService, 
  AvailableSatelliteService, 
  ServiceRequirement,
  LocalImageItem,
  ImageVerificationResult,
  AdminTemplateItem,
  RuntimeCapabilities,
  TemplateModelItem,
  TemplateCategory
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
  LucideRotateCw,
  LucideCopy,
  LucideSearch,
  LucideCheck,
  LucideCheckCircle2,
  LucideTerminal,
  LucideArrowLeft,
  LucideArrowRight,
  LucidePlus,
  LucideEdit,
  LucideTrash2,
  LucideTag
} from '@lucide/angular';

export interface ImageSuggestion {
  repoTag: string;
  isLocal: boolean;
  sizeMB?: number;
  description?: string;
  isOfficial: boolean;
  usageCount?: number;
}

export interface RamPreset {
  mb: number;
  label: string;
  desc: string;
}

export const RAM_PRESETS_IDE: RamPreset[] = [
  { mb: 512, label: '512 MB', desc: 'Ligera (C/Go)' },
  { mb: 1024, label: '1 GB', desc: 'Estándar (Web/Python)' },
  { mb: 2048, label: '2 GB', desc: 'Intensiva (Java/ML)' },
  { mb: 4096, label: '4 GB', desc: 'Datos & IA' }
];

export const RAM_PRESETS_JUDGE: RamPreset[] = [
  { mb: 128, label: '128 MB', desc: 'Ultra-ligera (C/C++)' },
  { mb: 256, label: '256 MB', desc: 'Recomendada (Python/Go)' },
  { mb: 512, label: '512 MB', desc: 'Completa (Java/JVM)' }
];

export type WizardSection = 'purpose' | 'identity' | 'image' | 'execution' | 'resources' | 'verification';

@Component({
  selector: 'solv-template-create-modal',
  standalone: true,
  imports: [
    CommonModule, 
    FormsModule, 
    RouterModule,
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
    LucideRotateCw,
    LucideCopy,
    LucideSearch,
    LucideCheck,
    LucideCheckCircle2,
    LucideTerminal,
    LucideArrowLeft,
    LucideArrowRight,
    LucidePlus,
    LucideEdit,
    LucideTrash2,
    LucideTag,
    RouterModule
  ],
  templateUrl: './template-create-modal.component.html',
  styleUrls: ['./template-create-modal.component.scss']
})
export class TemplateCreateModalComponent implements OnInit, OnDestroy {
  private templatesService = inject(AdminTemplatesService);
  private router = inject(Router);

  @Output() created = new EventEmitter<CreateOfficialTemplateDTO>();
  @Output() closed = new EventEmitter<void>();

  // Navegación modular de 6 pasos
  readonly WIZARD_SECTIONS: WizardSection[] = ['purpose', 'identity', 'image', 'execution', 'resources', 'verification'];
  activeSection = signal<WizardSection>('purpose');

  // Paso 1: Propósito del Entorno (IDE vs Juez)
  targetEnvironment = signal<'IDE_PERSISTENTE' | 'JUEZ_EFIMERO'>('IDE_PERSISTENTE');
  pendingPurposeChange = signal<'IDE_PERSISTENTE' | 'JUEZ_EFIMERO' | null>(null);
  showPurposeConfirmDialog = signal<boolean>(false);

  // Modo de creación de 3 puertas (ST-10, ST-11, ST-12, ST-13)
  creationMode = signal<'blank' | 'recipe' | 'duplicate'>('blank');
  toastMessage = signal<string | null>(null);

  // Campos principales del formulario
  name = signal<string>('');
  description = signal<string>('');
  dockerImage = signal<string>('');
  toolsDeclared = signal<string>('');
  setupScript = signal<string>('');
  entrypoint = signal<string>('');
  timeoutMS = signal<number>(5000);
  sampleInput = signal<string>('');
  baseRamMB = signal<number>(512);
  selectedCategoryId = signal<string | null>(null);
  selectedModelId = signal<string | null>(null);
  isSubmitting = signal<boolean>(false);

  // Categorías de plantillas (dato vivo en BD)
  categories = signal<TemplateCategory[]>([]);
  showCategoryManagerModal = signal<boolean>(false);
  categoryFormName = signal<string>('');
  categoryFormDescription = signal<string>('');
  editingCategoryId = signal<string | null>(null);
  categoryErrorMsg = signal<string | null>(null);
  isCategorySaving = signal<boolean>(false);
  quickCategoryName = signal<string>('');
  isQuickCategorySaving = signal<boolean>(false);

  // Job de prueba de entorno y diálogo de publicación
  activeEnvTestJob = signal<EnvTestJob | null>(null);
  showPublishDialog = signal<boolean>(false);

  // Autosave y Telemetría
  hasDraftToResume = signal<boolean>(false);
  private modalOpenTime = Date.now();
  private stepPath = signal<string[]>(['purpose']);
  recipeUsed = signal<string | null>(null);
  private envTestRan = signal<boolean>(false);

  // Verificación y Caché de Imagen
  verificationState = signal<'idle' | 'checking' | 'verified' | 'error'>('idle');
  verificationResult = signal<ImageVerificationResult | null>(null);
  verificationError = signal<string | null>(null);

  // Typeahead y Sugerencias de Imágenes
  localImages = signal<LocalImageItem[]>([]);
  usageMap = signal<Record<string, number>>({});
  isDropdownOpen = signal<boolean>(false);
  activeComboboxIndex = signal<number>(-1);
  showImagePopover = signal<boolean>(false);
  isHelpDrawerOpen = signal<boolean>(false);
  activeDrawerStep = signal<WizardSection>('purpose');
  copiedCommand = signal<boolean>(false);

  // Progressive Disclosure de Categorías
  isCreatingCategoryInline = signal<boolean>(false);
  inlineCategoryName = signal<string>('');
  previousCategoryId = signal<string | null>(null);

  // Modelos dinámicos consumidos desde GET /api/v1/admin/template-models
  models = signal<TemplateModelItem[]>([]);
  modelsSearch = signal<string>('');

  filteredModels = computed<TemplateModelItem[]>(() => {
    const currentEnv = this.targetEnvironment();
    const q = this.modelsSearch().trim().toLowerCase();
    return this.models().filter(m => {
      if (m.target_environment !== currentEnv) return false;
      if (!q) return true;
      return m.name.toLowerCase().includes(q) ||
        (m.category_name && m.category_name.toLowerCase().includes(q)) ||
        (m.description && m.description.toLowerCase().includes(q)) ||
        m.docker_image.toLowerCase().includes(q);
    });
  });

  availableCategories = computed<string[]>(() => {
    const set = new Set<string>();
    for (const m of this.filteredModels()) {
      if (m.category_name) {
        set.add(m.category_name);
      }
    }
    return Array.from(set);
  });

  modelsByCategory(catName: string): TemplateModelItem[] {
    return this.filteredModels().filter(m => m.category_name === catName);
  }

  modelsWithoutCategory = computed<TemplateModelItem[]>(() => {
    return this.filteredModels().filter(m => !m.category_name);
  });

  // Catálogo existente para puerta "Duplicar" filtrado por propósito activo (Adenda 4)
  existingCatalog = signal<AdminTemplateItem[]>([]);
  duplicateSearch = signal<string>('');

  filteredExistingCatalog = computed<AdminTemplateItem[]>(() => {
    const currentEnv = this.targetEnvironment();
    const q = this.duplicateSearch().trim().toLowerCase();
    const list = this.existingCatalog().filter(t => {
      const env = t.target_environment || 'IDE_PERSISTENTE';
      return env === currentEnv && (t.status === 'approved' || t.status === 'APROBADA');
    });

    if (!q) return list;
    return list.filter(t => 
      t.name.toLowerCase().includes(q) || 
      t.docker_image.toLowerCase().includes(q)
    );
  });

  // Puertas con confirmación si el form está editado (D1)
  pendingDoorChange = signal<'blank' | 'recipe' | 'duplicate' | null>(null);
  showDoorConfirmDialog = signal<boolean>(false);

  isFormDirty = computed<boolean>(() => {
    return this.name().trim().length > 0 || 
      this.dockerImage().trim().length > 0 || 
      this.entrypoint().trim().length > 0 || 
      this.setupScript().trim().length > 0 || 
      this.selectedServices().length > 0;
  });

  // Estado de borrador guardado y footer contextual (D2)
  isDraftSaved = signal<boolean>(false);

  footerActionState = computed<'save_draft' | 'publish_disabled' | 'publish_ready'>(() => {
    if (!this.isDraftSaved()) {
      return 'save_draft';
    }
    const job = this.activeEnvTestJob();
    const isVerified = job !== null && job.status === 'success';
    if (!isVerified || this.isInvalidFormat() || this.isLatestImage()) {
      return 'publish_disabled';
    }
    if (this.targetEnvironment() === 'JUEZ_EFIMERO' && !this.entrypoint().trim()) {
      return 'publish_disabled';
    }
    return 'publish_ready';
  });

  publishDisabledReason = computed<string>(() => {
    const reasons: string[] = [];
    if (!this.dockerImage().trim()) {
      reasons.push('Ingrese una imagen Docker');
    } else if (this.isLatestImage()) {
      reasons.push('Tag :latest prohibido');
    }
    if (this.targetEnvironment() === 'JUEZ_EFIMERO' && !this.entrypoint().trim()) {
      reasons.push('Comando de ejecución requerido para juez');
    }
    const job = this.activeEnvTestJob();
    if (!job) {
      reasons.push('Prueba de entorno no ejecutada');
    } else if (job.status === 'failed') {
      reasons.push('La prueba de entorno falló');
    } else if (job?.status === 'pulling' || job?.status === 'testing' || job?.status === 'pending') {
      reasons.push('Prueba de entorno en curso');
    }
    return reasons.length > 0 ? `Falta: ${reasons.join(' · ')}` : 'Complete las comprobaciones antes de publicar';
  });

  private dockerRegex = /^[a-z0-9]+(?:[._-][a-z0-9]+)*(?:\/[a-z0-9]+(?:[._-][a-z0-9]+)*)*:[a-zA-Z0-9_.-]+$/;

  private imageDebounce$ = new Subject<string>();
  private sub = new Subscription();

  // CONFIG: Semilla de sugerencias curadas para el typeahead de imágenes OCI.
  // Es una allowlist semilla documentada, no representa modelos de plantilla ni catálogo vivo.
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

  runtimeCapabilities = signal<RuntimeCapabilities | null>(null);

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

  // Presets de RAM dinámicos según el hardware real del entorno y el propósito
  activeRamPresets = computed<RamPreset[]>(() => {
    const caps = this.runtimeCapabilities();
    const isJudge = this.targetEnvironment() === 'JUEZ_EFIMERO';
    if (caps) {
      const presets = isJudge ? caps.judge_presets : caps.ide_presets;
      if (presets && presets.length > 0) {
        return presets;
      }
    }
    return isJudge ? RAM_PRESETS_JUDGE : RAM_PRESETS_IDE;
  });

  // Chips de validez por sección (6 pasos)
  purposeStatus = computed<'complete'>(() => 'complete');

  identityStatus = computed<'complete' | 'pending'>(() => {
    return this.name().trim().length >= 3 ? 'complete' : 'pending';
  });

  imageStatus = computed<'complete' | 'warning' | 'pending'>(() => {
    if (!this.dockerImage().trim()) return 'pending';
    if (this.isLatestImage() || this.isInvalidFormat() || this.isStorageBlocked() || this.isArchIncompatible()) {
      return 'warning';
    }
    return 'complete';
  });

  executionStatus = computed<'complete' | 'pending'>(() => {
    if (this.targetEnvironment() === 'JUEZ_EFIMERO') {
      return this.entrypoint().trim().length > 0 ? 'complete' : 'pending';
    }
    return 'complete';
  });

  resourcesStatus = computed<'complete' | 'pending'>(() => {
    return this.baseRamMB() > 0 && !this.isRamTooLow() ? 'complete' : 'pending';
  });

  verificationStatus = computed<'complete' | 'warning' | 'pending'>(() => {
    const job = this.activeEnvTestJob();
    if (!job) return 'pending';
    if (job.status === 'success') return 'complete';
    if (job.status === 'failed') return 'warning';
    return 'pending';
  });

  // Índice actual en el wizard
  currentSectionIndex = computed<number>(() => {
    return this.WIZARD_SECTIONS.indexOf(this.activeSection());
  });

  canGoPrev = computed<boolean>(() => this.currentSectionIndex() > 0);
  canGoNext = computed<boolean>(() => this.currentSectionIndex() < this.WIZARD_SECTIONS.length - 1);

  ngOnInit(): void {
    this.fetchLocalImages();
    this.loadModels();
    this.loadCategories();

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

    // Consulta únicamente plantillas aprobadas para la puerta de duplicación
    this.templatesService.getTemplates('approved').subscribe({
      next: (list) => {
        this.existingCatalog.set(list || []);
      },
      error: () => {}
    });

    // Consulta capacidades de hardware y servicios satélite reales del entorno
    this.templatesService.getRuntimeCapabilities().subscribe({
      next: (caps) => {
        if (caps) {
          this.runtimeCapabilities.set(caps);
          if (caps.satellite_services && caps.satellite_services.length > 0) {
            this.availableServices.set(
              caps.satellite_services.map(s => ({
                category: s.category,
                engine: s.engine,
                label: s.label,
                version: s.version,
                description: s.description,
                envVar: s.env_var,
                isAvailable: s.is_available
              }))
            );
          }
        }
      },
      error: () => {}
    });
  }

  loadModels(): void {
    this.templatesService.getModels().subscribe({
      next: (list) => this.models.set(list || []),
      error: () => {}
    });
  }

  loadCategories(): void {
    this.templatesService.getCategories().subscribe({
      next: (cats) => this.categories.set(cats || []),
      error: () => {}
    });
  }

  openCategoryManager(): void {
    this.categoryFormName.set('');
    this.categoryFormDescription.set('');
    this.editingCategoryId.set(null);
    this.categoryErrorMsg.set(null);
    this.showCategoryManagerModal.set(true);
  }

  closeCategoryManager(): void {
    this.showCategoryManagerModal.set(false);
    this.categoryErrorMsg.set(null);
    this.editingCategoryId.set(null);
  }

  startEditCategory(cat: TemplateCategory): void {
    this.editingCategoryId.set(cat.id);
    this.categoryFormName.set(cat.name);
    this.categoryFormDescription.set(cat.description || '');
    this.categoryErrorMsg.set(null);
  }

  cancelEditCategory(): void {
    this.editingCategoryId.set(null);
    this.categoryFormName.set('');
    this.categoryFormDescription.set('');
    this.categoryErrorMsg.set(null);
  }

  saveCategory(): void {
    const name = this.categoryFormName().trim();
    if (!name) return;
    this.isCategorySaving.set(true);
    this.categoryErrorMsg.set(null);

    const editId = this.editingCategoryId();
    if (editId) {
      this.templatesService.updateCategory(editId, {
        name,
        description: this.categoryFormDescription().trim()
      }).subscribe({
        next: () => {
          this.isCategorySaving.set(false);
          this.cancelEditCategory();
          this.loadCategories();
          this.loadModels();
        },
        error: (err) => {
          this.isCategorySaving.set(false);
          if (err?.status === 409) {
            this.categoryErrorMsg.set('Ya existe una categoría con este nombre.');
          } else {
            this.categoryErrorMsg.set('Error al actualizar la categoría.');
          }
        }
      });
    } else {
      this.templatesService.createCategory({
        name,
        description: this.categoryFormDescription().trim()
      }).subscribe({
        next: (created) => {
          this.isCategorySaving.set(false);
          this.categoryFormName.set('');
          this.categoryFormDescription.set('');
          this.loadCategories();
          if (!this.selectedCategoryId()) {
            this.selectedCategoryId.set(created.id);
          }
        },
        error: (err) => {
          this.isCategorySaving.set(false);
          if (err?.status === 409) {
            this.categoryErrorMsg.set('Ya existe una categoría con este nombre.');
          } else {
            this.categoryErrorMsg.set('Error al crear la categoría.');
          }
        }
      });
    }
  }

  deleteCategory(cat: TemplateCategory): void {
    this.categoryErrorMsg.set(null);
    this.templatesService.deleteCategory(cat.id).subscribe({
      next: () => {
        if (this.selectedCategoryId() === cat.id) {
          this.selectedCategoryId.set(null);
        }
        this.loadCategories();
        this.loadModels();
      },
      error: (err) => {
        if (err?.status === 409) {
          this.categoryErrorMsg.set(`No se puede eliminar "${cat.name}" porque está en uso.`);
        } else {
          this.categoryErrorMsg.set('Error al eliminar la categoría.');
        }
      }
    });
  }

  navigateToCategoryManager(): void {
    this.router.navigate(['/admin/modelos-categorias']);
    this.closed.emit();
  }

  onCategorySelect(val: string | null): void {
    if (val === '__new__') {
      this.previousCategoryId.set(this.selectedCategoryId());
      this.isCreatingCategoryInline.set(true);
      this.inlineCategoryName.set('');
    } else {
      this.isCreatingCategoryInline.set(false);
      this.selectedCategoryId.set(val);
    }
  }

  cancelInlineCategory(): void {
    this.isCreatingCategoryInline.set(false);
    this.selectedCategoryId.set(this.previousCategoryId());
    this.inlineCategoryName.set('');
  }

  createCategoryInline(): void {
    const name = this.inlineCategoryName().trim();
    if (!name) return;
    this.isQuickCategorySaving.set(true);
    this.templatesService.createCategory({ name }).subscribe({
      next: (cat) => {
        this.isQuickCategorySaving.set(false);
        this.categories.update(list => [...list, cat]);
        this.selectedCategoryId.set(cat.id);
        this.isCreatingCategoryInline.set(false);
        this.inlineCategoryName.set('');
      },
      error: () => {
        this.isQuickCategorySaving.set(false);
      }
    });
  }

  quickCreateCategory(): void {
    const name = this.quickCategoryName().trim();
    if (!name) return;
    this.isQuickCategorySaving.set(true);
    this.templatesService.createCategory({ name }).subscribe({
      next: (cat) => {
        this.isQuickCategorySaving.set(false);
        this.categories.update(list => [...list, cat]);
        this.selectedCategoryId.set(cat.id);
        this.quickCategoryName.set('');
      },
      error: () => {
        this.isQuickCategorySaving.set(false);
      }
    });
  }

  ngOnDestroy(): void {
    this.sub.unsubscribe();
  }

  // Telemetría de eventos (UX-14)
  private emitTelemetry(event: string, meta: Record<string, any> = {}): void {
    const payload = {
      event,
      timestamp: new Date().toISOString(),
      target_environment: this.targetEnvironment(),
      step_path: this.stepPath(),
      recipe_used: this.recipeUsed(),
      test_env_run: this.envTestRan(),
      tiempo_hasta_publicar: Math.round((Date.now() - this.modalOpenTime) / 1000),
      ...meta
    };
    console.debug('[SOLV Telemetry]', payload);
  }

  // Gestión de Autosave y Reanudación (UX-13)
  private checkDraft(): void {
    this.templatesService.getDraft().subscribe({
      next: (draft) => {
        if (draft && draft.form_data) {
          this.applyDraftData(draft.form_data);
          if (draft.form_data.name || draft.form_data.dockerImage) {
            this.hasDraftToResume.set(true);
          }
        }
      },
      error: (err) => {
        console.debug('[Draft] No active draft or error fetching draft', err);
      }
    });
  }

  private applyDraftData(d: any): void {
    if (!d) return;
    if (d.targetEnvironment) this.targetEnvironment.set(d.targetEnvironment);
    if (d.name) this.name.set(d.name);
    if (d.dockerImage) this.dockerImage.set(d.dockerImage);
    if (d.baseRamMB) this.baseRamMB.set(d.baseRamMB);
    if (d.setupScript) this.setupScript.set(d.setupScript);
    if (d.entrypoint) this.entrypoint.set(d.entrypoint);
    if (d.timeoutMS) this.timeoutMS.set(d.timeoutMS);
    if (d.sampleInput) this.sampleInput.set(d.sampleInput);
    if (d.description) this.description.set(d.description);
    if (d.toolsDeclared) this.toolsDeclared.set(d.toolsDeclared);
    if (d.selectedServices) this.selectedServices.set(d.selectedServices);
    if (d.selectedCategoryId) this.selectedCategoryId.set(d.selectedCategoryId);
    if (d.selectedModelId) this.selectedModelId.set(d.selectedModelId);
    if (d.dockerImage) this.triggerVerification(d.dockerImage, false);
  }

  resumeDraft(): void {
    this.hasDraftToResume.set(false);
  }

  discardDraft(): void {
    this.templatesService.deleteDraft().subscribe({
      next: () => {
        this.resetDraftForm();
        this.hasDraftToResume.set(false);
      },
      error: () => {
        this.resetDraftForm();
        this.hasDraftToResume.set(false);
      }
    });
  }

  private resetDraftForm(): void {
    this.name.set('');
    this.dockerImage.set('');
    this.description.set('');
    this.setupScript.set('');
    this.entrypoint.set('');
    this.sampleInput.set('');
    this.timeoutMS.set(3000);
    this.selectedServices.set([]);
    this.toolsDeclared.set('');
    this.selectedCategoryId.set(null);
    this.selectedModelId.set(null);
    this.verificationState.set('idle');
    this.verificationResult.set(null);
    this.verificationError.set(null);
  }

  saveDraft(): void {
    this.saveDraftToStorage();
  }

  private saveDraftToStorage(): void {
    const draft = {
      targetEnvironment: this.targetEnvironment(),
      name: this.name(),
      dockerImage: this.dockerImage(),
      baseRamMB: this.baseRamMB(),
      setupScript: this.setupScript(),
      entrypoint: this.entrypoint(),
      timeoutMS: this.timeoutMS(),
      sampleInput: this.sampleInput(),
      description: this.description(),
      toolsDeclared: this.toolsDeclared(),
      selectedServices: this.selectedServices(),
      selectedCategoryId: this.selectedCategoryId(),
      selectedModelId: this.selectedModelId(),
      updatedAt: new Date().toISOString()
    };
    this.templatesService.saveDraft(draft).subscribe({
      next: () => {},
      error: (err) => console.error('Error saving draft to backend:', err)
    });
  }

  // Navegación modular por pasos
  setSection(section: WizardSection): void {
    this.activeSection.set(section);
    this.stepPath.update((path: string[]) => [...path, section]);
    this.saveDraftToStorage();
    setTimeout(() => {
      document.getElementById(`step-title-${section}`)?.focus();
    }, 50);
  }

  toggleHelpDrawer(step?: WizardSection): void {
    const targetStep = step || this.activeSection();
    if (this.isHelpDrawerOpen() && this.activeDrawerStep() === targetStep) {
      this.isHelpDrawerOpen.set(false);
    } else {
      this.activeDrawerStep.set(targetStep);
      this.isHelpDrawerOpen.set(true);
    }
  }

  closeHelpDrawer(): void {
    this.isHelpDrawerOpen.set(false);
  }

  onPurposeKeydown(event: KeyboardEvent, current: 'IDE_PERSISTENTE' | 'JUEZ_EFIMERO'): void {
    const isHorizontalNext = event.key === 'ArrowRight' || event.key === 'ArrowDown';
    const isHorizontalPrev = event.key === 'ArrowLeft' || event.key === 'ArrowUp';

    if (isHorizontalNext || isHorizontalPrev) {
      event.preventDefault();
      const nextEnv = current === 'IDE_PERSISTENTE' ? 'JUEZ_EFIMERO' : 'IDE_PERSISTENTE';
      this.selectTargetEnvironment(nextEnv);
      const targetId = nextEnv === 'IDE_PERSISTENTE' ? 'purpose-card-ide' : 'purpose-card-judge';
      setTimeout(() => {
        document.getElementById(targetId)?.focus();
      }, 0);
    } else if (event.key === 'Enter') {
      event.preventDefault();
      this.nextSection();
    }
  }

  nextSection(): void {
    const idx = this.currentSectionIndex();
    if (idx < this.WIZARD_SECTIONS.length - 1) {
      this.setSection(this.WIZARD_SECTIONS[idx + 1]);
    }
  }

  prevSection(): void {
    const idx = this.currentSectionIndex();
    if (idx > 0) {
      this.setSection(this.WIZARD_SECTIONS[idx - 1]);
    }
  }

  // Cambio de Propósito con Diálogo de Confirmación (Adenda 3)
  selectTargetEnvironment(env: 'IDE_PERSISTENTE' | 'JUEZ_EFIMERO'): void {
    if (this.targetEnvironment() === env) return;
    if (this.isFormDirty()) {
      this.pendingPurposeChange.set(env);
      this.showPurposeConfirmDialog.set(true);
      return;
    }
    this.executePurposeChange(env);
  }

  confirmPurposeChange(): void {
    const nextEnv = this.pendingPurposeChange();
    if (nextEnv) {
      this.executePurposeChange(nextEnv);
    }
    this.showPurposeConfirmDialog.set(false);
    this.pendingPurposeChange.set(null);
  }

  cancelPurposeChange(): void {
    this.pendingPurposeChange.set(null);
    this.showPurposeConfirmDialog.set(false);
  }

  private executePurposeChange(newEnv: 'IDE_PERSISTENTE' | 'JUEZ_EFIMERO'): void {
    this.targetEnvironment.set(newEnv);
    // Matriz de reseteo estricta:
    // Identidad e imagen se conservan.
    this.recipeUsed.set(null);
    this.baseRamMB.set(newEnv === 'JUEZ_EFIMERO' ? 256 : 512);
    this.selectedServices.set([]);
    this.setupScript.set('');
    this.entrypoint.set('');
    this.sampleInput.set('');
    this.timeoutMS.set(newEnv === 'JUEZ_EFIMERO' ? 3000 : 5000);
    this.activeEnvTestJob.set(null);
    this.isDraftSaved.set(false);
    this.stepPath.update(path => [...path, 'target_env:' + newEnv]);
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

  // Puertas con confirmación si el form está editado (D1)
  selectCreationMode(mode: 'blank' | 'recipe' | 'duplicate'): void {
    if (this.creationMode() === mode) return;
    if (this.isFormDirty()) {
      this.pendingDoorChange.set(mode);
      this.showDoorConfirmDialog.set(true);
      return;
    }
    this.executeDoorChange(mode);
  }

  confirmDoorChange(): void {
    const nextMode = this.pendingDoorChange();
    if (nextMode) {
      this.name.set('');
      this.dockerImage.set('');
      this.description.set('');
      this.toolsDeclared.set('');
      this.baseRamMB.set(this.targetEnvironment() === 'JUEZ_EFIMERO' ? 256 : 512);
      this.selectedServices.set([]);
      this.setupScript.set('');
      this.entrypoint.set('');
      this.sampleInput.set('');
      this.activeEnvTestJob.set(null);
      this.isDraftSaved.set(false);
      this.executeDoorChange(nextMode);
    }
    this.showDoorConfirmDialog.set(false);
    this.pendingDoorChange.set(null);
  }

  cancelDoorChange(): void {
    this.pendingDoorChange.set(null);
    this.showDoorConfirmDialog.set(false);
  }

  private executeDoorChange(mode: 'blank' | 'recipe' | 'duplicate'): void {
    this.creationMode.set(mode);
    this.stepPath.update(path => [...path, 'mode:' + mode]);
  }

  saveDraftManually(): void {
    this.saveDraftToStorage();
    this.isDraftSaved.set(true);
    const msg = $localize`:@@ST-08:Borrador guardado.`;
    this.showToast(msg);
  }

  applyModel(model: TemplateModelItem): void {
    this.recipeUsed.set(model.id);
    this.selectedModelId.set(model.id);
    if (model.category_id) {
      this.selectedCategoryId.set(model.category_id);
    }
    this.name.set(model.name);
    this.dockerImage.set(model.docker_image);
    this.baseRamMB.set(model.base_ram_mb);
    this.description.set(model.description || '');
    if (model.entrypoint) this.entrypoint.set(model.entrypoint);
    if (model.timeout_ms) this.timeoutMS.set(model.timeout_ms);
    if (model.sample_input) this.sampleInput.set(model.sample_input);
    this.triggerVerification(model.docker_image, false);
    this.setSection('image');
    this.saveDraftToStorage();
    this.isDraftSaved.set(true);

    const msg = $localize`:@@ST-14:Modelo ${model.name}:name: aplicado. Edite lo que necesite.`;
    this.showToast(msg);
  }

  duplicateFromCatalog(template: AdminTemplateItem): void {
    this.name.set(`(Copia) ${template.name}`);
    this.dockerImage.set(template.docker_image);
    this.baseRamMB.set(template.base_ram_mb || (this.targetEnvironment() === 'JUEZ_EFIMERO' ? 256 : 512));
    this.description.set(template.description || '');
    this.toolsDeclared.set((template.tools_declared || []).join(', '));
    this.setupScript.set(template.setup_script || '');
    if (template.category_id) this.selectedCategoryId.set(template.category_id);
    if (template.entrypoint) this.entrypoint.set(template.entrypoint);
    if (template.timeout_ms) this.timeoutMS.set(template.timeout_ms);
    if (template.sample_input) this.sampleInput.set(template.sample_input);
    this.triggerVerification(template.docker_image, false);
    this.setSection('image');
    this.saveDraftToStorage();
    this.isDraftSaved.set(true);
    this.showToast(`Plantilla "${template.name}" duplicada. Ajuste la configuración.`);
  }

  onEnvTestCompleted(job: EnvTestJob): void {
    this.activeEnvTestJob.set(job);
    this.envTestRan.set(true);
    this.emitTelemetry('test_env_run', { status: job.status, duration_ms: job.result?.duration_ms });
  }

  openPublishDialog(): void {
    if (this.footerActionState() === 'publish_disabled') return;
    if (this.isFormDirty()) {
      this.saveDraftToStorage();
      this.isDraftSaved.set(true);
    }
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
    if (this.isHelpDrawerOpen()) {
      this.isHelpDrawerOpen.set(false);
      return;
    }
    if (this.showImagePopover()) {
      this.showImagePopover.set(false);
      return;
    }
    if (this.isDropdownOpen()) {
      this.isDropdownOpen.set(false);
      this.activeComboboxIndex.set(-1);
      return;
    }
    if (this.isCreatingCategoryInline()) {
      this.cancelInlineCategory();
      return;
    }
    this.closeModal();
  }

  @HostListener('document:click', ['$event'])
  onDocumentClick(event: MouseEvent): void {
    const target = event.target as HTMLElement;
    if (!target.closest('.image-combobox-wrapper') && !target.closest('.typeahead-container')) {
      this.isDropdownOpen.set(false);
      this.activeComboboxIndex.set(-1);
    }
    if (!target.closest('.label-with-help') && !target.closest('.image-popover-box')) {
      this.showImagePopover.set(false);
    }
  }

  fetchLocalImages(): void {
    this.templatesService.getLocalImages().subscribe({
      next: (res) => {
        this.localImages.set(res.images || []);
        this.usageMap.set(res.usage_map || {});
      },
      error: () => {}
    });
  }

  onImageInputChange(value: string): void {
    this.dockerImage.set(value);
    this.openCombobox();
    this.imageDebounce$.next(value.trim());
  }

  openCombobox(): void {
    this.isDropdownOpen.set(true);
    this.activeComboboxIndex.set(-1);
  }

  onComboboxBlur(): void {
    setTimeout(() => {
      this.isDropdownOpen.set(false);
      this.activeComboboxIndex.set(-1);
    }, 200);
  }

  toggleImagePopover(): void {
    this.showImagePopover.update(v => !v);
  }

  groupLocalImages = computed<ImageSuggestion[]>(() => {
    const query = this.dockerImage().trim().toLowerCase();
    const local = this.localImages();
    const usages = this.usageMap();

    const items: ImageSuggestion[] = [];
    for (const img of local) {
      if (img.has_latest_tag || img.repo_tag.toLowerCase().endsWith(':latest')) {
        continue;
      }
      if (!query || img.repo_tag.toLowerCase().includes(query)) {
        const count = img.usage_count ?? usages[img.repo_tag] ?? 0;
        items.push({
          repoTag: img.repo_tag,
          isLocal: true,
          sizeMB: img.size_mb,
          isOfficial: img.is_official,
          usageCount: count,
          description: 'En este servidor'
        });
      }
    }

    return items.sort((a, b) => {
      const diff = (b.usageCount || 0) - (a.usageCount || 0);
      if (diff !== 0) return diff;
      return (a.sizeMB || 0) - (b.sizeMB || 0);
    });
  });

  groupCuratedImages = computed<ImageSuggestion[]>(() => {
    const query = this.dockerImage().trim().toLowerCase();
    const local = this.localImages();
    const usages = this.usageMap();
    const localTags = new Set(local.map(l => l.repo_tag));

    const items: ImageSuggestion[] = [];
    for (const cur of this.curatedOfficialImages) {
      if (localTags.has(cur.repoTag)) {
        continue;
      }
      if (!query || cur.repoTag.toLowerCase().includes(query)) {
        const count = usages[cur.repoTag] ?? 0;
        items.push({
          ...cur,
          usageCount: count,
          isLocal: false
        });
      }
    }

    return items.sort((a, b) => (b.usageCount || 0) - (a.usageCount || 0));
  });

  allVisibleComboboxItems = computed<ImageSuggestion[]>(() => {
    return [...this.groupLocalImages(), ...this.groupCuratedImages()];
  });

  imageSuggestions = computed<ImageSuggestion[]>(() => {
    return this.allVisibleComboboxItems();
  });

  onComboboxKeydown(event: KeyboardEvent): void {
    if (!this.isDropdownOpen()) {
      if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
        event.preventDefault();
        this.openCombobox();
        this.activeComboboxIndex.set(0);
      }
      return;
    }

    const items = this.allVisibleComboboxItems();
    const total = items.length;

    if (event.key === 'ArrowDown') {
      event.preventDefault();
      if (total > 0) {
        this.activeComboboxIndex.update(idx => (idx + 1) % total);
      }
    } else if (event.key === 'ArrowUp') {
      event.preventDefault();
      if (total > 0) {
        this.activeComboboxIndex.update(idx => (idx <= 0 ? total - 1 : idx - 1));
      }
    } else if (event.key === 'Enter') {
      const idx = this.activeComboboxIndex();
      if (idx >= 0 && idx < total) {
        event.preventDefault();
        this.selectSuggestion(items[idx]);
      }
    } else if (event.key === 'Escape') {
      event.preventDefault();
      this.isDropdownOpen.set(false);
      this.activeComboboxIndex.set(-1);
    }
  }

  selectSuggestion(suggestion: ImageSuggestion): void {
    this.dockerImage.set(suggestion.repoTag);
    this.isDropdownOpen.set(false);
    this.activeComboboxIndex.set(-1);
    this.triggerVerification(suggestion.repoTag, false);
  }

  suggestedToolChips = computed<string[]>(() => {
    if (this.targetEnvironment() === 'JUEZ_EFIMERO') {
      return ['gcc', 'python3', 'javac'];
    }
    return ['python3', 'node', 'gcc'];
  });

  suggestedToolsPlaceholder = computed<string>(() => {
    return this.targetEnvironment() === 'JUEZ_EFIMERO'
      ? 'Ej: gcc, python3, javac'
      : 'Ej: python3, node, gcc, git';
  });

  isToolDeclared(tool: string): boolean {
    return this.toolsList().includes(tool);
  }

  addToolDeclared(tool: string): void {
    const current = this.toolsList();
    if (!current.includes(tool)) {
      const updated = [...current, tool].join(', ');
      this.toolsDeclared.set(updated);
    }
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
    const min = this.targetEnvironment() === 'JUEZ_EFIMERO' ? 64 : 256;
    return this.baseRamMB() < min;
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
    const isJudge = this.targetEnvironment() === 'JUEZ_EFIMERO';
    const ram = this.baseRamMB() || (isJudge ? 256 : 512);
    
    // Obtener la memoria libre/disponible del host reportada dinámicamente por la API del backend
    const caps = this.runtimeCapabilities();
    const hostFreeRamMB = caps?.host_memory?.available_ram_mb ?? (caps?.host_memory?.total_ram_mb ? caps.host_memory.total_ram_mb * 0.20 : 1024);
    
    if (isJudge) {
      // RAM mínima reservada por el kernel + sandbox del Juez (viene del backend; fallback: 32 MB)
      const runtimeBase = caps?.runtime_base_mb ?? 32;
      const usable = Math.max(0, ram - runtimeBase);
      const estimatedConcurrentEvaluations = Math.max(1, Math.floor(hostFreeRamMB / ram));
      return {
        ram,
        editorBase: runtimeBase,
        usable,
        estimatedCapacity: estimatedConcurrentEvaluations,
        isJudge: true
      };
    }

    // RAM mínima consumida por el editor OpenVSCode Server (viene del backend; fallback: 210 MB)
    const editorBase = caps?.editor_base_mb ?? 210;
    const usable = Math.max(0, ram - editorBase);
    const estimatedStudents = Math.max(1, Math.floor(hostFreeRamMB / ram));
    return {
      ram,
      editorBase,
      usable,
      estimatedCapacity: estimatedStudents,
      isJudge: false
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

  canSaveDraft = computed(() => {
    const hasName = this.name().trim().length >= 3;
    const hasImage = this.dockerImage().trim().length > 0;
    const formatValid = !this.isInvalidFormat() && !this.isLatestImage();
    const ramValid = !this.isRamTooLow();
    const judgeValid = this.targetEnvironment() !== 'JUEZ_EFIMERO' || this.entrypoint().trim().length > 0;
    return hasName && hasImage && formatValid && ramValid && judgeValid && !this.isSubmitting();
  });

  saveDraftDisabledTooltip = computed(() => {
    if (this.canSaveDraft()) return '';
    if (this.name().trim().length < 3) return 'Completá el nombre de la plantilla (mínimo 3 caracteres)';
    if (!this.dockerImage().trim()) return 'Ingresá la imagen Docker requerida';
    if (this.isLatestImage()) return 'El tag :latest está prohibido';
    if (this.isInvalidFormat()) return 'Formato OCI inválido (ej: python:3.12-slim)';
    if (this.targetEnvironment() === 'JUEZ_EFIMERO' && !this.entrypoint().trim()) {
      return $localize`:@@PU-13:Completá el nombre y comando de ejecución para habilitar el guardado`;
    }
    if (this.isRamTooLow()) return 'La memoria RAM asignada está por debajo del mínimo permitido';
    return 'Completá los campos requeridos para habilitar el guardado';
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
    if (this.targetEnvironment() === 'JUEZ_EFIMERO' || service.isAvailable === false) return;
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

    this.templatesService.deleteDraft().subscribe({ error: () => {} });
    this.emitTelemetry('template_saved', { name: this.name(), image: this.dockerImage() });

    this.created.emit({
      name: this.name().trim(),
      docker_image: this.dockerImage().trim(),
      base_ram_mb: this.baseRamMB(),
      description: this.description().trim(),
      target_environment: this.targetEnvironment(),
      entrypoint: this.entrypoint().trim(),
      timeout_ms: this.timeoutMS(),
      sample_input: this.sampleInput().trim(),
      category_id: this.selectedCategoryId() || undefined,
      model_id: this.selectedModelId() || undefined,
      setup_script: this.setupScript().trim(),
      tools_declared: this.toolsList(),
      services_config: {
        services: this.targetEnvironment() === 'JUEZ_EFIMERO' ? [] : this.selectedServices()
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
