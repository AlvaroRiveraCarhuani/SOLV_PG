import { Component, OnInit, OnDestroy, computed, inject, signal } from '@angular/core';
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
import { TechLogoComponent } from '../components/tech-logo/tech-logo.component';
import { ComponentFormModalComponent } from '../components/model-library/components/component-form-modal/component-form-modal.component';
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
  LucidePencil
} from '@lucide/angular';
import {
  analyzeDockerfile,
  buildStudioDatabaseOptions,
  CapacityOption,
  computeCompatibleAddonsForSelection,
  computeMinimumFloorMBForStack,
  computeRecommendedRamMBForStack,
  computeVerificationFingerprint,
  DockerfileDiagnostic,
  downloadBrowserTextFile,
  filterAndPrioritizeAddonItems,
  filterAndPrioritizeRuntimeItems,
  fnv1aHex,
  importBrowserTextFile,
  StudioActionBlocker,
  StudioDatabaseOption,
  synthesizeVisualStackState
} from './template-studio.utils';
import { StudioPurposeStepComponent } from './components/studio-purpose-step/studio-purpose-step.component';
import { StudioCapacitySelectorComponent } from './components/studio-capacity-selector/studio-capacity-selector.component';
import { StudioSummarySidebarComponent } from './components/studio-summary-sidebar/studio-summary-sidebar.component';

export type { CapacityOption, StudioDatabaseOption, StudioActionBlocker };

@Component({
  selector: 'template-studio',
  imports: [
    FormsModule,
    TechLogoComponent,
    ComponentFormModalComponent,
    CodeEditorComponent,
    MachineDataDirective,
    StudioPurposeStepComponent,
    StudioCapacitySelectorComponent,
    StudioSummarySidebarComponent,
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
    LucidePencil
  ],
  templateUrl: './template-studio.component.html',
  styleUrls: ['./template-studio.component.scss']
})
export class TemplateStudioComponent implements OnInit, OnDestroy {
  private readonly templatesService = inject(AdminTemplatesService);
  private readonly envTestService = inject(EnvTestJobService);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);

  private readonly sub = new Subscription();
  private pollSub?: Subscription;
  private loadedTemplateItem: AdminTemplateItem | null = null;
  private isDuplicateLoad = false;

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
      ramMB: this.baseRamMB()
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
      this.isConfigModifiedAfterVerification()
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
  readonly modalDefaultSubCategory = signal<'backend' | 'web'>('backend');
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

  readonly intensiveRamMB = computed<number>(() => {
    const isJudge = this.targetEnvironment() === 'JUEZ_EFIMERO';
    const rec = this.recommendedRamMB();
    const maxAllowed = this.maxAllowedRamMB();
    const tiers = isJudge ? [128, 256, 512, 1024] : [384, 512, 768, 1024, 1536, 2048];
    const nextTier = tiers.find(t => t > rec) ?? rec;
    return Math.min(nextTier, maxAllowed);
  });

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
    const next = this.selectedRuntimeId() === id ? null : id;
    this.selectedRuntimeId.set(next);
    const validIds = new Set(this.compatibleAddons().map(a => a.id));
    const validSlugs = new Set(this.compatibleAddons().map(a => a.slug));
    this.selectedAddonIds.update(curr => curr.filter(x => validIds.has(x) || validSlugs.has(x)));
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
    const next = this.selectedWebPlatformId() === id ? null : id;
    this.selectedWebPlatformId.set(next);
    const validIds = new Set(this.compatibleAddons().map(a => a.id));
    const validSlugs = new Set(this.compatibleAddons().map(a => a.slug));
    this.selectedAddonIds.update(curr => curr.filter(x => validIds.has(x) || validSlugs.has(x)));
    this.recalculateFromVisualSelection();
  }

  toggleAddon(id: string): void {
    this.selectedAddonIds.update(curr => (curr.includes(id) ? curr.filter(x => x !== id) : [...curr, id]));
    this.recalculateFromVisualSelection();
  }

  isAddonSelected(id: string): boolean {
    return this.selectedAddonIds().includes(id);
  }

  selectDatabase(id: string): void {
    const db = this.databaseOptions().find(i => i.id === id || i.slug === id || i.service_engine === id);
    if (!db || !db.is_available) return;
    this.selectedDatabaseId.set(db.id);
    this.recalculateFromVisualSelection();
  }

  onNameInput(val: string): void {
    this.isNameManuallyEdited.set(val.trim().length > 0);
    this.name.set(val);
  }

  selectRamPreset(mb: number): void {
    this.isRamManuallyEdited.set(true);
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
      this.isRamManuallyEdited.set(false);
    }
  }

  readonly editingComponent = signal<TemplateComponent | null>(null);

  openNewComponentModal(layer: ComponentLayer, subCategory: 'backend' | 'web' = 'backend'): void {
    this.editingComponent.set(null);
    this.modalDefaultLayer.set(layer);
    this.modalDefaultSubCategory.set(subCategory);
    this.showComponentModal.set(true);
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
      dockerfile_content: this.needsLocalBuild() ? this.dockerfileContent() : undefined
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
      stack_config: stackConfig
    };
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
