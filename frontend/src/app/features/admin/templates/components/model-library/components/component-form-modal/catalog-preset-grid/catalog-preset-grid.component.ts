import { Component, input, output } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { TemplateComponent } from '../../../../../../services/admin-templates.service';
import { TechLogoComponent } from '../../../../tech-logo/tech-logo.component';
import {
  ComponentCatalogKind,
  EcosystemVersionOption,
  OfficialAddonPreset,
  OfficialEcosystemProfile,
  OfficialWebProfile
} from '../component-catalog.presets';
import { findCatalogMatchInList } from '../component-ecosystem.utils';

@Component({
  selector: 'catalog-preset-grid',
  imports: [FormsModule, TechLogoComponent],
  templateUrl: './catalog-preset-grid.component.html',
  styleUrls: ['./catalog-preset-grid.component.scss']
})
export class CatalogPresetGridComponent {
  readonly catalogKind = input.required<ComponentCatalogKind>();
  readonly ecosystemProfiles = input.required<OfficialEcosystemProfile[]>();
  readonly webProfiles = input.required<OfficialWebProfile[]>();
  readonly addonPresets = input.required<OfficialAddonPreset[]>();
  readonly availableRuntimes = input<TemplateComponent[]>([]);

  readonly selectedEcosystemProfileId = input<string>('');
  readonly selectedWebProfileId = input<string>('');
  readonly selectedAddonPresetId = input<string>('');
  readonly isCustomPresetSelected = input<boolean>(false);

  readonly availableVersions = input<EcosystemVersionOption[]>([]);
  readonly selectedVersion = input<string>('');
  readonly existingCatalogMatch = input<TemplateComponent | null>(null);

  readonly selectEcosystemProfile = output<string>();
  readonly selectWebProfile = output<string>();
  readonly selectAddonPreset = output<string>();
  readonly selectCustomMode = output<void>();
  readonly versionChange = output<string>();

  isPresetInCatalog(presetId: string, presetName: string): boolean {
    return !!findCatalogMatchInList(this.availableRuntimes(), presetId, presetName);
  }
}
