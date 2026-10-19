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

export type ComponentMemoryProfile = 'library' | 'devserver' | 'heavy' | 'light' | 'standard';

@Component({
  selector: 'component-form-modal',
  imports: [
    FormsModule,
    ModalShellComponent,
    FormFieldComponent,
    TechLogoComponent,
    LucideBoxes,
    LucideAlertTriangle
  ],
  templateUrl: './component-form-modal.component.html',
  styleUrls: ['./component-form-modal.component.scss']
})
export class ComponentFormModalComponent implements OnInit {
  private readonly templatesService = inject(AdminTemplatesService);

  @Input() defaultLayer: ComponentLayer = 'addon';

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
  readonly packageNamesInput = signal<string>('');
  readonly customBuildSnippet = signal<string>('');
  readonly toolsProvidedInput = signal<string>('');
  readonly serviceEngine = signal<string>('postgres');
  readonly memoryProfile = signal<ComponentMemoryProfile>('library');
  readonly ramDeltaMB = signal<number>(0);

  readonly isSubmitting = signal<boolean>(false);
  readonly errorMessage = signal<string | null>(null);

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
      this.setMemoryProfile('standard');
    } else if (newLayer === 'addon') {
      this.subCategory.set('framework');
      this.compatibleRuntime.set('');
      this.installMode.set('pip');
      this.setMemoryProfile('library');
    } else {
      this.subCategory.set('relational');
      this.installMode.set('none');
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

    const pkgs = this.packageNamesInput()
      .split(/[\s,]+/)
      .map(s => s.trim())
      .filter(Boolean);

    if (currentLayer === 'addon' && ['pip', 'npm', 'apt'].includes(this.installMode()) && pkgs.length === 0) {
      this.errorMessage.set('Indique al menos un nombre de paquete para instalar.');
      return;
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
      install_mode: currentLayer === 'addon' ? this.installMode() : 'none',
      package_names: pkgs,
      build_layer_snippet: this.installMode() === 'custom' ? this.customBuildSnippet().trim() : '',
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
