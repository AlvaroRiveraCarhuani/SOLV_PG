import { Component, EventEmitter, Input, OnInit, Output, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { LucideBoxes, LucideAlertTriangle } from '@lucide/angular';
import {
  AdminTemplatesService,
  ComponentLayer,
  CreateTemplateComponentDTO,
  UpdateTemplateComponentDTO,
  InstallMode,
  TemplateComponent
} from '../../../../../services/admin-templates.service';
import { CURATED_TECH_ICONS } from '../../../tech-logo/tech-logo.component';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import {
  ADDON_MEMORY_PROFILE_OPTIONS,
  CATALOG_KIND_OPTIONS,
  ComponentCatalogKind,
  ComponentMemoryProfile,
  DATABASE_ENGINE_OPTIONS,
  EcosystemVersionOption,
  INSTALL_PRESET_OPTIONS,
  LAYER_OPTIONS,
  OFFICIAL_ADDON_PRESETS,
  OFFICIAL_ECOSYSTEM_PROFILES,
  OFFICIAL_WEB_PROFILES,
  OfficialAddonPreset,
  OfficialEcosystemProfile,
  OfficialWebProfile,
  RUNTIME_MEMORY_PROFILE_OPTIONS,
  SUB_CATEGORY_OPTIONS
} from './component-catalog.presets';
import {
  computeCompatibleRuntimesForComponent,
  computeComponentUiCopy,
  filterAddonPresetsByFamily,
  filterCuratedIconsByContext,
  filterInstallPresetOptionsByFamily,
  findCatalogMatchInList,
  inferAddonInstallPresetFromRuntime,
  resolveAutoIconKeyFromContext,
  resolveDynamicWebBaseImageFromCatalog,
  resolveEcosystemFamilyFromContext,
  slugifyComponentName,
  synthesizeInstallModeAndSnippet
} from './component-ecosystem.utils';
import { CatalogPresetGridComponent } from './catalog-preset-grid/catalog-preset-grid.component';
import { SmartIconPickerComponent } from './smart-icon-picker/smart-icon-picker.component';

export type {
  ComponentCatalogKind,
  ComponentMemoryProfile,
  EcosystemVersionOption,
  OfficialAddonPreset,
  OfficialEcosystemProfile,
  OfficialWebProfile
};

export {
  OFFICIAL_ADDON_PRESETS,
  OFFICIAL_ECOSYSTEM_PROFILES,
  OFFICIAL_WEB_PROFILES
};

@Component({
  selector: 'component-form-modal',
  imports: [
    FormsModule,
    ModalShellComponent,
    FormFieldComponent,
    ComboboxComponent,
    CatalogPresetGridComponent,
    SmartIconPickerComponent,
    LucideBoxes,
    LucideAlertTriangle
  ],
  templateUrl: './component-form-modal.component.html',
  styleUrls: ['./component-form-modal.component.scss']
})
export class ComponentFormModalComponent implements OnInit {
  private readonly templatesService = inject(AdminTemplatesService);

  @Input() defaultLayer: ComponentLayer = 'addon';
  @Input() defaultSubCategory: 'backend' | 'web' = 'backend';
  @Input() defaultCompatibleRuntime: string = '';
  @Input() availableRuntimes: TemplateComponent[] = [];
  @Input() editingComponent: TemplateComponent | null = null;
  @Input() lockContext: boolean = false;

  @Output() closed = new EventEmitter<void>();
  @Output() saved = new EventEmitter<TemplateComponent>();

  readonly curatedIcons = CURATED_TECH_ICONS;
  readonly officialEcosystemProfiles = OFFICIAL_ECOSYSTEM_PROFILES;
  readonly officialWebProfiles = OFFICIAL_WEB_PROFILES;
  readonly officialAddonPresets = OFFICIAL_ADDON_PRESETS;
  readonly catalogKindOptions = CATALOG_KIND_OPTIONS;
  readonly layerOptions = LAYER_OPTIONS;
  readonly subCategoryOptions = SUB_CATEGORY_OPTIONS;
  readonly installPresetOptions = INSTALL_PRESET_OPTIONS;
  readonly databaseEngineOptions = DATABASE_ENGINE_OPTIONS;
  readonly addonMemoryProfileOptions = ADDON_MEMORY_PROFILE_OPTIONS;
  readonly runtimeMemoryProfileOptions = RUNTIME_MEMORY_PROFILE_OPTIONS;

  readonly isContextLocked = signal<boolean>(false);
  readonly showExpertMode = signal<boolean>(false);
  readonly selectedEcosystemProfileId = signal<string>('');
  readonly selectedWebProfileId = signal<string>('');
  readonly selectedAddonPresetId = signal<string>('');
  readonly selectedVersion = signal<string>('');
  readonly isCustomPresetSelected = signal<boolean>(false);
  readonly existingCatalogMatch = signal<TemplateComponent | null>(null);

  readonly layer = signal<ComponentLayer>('addon');
  readonly subCategory = signal<string>('framework');
  readonly name = signal<string>('');
  readonly description = signal<string>('');
  readonly iconKey = signal<string>('docker');
  readonly isIconManuallySelected = signal<boolean>(false);
  readonly customIconUrl = signal<string>('');
  readonly showIconPickerGrid = signal<boolean>(false);
  readonly showCustomIconInput = signal<boolean>(false);
  readonly iconSearchQuery = signal<string>('');
  readonly baseImage = signal<string>('');
  readonly compatibleRuntime = signal<string>('');
  readonly installMode = signal<InstallMode>('pip');
  readonly installPreset = signal<string>('pip');
  readonly packageNamesInput = signal<string>('');
  readonly customBuildSnippet = signal<string>('');
  readonly toolsProvidedInput = signal<string>('');
  readonly serviceEngine = signal<string>('postgres');
  readonly judgeEntrypoint = signal<string>('');
  readonly memoryProfile = signal<ComponentMemoryProfile>('library');
  readonly ramDeltaMB = signal<number>(0);

  readonly isSubmitting = signal<boolean>(false);
  readonly errorMessage = signal<string | null>(null);

  readonly ecosystemProfileOptions: ComboboxOption<string>[] = OFFICIAL_ECOSYSTEM_PROFILES.map(p => ({
    id: p.id,
    label: `${p.name} — ${p.baseImage}`,
    value: p.id
  }));

  readonly catalogKind = computed<ComponentCatalogKind>(() => {
    if (this.layer() === 'addon') return 'addon';
    return this.subCategory() === 'web' ? 'web' : 'backend';
  });

  readonly uiCopy = computed(() =>
    computeComponentUiCopy(this.catalogKind(), !!this.editingComponent, this.installPreset())
  );

  readonly resolvedEcosystemFamily = computed<string>(() =>
    resolveEcosystemFamilyFromContext({
      layer: this.layer(),
      subCategory: this.subCategory(),
      compatibleRuntime: this.compatibleRuntime(),
      availableRuntimes: this.availableRuntimes,
      iconKey: this.iconKey(),
      name: this.name(),
      baseImage: this.baseImage()
    })
  );

  readonly filteredInstallPresetOptions = computed<ComboboxOption<string>[]>(() =>
    filterInstallPresetOptionsByFamily(
      this.resolvedEcosystemFamily(),
      this.layer() === 'runtime',
      this.showExpertMode() || this.installPreset() === 'custom',
      this.installPresetOptions
    )
  );

  readonly autoResolvedSummary = computed<string>(() => {
    if (this.layer() === 'runtime') {
      const img = this.baseImage().trim() || 'Pendiente de selección';
      const ram = this.ramDeltaMB() || 512;
      const judge = this.judgeEntrypoint().trim();
      return judge
        ? `Imagen: ${img} · Evaluador: ${judge} · Memoria base: ${ram} MB`
        : `Imagen: ${img} · Memoria base: ${ram} MB`;
    }
    const preset = this.installPreset();
    const installerLabel = this.installPresetOptions.find(o => o.id === preset)?.label || preset;
    return `${installerLabel} · Sin recargo de memoria residente (0 MB extra)`;
  });

  readonly compatibleRuntimeOptions = computed<ComboboxOption<string>[]>(() => {
    const runtimes = (this.availableRuntimes || []).filter(r => r.layer === 'runtime');
    const seen = new Set<string>();
    const options: ComboboxOption<string>[] = [];

    for (const r of runtimes) {
      const key = (r.slug || r.icon_key || r.name).toLowerCase();
      if (!seen.has(key)) {
        seen.add(key);
        options.push({ id: key, label: r.name, value: key });
      }
    }

    options.push({
      id: 'universal',
      label: 'Universal (Compatible con cualquier tecnología)',
      value: 'universal'
    });
    return options;
  });

  readonly effectiveIconKey = computed<string>(() => {
    const custom = this.customIconUrl().trim();
    return custom ? custom : (this.iconKey().trim() || 'docker');
  });

  findCatalogMatch(presetId: string, presetName: string): TemplateComponent | null {
    return findCatalogMatchInList(this.availableRuntimes, presetId, presetName);
  }

  isPresetInCatalog(presetId: string, presetName: string): boolean {
    return !!this.findCatalogMatch(presetId, presetName);
  }

  resolveDynamicWebBaseImage(): string {
    return resolveDynamicWebBaseImageFromCatalog(this.availableRuntimes, this.officialEcosystemProfiles);
  }

  readonly availableVersionsForSelectedProfile = computed<EcosystemVersionOption[]>(() => {
    const kind = this.catalogKind();
    if (kind === 'backend') {
      return this.officialEcosystemProfiles.find(p => p.id === this.selectedEcosystemProfileId())?.suggestedVersions || [];
    }
    if (kind === 'web') {
      return this.officialWebProfiles.find(p => p.id === this.selectedWebProfileId())?.suggestedVersions || [];
    }
    return [];
  });

  readonly versionComboboxOptions = computed<ComboboxOption<string>[]>(() =>
    this.availableVersionsForSelectedProfile().map(v => ({
      id: v.version,
      label: v.label,
      value: v.version
    }))
  );

  onVersionChange(rawVersion: string): void {
    const cleanVer = (rawVersion || '').trim();
    this.selectedVersion.set(cleanVer);
    if (!cleanVer) return;

    const kind = this.catalogKind();
    if (kind === 'backend') {
      const prof = this.officialEcosystemProfiles.find(p => p.id === this.selectedEcosystemProfileId());
      if (prof) {
        const resolvedImg = prof.imageTemplate.replace('{version}', cleanVer);
        const majorLabel =
          cleanVer.split('.')[0] && (prof.id === 'c-gcc13' || prof.id === 'cpp-gcc13')
            ? cleanVer.split('.')[0]
            : cleanVer;
        const resolvedName = prof.nameTemplate.replace('{version}', majorLabel);
        this.baseImage.set(resolvedImg);
        this.name.set(resolvedName);
        this.existingCatalogMatch.set(this.findCatalogMatch(slugifyComponentName(resolvedName), resolvedName));
      }
    } else if (kind === 'web') {
      const webProf = this.officialWebProfiles.find(p => p.id === this.selectedWebProfileId());
      if (webProf) {
        if (webProf.packageTemplate?.length) {
          const pkgs = webProf.packageTemplate.map(t => t.replace('{version}', cleanVer));
          this.packageNamesInput.set(pkgs.join(', '));
        }
        if (webProf.nameTemplate) {
          const isDefault = cleanVer === webProf.defaultVersion;
          const resolvedName = isDefault ? webProf.name : webProf.nameTemplate.replace('{version}', cleanVer);
          this.name.set(resolvedName);
          this.existingCatalogMatch.set(this.findCatalogMatch(slugifyComponentName(resolvedName), resolvedName));
        }
      }
    }
  }

  readonly sortedEcosystemProfiles = computed<OfficialEcosystemProfile[]>(() =>
    [...this.officialEcosystemProfiles].sort(
      (a, b) => (this.isPresetInCatalog(a.id, a.name) ? 1 : 0) - (this.isPresetInCatalog(b.id, b.name) ? 1 : 0)
    )
  );

  readonly sortedWebProfiles = computed<OfficialWebProfile[]>(() =>
    [...this.officialWebProfiles].sort(
      (a, b) => (this.isPresetInCatalog(a.id, a.name) ? 1 : 0) - (this.isPresetInCatalog(b.id, b.name) ? 1 : 0)
    )
  );

  readonly contextualAddonPresets = computed<OfficialAddonPreset[]>(() => {
    const filtered = filterAddonPresetsByFamily(this.officialAddonPresets, this.resolvedEcosystemFamily());
    return [...filtered].sort(
      (a, b) => (this.isPresetInCatalog(a.id, a.name) ? 1 : 0) - (this.isPresetInCatalog(b.id, b.name) ? 1 : 0)
    );
  });

  readonly filteredCuratedIcons = computed(() =>
    filterCuratedIconsByContext(
      this.curatedIcons,
      this.iconSearchQuery(),
      this.catalogKind(),
      this.resolvedEcosystemFamily()
    )
  );

  readonly activeIconLabel = computed<string>(() => {
    if (this.customIconUrl().trim()) return 'Icono personalizado (URL / SVG)';
    const key = this.iconKey().trim().toLowerCase();
    return this.curatedIcons.find(ic => ic.key.toLowerCase() === key)?.label || this.iconKey() || 'Docker';
  });

  readonly iconResolutionSource = computed<string>(() => {
    if (this.customIconUrl().trim()) return 'URL personalizada';
    if (this.isIconManuallySelected()) return 'Seleccionado manualmente';
    if (this.iconKey() && this.iconKey() !== 'docker') return 'Detectado automáticamente';
    return 'Icono estándar por defecto';
  });

  ngOnInit(): void {
    this.isContextLocked.set(this.lockContext && !this.editingComponent);

    if (this.editingComponent) {
      this.populateEditingState(this.editingComponent);
      return;
    }

    const initialKind: ComponentCatalogKind =
      this.defaultLayer === 'addon' ? 'addon' : (this.defaultSubCategory === 'web' ? 'web' : 'backend');
    this.onCatalogKindChange(initialKind);

    if (this.defaultLayer === 'addon') {
      if (this.defaultCompatibleRuntime?.trim()) {
        this.onCompatibleRuntimeChange(this.defaultCompatibleRuntime.trim().toLowerCase());
      } else {
        const firstRuntime = (this.availableRuntimes || []).find(r => r.layer === 'runtime' && r.is_active !== false);
        if (firstRuntime) {
          this.onCompatibleRuntimeChange((firstRuntime.slug || firstRuntime.icon_key || firstRuntime.name).toLowerCase());
        }
      }
    } else if (this.defaultCompatibleRuntime?.trim()) {
      this.compatibleRuntime.set(this.defaultCompatibleRuntime.trim().toLowerCase());
    }
  }

  private populateEditingState(comp: TemplateComponent): void {
    this.layer.set(comp.layer);
    this.subCategory.set(comp.sub_category || (comp.layer === 'runtime' ? 'backend' : 'framework'));
    this.name.set(comp.name);
    this.description.set(comp.description || '');
    this.iconKey.set(comp.icon_key || 'docker');
    if (comp.icon_key && (comp.icon_key.startsWith('/') || comp.icon_key.startsWith('http'))) {
      this.customIconUrl.set(comp.icon_key);
      this.showCustomIconInput.set(true);
    } else if (comp.icon_key && comp.icon_key !== 'docker') {
      this.isIconManuallySelected.set(true);
    } else {
      this.isIconManuallySelected.set(false);
      this.iconKey.set(this.resolveAutoIconKey(comp.name, comp.compatible_runtimes?.[0] || ''));
    }
    this.baseImage.set(comp.base_image || '');
    if (comp.compatible_runtimes?.length) {
      this.compatibleRuntime.set(comp.compatible_runtimes[0]);
    }
    this.installMode.set(comp.install_mode || 'none');
    if (comp.install_mode === 'custom') {
      this.installPreset.set('custom');
      this.customBuildSnippet.set(comp.build_layer_snippet || '');
      this.showExpertMode.set(true);
    } else {
      this.installPreset.set(comp.install_mode || 'none');
    }
    if (comp.package_names?.length) {
      this.packageNamesInput.set(comp.package_names.join(', '));
    }
    if (comp.tools_provided?.length) {
      this.toolsProvidedInput.set(comp.tools_provided.join(', '));
    }
    this.serviceEngine.set(comp.service_engine || 'postgres');
    this.judgeEntrypoint.set(comp.judge_entrypoint || '');
    this.ramDeltaMB.set(comp.ram_delta_mb || 0);

    if (comp.layer === 'addon') {
      this.memoryProfile.set(comp.ram_delta_mb >= 384 ? 'heavy' : (comp.ram_delta_mb >= 256 ? 'devserver' : 'library'));
    } else if (comp.layer === 'runtime') {
      this.memoryProfile.set(comp.ram_delta_mb >= 1024 ? 'heavy' : (comp.ram_delta_mb <= 384 ? 'light' : 'standard'));
    }
  }

  onSelectEcosystemProfile(profileId: string): void {
    this.isCustomPresetSelected.set(false);
    this.selectedEcosystemProfileId.set(profileId);
    const profile = this.officialEcosystemProfiles.find(p => p.id === profileId);
    if (!profile) return;
    this.selectedVersion.set(profile.defaultVersion);
    this.existingCatalogMatch.set(this.findCatalogMatch(profile.id, profile.name));
    this.name.set(profile.name);
    this.description.set(profile.description);
    this.iconKey.set(profile.iconKey);
    this.isIconManuallySelected.set(false);
    this.baseImage.set(profile.baseImage);
    this.judgeEntrypoint.set(profile.judgeEntrypoint);
    this.toolsProvidedInput.set(profile.toolsProvided.join(', '));
    this.setMemoryProfile(profile.memoryProfile);
    this.installPreset.set('none');
    this.installMode.set('none');
  }

  onSelectWebProfile(profileId: string): void {
    this.isCustomPresetSelected.set(false);
    this.selectedWebProfileId.set(profileId);
    const profile = this.officialWebProfiles.find(p => p.id === profileId);
    if (!profile) return;
    this.selectedVersion.set(profile.defaultVersion || '');
    this.existingCatalogMatch.set(this.findCatalogMatch(profile.id, profile.name));
    this.name.set(profile.name);
    this.description.set(profile.description);
    this.iconKey.set(profile.iconKey);
    this.isIconManuallySelected.set(false);
    this.baseImage.set(this.resolveDynamicWebBaseImage());
    this.installPreset.set(profile.installPreset);
    this.installMode.set(profile.installPreset as InstallMode);
    this.packageNamesInput.set(profile.packageNames.join(', '));
    this.toolsProvidedInput.set(profile.toolsProvided.join(', '));
    this.setMemoryProfile(profile.memoryProfile);
  }

  onSelectAddonPreset(presetId: string): void {
    this.isCustomPresetSelected.set(false);
    this.selectedAddonPresetId.set(presetId);
    const preset = this.officialAddonPresets.find(p => p.id === presetId);
    if (!preset) return;
    this.existingCatalogMatch.set(this.findCatalogMatch(preset.id, preset.name));
    this.name.set(preset.name);
    this.description.set(preset.description);
    this.iconKey.set(preset.iconKey);
    this.isIconManuallySelected.set(false);
    this.installPreset.set(preset.installPreset);
    this.packageNamesInput.set(preset.packageNames.join(', '));
    this.toolsProvidedInput.set(preset.toolsProvided.join(', '));
    this.setMemoryProfile(preset.memoryProfile);
  }

  onSelectCustomMode(): void {
    this.isCustomPresetSelected.set(true);
    this.selectedEcosystemProfileId.set('');
    this.selectedWebProfileId.set('');
    this.selectedAddonPresetId.set('');
    this.selectedVersion.set('');
    this.existingCatalogMatch.set(null);
    this.name.set('');
    this.description.set('');
    this.iconKey.set('docker');
    this.isIconManuallySelected.set(false);
    this.packageNamesInput.set('');
    this.toolsProvidedInput.set('');
    this.judgeEntrypoint.set('');
    if (this.catalogKind() === 'web') {
      this.baseImage.set(this.resolveDynamicWebBaseImage());
      this.installPreset.set('npm');
      this.installMode.set('npm');
    } else if (this.catalogKind() === 'backend') {
      this.baseImage.set('');
      this.installPreset.set('none');
      this.installMode.set('none');
      this.showExpertMode.set(true);
    } else {
      this.baseImage.set('');
    }
  }

  onCatalogKindChange(kind: ComponentCatalogKind): void {
    const dynamicWebImg = this.resolveDynamicWebBaseImage();
    if (kind === 'backend') {
      this.layer.set('runtime');
      this.subCategory.set('backend');
      this.installMode.set('none');
      this.installPreset.set('none');
      this.setMemoryProfile('standard');
      if (!this.editingComponent && this.baseImage().trim() === dynamicWebImg) {
        this.baseImage.set('');
      }
    } else if (kind === 'web') {
      this.layer.set('runtime');
      this.subCategory.set('web');
      if (!this.baseImage().trim()) {
        this.baseImage.set(dynamicWebImg);
      }
      this.installMode.set('npm');
      this.installPreset.set('npm');
      this.setMemoryProfile('standard');
    } else if (kind === 'addon') {
      this.layer.set('addon');
      this.subCategory.set('framework');
      this.setMemoryProfile('library');
      this.onCompatibleRuntimeChange(this.defaultCompatibleRuntime || this.compatibleRuntime() || '');
    }

    if (!this.isIconManuallySelected() && !this.customIconUrl().trim()) {
      this.iconKey.set(this.resolveAutoIconKey(this.name(), this.compatibleRuntime()));
    }
  }

  private resolveAutoIconKey(nameVal: string, compatRuntimeKey: string): string {
    return resolveAutoIconKeyFromContext(
      nameVal,
      compatRuntimeKey,
      this.layer(),
      this.availableRuntimes,
      this.curatedIcons
    );
  }

  onCompatibleRuntimeChange(runtimeKey: string): void {
    this.compatibleRuntime.set(runtimeKey);
    if (!this.isIconManuallySelected() && !this.customIconUrl().trim()) {
      this.iconKey.set(this.resolveAutoIconKey(this.name(), runtimeKey));
    }
    if (this.layer() === 'addon') {
      const inferredPreset = inferAddonInstallPresetFromRuntime(runtimeKey, this.availableRuntimes);
      if (inferredPreset) {
        this.installPreset.set(inferredPreset);
      }
    }
  }

  onSubCategoryChange(newSubCat: string): void {
    if (this.layer() === 'runtime') {
      this.onCatalogKindChange(newSubCat === 'web' ? 'web' : 'backend');
    } else {
      this.subCategory.set(newSubCat);
    }
  }

  onNameChange(val: string): void {
    this.name.set(val);
    if (!this.isIconManuallySelected() && !this.customIconUrl().trim()) {
      this.iconKey.set(this.resolveAutoIconKey(val, this.compatibleRuntime()));
    }
    if (!this.editingComponent && this.layer() === 'runtime' && this.subCategory() === 'backend') {
      const lower = (val || '').trim().toLowerCase();
      const matchedProfile = this.officialEcosystemProfiles.find(p => {
        if (p.id === 'cpp-gcc13' && (lower.includes('c++') || lower.includes('g++'))) return true;
        if (p.id === 'c-gcc13' && /^c(\s|\(|$)/.test(lower) && !lower.includes('c++') && !lower.includes('c#')) return true;
        return lower.startsWith(p.iconKey) && p.iconKey.length >= 2;
      });
      if (matchedProfile) {
        if (!this.baseImage().trim()) this.baseImage.set(matchedProfile.baseImage);
        if (!this.judgeEntrypoint().trim()) this.judgeEntrypoint.set(matchedProfile.judgeEntrypoint);
        if (!this.toolsProvidedInput().trim()) this.toolsProvidedInput.set(matchedProfile.toolsProvided.join(', '));
        this.setMemoryProfile(matchedProfile.memoryProfile);
      }
    }
  }

  selectCuratedIcon(key: string): void {
    this.isIconManuallySelected.set(true);
    this.iconKey.set(key);
    this.customIconUrl.set('');
    this.showCustomIconInput.set(false);
    this.showIconPickerGrid.set(false);

    if (!this.editingComponent) {
      const kind = this.catalogKind();
      if (kind === 'backend') {
        const matchedProfile = this.officialEcosystemProfiles.find(p => p.iconKey === key);
        if (matchedProfile) {
          this.onSelectEcosystemProfile(matchedProfile.id);
          this.isIconManuallySelected.set(true);
        }
      } else if (kind === 'web') {
        const matchedWeb = this.officialWebProfiles.find(p => p.iconKey === key);
        if (matchedWeb) {
          this.onSelectWebProfile(matchedWeb.id);
          this.isIconManuallySelected.set(true);
        }
      } else if (kind === 'addon') {
        const matchedAddon = this.contextualAddonPresets().find(
          p => p.iconKey === key && p.iconKey !== 'cpp' && p.iconKey !== 'python'
        );
        if (matchedAddon) {
          this.onSelectAddonPreset(matchedAddon.id);
          this.isIconManuallySelected.set(true);
        }
      }
    }
  }

  toggleIconPickerGrid(): void {
    this.showIconPickerGrid.update(v => !v);
    if (this.showIconPickerGrid()) {
      this.showCustomIconInput.set(false);
    }
  }

  toggleCustomIconInput(): void {
    this.showCustomIconInput.update(v => !v);
    if (this.showCustomIconInput()) {
      this.showIconPickerGrid.set(false);
    }
  }

  toggleExpertMode(): void {
    this.showExpertMode.update(v => !v);
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
      this.ramDeltaMB.set(profile === 'light' ? 384 : (profile === 'heavy' ? 1024 : 512));
    }
  }

  setLayerDefaults(newLayer: ComponentLayer): void {
    if (newLayer === 'runtime') {
      this.onCatalogKindChange('backend');
    } else if (newLayer === 'addon') {
      this.onCatalogKindChange('addon');
    } else {
      this.layer.set(newLayer);
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
      this.errorMessage.set('Debe seleccionar un perfil tecnológico o indicar la imagen base para el lenguaje principal.');
      return;
    }

    if (currentLayer === 'addon' && !this.compatibleRuntime().trim()) {
      this.errorMessage.set('Seleccione el lenguaje principal con el que es compatible esta tecnología.');
      return;
    }

    let preset = this.installPreset();
    const pkgs = this.packageNamesInput().split(/[\s,]+/).map(s => s.trim()).filter(Boolean);
    let tools = this.toolsProvidedInput().split(/[\s,]+/).map(s => s.trim()).filter(Boolean);

    if (currentLayer === 'runtime' && preset === 'none' && pkgs.length > 0) {
      const fam = this.resolvedEcosystemFamily();
      preset = fam === 'python' ? 'pip' : (fam === 'node' || fam === 'web' ? 'npm' : 'apt');
    }

    const safePkgRegex = /^[a-zA-Z0-9@_./=<>-]+$/;
    for (const pkg of pkgs) {
      if (!safePkgRegex.test(pkg)) {
        this.errorMessage.set(`El nombre de paquete "${pkg}" contiene caracteres inválidos.`);
        return;
      }
    }

    if (preset !== 'custom' && preset !== 'none' && pkgs.length === 0) {
      this.errorMessage.set('Indique al menos un nombre de paquete para instalar.');
      return;
    }

    if (preset === 'custom' && !this.customBuildSnippet().trim()) {
      this.errorMessage.set('Indique la instrucción RUN personalizada.');
      return;
    }

    const { installMode: resolvedInstallMode, snippet: resolvedSnippet } =
      synthesizeInstallModeAndSnippet(preset, pkgs, this.customBuildSnippet());

    if (resolvedInstallMode === 'apt' && tools.length === 0 && pkgs.length > 0) {
      tools = [...pkgs];
    }

    const generatedSlug = slugifyComponentName(trimmedName);
    const compatibleRuntimes = computeCompatibleRuntimesForComponent({
      layer: currentLayer,
      subCategory: this.subCategory(),
      compatibleRuntime: this.compatibleRuntime(),
      iconKey: this.iconKey(),
      editingSlug: this.editingComponent?.slug,
      generatedSlug
    });

    const judgeEntrypointValue =
      currentLayer === 'runtime' && this.subCategory() === 'backend' ? this.judgeEntrypoint().trim() : '';

    this.isSubmitting.set(true);
    this.errorMessage.set(null);

    const targetExisting = this.editingComponent || this.existingCatalogMatch();
    if (targetExisting) {
      const updateDto: UpdateTemplateComponentDTO = {
        name: trimmedName,
        sub_category: this.subCategory(),
        description: this.description().trim(),
        icon_key: this.effectiveIconKey(),
        base_image: currentLayer === 'runtime' ? this.baseImage().trim() : '',
        compatible_runtimes: compatibleRuntimes,
        install_mode: resolvedInstallMode,
        package_names: pkgs,
        build_layer_snippet: resolvedSnippet,
        setup_snippet: targetExisting.setup_snippet,
        judge_entrypoint: judgeEntrypointValue,
        tools_provided: tools,
        service_engine: currentLayer === 'database' ? this.serviceEngine().trim() : '',
        ram_delta_mb: Number(this.ramDeltaMB()) || 0,
        is_active: true
      };

      this.templatesService.updateTechCatalogItem(targetExisting.id, updateDto).subscribe({
        next: updated => {
          this.isSubmitting.set(false);
          this.saved.emit(updated);
        },
        error: err => {
          this.isSubmitting.set(false);
          this.errorMessage.set(err.error?.error?.message || err.message || 'No se pudo actualizar la tecnología.');
        }
      });
      return;
    }

    const createDto: CreateTemplateComponentDTO = {
      name: trimmedName,
      slug: generatedSlug,
      layer: currentLayer,
      sub_category: this.subCategory(),
      description: this.description().trim(),
      icon_key: this.effectiveIconKey(),
      base_image: currentLayer === 'runtime' ? this.baseImage().trim() : '',
      compatible_runtimes: compatibleRuntimes,
      install_mode: resolvedInstallMode,
      package_names: pkgs,
      build_layer_snippet: resolvedSnippet,
      judge_entrypoint: judgeEntrypointValue,
      tools_provided: tools,
      service_engine: currentLayer === 'database' ? this.serviceEngine().trim() : '',
      ram_delta_mb: Number(this.ramDeltaMB()) || 0
    };

    this.templatesService.createTechCatalogItem(createDto).subscribe({
      next: created => {
        this.isSubmitting.set(false);
        this.saved.emit(created);
      },
      error: err => {
        this.isSubmitting.set(false);
        this.errorMessage.set(err.error?.error?.message || err.message || 'No se pudo registrar la tecnología.');
      }
    });
  }
}
