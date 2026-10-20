import { Component, EventEmitter, Input, OnInit, Output, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { LucideBoxes, LucideAlertTriangle } from '@lucide/angular';
import {
  AdminTemplatesService,
  ComponentLayer,
  CreateTemplateComponentDTO,
  InstallMode,
  TemplateComponent
} from '../../../../../services/admin-templates.service';
import { CURATED_TECH_ICONS, TechLogoComponent } from '../../../tech-logo/tech-logo.component';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';

export type ComponentMemoryProfile = 'library' | 'devserver' | 'heavy' | 'light' | 'standard';

@Component({
  selector: 'component-form-modal',
  imports: [
    FormsModule,
    ModalShellComponent,
    FormFieldComponent,
    TechLogoComponent,
    ComboboxComponent,
    LucideBoxes,
    LucideAlertTriangle
  ],
  templateUrl: './component-form-modal.component.html',
  styleUrls: ['./component-form-modal.component.scss']
})
export class ComponentFormModalComponent implements OnInit {
  private readonly templatesService = inject(AdminTemplatesService);

  @Input() defaultLayer: ComponentLayer = 'addon';
  @Input() defaultCompatibleRuntime: string = '';
  @Input() availableRuntimes: TemplateComponent[] = [];

  @Output() closed = new EventEmitter<void>();
  @Output() saved = new EventEmitter<TemplateComponent>();

  readonly curatedIcons = CURATED_TECH_ICONS;

  readonly layer = signal<ComponentLayer>('addon');
  readonly subCategory = signal<string>('framework');
  readonly name = signal<string>('');
  readonly description = signal<string>('');
  readonly iconKey = signal<string>('docker');
  readonly isIconManuallySelected = signal<boolean>(false);
  readonly customIconUrl = signal<string>('');
  readonly baseImage = signal<string>('');
  readonly compatibleRuntime = signal<string>('');
  readonly installMode = signal<InstallMode>('pip');
  readonly installPreset = signal<string>('pip');
  readonly packageNamesInput = signal<string>('');
  readonly customBuildSnippet = signal<string>('');
  readonly toolsProvidedInput = signal<string>('');
  readonly serviceEngine = signal<string>('postgres');
  readonly memoryProfile = signal<ComponentMemoryProfile>('library');
  readonly ramDeltaMB = signal<number>(0);

  readonly isSubmitting = signal<boolean>(false);
  readonly errorMessage = signal<string | null>(null);

  readonly layerOptions: ComboboxOption<ComponentLayer>[] = [
    { id: 'runtime', label: '1. Tecnología Principal o Interfaz Web', value: 'runtime' },
    { id: 'addon', label: '2. Complemento / Framework / Librería', value: 'addon' }
  ];

  readonly subCategoryOptions: ComboboxOption<string>[] = [
    { id: 'backend', label: 'Lenguaje Principal (Backend / General)', value: 'backend' },
    { id: 'web', label: 'Plataforma Web / Frontend', value: 'web' }
  ];

  readonly installPresetOptions: ComboboxOption<string>[] = [
    { id: 'pip', label: 'Automático: Paquete de Python (pip)', value: 'pip' },
    { id: 'uv', label: 'Automático: Paquete de Python ultrarrápido (uv)', value: 'uv' },
    { id: 'npm', label: 'Automático: Paquete de Node.js (npm)', value: 'npm' },
    { id: 'apt', label: 'Automático: Programa del Sistema (apt / apk)', value: 'apt' },
    { id: 'cargo', label: 'Automático: Paquete de Rust (cargo)', value: 'cargo' },
    { id: 'go', label: 'Automático: Paquete de Go (go install)', value: 'go' },
    { id: 'composer', label: 'Automático: Paquete de PHP (composer)', value: 'composer' },
    { id: 'custom', label: 'Avanzado: Instrucción personalizada (RUN)', value: 'custom' }
  ];

  readonly databaseEngineOptions: ComboboxOption<string>[] = [
    { id: 'postgres', label: 'PostgreSQL', value: 'postgres' },
    { id: 'mysql', label: 'MySQL / MariaDB', value: 'mysql' },
    { id: 'sqlite', label: 'SQLite (Archivo local)', value: 'sqlite' },
    { id: 'mongodb', label: 'MongoDB', value: 'mongodb' },
    { id: 'redis', label: 'Redis (Caché)', value: 'redis' }
  ];

  readonly addonMemoryProfileOptions: ComboboxOption<ComponentMemoryProfile>[] = [
    { id: 'library', label: 'Librería o paquete en disco (0 MB residentes extra — solo ocupa disco)', value: 'library' },
    { id: 'devserver', label: 'Servidor de desarrollo activo en segundo plano (ej. Vite, Webpack, CLI en vivo)', value: 'devserver' },
    { id: 'heavy', label: 'Motor pesado en memoria (ej. Ciencia de Datos / Machine Learning / JVM)', value: 'heavy' }
  ];

  readonly runtimeMemoryProfileOptions: ComboboxOption<ComponentMemoryProfile>[] = [
    { id: 'light', label: 'Ligero — Compilador nativo o consola (ej. C, C++, Bash)', value: 'light' },
    { id: 'standard', label: 'Estándar — Lenguaje interpretado o compilado moderno (ej. Python, Go, PHP, Node)', value: 'standard' },
    { id: 'heavy', label: 'Intensivo — Máquina Virtual o SDK pesado (ej. Java JVM, .NET)', value: 'heavy' }
  ];

  readonly compatibleRuntimeOptions = computed<ComboboxOption<string>[]>(() => {
    const baseList: { id: string; label: string }[] = [
      { id: 'python', label: 'Python' },
      { id: 'node', label: 'Node.js / Web (Angular, React, Vue)' },
      { id: 'java', label: 'Java' },
      { id: 'go', label: 'Go' },
      { id: 'cpp', label: 'C / C++' },
      { id: 'rust', label: 'Rust' }
    ];
    const seen = new Set(baseList.map(b => b.id));
    const dynamic: { id: string; label: string }[] = [];
    for (const r of this.availableRuntimes) {
      const key = (r.icon_key || r.slug || r.name).toLowerCase();
      if (!seen.has(key)) {
        seen.add(key);
        dynamic.push({ id: key, label: r.name });
      }
    }
    const all = [...baseList, ...dynamic, { id: 'universal', label: 'Todas las tecnologías (Universal)' }];
    return all.map(item => ({
      id: item.id,
      label: item.label,
      value: item.id
    }));
  });

  readonly effectiveIconKey = computed<string>(() => {
    const custom = this.customIconUrl().trim();
    if (custom) {
      return custom;
    }
    return this.iconKey().trim() || 'docker';
  });

  ngOnInit(): void {
    const initial = this.defaultLayer === 'runtime' ? 'runtime' : 'addon';
    this.setLayerDefaults(initial);
    if (this.defaultCompatibleRuntime) {
      this.compatibleRuntime.set(this.defaultCompatibleRuntime);
    }
  }

  onNameChange(val: string): void {
    this.name.set(val);
    if (!this.isIconManuallySelected() && !this.customIconUrl().trim()) {
      const normalized = val.trim().toLowerCase();
      const matched = this.curatedIcons.find(ic => {
        if (ic.key === 'docker') return false;
        const labelFirstWord = ic.label.split(/[\s/]+/)[0].toLowerCase();
        return (
          normalized.includes(ic.key.toLowerCase()) ||
          (labelFirstWord.length >= 2 && normalized.includes(labelFirstWord))
        );
      });
      this.iconKey.set(matched ? matched.key : 'docker');
    }
  }

  selectCuratedIcon(key: string): void {
    this.isIconManuallySelected.set(true);
    this.iconKey.set(key);
    this.customIconUrl.set('');
  }

  setMemoryProfile(profile: ComponentMemoryProfile): void {
    this.memoryProfile.set(profile);
    if (this.layer() === 'addon') {
      if (profile === 'devserver') {
        this.ramDeltaMB.set(256);
        this.subCategory.set('devserver');
      } else if (profile === 'heavy') {
        this.ramDeltaMB.set(384);
        this.subCategory.set('heavy');
      } else {
        this.ramDeltaMB.set(0);
        this.subCategory.set('framework');
      }
    } else if (this.layer() === 'runtime') {
      if (profile === 'light') {
        this.ramDeltaMB.set(384);
      } else if (profile === 'heavy') {
        this.ramDeltaMB.set(1024);
      } else {
        this.ramDeltaMB.set(512);
      }
    }
  }

  setLayerDefaults(newLayer: ComponentLayer): void {
    this.layer.set(newLayer);
    if (newLayer === 'runtime') {
      this.subCategory.set('backend');
      this.installMode.set('none');
      this.installPreset.set('none');
      this.setMemoryProfile('standard');
    } else if (newLayer === 'addon') {
      this.subCategory.set('framework');
      this.compatibleRuntime.set(this.defaultCompatibleRuntime || '');
      this.installMode.set('pip');
      this.installPreset.set('pip');
      this.setMemoryProfile('library');
    } else {
      this.subCategory.set('relational');
      this.installMode.set('none');
      this.installPreset.set('none');
      this.ramDeltaMB.set(50);
    }
  }

  onClose(): void {
    this.closed.emit();
  }

  onSave(): void {
    const trimmedName = this.name().trim();
    if (trimmedName.length < 2) {
      this.errorMessage.set('El nombre de la tecnología debe tener al menos 2 caracteres.');
      return;
    }

    const currentLayer = this.layer();
    if (currentLayer === 'runtime' && !this.baseImage().trim()) {
      this.errorMessage.set('Debe indicar la imagen base para el lenguaje principal.');
      return;
    }

    if (currentLayer === 'addon' && !this.compatibleRuntime().trim()) {
      this.errorMessage.set('Seleccione el lenguaje principal con el que es compatible esta tecnología.');
      return;
    }

    const preset = this.installPreset();
    const pkgs = this.packageNamesInput()
      .split(/[\s,]+/)
      .map(s => s.trim())
      .filter(Boolean);

    const safePkgRegex = /^[a-zA-Z0-9@_./=<>-]+$/;
    for (const pkg of pkgs) {
      if (!safePkgRegex.test(pkg)) {
        this.errorMessage.set(`El nombre de paquete "${pkg}" contiene caracteres inválidos.`);
        return;
      }
    }

    if (currentLayer === 'addon' && preset !== 'custom' && pkgs.length === 0) {
      this.errorMessage.set('Indique al menos un nombre de paquete para instalar.');
      return;
    }

    let resolvedInstallMode: InstallMode = 'none';
    let resolvedSnippet = '';

    if (currentLayer === 'addon') {
      if (preset === 'pip') {
        resolvedInstallMode = 'pip';
      } else if (preset === 'npm') {
        resolvedInstallMode = 'npm';
      } else if (preset === 'apt') {
        resolvedInstallMode = 'apt';
      } else if (preset === 'uv') {
        resolvedInstallMode = 'custom';
        resolvedSnippet = `RUN uv pip install --system --no-cache ${pkgs.join(' ')}`;
      } else if (preset === 'cargo') {
        resolvedInstallMode = 'custom';
        resolvedSnippet = `RUN cargo install ${pkgs.join(' ')}`;
      } else if (preset === 'go') {
        resolvedInstallMode = 'custom';
        resolvedSnippet = `RUN go install ${pkgs.join(' ')}`;
      } else if (preset === 'composer') {
        resolvedInstallMode = 'custom';
        resolvedSnippet = `RUN composer global require ${pkgs.join(' ')}`;
      } else {
        resolvedInstallMode = 'custom';
        resolvedSnippet = this.customBuildSnippet().trim();
      }
    }

    const tools = this.toolsProvidedInput()
      .split(',')
      .map(s => s.trim())
      .filter(Boolean);

    const dto: CreateTemplateComponentDTO = {
      name: trimmedName,
      layer: currentLayer,
      sub_category: this.subCategory(),
      description: this.description().trim(),
      icon_key: this.effectiveIconKey(),
      base_image: currentLayer === 'runtime' ? this.baseImage().trim() : '',
      compatible_runtimes:
        currentLayer === 'database'
          ? ['universal']
          : currentLayer === 'addon'
            ? [this.compatibleRuntime().trim()]
            : this.subCategory() === 'web'
              ? ['node', 'web']
              : [this.effectiveIconKey().toLowerCase()],
      install_mode: resolvedInstallMode,
      package_names: pkgs,
      build_layer_snippet: resolvedSnippet,
      tools_provided: tools,
      service_engine: currentLayer === 'database' ? this.serviceEngine().trim() : '',
      ram_delta_mb: Number(this.ramDeltaMB()) || 0
    };

    this.isSubmitting.set(true);
    this.errorMessage.set(null);

    this.templatesService.createTechCatalogItem(dto).subscribe({
      next: (created) => {
        this.isSubmitting.set(false);
        this.saved.emit(created);
      },
      error: (err) => {
        this.isSubmitting.set(false);
        const msg = err.error?.error?.message || err.message || 'No se pudo registrar la tecnología.';
        this.errorMessage.set(msg);
      }
    });
  }
}
