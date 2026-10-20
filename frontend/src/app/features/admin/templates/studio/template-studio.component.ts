import { Component, OnInit, OnDestroy, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterModule } from '@angular/router';
import { Subscription } from 'rxjs';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { AdminTemplatesService } from '../../services/admin-templates.service';
import {
  AdminTemplateItem,
  ComponentLayer,
  CreateOfficialTemplateDTO,
  RuntimeCapabilities,
  ServiceRequirement,
  StackConfig,
  TargetEnvironment,
  TemplateCategory,
  TemplateComponent
} from '@features/admin/models/admin-templates.models';
import { EnvTestJob, EnvTestJobService } from '../../services/env-test-job.service';
import { TechLogoComponent } from '../components/tech-logo/tech-logo.component';
import { ComponentFormModalComponent } from '../components/model-library/components/component-form-modal/component-form-modal.component';
import { CodeEditorComponent } from '@shared/components/code-editor/code-editor.component';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import {
  LucideArrowLeft,
  LucideCheckCircle2,
  LucideAlertCircle,
  LucideCode,
  LucideLayers,
  LucidePlay,
  LucideSend,
  LucideRefreshCw,
  LucideDownload,
  LucideUpload,
  LucideRotateCcw,
  LucideAlertTriangle,
  LucideCopy,
  LucideXCircle
} from '@lucide/angular';
import {
  analyzeDockerfile,
  computeVerificationFingerprint,
  fnv1aHex,
  DockerfileDiagnostic
} from './template-studio.utils';

export interface CapacityOption {
  mb: number;
  title: string;
  desc: string;
  isRecommended: boolean;
}

export interface StudioDatabaseOption extends TemplateComponent {
  is_available: boolean;
  version?: string;
}

@Component({
  selector: 'template-studio',
  imports: [
    CommonModule,
    FormsModule,
    RouterModule,
    MachineDataDirective,
    TechLogoComponent,
    ComponentFormModalComponent,
    CodeEditorComponent,
    ComboboxComponent,
    LucideArrowLeft,
    LucideCheckCircle2,
    LucideAlertCircle,
    LucideCode,
    LucideLayers,
    LucidePlay,
    LucideSend,
    LucideRefreshCw,
    LucideDownload,
    LucideUpload,
    LucideRotateCcw,
    LucideAlertTriangle,
    LucideCopy,
    LucideXCircle
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

  // Modo de edición / duplicación
  readonly editingTemplateId = signal<string | null>(null);
  readonly isEditMode = computed(() => !!this.editingTemplateId());

  // Estado principal del constructor
  readonly targetEnvironment = signal<TargetEnvironment>('IDE_PERSISTENTE');
  readonly studioMode = signal<'visual' | 'code'>('visual');
  readonly blueprintTab = signal<'human' | 'manifest'>('human');

  // Campos de identidad
  readonly name = signal<string>('');
  readonly isNameManuallyEdited = signal<boolean>(false);
  readonly description = signal<string>('');
  readonly selectedCategoryId = signal<string | null>(null);
  readonly categories = signal<TemplateCategory[]>([]);
  readonly categoryOptions = computed<ComboboxOption<string>[]>(() =>
    this.categories().map(c => ({ id: c.id, label: c.name, value: c.id }))
  );

  // Catálogo de componentes tecnológicos y selección activa
  readonly techComponents = signal<TemplateComponent[]>([]);
  readonly selectedRuntimeId = signal<string | null>(null);
  readonly selectedWebPlatformId = signal<string | null>(null);
  readonly selectedAddonIds = signal<string[]>([]);
  readonly selectedDatabaseId = signal<string>('db-none');

  // Estado sincronizado del Modo Avanzado (Dockerfile / Scripts)
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
    this.selectedRuntimeComponent() !== null || this.dockerImage().trim().length > 0
  );
  readonly showManualRamInput = signal<boolean>(false);

  readonly dockerfileDiagnostics = computed<DockerfileDiagnostic[]>(() =>
    analyzeDockerfile(this.dockerfileContent())
  );

  readonly hasBlockingDockerfileError = computed<boolean>(() =>
    this.dockerfileDiagnostics().some(d => d.severity === 'error')
  );

  // Huella determinista y estado de verificación Zero-Trust
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
    const verified = this.verifiedFingerprint();
    if (!verified) return false;
    return this.currentVerificationFingerprint() !== verified;
  });

  readonly requiresVerificationBeforePublish = computed<boolean>(() => {
    const isLocal = this.needsLocalBuild() || this.dockerImage().startsWith('solv-local/stack:');
    if (!isLocal) return false;
    return (
      this.activeEnvTestJob()?.status !== 'success' ||
      this.isConfigModifiedAfterVerification()
    );
  });

  // Estado del host y comprobación del entorno
  readonly runtimeCapabilities = signal<RuntimeCapabilities | null>(null);
  readonly activeEnvTestJob = signal<EnvTestJob | null>(null);
  readonly isVerifying = signal<boolean>(false);
  readonly isSubmitting = signal<boolean>(false);
  readonly showTechnicalLogs = signal<boolean>(false);
  readonly toast = signal<{ message: string; type: 'success' | 'error' } | null>(null);

  // Modal para registrar una nueva tecnología en el catálogo
  readonly showComponentModal = signal<boolean>(false);
  readonly modalDefaultLayer = signal<ComponentLayer>('addon');
  readonly modalDefaultCompatibleRuntime = computed<string>(() => {
    const rt = this.selectedRuntimeComponent();
    if (!rt) return '';
    const name = (rt.icon_key || rt.slug || rt.name).toLowerCase();
    if (name.includes('python')) return 'python';
    if (name.includes('node') || name.includes('javascript') || name.includes('typescript')) return 'node';
    if (name.includes('java')) return 'java';
    if (name.includes('go')) return 'go';
    if (name.includes('c++') || name.includes('cpp')) return 'cpp';
    if (name.includes('rust')) return 'rust';
    return rt.icon_key || 'universal';
  });

  // Columnas computadas del Constructor de Entornos
  readonly backendRuntimes = computed<TemplateComponent[]>(() =>
    this.techComponents().filter(i => i.layer === 'runtime' && i.sub_category !== 'web')
  );

  readonly webPlatforms = computed<TemplateComponent[]>(() =>
    this.techComponents().filter(i => i.layer === 'runtime' && i.sub_category === 'web')
  );

  readonly compatibleAddons = computed<TemplateComponent[]>(() => {
    const rt = this.selectedRuntimeComponent();
    if (!rt) {
      return [];
    }
    const web = this.selectedWebComponent();
    const activeFamilies = new Set<string>(['universal']);

    for (const fam of rt.compatible_runtimes || []) {
      activeFamilies.add(fam.toLowerCase());
    }
    if (rt.icon_key) activeFamilies.add(rt.icon_key.toLowerCase());
    if (web && this.targetEnvironment() === 'IDE_PERSISTENTE') {
      activeFamilies.add('node');
      activeFamilies.add('web');
    }

    return this.techComponents().filter(i => {
      if (i.layer !== 'addon') return false;
      const compat = i.compatible_runtimes || [];
      if (compat.length === 0) return true;
      return compat.some(c => activeFamilies.has(c.toLowerCase()));
    });
  });

  readonly databaseOptions = computed<StudioDatabaseOption[]>(() => {
    const caps = this.runtimeCapabilities();
    const satellites = caps?.satellite_services || [];
    const catalogDbs = this.techComponents().filter(i => i.layer === 'database');

    const noneOption: StudioDatabaseOption = {
      id: 'db-none',
      slug: 'db-none',
      layer: 'database',
      sub_category: 'none',
      name: 'Sin Base de Datos',
      description: 'Solo archivos de código en la carpeta del alumno',
      icon_key: 'docker',
      base_image: '',
      compatible_runtimes: ['universal'],
      install_mode: 'none',
      package_names: [],
      build_layer_snippet: '',
      setup_snippet: '',
      judge_entrypoint: '',
      tools_provided: [],
      service_engine: 'none',
      ram_delta_mb: 0,
      is_system: true,
      is_active: true,
      sort_order: 300,
      is_available: true
    };

    if (satellites.length > 0) {
      const availableSatellites = satellites.filter(sat => sat.is_available);
      const mappedSatellites = availableSatellites.map((sat, idx): StudioDatabaseOption => {
        const engine = (sat.engine || '').toLowerCase();
        const catMatch = catalogDbs.find(c => (c.service_engine || '').toLowerCase() === engine);
        const verSuffix = sat.version ? ` v${sat.version}` : '';
        return {
          id: catMatch?.id || `db-${engine}`,
          slug: catMatch?.slug || `${engine}-service`,
          layer: 'database',
          sub_category: sat.category || 'database',
          name: `${sat.label}${verSuffix}`,
          description: sat.description || catMatch?.description || '',
          icon_key: catMatch?.icon_key || engine,
          base_image: '',
          compatible_runtimes: ['universal'],
          install_mode: 'none',
          package_names: [],
          build_layer_snippet: '',
          setup_snippet: '',
          judge_entrypoint: '',
          tools_provided: catMatch?.tools_provided || [],
          service_engine: engine,
          ram_delta_mb: sat.base_ram_mb ?? catMatch?.ram_delta_mb ?? 0,
          is_system: true,
          is_active: true,
          sort_order: 310 + idx * 10,
          is_available: true,
          version: sat.version
        };
      });
      return [noneOption, ...mappedSatellites];
    }

    const fallbackList = catalogDbs
      .filter(item => {
        const engine = (item.service_engine || '').toLowerCase();
        return engine === 'none' || engine === 'sqlite' || engine === 'postgres';
      })
      .map((item): StudioDatabaseOption => ({
        ...item,
        is_available: true
      }));

    if (fallbackList.some(d => d.id === 'db-none' || d.service_engine === 'none')) {
      return fallbackList;
    }
    return [noneOption, ...fallbackList];
  });

  readonly selectedRuntimeComponent = computed<TemplateComponent | null>(() => {
    const id = this.selectedRuntimeId();
    if (!id) return null;
    return this.techComponents().find(i => i.id === id || i.slug === id) || null;
  });

  readonly selectedWebComponent = computed<TemplateComponent | null>(() => {
    if (this.targetEnvironment() === 'JUEZ_EFIMERO') return null;
    const id = this.selectedWebPlatformId();
    if (!id) return null;
    return this.techComponents().find(i => i.id === id || i.slug === id) || null;
  });

  readonly selectedAddonComponents = computed<TemplateComponent[]>(() => {
    const ids = new Set(this.selectedAddonIds());
    const validAddons = this.compatibleAddons();
    return validAddons.filter(a => ids.has(a.id) || ids.has(a.slug));
  });

  readonly selectedDatabaseOption = computed<StudioDatabaseOption | null>(() => {
    if (this.targetEnvironment() === 'JUEZ_EFIMERO') return null;
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
    this.toolsDeclared()
      .split(',')
      .map(t => t.trim())
      .filter(Boolean)
  );


  private isCppRuntime(rt: TemplateComponent | null): boolean {
    if (!rt) return false;
    if ((rt.icon_key || '').toLowerCase() === 'cpp') return true;
    if ((rt.compatible_runtimes || []).some(c => c.toLowerCase() === 'cpp')) return true;
    return rt.ram_delta_mb > 0 && rt.ram_delta_mb <= 384;
  }

  private isJavaRuntime(rt: TemplateComponent | null): boolean {
    if (!rt) return false;
    if ((rt.icon_key || '').toLowerCase() === 'java') return true;
    if ((rt.compatible_runtimes || []).some(c => c.toLowerCase() === 'java')) return true;
    return rt.ram_delta_mb >= 1024;
  }

  private isResidentAddon(addon: TemplateComponent): boolean {
    const sub = (addon.sub_category || '').toLowerCase();
    const slug = (addon.slug || '').toLowerCase();
    if (
      sub === 'devserver' ||
      sub === 'heavy' ||
      sub === 'data' ||
      slug.includes('datascience') ||
      slug.includes('data-science')
    ) {
      return true;
    }
    return (addon.ram_delta_mb || 0) >= 256;
  }

  readonly recommendedRamMB = computed<number>(() => {
    const isJudge = this.targetEnvironment() === 'JUEZ_EFIMERO';
    const rt = this.selectedRuntimeComponent();
    const web = this.selectedWebComponent();
    const addons = this.selectedAddonComponents();
    const db = this.selectedDatabaseOption();
    const caps = this.runtimeCapabilities();

    if (isJudge) {
      if (!rt) return 256;
      const runtimeBaseMB = caps?.runtime_base_mb ?? 32;
      let processFootprintMB = 224;
      if (this.isJavaRuntime(rt)) {
        processFootprintMB = 480;
      } else if (this.isCppRuntime(rt)) {
        processFootprintMB = 96;
      }
      for (const a of addons) {
        if (!this.isResidentAddon(a)) continue;
        if ((a.sub_category || '').toLowerCase() === 'heavy' || a.ram_delta_mb >= 384) {
          processFootprintMB = Math.max(processFootprintMB, 480);
        } else {
          processFootprintMB = Math.max(processFootprintMB, 224);
        }
      }
      const totalJudge = runtimeBaseMB + processFootprintMB;
      return totalJudge <= 128 ? 128 : totalJudge <= 256 ? 256 : 512;
    }

    const editorBaseMB = caps?.editor_base_mb ?? 210;
    const satelliteBaseMB = db?.ram_delta_mb || 0;
    if (!rt) {
      return satelliteBaseMB > 0 ? 768 : 512;
    }

    let activeProcessFootprintMB = 256;
    if (this.isCppRuntime(rt)) {
      activeProcessFootprintMB = 128;
    }
    if (this.isJavaRuntime(rt)) {
      activeProcessFootprintMB = web || addons.length > 0 || satelliteBaseMB > 0 ? 1280 : 640;
    }
    if (web) {
      activeProcessFootprintMB = Math.max(activeProcessFootprintMB, 640);
    }

    for (const a of addons) {
      if (!this.isResidentAddon(a)) continue;
      if ((a.sub_category || '').toLowerCase() === 'heavy' || a.ram_delta_mb >= 384) {
        activeProcessFootprintMB = Math.max(
          activeProcessFootprintMB,
          web || this.isJavaRuntime(rt) ? 1280 : 512
        );
      } else {
        activeProcessFootprintMB = Math.max(activeProcessFootprintMB, 384);
      }
    }

    const rawTotal = editorBaseMB + satelliteBaseMB + activeProcessFootprintMB;
    if (rawTotal <= 384) return 384;
    if (rawTotal <= 512) return 512;
    if (rawTotal <= 768) return 768;
    if (rawTotal <= 1024) return 1024;
    if (rawTotal <= 2048) return 2048;
    return 4096;
  });

  readonly minimumFloorMB = computed<number>(() => {
    const isJudge = this.targetEnvironment() === 'JUEZ_EFIMERO';
    const rt = this.selectedRuntimeComponent();
    if (isJudge) {
      return this.isJavaRuntime(rt) ? 128 : 64;
    }
    const editorBase = this.runtimeCapabilities()?.editor_base_mb ?? 210;
    const satelliteMB = this.selectedDatabaseOption()?.ram_delta_mb || 0;
    const web = this.selectedWebComponent();
    const addons = this.selectedAddonComponents();

    let minRuntimeProcessMB = 0;
    if (rt) {
      minRuntimeProcessMB = this.isJavaRuntime(rt) ? 500 : this.isCppRuntime(rt) ? 128 : 174;
    }
    if (web || addons.some(a => this.isResidentAddon(a))) {
      minRuntimeProcessMB = Math.max(minRuntimeProcessMB, 300);
    }
    return editorBase + satelliteMB + minRuntimeProcessMB;
  });


  readonly maxAllowedRamMB = computed<number>(() => {
    const caps = this.runtimeCapabilities();
    if (caps?.max_allowed_ram_mb && caps.max_allowed_ram_mb > 0) {
      return caps.max_allowed_ram_mb;
    }
    const total = caps?.host_memory?.total_ram_mb ?? 8192;
    return Math.max(512, Math.floor(total * 0.75));
  });

  readonly hostFreeRamMB = computed<number>(() => {
    return this.runtimeCapabilities()?.host_memory?.available_ram_mb ?? 4096;
  });

  readonly isRamBelowFloor = computed<boolean>(() => this.baseRamMB() < this.minimumFloorMB());

  readonly isRamExceedingHost = computed<boolean>(() => this.baseRamMB() > this.maxAllowedRamMB());

  readonly ramExcessMB = computed<number>(() => Math.max(0, this.baseRamMB() - this.maxAllowedRamMB()));

  readonly isRamExceedingCurrentFree = computed<boolean>(() => {
    if (this.isRamExceedingHost()) return false;
    return this.baseRamMB() > this.hostFreeRamMB();
  });

  readonly intensiveRamMB = computed<number>(() => {
    const isJudge = this.targetEnvironment() === 'JUEZ_EFIMERO';
    const rec = this.recommendedRamMB();
    const maxAllowed = this.maxAllowedRamMB();
    const tiers = isJudge ? [128, 256, 512, 1024] : [384, 512, 768, 1024, 1536, 2048];
    const nextTier = tiers.find(t => t > rec) ?? rec;
    return Math.min(nextTier, maxAllowed);
  });

  readonly capacityOptions = computed<CapacityOption[]>(() => {
    if (!this.hasSelectedStack()) {
      return [];
    }
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
    if (!rt) return 'Ninguna';
    const parts: string[] = [rt.name];
    const web = this.selectedWebComponent();
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
    return {
      isJudge,
      totalMB,
      editorBaseMB,
      stackBaseMB,
      usableMB
    };
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

  readonly liveManifestJson = computed<string>(() => {
    const payload = this.buildCreatePayload();
    return JSON.stringify(payload, null, 2);
  });

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
      next: (items) => {
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
      next: (cats) => {
        this.categories.set(cats || []);
      },
      error: () => {}
    });

    this.templatesService.getRuntimeCapabilities().subscribe({
      next: (caps) => {
        if (caps) this.runtimeCapabilities.set(caps);
      },
      error: () => {}
    });
  }

  private loadExistingTemplate(id: string, isDuplicate: boolean): void {
    this.templatesService.getTemplateById(id).subscribe({
      next: (tpl) => {
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

    if (cfg && cfg.runtime_id) {
      this.selectedRuntimeId.set(cfg.runtime_id);
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
      if (tpl.tools_declared && tpl.tools_declared.length > 0) {
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
    if (env === 'JUEZ_EFIMERO') {
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
    this.selectedRuntimeId.set(id);
    // Conservar únicamente los complementos que sigan siendo compatibles
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
    const blob = new Blob([content], { type: mimeType });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  }

  importTextFile(accept: string, callback: (text: string) => void): void {
    const input = document.createElement('input');
    input.type = 'file';
    input.accept = accept;
    input.onchange = (e: Event) => {
      const file = (e.target as HTMLInputElement).files?.[0];
      if (!file) return;
      const reader = new FileReader();
      reader.onload = (event) => {
        const text = event.target?.result;
        if (typeof text === 'string') {
          callback(text);
        }
      };
      reader.readAsText(file);
    };
    input.click();
  }

  downloadDockerfile(): void {
    const content = this.dockerfileContent();
    if (!content.trim()) return;
    this.downloadTextFile('Dockerfile', content);
  }

  importDockerfile(): void {
    this.importTextFile('.dockerfile,Dockerfile,text/plain,*', (content) => {
      this.onDockerfileManualChange(content);
    });
  }

  downloadSetupScript(): void {
    const content = this.setupScript();
    if (!content.trim()) return;
    this.downloadTextFile('setup.sh', content);
  }

  importSetupScript(): void {
    this.importTextFile('.sh,text/x-shellscript,text/plain,*', (content) => {
      this.onSetupScriptManualChange(content);
    });
  }

  downloadManifest(): void {
    this.downloadTextFile('template-manifest.json', this.liveManifestJson(), 'application/json');
  }

  selectWebPlatform(id: string | null): void {
    const next = this.selectedWebPlatformId() === id ? null : id;
    this.selectedWebPlatformId.set(next);
    this.recalculateFromVisualSelection();
  }

  toggleAddon(id: string): void {
    this.selectedAddonIds.update(curr =>
      curr.includes(id) ? curr.filter(x => x !== id) : [...curr, id]
    );
    this.recalculateFromVisualSelection();
  }

  isAddonSelected(id: string): boolean {
    return this.selectedAddonIds().includes(id);
  }

  selectDatabase(id: string): void {
    const db = this.databaseOptions().find(
      i => i.id === id || i.slug === id || i.service_engine === id
    );
    if (!db || !db.is_available) {
      return;
    }
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
    if (!rt) {
      if (!this.isNameManuallyEdited()) {
        this.name.set('');
      }
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

    const isJudge = this.targetEnvironment() === 'JUEZ_EFIMERO';
    const web = this.selectedWebComponent();
    const addons = this.selectedAddonComponents();
    const db = this.selectedDatabaseOption();

    // 1. Nombre sugerido automáticamente
    if (!this.isNameManuallyEdited()) {
      const parts = [rt.name];
      if (web) parts.push(web.name);
      for (const a of addons) {
        parts.push(a.name.split(' ')[0]);
      }
      if (db) parts.push(db.name.split(' ')[0]);
      const prefix = isJudge ? 'Evaluador: ' : 'Laboratorio: ';
      this.name.set(prefix + parts.join(' + '));
    }

    // 2. Capas de construcción y resolución determinista de imagen
    const baseImage = rt.base_image || '';
    const buildLayers: string[] = [];
    if (rt.build_layer_snippet) {
      buildLayers.push(rt.build_layer_snippet);
    }
    if (web && !isJudge) {
      if (!baseImage.toLowerCase().includes('node')) {
        buildLayers.push('RUN (command -v apk >/dev/null 2>&1 && apk add --no-cache nodejs npm) || (apt-get update && apt-get install -y --no-install-recommends nodejs npm && rm -rf /var/lib/apt/lists/*)');
      }
      if (web.build_layer_snippet) {
        buildLayers.push(web.build_layer_snippet);
      }
    }
    for (const addon of addons) {
      if (addon.build_layer_snippet) {
        buildLayers.push(addon.build_layer_snippet);
      }
    }

    const needsBuild = buildLayers.length > 0;
    this.needsLocalBuild.set(needsBuild);
    if (!this.hasCustomDockerfile()) {
      if (needsBuild) {
        const canonical = `FROM ${baseImage}\n${buildLayers.join('\n')}`;
        const hash = fnv1aHex(canonical);
        this.dockerImage.set(`solv-local/stack:${hash}`);
        this.dockerfileContent.set(`# Generado automáticamente por el Constructor de Entornos SOLV\nFROM ${baseImage}\n${buildLayers.join('\n')}`);
      } else {
        this.dockerImage.set(baseImage);
        this.dockerfileContent.set(baseImage ? `FROM ${baseImage}` : '');
      }
    }

    // 3. Unión de programas verificables
    const toolsSet = new Set<string>(rt.tools_provided || []);
    if (!isJudge) toolsSet.add('git');
    if (web && !isJudge) {
      for (const t of web.tools_provided || []) toolsSet.add(t);
    }
    for (const a of addons) {
      for (const t of a.tools_provided || []) toolsSet.add(t);
    }
    if (db && !isJudge) {
      for (const t of db.tools_provided || []) toolsSet.add(t);
    }
    this.toolsDeclared.set(Array.from(toolsSet).join(', '));

    // 4. Script de inicialización opcional definido por los componentes tecnológicos
    if (!isJudge) {
      const lines: string[] = [];
      if (rt.setup_snippet) lines.push(rt.setup_snippet);
      if (web?.setup_snippet) lines.push(web.setup_snippet);
      for (const a of addons) {
        if (a.setup_snippet) lines.push(a.setup_snippet);
      }
      this.setupScript.set(lines.join('\n'));
    } else {
      this.setupScript.set('');
      this.entrypoint.set(rt.judge_entrypoint || '');
    }

    // 5. Memoria recomendada basada en la selección tecnológica
    if (!this.isRamManuallyEdited() || this.baseRamMB() < this.minimumFloorMB()) {
      this.baseRamMB.set(this.recommendedRamMB());
      this.isRamManuallyEdited.set(false);
    }
  }

  openNewComponentModal(layer: ComponentLayer): void {
    this.modalDefaultLayer.set(layer);
    this.showComponentModal.set(true);
  }

  handleNewComponentSaved(created: TemplateComponent): void {
    this.showComponentModal.set(false);
    this.techComponents.update(list => [...list, created]);
    if (created.layer === 'runtime') {
      if (created.sub_category === 'web') {
        this.selectWebPlatform(created.id);
      } else {
        this.selectRuntime(created.id);
      }
    } else if (created.layer === 'addon') {
      this.toggleAddon(created.id);
    } else {
      this.selectDatabase(created.id);
    }
    this.showToast(`Tecnología "${created.name}" agregada y seleccionada.`);
  }

  runVerificationTest(): void {
    const hasPrimaryRuntime =
      this.studioMode() === 'visual'
        ? !!this.selectedRuntimeComponent()
        : this.dockerImage().trim().length > 0;
    if (!hasPrimaryRuntime || !this.dockerImage().trim()) {
      this.showToast('Seleccione un lenguaje principal o indique una imagen base antes de comprobar el entorno.', 'error');
      return;
    }

    if (this.hasBlockingDockerfileError()) {
      this.showToast('Corrija los errores del Dockerfile antes de comprobar el entorno.', 'error');
      return;
    }

    if (this.isRamBelowFloor() || this.isRamExceedingHost()) {
      this.showToast('Ajuste la memoria RAM dentro de los límites permitidos antes de comprobar el entorno.', 'error');
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
      next: (job) => {
        this.activeEnvTestJob.set(job);
        this.pollSub = this.envTestService.pollJob(job.id).subscribe({
          next: (updated) => {
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
      error: (err) => {
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
      next: () => {
        this.showToast('Comprobación de entorno cancelada.');
      },
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
    const content = this.liveManifestJson();
    navigator.clipboard.writeText(content).then(() => {
      this.showToast('Manifiesto JSON copiado al portapapeles.');
    });
  }

  publishTemplate(): void {
    const hasPrimaryRuntime =
      this.studioMode() === 'visual'
        ? !!this.selectedRuntimeComponent()
        : this.dockerImage().trim().length > 0;
    if (!hasPrimaryRuntime || !this.dockerImage().trim()) {
      this.showToast('Seleccione un lenguaje principal o indique una imagen base antes de publicar la plantilla.', 'error');
      return;
    }

    if (this.hasBlockingDockerfileError()) {
      this.showToast('Corrija los errores del Dockerfile antes de publicar la plantilla.', 'error');
      return;
    }

    if (this.requiresVerificationBeforePublish()) {
      this.showToast('Esta combinación construye una imagen local a medida. Ejecute primero «Comprobar funcionamiento» para construirla y habilitar su publicación.', 'error');
      return;
    }

    if (this.isConfigModifiedAfterVerification()) {
      this.showToast('La configuración del entorno cambió después de la verificación. Compruebe el funcionamiento nuevamente antes de publicar.', 'error');
      return;
    }

    if (this.name().trim().length < 3) {
      this.showToast('El nombre del entorno debe tener al menos 3 caracteres.', 'error');
      return;
    }

    if (this.isRamBelowFloor() || this.isRamExceedingHost()) {
      this.showToast('La memoria RAM configurada viola los límites de seguridad del contenedor o del servidor.', 'error');
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
        this.router.navigate(['/admin/plantillas']);
      },
      error: (err) => {
        this.isSubmitting.set(false);
        const msg = err.error?.error?.message || 'Error al guardar la plantilla en el catálogo.';
        this.showToast(msg, 'error');
      }
    });
  }

  buildCreatePayload(): CreateOfficialTemplateDTO {
    const db = this.selectedDatabaseOption();
    const services: ServiceRequirement[] = [];
    if (db && db.service_engine && db.service_engine !== 'none') {
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
