import { Component, OnInit, OnDestroy, computed, effect, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { Subscription } from 'rxjs';
import {
  AdminTemplatesService,
  AdminTemplateItem,
  ComponentLayer,
  CreateOfficialTemplateDTO,
  RuntimeCapabilities,
  ServiceRequirement,
  StackConfig,
  TargetEnvironment,
  TemplateCategory,
  TemplateComponent
} from '../../services/admin-templates.service';
import { EnvTestJob, EnvTestJobService } from '../../services/env-test-job.service';
import { StudioCopilotService, StudioDraft } from './studio-copilot.service';
import { ToastService } from '@shared/services/toast.service';
import { AiOperation, AiOperations } from '@shared/services/ai-operation';
import { TechLogoComponent } from '../components/tech-logo/tech-logo.component';
import { ComponentFormModalComponent } from '../components/model-library/components/component-form-modal/component-form-modal.component';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { AdminConfigIaService, AIMissingComponent, AIRepairTemplateResponse, AIStatusDTO } from '../../configuracion/admin-config-ia.service';
import { ComboboxOption } from '@shared/components/combobox/combobox.component';
import { CodeEditorComponent } from '@shared/components/code-editor/code-editor.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import {
  LucideArrowLeft,
  LucideCheckCircle2,
  LucideAlertCircle,
  LucideCode,
  LucideLayers,
  LucideUpload,
  LucideDownload,
  LucideRotateCcw,
  LucideAlertTriangle,
  LucideCopy,
  LucidePencil,
  LucideSparkles,
  LucideWrench,
  LucideNetwork,
  LucideInfo,
  LucideCheck
} from '@lucide/angular';
import {
  AiProposalView,
  analyzeDockerfile,
  buildStudioDatabaseOptions,
  CapacityOption,
  classifyAddonComponent,
  computeCompatibleAddonsForSelection,
  computeIntensiveRamMB,
  buildConstellationFromProposal,
  buildConstellationFromSelection,
  computeMinimumFloorMBForStack,
  computeRecommendedRamMBForStack,
  computeVerificationFingerprint,
  DockerfileDiagnostic,
  downloadBrowserTextFile,
  filterAndPrioritizeAddonItems,
  filterAndPrioritizeRuntimeItems,
  fnv1aHex,
  getComponentRoleBadge,
  importBrowserTextFile,
  isFrameworkAddon,
  StackConstellation,
  StudioActionBlocker,
  StudioDatabaseOption,
  synthesizeVisualStackState
} from './template-studio.utils';
import { StudioPurposeStepComponent } from './components/studio-purpose-step/studio-purpose-step.component';
import { StudioCapacitySelectorComponent } from './components/studio-capacity-selector/studio-capacity-selector.component';
import { StudioSummarySidebarComponent } from './components/studio-summary-sidebar/studio-summary-sidebar.component';
import { StudioAiCopilotModalComponent } from './components/studio-ai-copilot-modal/studio-ai-copilot-modal.component';
import { TechGraphModalComponent } from './components/tech-graph-modal/tech-graph-modal.component';
import {
  findCatalogComponentMatch,
  findDatabaseOptionMatch
} from './template-studio.utils';

// Resolución de una propuesta del Copiloto contra el catálogo real del tenant:
// muestra qué piezas se pueden aplicar y cuáles faltan (no disponibles).
interface StudioAiProposalView extends AiProposalView {
  latencyMs: number;
}

export type { CapacityOption, StudioDatabaseOption, StudioActionBlocker };

@Component({
  selector: 'template-studio',
  imports: [
    FormsModule,
    TechLogoComponent,
    ComponentFormModalComponent,
    ModalShellComponent,
    CodeEditorComponent,
    MachineDataDirective,
    StudioPurposeStepComponent,
    StudioCapacitySelectorComponent,
    StudioSummarySidebarComponent,
    StudioAiCopilotModalComponent,
    TechGraphModalComponent,
    LucideArrowLeft,
    LucideCheckCircle2,
    LucideAlertCircle,
    LucideCode,
    LucideLayers,
    LucideUpload,
    LucideDownload,
    LucideRotateCcw,
    LucideAlertTriangle,
    LucideCopy,
    LucidePencil,
    LucideSparkles,
    LucideWrench,
    LucideNetwork,
    LucideInfo,
    LucideCheck
  ],
  templateUrl: './template-studio.component.html',
  styleUrls: ['./template-studio.component.scss']
})
export class TemplateStudioComponent implements OnInit, OnDestroy {
  private readonly templatesService = inject(AdminTemplatesService);
  private readonly envTestService = inject(EnvTestJobService);
  private readonly aiService = inject(AdminConfigIaService);
  private readonly copilot = inject(StudioCopilotService);
  private readonly aiOperations = inject(AiOperations);
  private readonly toastService = inject(ToastService);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);

  private readonly sub = new Subscription();
  private pollSub?: Subscription;
  private loadedTemplateItem: AdminTemplateItem | null = null;
  private isDuplicateLoad = false;
  private readonly pendingDraft: StudioDraft | null;

  constructor() {
    // Se lee antes de cualquier flush de effects para no pisar el borrador
    // guardado por una visita anterior al Studio.
    this.pendingDraft = this.copilot.draft();
    effect(() => {
      if (this.editingTemplateId()) return;
      this.copilot.saveDraft(this.captureDraft());
    });
  }

  readonly editingTemplateId = signal<string | null>(null);
  readonly isEditMode = computed(() => !!this.editingTemplateId());

  readonly targetEnvironment = signal<TargetEnvironment>('IDE_PERSISTENTE');
  readonly studioMode = signal<'visual' | 'code'>('visual');
  readonly blueprintTab = signal<'human' | 'manifest'>('human');

  readonly name = signal<string>('');
  readonly isNameManuallyEdited = signal<boolean>(false);
  readonly description = signal<string>('');
  readonly selectedCategoryId = signal<string | null>(null);
  readonly categories = signal<TemplateCategory[]>([]);
  readonly categoryOptions = computed<ComboboxOption<string>[]>(() =>
    this.categories().map(c => ({ id: c.id, label: c.name, value: c.id }))
  );

  readonly techComponents = signal<TemplateComponent[]>([]);
  readonly selectedRuntimeId = signal<string | null>(null);
  readonly selectedWebPlatformId = signal<string | null>(null);
  readonly selectedAddonIds = signal<string[]>([]);
  readonly selectedDatabaseId = signal<string>('db-none');

  readonly dockerImage = signal<string>('');
  readonly dockerfileContent = signal<string>('');
  readonly hasCustomDockerfile = signal<boolean>(false);
  readonly needsLocalBuild = signal<boolean>(false);
  readonly setupScript = signal<string>('');
  readonly toolsDeclared = signal<string>('');
  readonly entrypoint = signal<string>('');
  readonly timeoutMS = signal<number>(5000);
  readonly sampleInput = signal<string>('');
  readonly baseRamMB = signal<number>(512);
  readonly isRamManuallyEdited = signal<boolean>(false);
  readonly hasSelectedStack = computed<boolean>(() =>
    this.effectivePrimaryComponent() !== null || this.dockerImage().trim().length > 0
  );
  readonly showManualRamInput = signal<boolean>(false);

  readonly dockerfileDiagnostics = computed<DockerfileDiagnostic[]>(() =>
    analyzeDockerfile(this.dockerfileContent())
  );

  // Copiloto IA y Autorreparación
  readonly aiEnabled = signal<boolean>(false);
  readonly isAiCopilotModalOpen = signal<boolean>(false);
  readonly isTechGraphOpen = signal<boolean>(false);
  // El estado del Copiloto vive en StudioCopilotService: sobrevive a la
  // navegación y al cierre del modal. Se exponen los mismos signals para no
  // cambiar los bindings del template ni la API que consumen los tests.
  readonly aiPrompt = this.copilot.prompt;
  readonly isSuggestingWithAi = this.copilot.isSuggesting;
  readonly aiProposal = this.copilot.proposal;
  readonly aiProposalApplied = this.copilot.applied;
  readonly aiElapsedSeconds = this.copilot.elapsedSeconds;
  readonly aiErrorMessage = this.copilot.errorMessage;
  // La autorreparación también es una operación con dueño en la raíz.
  private readonly repairOperation: AiOperation<AIRepairTemplateResponse> = this.aiOperations.get(
    'repair-template',
    {
      resolveError: (err: unknown) => this.aiService.resolveError(err),
      onSuccess: (res: AIRepairTemplateResponse) => this.applyRepairResult(res),
      notify: {
        success: 'Entorno reparado por Copiloto IA. Verifique el manifiesto en Modo Avanzado.',
        error: (message: string) => message,
        cancel: 'Reparación cancelada.'
      }
    }
  );
  readonly isRepairingWithAi = this.repairOperation.isRunning;
  readonly repairElapsedSeconds = this.repairOperation.elapsedSeconds;
  readonly repairDiagnosis = signal<{ rationale: string; latencyMs: number } | null>(null);

  readonly hasBlockingDockerfileError = computed<boolean>(() =>
    this.dockerfileDiagnostics().some(d => d.severity === 'error')
  );

  readonly currentVerificationFingerprint = computed<string>(() =>
    computeVerificationFingerprint({
      targetEnvironment: this.targetEnvironment(),
      dockerImage: this.dockerImage(),
      dockerfileContent: this.dockerfileContent(),
      setupScript: this.setupScript(),
      entrypoint: this.entrypoint(),
      toolsDeclared: this.toolsDeclared(),
      timeoutMS: this.timeoutMS(),
      ramMB: this.baseRamMB(),
      sampleInput: this.sampleInput()
    })
  );

  readonly verifiedFingerprint = signal<string | null>(null);

  readonly isConfigModifiedAfterVerification = computed<boolean>(() => {
    const vfp = this.verifiedFingerprint();
    if (!vfp) return false;
    return vfp !== this.currentVerificationFingerprint();
  });

  readonly requiresVerificationBeforePublish = computed<boolean>(() => {
    const isLocal = this.needsLocalBuild() || this.dockerImage().startsWith('solv-local/stack:');
    if (!isLocal) return false;
    return (
      this.activeEnvTestJob()?.status !== 'success' ||
      this.verifiedFingerprint() !== this.currentVerificationFingerprint()
    );
  });

  readonly runtimeCapabilities = signal<RuntimeCapabilities | null>(null);
  readonly activeEnvTestJob = signal<EnvTestJob | null>(null);
  readonly isVerifying = signal<boolean>(false);
  readonly isSubmitting = signal<boolean>(false);
  readonly showTechnicalLogs = signal<boolean>(false);
  readonly toast = signal<{ message: string; type: 'success' | 'error' } | null>(null);

  readonly showComponentModal = signal<boolean>(false);
  readonly modalDefaultLayer = signal<ComponentLayer>('addon');
  readonly modalDefaultSubCategory = signal<string>('backend');
  readonly modalInitialAiPrompt = signal<string>('');
  readonly modalDefaultCompatibleRuntime = computed<string>(() => {
    const rt = this.effectivePrimaryComponent();
    if (!rt) return '';
    return (rt.slug || rt.icon_key || rt.name).toLowerCase();
  });

  readonly backendRuntimes = computed<TemplateComponent[]>(() =>
    this.techComponents().filter(i => i.layer === 'runtime' && i.sub_category !== 'web')
  );

  readonly webPlatforms = computed<TemplateComponent[]>(() =>
    this.techComponents().filter(i => i.layer === 'runtime' && i.sub_category === 'web')
  );

  readonly compatibleAddons = computed<TemplateComponent[]>(() =>
    computeCompatibleAddonsForSelection(
      this.techComponents(),
      this.selectedRuntimeComponent(),
      this.selectedWebComponent(),
      this.targetEnvironment()
    )
  );

  readonly col1SearchQuery = signal<string>('');
  readonly col2SearchQuery = signal<string>('');
  readonly col2CategoryFilter = signal<'all' | 'framework' | 'orm' | 'toolkit'>('all');

  readonly filteredBackendRuntimes = computed<TemplateComponent[]>(() =>
    filterAndPrioritizeRuntimeItems(
      this.backendRuntimes(),
      this.col1SearchQuery(),
      this.selectedRuntimeComponent()?.id
    )
  );

  readonly filteredWebPlatforms = computed<TemplateComponent[]>(() =>
    filterAndPrioritizeRuntimeItems(
      this.webPlatforms(),
      this.col1SearchQuery(),
      this.selectedWebComponent()?.id
    )
  );

  readonly filteredCompatibleAddons = computed<TemplateComponent[]>(() =>
    filterAndPrioritizeAddonItems(
      this.compatibleAddons(),
      this.col2SearchQuery(),
      this.selectedAddonIds()
    )
  );

  readonly filteredFrameworkAddons = computed<TemplateComponent[]>(() =>
    this.filteredCompatibleAddons().filter(a => classifyAddonComponent(a) === 'framework')
  );

  readonly filteredOrmAddons = computed<TemplateComponent[]>(() =>
    this.filteredCompatibleAddons().filter(a => classifyAddonComponent(a) === 'orm')
  );

  readonly filteredToolkitAddons = computed<TemplateComponent[]>(() =>
    this.filteredCompatibleAddons().filter(a => classifyAddonComponent(a) === 'toolkit')
  );

  readonly databaseOptions = computed<StudioDatabaseOption[]>(() =>
    buildStudioDatabaseOptions(this.runtimeCapabilities(), this.techComponents())
  );

  readonly selectedRuntimeComponent = computed<TemplateComponent | null>(() => {
    const id = this.selectedRuntimeId();
    if (!id) return null;
    return this.techComponents().find(i => i.id === id || i.slug === id) || null;
  });

  readonly selectedWebComponent = computed<TemplateComponent | null>(() => {
    if (this.targetEnvironment() === 'JUEZ_EFIMERO' || this.targetEnvironment() === 'AMBOS') return null;
    const id = this.selectedWebPlatformId();
    if (!id) return null;
    return this.techComponents().find(i => i.id === id || i.slug === id) || null;
  });

  readonly effectivePrimaryComponent = computed<TemplateComponent | null>(() =>
    this.selectedRuntimeComponent() || this.selectedWebComponent()
  );

  readonly selectedAddonComponents = computed<TemplateComponent[]>(() => {
    const ids = new Set(this.selectedAddonIds());
    return this.compatibleAddons().filter(a => ids.has(a.id) || ids.has(a.slug));
  });

  readonly selectedFrameworkAddon = computed<TemplateComponent | null>(() =>
    this.selectedAddonComponents().find(a => classifyAddonComponent(a) === 'framework') || null
  );

  readonly selectedOrmAddons = computed<TemplateComponent[]>(() =>
    this.selectedAddonComponents().filter(a => classifyAddonComponent(a) === 'orm')
  );

  readonly selectedToolkitAddons = computed<TemplateComponent[]>(() =>
    this.selectedAddonComponents().filter(a => classifyAddonComponent(a) === 'toolkit')
  );

  readonly showScopeInfoModal = signal<boolean>(false);

  openScopeInfoModal(): void {
    this.showScopeInfoModal.set(true);
  }

  closeScopeInfoModal(): void {
    this.showScopeInfoModal.set(false);
  }

  readonly selectedDatabaseOption = computed<StudioDatabaseOption | null>(() => {
    if (this.targetEnvironment() === 'JUEZ_EFIMERO' || this.targetEnvironment() === 'AMBOS') return null;
    const id = this.selectedDatabaseId();
    if (!id || id === 'db-none' || id === 'none') return null;
    return (
      this.databaseOptions().find(
        i => (i.id === id || i.slug === id || i.service_engine === id) && i.service_engine !== 'none'
      ) || null
    );
  });

  readonly selectedStackComponents = computed<TemplateComponent[]>(() => {
    const list: TemplateComponent[] = [];
    const rt = this.selectedRuntimeComponent();
    if (rt) list.push(rt);
    const web = this.selectedWebComponent();
    if (web) list.push(web);
    list.push(...this.selectedAddonComponents());
    const db = this.selectedDatabaseOption();
    if (db) list.push(db);
    return list;
  });

  readonly toolsList = computed<string[]>(() =>
    this.toolsDeclared().split(',').map(t => t.trim()).filter(Boolean)
  );

  readonly recommendedRamMB = computed<number>(() =>
    computeRecommendedRamMBForStack({
      targetEnvironment: this.targetEnvironment(),
      rt: this.selectedRuntimeComponent(),
      web: this.selectedWebComponent(),
      addons: this.selectedAddonComponents(),
      db: this.selectedDatabaseOption(),
      caps: this.runtimeCapabilities()
    })
  );

  readonly minimumFloorMB = computed<number>(() =>
    computeMinimumFloorMBForStack({
      targetEnvironment: this.targetEnvironment(),
      rt: this.selectedRuntimeComponent(),
      web: this.selectedWebComponent(),
      addons: this.selectedAddonComponents(),
      db: this.selectedDatabaseOption(),
      caps: this.runtimeCapabilities()
    })
  );

  readonly maxAllowedRamMB = computed<number>(() => {
    const caps = this.runtimeCapabilities();
    if (caps?.max_allowed_ram_mb && caps.max_allowed_ram_mb > 0) {
      return caps.max_allowed_ram_mb;
    }
    const total = caps?.host_memory?.total_ram_mb ?? 8192;
    return Math.max(512, Math.floor(total * 0.75));
  });

  readonly hostFreeRamMB = computed<number>(() =>
    this.runtimeCapabilities()?.host_memory?.available_ram_mb ?? 4096
  );

  readonly isRamBelowFloor = computed<boolean>(() => this.baseRamMB() < this.minimumFloorMB());
  readonly isRamExceedingHost = computed<boolean>(() => this.baseRamMB() > this.maxAllowedRamMB());
  readonly ramExcessMB = computed<number>(() => Math.max(0, this.baseRamMB() - this.maxAllowedRamMB()));
  readonly isRamExceedingCurrentFree = computed<boolean>(() => {
    if (this.isRamExceedingHost()) return false;
    return this.baseRamMB() > this.hostFreeRamMB();
  });

  readonly verificationBlocker = computed<StudioActionBlocker | null>(() => {
    if (this.hasBlockingDockerfileError()) {
      return {
        reason: 'Corrija los errores del Dockerfile antes de comprobar o publicar el entorno.',
        tone: 'error'
      };
    }

    if (this.studioMode() === 'visual') {
      if ((this.targetEnvironment() === 'JUEZ_EFIMERO' || this.targetEnvironment() === 'AMBOS') && !this.selectedRuntimeComponent()) {
        return {
          reason: 'Seleccione un Lenguaje Principal en la Columna 1 para configurar el Evaluador Automático.',
          tone: 'neutral'
        };
      }
      if (!this.effectivePrimaryComponent()) {
        return {
          reason: 'Seleccione una tecnología principal en el Paso 2 para habilitar la comprobación y publicación.',
          tone: 'neutral'
        };
      }
    }

    if (!this.dockerImage().trim()) {
      return { reason: 'Indique una imagen Docker base válida en el Paso 2.', tone: 'warning' };
    }

    if ((this.targetEnvironment() === 'JUEZ_EFIMERO' || this.targetEnvironment() === 'AMBOS') && !this.entrypoint().trim()) {
      return {
        reason: 'Indique el comando de compilación y ejecución del Evaluador Automático.',
        tone: 'warning'
      };
    }

    if (this.isRamBelowFloor() || this.isRamExceedingHost()) {
      return { reason: 'Ajuste la memoria RAM dentro de los límites permitidos en el Paso 3.', tone: 'error' };
    }

    return null;
  });

  readonly publishBlocker = computed<StudioActionBlocker | null>(() => {
    const vBlocker = this.verificationBlocker();
    if (vBlocker) return vBlocker;

    if (this.name().trim().length < 3) {
      return {
        reason: 'Complete el nombre de la plantilla (mínimo 3 caracteres) en el Paso 1 para habilitar la publicación.',
        tone: 'neutral'
      };
    }

    if (this.isConfigModifiedAfterVerification()) {
      return {
        reason: 'La configuración cambió después de la verificación. Compruebe el funcionamiento para habilitar la publicación.',
        tone: 'warning'
      };
    }

    if (this.requiresVerificationBeforePublish()) {
      return {
        reason: 'Esta combinación construye una imagen local a medida. Ejecute primero «Comprobar funcionamiento» para construirla y habilitar su publicación.',
        tone: 'neutral'
      };
    }

    return null;
  });

  readonly canRunVerification = computed<boolean>(
    () => !this.isVerifying() && !this.isSubmitting() && this.verificationBlocker() === null
  );

  readonly canPublishTemplate = computed<boolean>(
    () => !this.isVerifying() && !this.isSubmitting() && this.publishBlocker() === null
  );

  readonly intensiveRamMB = computed<number>(() =>
    computeIntensiveRamMB({
      isJudge: this.targetEnvironment() === 'JUEZ_EFIMERO',
      recommendedRamMB: this.recommendedRamMB(),
      maxAllowedRamMB: this.maxAllowedRamMB()
    })
  );

  readonly capacityOptions = computed<CapacityOption[]>(() => {
    if (!this.hasSelectedStack()) return [];
    const isJudge = this.targetEnvironment() === 'JUEZ_EFIMERO';
    const rec = this.recommendedRamMB();
    const intensive = this.intensiveRamMB();

    const options: CapacityOption[] = [
      {
        mb: rec,
        title: isJudge ? 'Entregas y Ejercicios Estándar' : 'Prácticas y Ejercicios de Clase',
        desc: isJudge
          ? 'Memoria óptima para evaluar algoritmos y resoluciones habituales con alta velocidad de respuesta.'
          : 'Asignación óptima para tareas semanales, algoritmos y ejercicios guiados. Permite la mayor cantidad de alumnos conectados a la vez.',
        isRecommended: true
      }
    ];

    if (intensive > rec) {
      options.push({
        mb: intensive,
        title: isJudge ? 'Pruebas con Grandes Volúmenes de Datos' : 'Proyectos Integradores y Trabajos Pesados',
        desc: isJudge
          ? 'Amplía el margen de memoria para ejercicios que procesan estructuras o archivos extensos durante la evaluación.'
          : 'Otorga un margen extra de memoria libre para compilaciones grandes, ejecución de múltiples pruebas o manejo de más datos.',
        isRecommended: false
      });
    }

    return options;
  });

  readonly stackSummaryLabel = computed<string>(() => {
    const rt = this.selectedRuntimeComponent();
    const web = this.selectedWebComponent();
    if (!rt && !web) return 'Ninguna';
    const parts: string[] = [];
    if (rt) parts.push(rt.name);
    if (web) parts.push(web.name);
    for (const a of this.selectedAddonComponents()) {
      parts.push(a.name.split(' ')[0]);
    }
    const db = this.selectedDatabaseOption();
    if (db) parts.push(db.name.split(' ')[0]);
    return parts.join(' + ');
  });

  readonly resourceBreakdown = computed(() => {
    const isJudge = this.targetEnvironment() === 'JUEZ_EFIMERO';
    const caps = this.runtimeCapabilities();
    const totalMB = this.baseRamMB();
    const editorBaseMB = isJudge ? (caps?.runtime_base_mb ?? 32) : (caps?.editor_base_mb ?? 210);
    const floorMB = this.minimumFloorMB();
    const stackBaseMB = Math.max(0, floorMB - editorBaseMB);
    const usableMB = Math.max(0, totalMB - editorBaseMB - stackBaseMB);
    return { isJudge, totalMB, editorBaseMB, stackBaseMB, usableMB };
  });

  readonly nominalConcurrentStudents = computed<number>(() => {
    const ram = Math.max(1, this.baseRamMB());
    return Math.max(1, Math.floor(this.maxAllowedRamMB() / ram));
  });

  readonly estimatedConcurrentStudents = computed<number>(() => {
    const ram = Math.max(1, this.baseRamMB());
    return Math.max(1, Math.floor(this.hostFreeRamMB() / ram));
  });

  estimateStudentsForPreset(mb: number): number {
    return Math.max(1, Math.floor(this.maxAllowedRamMB() / Math.max(1, mb)));
  }

  toggleManualRamInput(): void {
    this.showManualRamInput.update(v => !v);
  }

  readonly liveManifestJson = computed<string>(() =>
    JSON.stringify(this.buildCreatePayload(), null, 2)
  );

  ngOnInit(): void {
    this.loadInitialCatalog();

    const editId = this.route.snapshot.paramMap.get('id');
    const duplicateId = this.route.snapshot.queryParamMap.get('duplicar');
    if (editId) {
      this.editingTemplateId.set(editId);
      this.loadExistingTemplate(editId, false);
    } else if (duplicateId) {
      this.loadExistingTemplate(duplicateId, true);
    } else if (this.pendingDraft && this.isMeaningfulDraft(this.pendingDraft)) {
      this.applyDraft(this.pendingDraft);
      this.toastService.show('Borrador del Studio restaurado.', 'info');
    }

    // Una reparación que terminó mientras el administrador estaba en otra
    // vista se aplica al volver. `take()` la reclama y la consume, así el
    // resultado no se aplica dos veces.
    const pendingRepair = this.repairOperation.take();
    if (pendingRepair) {
      this.applyRepairResult(pendingRepair);
    }
  }

  ngOnDestroy(): void {
    this.pollSub?.unsubscribe();
    this.sub.unsubscribe();
  }

  private loadInitialCatalog(): void {
    this.templatesService.getTechCatalog().subscribe({
      next: items => {
        this.techComponents.set(items || []);
        if (this.loadedTemplateItem) {
          this.applyTemplateToStudio(this.loadedTemplateItem, this.isDuplicateLoad);
        } else {
          this.recalculateFromVisualSelection();
        }
      },
      error: () => {
        this.techComponents.set([]);
        this.showToast('No se pudo cargar el catálogo de tecnologías.', 'error');
      }
    });

    this.templatesService.getCategories().subscribe({
      next: cats => this.categories.set(cats || []),
      error: () => {}
    });

    this.templatesService.getRuntimeCapabilities().subscribe({
      next: caps => {
        if (caps) this.runtimeCapabilities.set(caps);
      },
      error: () => {}
    });

    this.aiService.getStatus().subscribe({
      next: (status: AIStatusDTO) => this.aiEnabled.set(!!status?.enabled),
      error: () => this.aiEnabled.set(false)
    });
  }

  private loadExistingTemplate(id: string, isDuplicate: boolean): void {
    this.templatesService.getTemplateById(id).subscribe({
      next: tpl => {
        if (!tpl) return;
        this.loadedTemplateItem = tpl;
        this.isDuplicateLoad = isDuplicate;
        this.applyTemplateToStudio(tpl, isDuplicate);
      },
      error: () => {}
    });
  }

  private applyTemplateToStudio(tpl: AdminTemplateItem, isDuplicate: boolean): void {
    this.isNameManuallyEdited.set(true);
    this.name.set(isDuplicate ? `(Copia) ${tpl.name}` : tpl.name);
    this.description.set(tpl.description || '');
    this.targetEnvironment.set((tpl.target_environment as TargetEnvironment) || 'IDE_PERSISTENTE');
    if (tpl.category_id) this.selectedCategoryId.set(tpl.category_id);

    const cfg = tpl.stack_config;
    if (cfg?.mode === 'code' || cfg?.mode === 'visual') {
      this.studioMode.set(cfg.mode);
    }

    if (cfg && (cfg.runtime_id || cfg.web_platform_id)) {
      this.selectedRuntimeId.set(cfg.runtime_id || null);
      this.selectedWebPlatformId.set(cfg.web_platform_id || null);
      this.selectedAddonIds.set(cfg.addon_ids || []);
      this.selectedDatabaseId.set(cfg.database_id || 'db-none');
      this.recalculateFromVisualSelection();

      if (cfg.dockerfile_content && cfg.dockerfile_content !== this.dockerfileContent()) {
        this.dockerfileContent.set(cfg.dockerfile_content);
        this.hasCustomDockerfile.set(true);
        this.dockerImage.set(tpl.docker_image);
        this.needsLocalBuild.set(tpl.docker_image.startsWith('solv-local/stack:'));
      }

      if (tpl.setup_script !== undefined && tpl.setup_script !== null) {
        this.setupScript.set(tpl.setup_script);
      }
      if (tpl.tools_declared?.length) {
        this.toolsDeclared.set(tpl.tools_declared.join(', '));
      }
      if (tpl.entrypoint !== undefined && tpl.entrypoint !== null) {
        this.entrypoint.set(tpl.entrypoint);
      }
      if (tpl.timeout_ms) {
        this.timeoutMS.set(tpl.timeout_ms);
      }
      if (tpl.sample_input !== undefined && tpl.sample_input !== null) {
        this.sampleInput.set(tpl.sample_input);
      }
    } else {
      this.dockerImage.set(tpl.docker_image);
      this.dockerfileContent.set(cfg?.dockerfile_content || (tpl.docker_image ? 'FROM ' + tpl.docker_image : ''));
      this.needsLocalBuild.set(tpl.docker_image.startsWith('solv-local/stack:'));
      this.hasCustomDockerfile.set(!!cfg?.dockerfile_content);
      this.setupScript.set(tpl.setup_script || '');
      this.toolsDeclared.set((tpl.tools_declared || []).join(', '));
      this.entrypoint.set(tpl.entrypoint || '');
      this.timeoutMS.set(tpl.timeout_ms || 5000);
      this.sampleInput.set(tpl.sample_input || '');
    }

    if (tpl.base_ram_mb) {
      this.isRamManuallyEdited.set(true);
      this.baseRamMB.set(tpl.base_ram_mb);
    }
  }

  setTargetEnvironment(env: TargetEnvironment): void {
    this.targetEnvironment.set(env);
    if (env === 'JUEZ_EFIMERO' || env === 'AMBOS') {
      this.selectedWebPlatformId.set(null);
      this.selectedDatabaseId.set('db-none');
    }
    this.isRamManuallyEdited.set(false);
    this.recalculateFromVisualSelection();
  }

  setStudioMode(mode: 'visual' | 'code'): void {
    this.studioMode.set(mode);
  }

  selectRuntime(id: string): void {
    this.hasCustomDockerfile.set(false);
    this.col2SearchQuery.set('');
    this.col2CategoryFilter.set('all');
    this.isRamManuallyEdited.set(false);
    const next = this.selectedRuntimeId() === id ? null : id;
    this.selectedRuntimeId.set(next);
    const validIds = new Set(this.compatibleAddons().map(a => a.id));
    const validSlugs = new Set(this.compatibleAddons().map(a => a.slug));
    const allComponents = this.techComponents();
    let hasFramework = false;
    this.selectedAddonIds.update(curr => {
      const valid = curr.filter(x => validIds.has(x) || validSlugs.has(x));
      return valid.filter(addonId => {
        const comp = allComponents.find(c => c.id === addonId || c.slug === addonId);
        if (comp && isFrameworkAddon(comp)) {
          if (hasFramework) {
            return false;
          }
          hasFramework = true;
        }
        return true;
      });
    });
    this.recalculateFromVisualSelection();
  }

  onDockerfileManualChange(content: string): void {
    this.dockerfileContent.set(content);
    this.hasCustomDockerfile.set(true);
    if (this.activeEnvTestJob()?.status !== 'success') {
      this.activeEnvTestJob.set(null);
    }

    const activeLines = content
      .split('\n')
      .map(l => l.trim())
      .filter(l => l.length > 0 && !l.startsWith('#'));

    const fromLine = activeLines.find(l => l.toUpperCase().startsWith('FROM '));
    if (fromLine) {
      const parts = fromLine.split(/\s+/).filter(Boolean);
      const imageToken = parts[1] || '';
      const otherInstructions = activeLines.filter(l => !l.toUpperCase().startsWith('FROM '));
      if (otherInstructions.length > 0) {
        this.needsLocalBuild.set(true);
        const hash = fnv1aHex(content.trim());
        this.dockerImage.set(`solv-local/stack:${hash}`);
      } else {
        this.needsLocalBuild.set(false);
        this.dockerImage.set(imageToken);
      }
    }
  }

  resetDockerfileFromVisual(): void {
    this.hasCustomDockerfile.set(false);
    this.recalculateFromVisualSelection();
  }

  onSetupScriptManualChange(content: string): void {
    this.setupScript.set(content);
  }

  downloadTextFile(filename: string, content: string, mimeType: string = 'text/plain'): void {
    downloadBrowserTextFile(filename, content, mimeType);
  }

  importTextFile(accept: string, callback: (text: string) => void): void {
    importBrowserTextFile(accept, callback);
  }

  downloadDockerfile(): void {
    const content = this.dockerfileContent();
    if (!content.trim()) return;
    this.downloadTextFile('Dockerfile', content);
  }

  importDockerfile(): void {
    this.importTextFile('.dockerfile,Dockerfile,text/plain,*', content => {
      this.onDockerfileManualChange(content);
    });
  }

  downloadSetupScript(): void {
    const content = this.setupScript();
    if (!content.trim()) return;
    this.downloadTextFile('setup.sh', content);
  }

  importSetupScript(): void {
    this.importTextFile('.sh,text/x-shellscript,text/plain,*', content => {
      this.onSetupScriptManualChange(content);
    });
  }

  downloadManifest(): void {
    this.downloadTextFile('template-manifest.json', this.liveManifestJson(), 'application/json');
  }

  selectWebPlatform(id: string | null): void {
    this.hasCustomDockerfile.set(false);
    this.col2SearchQuery.set('');
    this.col2CategoryFilter.set('all');
    const next = this.selectedWebPlatformId() === id ? null : id;
    this.selectedWebPlatformId.set(next);
    const validIds = new Set(this.compatibleAddons().map(a => a.id));
    const validSlugs = new Set(this.compatibleAddons().map(a => a.slug));
    this.selectedAddonIds.update(curr => curr.filter(x => validIds.has(x) || validSlugs.has(x)));
    this.recalculateFromVisualSelection();
  }

  getRoleBadge(comp: TemplateComponent | StudioDatabaseOption): string {
    return getComponentRoleBadge(comp);
  }

  toggleAddon(id: string): void {
    const isCurrentlySelected = this.isAddonSelected(id);
    if (isCurrentlySelected) {
      const comp = this.techComponents().find(c => c.id === id || c.slug === id);
      const toRemove = new Set<string>(comp ? [comp.id, comp.slug] : [id]);
      this.selectedAddonIds.update(curr => curr.filter(x => !toRemove.has(x)));
    } else {
      const compToSelect = this.techComponents().find(c => c.id === id || c.slug === id);
      if (compToSelect && isFrameworkAddon(compToSelect)) {
        const allComponents = this.techComponents();
        this.selectedAddonIds.update(curr => {
          const nonFrameworks = curr.filter(currId => {
            const c = allComponents.find(item => item.id === currId || item.slug === currId);
            return !c || !isFrameworkAddon(c);
          });
          return [...nonFrameworks, id];
        });
      } else {
        this.selectedAddonIds.update(curr => [...curr, id]);
      }
    }
    this.recalculateFromVisualSelection();
  }

  isAddonSelected(id: string): boolean {
    const ids = this.selectedAddonIds();
    if (ids.includes(id)) return true;
    const comp = this.techComponents().find(c => c.id === id || c.slug === id);
    if (comp) {
      return ids.includes(comp.id) || ids.includes(comp.slug);
    }
    return false;
  }

  selectDatabase(id: string): void {
    const db = this.databaseOptions().find(i => i.id === id || i.slug === id || i.service_engine === id);
    if (!db || !db.is_available) return;
    this.selectedDatabaseId.set(db.id);
    this.recalculateFromVisualSelection();
  }

  getDatabaseDriverName(engine?: string): string {
    if (!engine || engine === 'none') return '';
    const rtSlug = (this.selectedRuntimeComponent()?.slug || this.selectedRuntimeComponent()?.icon_key || '').toLowerCase();

    if (rtSlug.includes('node') || rtSlug.includes('typescript')) {
      switch (engine.toLowerCase()) {
        case 'postgres': return 'pg';
        case 'mongodb': return 'mongodb';
        case 'redis': return 'ioredis';
        case 'mysql':
        case 'mariadb': return 'mysql2';
        default: return 'conector oficial';
      }
    }
    if (rtSlug.includes('go')) {
      switch (engine.toLowerCase()) {
        case 'postgres': return 'pgx';
        case 'mongodb': return 'mongo-driver';
        case 'redis': return 'go-redis';
        case 'mysql':
        case 'mariadb': return 'go-sql-driver';
        default: return 'conector oficial';
      }
    }
    if (rtSlug.includes('java')) {
      switch (engine.toLowerCase()) {
        case 'postgres': return 'postgresql-jdbc';
        case 'mongodb': return 'mongo-java-driver';
        case 'redis': return 'jedis';
        case 'mysql':
        case 'mariadb': return 'mysql-connector-j';
        default: return 'conector oficial';
      }
    }
    // Default Python
    switch (engine.toLowerCase()) {
      case 'postgres': return 'psycopg / SQLAlchemy';
      case 'mongodb': return 'pymongo';
      case 'redis': return 'redis-py';
      case 'mysql':
      case 'mariadb': return 'pymysql';
      default: return 'conector oficial';
    }
  }

  onNameInput(val: string): void {
    this.isNameManuallyEdited.set(val.trim().length > 0);
    this.name.set(val);
  }

  selectRamPreset(mb: number): void {
    this.isRamManuallyEdited.set(mb !== this.recommendedRamMB());
    this.baseRamMB.set(mb);
    if (this.activeEnvTestJob()?.status !== 'success') {
      this.activeEnvTestJob.set(null);
    }
  }

  onCustomRamChange(val: number | string): void {
    const parsed = typeof val === 'number' ? val : parseInt(String(val).replace(/\D/g, ''), 10);
    if (!isNaN(parsed) && parsed > 0) {
      this.isRamManuallyEdited.set(true);
      this.baseRamMB.set(parsed);
      if (this.activeEnvTestJob()?.status !== 'success') {
        this.activeEnvTestJob.set(null);
      }
    }
  }

  recalculateFromVisualSelection(): void {
    const rt = this.selectedRuntimeComponent();
    const web = this.selectedWebComponent();
    if (!rt && !web) {
      if (!this.isNameManuallyEdited()) this.name.set('');
      if (!this.hasCustomDockerfile()) {
        this.needsLocalBuild.set(false);
        this.dockerImage.set('');
        this.dockerfileContent.set('');
      }
      this.toolsDeclared.set('');
      this.setupScript.set('');
      this.entrypoint.set('');
      this.activeEnvTestJob.set(null);
      this.verifiedFingerprint.set(null);
      if (!this.isRamManuallyEdited()) {
        this.baseRamMB.set(512);
      }
      return;
    }

    const synthesized = synthesizeVisualStackState({
      targetEnvironment: this.targetEnvironment(),
      rt,
      web,
      addons: this.selectedAddonComponents(),
      db: this.selectedDatabaseOption()
    });

    if (!this.isNameManuallyEdited()) {
      this.name.set(synthesized.suggestedName);
    }

    this.needsLocalBuild.set(synthesized.needsBuild);
    if (!this.hasCustomDockerfile()) {
      this.dockerImage.set(synthesized.dockerImage);
      this.dockerfileContent.set(synthesized.dockerfileContent);
    }

    this.toolsDeclared.set(synthesized.toolsDeclared);
    this.setupScript.set(synthesized.setupScript);
    this.entrypoint.set(synthesized.entrypoint);

    if (!this.isRamManuallyEdited() || this.baseRamMB() < this.minimumFloorMB()) {
      this.baseRamMB.set(this.recommendedRamMB());
      if (this.baseRamMB() < this.minimumFloorMB()) {
        this.isRamManuallyEdited.set(false);
      }
    }
  }

  readonly editingComponent = signal<TemplateComponent | null>(null);

  openNewComponentModal(layer: ComponentLayer, subCategory: string = 'backend', initialPrompt: string = ''): void {
    this.editingComponent.set(null);
    this.modalDefaultLayer.set(layer);
    this.modalDefaultSubCategory.set(subCategory);
    this.modalInitialAiPrompt.set(initialPrompt);
    this.showComponentModal.set(true);
  }

  openCreateMissingFromProposal(event: { name: string; missing?: AIMissingComponent }): void {
    const layer = (event.missing?.layer as ComponentLayer) || 'addon';
    const sub = (event.missing?.category === 'web' ? 'web' : 'backend') as 'backend' | 'web';
    const prompt = event.missing?.reason
      ? `Configurar ${event.name} (${layer}): ${event.missing.reason}`
      : `Configurar ${event.name} como ${layer} para el catálogo oficial`;
    this.openNewComponentModal(layer, sub, prompt);
  }

  openEditComponentModal(comp: TemplateComponent): void {
    this.editingComponent.set(comp);
    this.modalDefaultLayer.set(comp.layer);
    this.modalDefaultSubCategory.set(comp.sub_category === 'web' ? 'web' : 'backend');
    this.showComponentModal.set(true);
  }

  handleModalClosed(): void {
    this.showComponentModal.set(false);
    this.editingComponent.set(null);
    this.modalInitialAiPrompt.set('');
  }

  handleModalComponentSaved(saved: TemplateComponent): void {
    this.showComponentModal.set(false);
    if (saved.layer === 'runtime') {
      this.col1SearchQuery.set('');
    } else if (saved.layer === 'addon') {
      this.col2SearchQuery.set('');
    }
    const editing = this.editingComponent();

    if (editing) {
      this.techComponents.update(list =>
        list.map(item => (item.id === saved.id || item.slug === saved.slug ? saved : item))
      );
      const isSelectedRuntime = this.selectedRuntimeId() === saved.id || this.selectedRuntimeId() === saved.slug;
      const isSelectedWeb = this.selectedWebPlatformId() === saved.id || this.selectedWebPlatformId() === saved.slug;
      const isSelectedAddon = this.selectedAddonIds().includes(saved.id) || this.selectedAddonIds().includes(saved.slug);
      const isSelectedDb = this.selectedDatabaseId() === saved.id || this.selectedDatabaseId() === saved.slug;

      if (isSelectedRuntime || isSelectedWeb || isSelectedAddon || isSelectedDb) {
        this.recalculateFromVisualSelection();
      }
      this.editingComponent.set(null);
      this.showToast(`Tecnología "${saved.name}" actualizada.`);
    } else {
      this.techComponents.update(list => [...list, saved]);
      // Si la propuesta del copiloto está abierta, sincronizar la nueva pieza en la propuesta activa
      const currentProp = this.aiProposal();
      if (currentProp) {
        if (saved.layer === 'addon') {
          this.aiProposal.set({
            ...currentProp,
            addon_slugs: [...new Set([...(currentProp.addon_slugs || []), saved.slug || saved.id])]
          });
        } else if (saved.layer === 'runtime' && saved.sub_category !== 'web' && !currentProp.runtime_slug) {
          this.aiProposal.set({
            ...currentProp,
            runtime_slug: saved.slug || saved.id
          });
        }
      }

      if (saved.layer === 'runtime') {
        if (saved.sub_category === 'web') {
          this.selectWebPlatform(saved.id);
        } else {
          this.selectRuntime(saved.id);
        }
      } else if (saved.layer === 'addon') {
        this.toggleAddon(saved.id);
      } else {
        this.selectDatabase(saved.id);
      }
      this.showToast(`Tecnología "${saved.name}" agregada y seleccionada.`);
    }
  }

  handleNewComponentSaved(created: TemplateComponent): void {
    this.handleModalComponentSaved(created);
  }

  runVerificationTest(): void {
    const blocker = this.verificationBlocker();
    if (blocker) {
      this.showToast(blocker.reason, 'error');
      return;
    }

    const testedFingerprint = this.currentVerificationFingerprint();
    this.isVerifying.set(true);
    this.pollSub?.unsubscribe();
    this.pollSub = undefined;

    const req = {
      image: this.dockerImage().trim(),
      tools: this.toolsList(),
      target_environment: this.targetEnvironment(),
      entrypoint: this.entrypoint().trim(),
      timeout_ms: this.timeoutMS(),
      sample_input: this.sampleInput(),
      base_ram_mb: this.baseRamMB(),
      dockerfile_content: this.dockerfileContent(),
      setup_script: this.setupScript().trim()
    };

    this.envTestService.startJob(req).subscribe({
      next: job => {
        this.activeEnvTestJob.set(job);
        this.pollSub = this.envTestService.pollJob(job.id).subscribe({
          next: updated => {
            this.activeEnvTestJob.set(updated);
            if (updated.status === 'success') {
              this.verifiedFingerprint.set(testedFingerprint);
            }
            if (this.envTestService.isTerminal(updated.status)) {
              this.isVerifying.set(false);
              this.pollSub?.unsubscribe();
              this.pollSub = undefined;
            }
          },
          error: () => {
            this.isVerifying.set(false);
            this.pollSub?.unsubscribe();
            this.pollSub = undefined;
          }
        });
        this.sub.add(this.pollSub);
      },
      error: err => {
        this.isVerifying.set(false);
        this.showToast(err.error?.error?.message || 'No se pudo iniciar la comprobación del entorno.', 'error');
      }
    });
  }

  cancelEnvTest(): void {
    const job = this.activeEnvTestJob();
    if (!job || !this.isVerifying()) return;
    this.pollSub?.unsubscribe();
    this.pollSub = undefined;
    this.activeEnvTestJob.update(j => (j ? { ...j, status: 'canceled' } : null));
    this.isVerifying.set(false);
    this.envTestService.cancelJob(job.id).subscribe({
      next: () => this.showToast('Comprobación de entorno cancelada.'),
      error: () => {}
    });
  }

  copyDockerfileToClipboard(): void {
    const content = this.dockerfileContent();
    if (!content) return;
    navigator.clipboard.writeText(content).then(() => {
      this.showToast('Dockerfile copiado al portapapeles.');
    });
  }

  copyManifestToClipboard(): void {
    navigator.clipboard.writeText(this.liveManifestJson()).then(() => {
      this.showToast('Manifiesto JSON copiado al portapapeles.');
    });
  }

  publishTemplate(): void {
    const blocker = this.publishBlocker();
    if (blocker) {
      this.showToast(blocker.reason, 'error');
      return;
    }

    const dto = this.buildCreatePayload();
    this.isSubmitting.set(true);

    const editId = this.editingTemplateId();
    const obs$ = editId
      ? this.templatesService.updateOfficialTemplate(editId, dto)
      : this.templatesService.createOfficialTemplate(dto);

    obs$.subscribe({
      next: () => {
        this.isSubmitting.set(false);
        this.copilot.clearDraft();
        this.router.navigate(['/admin/plantillas'], { queryParams: { tab: 'catalog' } });
      },
      error: err => {
        this.isSubmitting.set(false);
        this.showToast(err.error?.error?.message || 'Error al guardar la plantilla en el catálogo.', 'error');
      }
    });
  }

  buildCreatePayload(): CreateOfficialTemplateDTO {
    const db = this.selectedDatabaseOption();
    const services: ServiceRequirement[] = [];
    if (db?.service_engine && db.service_engine !== 'none') {
      services.push({
        category: db.service_engine === 'redis' ? 'cache' : 'database',
        engine: db.service_engine
      });
    }

    const stackConfig: StackConfig = {
      mode: this.studioMode(),
      runtime_id: this.selectedRuntimeId() || undefined,
      web_platform_id: this.selectedWebPlatformId() || undefined,
      addon_ids: this.selectedAddonIds(),
      database_id: this.selectedDatabaseId(),
      dockerfile_content: this.dockerfileContent().trim() || undefined
    };

    const minMB = this.minimumFloorMB();
    const maxMB = this.baseRamMB();
    const highMB = Math.max(minMB, Math.floor(maxMB * 0.85));

    return {
      name: this.name().trim(),
      docker_image: this.dockerImage().trim(),
      base_ram_mb: this.baseRamMB(),
      description: this.description().trim(),
      target_environment: this.targetEnvironment(),
      category_id: this.selectedCategoryId() || undefined,
      entrypoint: this.entrypoint().trim() || undefined,
      timeout_ms: this.timeoutMS(),
      sample_input: this.sampleInput() || undefined,
      services_config: { services },
      resource_profile: { min_mb: minMB, high_mb: highMB, max_mb: maxMB },
      setup_script: this.setupScript().trim(),
      tools_declared: this.toolsList(),
      stack_config: stackConfig,
      verification_job_id: this.needsLocalBuild() || this.dockerImage().startsWith('solv-local/stack:')
        ? this.activeEnvTestJob()?.id
        : undefined
    };
  }

  // Flujo de Copiloto IA (Sugerencia de Stack)
  // La operación vive en StudioCopilotService para que sobreviva a la
  // navegación y al cierre del modal; acá solo se dispara y se refleja.
  suggestStackWithAi(): void {
    this.copilot.suggest(this.targetEnvironment());
  }

  // Resuelve la propuesta del Copiloto contra el catálogo real del tenant.
  // Cada slug propuesto se empareja por id, slug o icon_key; los que no existen
  // se marcan en `missing` para que el administrador los vea antes de aplicar.
  readonly aiProposalView = computed<StudioAiProposalView | null>(() => {
    const res = this.aiProposal();
    if (!res) return null;

    const runtime = res.runtime_slug
      ? findCatalogComponentMatch(this.techComponents(), res.runtime_slug)
      : null;
    const web =
      this.targetEnvironment() === 'IDE_PERSISTENTE' && res.web_platform_slug
        ? findCatalogComponentMatch(this.techComponents(), res.web_platform_slug)
        : null;
    const addons: TemplateComponent[] = [];
    const addonSlugs: string[] = [];
    const missing: string[] = [];
    if (Array.isArray(res.addon_slugs)) {
      for (const slug of res.addon_slugs) {
        addonSlugs.push(slug);
        const comp = findCatalogComponentMatch(this.techComponents(), slug);
        if (comp) {
          addons.push(comp);
        } else {
          missing.push(slug);
        }
      }
    }
    if (!runtime && res.runtime_slug) missing.push(res.runtime_slug);
    if (!web && res.web_platform_slug && this.targetEnvironment() === 'IDE_PERSISTENTE') {
      missing.push(res.web_platform_slug);
    }
    const database = res.database_slug
      ? findDatabaseOptionMatch(this.databaseOptions(), res.database_slug)
      : null;
    if (!database && res.database_slug) missing.push(res.database_slug);

    const proposalAddonChoices = computeCompatibleAddonsForSelection(
      this.techComponents(),
      runtime,
      web,
      this.targetEnvironment()
    );

    return {
      runtime,
      runtimeSlug: res.runtime_slug || '',
      web,
      webSlug: res.web_platform_slug || '',
      addons,
      addonSlugs,
      database,
      databaseSlug: res.database_slug || '',
      missing: [...new Set(missing)],
      missingComponents: res.missing_components,
      runtimeChoices: this.backendRuntimes(),
      webChoices: this.webPlatforms(),
      addonChoices: proposalAddonChoices,
      databaseChoices: this.databaseOptions(),
      ramMB: res.base_ram_mb,
      hasSetupScript: !!res.setup_script?.trim(),
      rationale: res.rationale,
      latencyMs: res.latency_ms
    };
  });

  // Constelación visual del stack: usa la propuesta del Copiloto cuando existe;
  // si no, refleja la selección actual del Paso 2 (todo compatible por diseño).
  readonly stackConstellation = computed<StackConstellation | null>(() => {
    const proposal = this.aiProposalView();
    if (proposal) {
      return buildConstellationFromProposal(proposal, this.targetEnvironment());
    }
    return buildConstellationFromSelection(
      this.selectedRuntimeComponent(),
      this.selectedWebComponent(),
      this.selectedAddonComponents(),
      this.selectedDatabaseOption(),
      this.targetEnvironment()
    );
  });

  updateAiProposalView(view: StudioAiProposalView): void {
    const current = this.aiProposal();
    if (!current) return;

    this.aiProposal.set({
      ...current,
      runtime_slug: view.runtime?.slug ?? view.runtimeSlug,
      web_platform_slug: view.web?.slug ?? view.webSlug,
      addon_slugs: view.addons.map(addon => addon.slug).concat(view.missing.filter(slug => view.addonSlugs.includes(slug))),
      database_slug: view.database?.service_engine ?? view.databaseSlug,
      base_ram_mb: view.ramMB
    });
  }

  // Aplica la propuesta revisada: sincroniza el constructor visual y el
  // manifiesto técnico del Modo Avanzado (Dockerfile, script de setup, RAM).
  applyAiProposal(): void {
    const res = this.aiProposal();
    const view = this.aiProposalView();
    if (!res || !view || this.isSuggestingWithAi()) return;

    // 2. Componente runtime (id real del catálogo)
    this.selectedRuntimeId.set(view.runtime ? view.runtime.id : null);

    // 3. Plataforma web para IDE_PERSISTENTE
    this.selectedWebPlatformId.set(view.web ? view.web.id : null);

    // 4. Addons (solo los presentes en el catálogo)
    this.selectedAddonIds.set(view.addons.map((a) => a.id));

    // 5. Database
    this.selectedDatabaseId.set(view.database ? view.database.id : 'db-none');

    // 6. Sincronizar visual + avanzado
    this.hasCustomDockerfile.set(false);
    this.recalculateFromVisualSelection();

    // 1. Nombre y descripción si no fueron editados manualmente
    if (res.name && !this.isNameManuallyEdited()) {
      this.name.set(res.name);
    }
    if (res.description) {
      this.description.set(res.description);
    }

    // 7. Script de setup y RAM base
    if (view.hasSetupScript) {
      this.setupScript.set(res.setup_script);
    }
    if (typeof res.base_ram_mb === 'number' && res.base_ram_mb >= 256) {
      this.baseRamMB.set(res.base_ram_mb);
    }

    this.aiProposalApplied.set(true);
    this.showToast('Configuración asistida aplicada al constructor.');
  }

  // Descarta la propuesta del Copiloto sin tocar el constructor.
  discardAiProposal(): void {
    this.copilot.discard();
    this.showToast('Propuesta descartada; el constructor no se modificó.');
  }

  cancelAiSuggestion(): void {
    this.copilot.cancel();
  }

  openAiCopilotModal(): void {
    this.isAiCopilotModalOpen.set(true);
  }

  openTechGraph(): void {
    this.isTechGraphOpen.set(true);
  }

  onApplyAiCopilotModal(): void {
    this.applyAiProposal();
    this.isAiCopilotModalOpen.set(false);
  }

  // Flujo de Autorreparación (Corregir con Copiloto IA)
  // Igual que la sugerencia de stack: la llamada vive en una operación con
  // dueño en la raíz, así que navegar no la aborta y el resultado se aplica
  // al volver.
  repairWithAi(): void {
    const job = this.activeEnvTestJob();
    if (!job || job.status !== 'failed' || this.isRepairingWithAi()) return;

    this.repairDiagnosis.set(null);

    const failedTools: string[] = (job.result?.tools || [])
      .filter((t) => !t.present)
      .map((t) => t.name);

    this.repairOperation.run(
      this.aiService.repairTemplate({
        name: this.name(),
        description: this.description(),
        target_environment: this.targetEnvironment(),
        docker_image: this.dockerImage(),
        dockerfile_content: this.dockerfileContent(),
        setup_script: this.setupScript(),
        tools_declared: this.toolsList(),
        entrypoint: this.entrypoint(),
        base_ram_mb: this.baseRamMB(),
        error_message: job.error_message || '',
        failed_tools: failedTools
      })
    );
  }

  cancelRepairWithAi(): void {
    this.repairOperation.cancel();
  }

  private applyRepairResult(res: AIRepairTemplateResponse): void {
    // 1. Aplica las correcciones en el Modo Avanzado
    this.dockerfileContent.set(res.dockerfile_content);
    this.hasCustomDockerfile.set(true);
    this.dockerImage.set(res.docker_image);
    this.needsLocalBuild.set(res.docker_image.startsWith('solv-local/stack:'));
    this.setupScript.set(res.setup_script);
    this.toolsDeclared.set((res.tools_declared || []).join(', '));
    this.entrypoint.set(res.entrypoint);
    this.baseRamMB.set(res.base_ram_mb);

    // 2. Conmuta automáticamente a Modo Avanzado y muestra diagnóstico
    this.studioMode.set('code');
    this.repairDiagnosis.set({
      rationale: res.rationale,
      latencyMs: res.latency_ms
    });
  }

  // --- Persistencia del borrador del Studio (sobrevive a la navegación) ---
  private captureDraft(): StudioDraft {
    return {
      name: this.name(),
      description: this.description(),
      selectedCategoryId: this.selectedCategoryId(),
      targetEnvironment: this.targetEnvironment(),
      studioMode: this.studioMode(),
      blueprintTab: this.blueprintTab(),
      selectedRuntimeId: this.selectedRuntimeId(),
      selectedWebPlatformId: this.selectedWebPlatformId(),
      selectedAddonIds: [...this.selectedAddonIds()],
      selectedDatabaseId: this.selectedDatabaseId(),
      dockerImage: this.dockerImage(),
      dockerfileContent: this.dockerfileContent(),
      hasCustomDockerfile: this.hasCustomDockerfile(),
      needsLocalBuild: this.needsLocalBuild(),
      setupScript: this.setupScript(),
      toolsDeclared: this.toolsDeclared(),
      entrypoint: this.entrypoint(),
      timeoutMS: this.timeoutMS(),
      sampleInput: this.sampleInput(),
      baseRamMB: this.baseRamMB(),
      isNameManuallyEdited: this.isNameManuallyEdited(),
      isRamManuallyEdited: this.isRamManuallyEdited()
    };
  }

  private applyDraft(draft: StudioDraft): void {
    this.name.set(draft.name);
    this.description.set(draft.description);
    this.selectedCategoryId.set(draft.selectedCategoryId);
    this.targetEnvironment.set(draft.targetEnvironment);
    this.studioMode.set(draft.studioMode);
    this.blueprintTab.set(draft.blueprintTab);
    this.selectedRuntimeId.set(draft.selectedRuntimeId);
    this.selectedWebPlatformId.set(draft.selectedWebPlatformId);
    this.selectedAddonIds.set([...draft.selectedAddonIds]);
    this.selectedDatabaseId.set(draft.selectedDatabaseId);
    this.dockerImage.set(draft.dockerImage);
    this.dockerfileContent.set(draft.dockerfileContent);
    this.hasCustomDockerfile.set(draft.hasCustomDockerfile);
    this.needsLocalBuild.set(draft.needsLocalBuild);
    this.setupScript.set(draft.setupScript);
    this.toolsDeclared.set(draft.toolsDeclared);
    this.entrypoint.set(draft.entrypoint);
    this.timeoutMS.set(draft.timeoutMS);
    this.sampleInput.set(draft.sampleInput);
    this.baseRamMB.set(draft.baseRamMB);
    this.isNameManuallyEdited.set(draft.isNameManuallyEdited);
    this.isRamManuallyEdited.set(draft.isRamManuallyEdited);
  }

  private isMeaningfulDraft(draft: StudioDraft): boolean {
    return Boolean(
      draft.name.trim() ||
        draft.description.trim() ||
        draft.dockerImage.trim() ||
        draft.selectedRuntimeId ||
        draft.selectedAddonIds.length > 0 ||
        draft.setupScript.trim()
    );
  }

  goBack(): void {
    this.router.navigate(['/admin/plantillas']);
  }

  private showToast(message: string, type: 'success' | 'error' = 'success'): void {
    this.toast.set({ message, type });
    setTimeout(() => {
      if (this.toast()?.message === message) {
        this.toast.set(null);
      }
    }, 4000);
  }
}
