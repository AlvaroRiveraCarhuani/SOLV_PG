import { Component, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import {
  LucidePlus,
  LucideDatabase,
  LucideLayers,
  LucideLaptop
} from '@lucide/angular';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { TechLogoComponent, TechIconKey } from '@features/admin/templates/components/tech-logo/tech-logo.component';
import {
  detectTemplateTech,
  inferTemplateTechnologies
} from '@features/admin/templates/template-tech-detector.utils';
import { ApprovedTemplate } from '../../step-ide-environment.component';

export interface TechBadgeItem {
  label: string;
  icon: TechIconKey;
}

@Component({
  selector: 'ide-template-selector',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucidePlus,
    LucideDatabase,
    LucideLayers,
    LucideLaptop,
    ComboboxComponent,
    TechLogoComponent
  ],
  templateUrl: './ide-template-selector.component.html',
  styleUrl: './ide-template-selector.component.scss'
})
export class IdeTemplateSelectorComponent {
  approvedTemplates = input<ApprovedTemplate[]>([]);
  selectedTemplate = input<ApprovedTemplate | null>(null);
  templateOptions = input<ComboboxOption[]>([]);

  templateSelected = output<string>();
  requestTemplate = output<void>();

  onTemplateOptionSelected(opt: ComboboxOption): void {
    if (opt?.value) {
      this.templateSelected.emit(opt.value);
    }
  }

  getPrimaryTechIcon(tpl: ApprovedTemplate): TechIconKey {
    return detectTemplateTech(tpl).iconType;
  }

  getTechBadges(tpl: ApprovedTemplate): TechBadgeItem[] {
    const hasDb = this.hasDatabaseService(tpl);
    const inferred = inferTemplateTechnologies(tpl);

    return inferred
      .filter(tech => !(hasDb && this.isDatabaseTech(tech.id)))
      .map(item => ({
        label: item.name,
        icon: item.iconType
      }));
  }

  private isDatabaseTech(techId: string): boolean {
    return ['postgres', 'mysql', 'mariadb', 'mongodb', 'redis', 'sqlite', 'db'].includes(techId);
  }

  hasDatabaseService(tpl: ApprovedTemplate): boolean {
    return this.databaseEngine(tpl) !== '';
  }

  databaseEngine(tpl: ApprovedTemplate): string {
    const config = tpl?.services_config;
    if (!config) return '';
    if (config.database?.enabled) return config.database.engine || 'PostgreSQL';
    const services = Array.isArray(config.services) ? config.services : [];
    const db = services.find(
      s => s.category === 'database' && !!s.engine && s.engine !== 'none'
    );
    if (!db) return '';
    if (db.engine === 'postgres') return 'PostgreSQL';
    return db.engine || '';
  }
}
