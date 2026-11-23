import { Component, computed, input, output, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { LucideChevronDown, LucideChevronRight } from '@lucide/angular';
import { TemplateComponent } from '../../../../../../services/admin-templates.service';
import { TechLogoComponent } from '../../../../tech-logo/tech-logo.component';
import {
  ComponentCatalogKind,
  EcosystemVersionOption,
  OfficialAddonPreset,
  OfficialDatabaseProfile,
  OfficialEcosystemProfile,
  OfficialWebProfile
} from '../component-catalog.presets';
import { findCatalogMatchInList } from '../component-ecosystem.utils';

@Component({
  selector: 'catalog-preset-grid',
  imports: [FormsModule, TechLogoComponent, LucideChevronDown, LucideChevronRight],
  templateUrl: './catalog-preset-grid.component.html',
  styleUrls: ['./catalog-preset-grid.component.scss']
})
export class CatalogPresetGridComponent {
  readonly catalogKind = input.required<ComponentCatalogKind>();
  readonly ecosystemProfiles = input.required<OfficialEcosystemProfile[]>();
  readonly webProfiles = input.required<OfficialWebProfile[]>();
  readonly addonPresets = input.required<OfficialAddonPreset[]>();
  readonly databaseProfiles = input<OfficialDatabaseProfile[]>([]);
  readonly availableRuntimes = input<TemplateComponent[]>([]);

  readonly selectedEcosystemProfileId = input<string>('');
  readonly selectedWebProfileId = input<string>('');
  readonly selectedAddonPresetId = input<string>('');
  readonly selectedDatabaseProfileId = input<string>('');
  readonly isCustomPresetSelected = input<boolean>(false);

  readonly availableVersions = input<EcosystemVersionOption[]>([]);
  readonly selectedVersion = input<string>('');
  readonly existingCatalogMatch = input<TemplateComponent | null>(null);

  readonly selectEcosystemProfile = output<string>();
  readonly selectWebProfile = output<string>();
  readonly selectAddonPreset = output<string>();
  readonly selectDatabaseProfile = output<string>();
  readonly selectCustomMode = output<void>();
  readonly versionChange = output<string>();

  readonly showInstalled = signal<boolean>(false);

  readonly pendingEcosystemProfiles = computed(() =>
    this.ecosystemProfiles().filter(p => !this.isPresetInCatalog(p.id, p.name))
  );
  readonly installedEcosystemProfiles = computed(() =>
    this.ecosystemProfiles().filter(p => this.isPresetInCatalog(p.id, p.name))
  );

  readonly pendingWebProfiles = computed(() =>
    this.webProfiles().filter(p => !this.isPresetInCatalog(p.id, p.name))
  );
  readonly installedWebProfiles = computed(() =>
    this.webProfiles().filter(p => this.isPresetInCatalog(p.id, p.name))
  );

  readonly pendingDatabaseProfiles = computed(() =>
    this.databaseProfiles().filter(p => !this.isPresetInCatalog(p.id, p.name))
  );
  readonly installedDatabaseProfiles = computed(() =>
    this.databaseProfiles().filter(p => this.isPresetInCatalog(p.id, p.name))
  );

  readonly pendingAddonPresets = computed(() =>
    this.addonPresets().filter(p => !this.isPresetInCatalog(p.id, p.name))
  );
  readonly installedAddonPresets = computed(() =>
    this.addonPresets().filter(p => this.isPresetInCatalog(p.id, p.name))
  );

  isPresetInCatalog(presetId: string, presetName: string): boolean {
    return !!findCatalogMatchInList(this.availableRuntimes(), presetId, presetName);
  }

  isInstalledOpen(installedCount: number, pendingCount: number, isSelectedInInstalled: boolean): boolean {
    if (pendingCount === 0 && installedCount > 0) return true;
    if (isSelectedInInstalled) return true;
    return this.showInstalled();
  }

  toggleInstalled(): void {
    this.showInstalled.update(v => !v);
  }
}
