import { Component, OnInit, inject, input, output, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import {
  LucidePlus,
  LucideLayers,
  LucideLaptop
} from '@lucide/angular';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { TechLogoComponent, TechIconKey } from '@features/admin/templates/components/tech-logo/tech-logo.component';
import {
  detectTemplateTech,
  inferTemplateTechnologies,
  resolveTemplateStack
} from '@features/admin/templates/template-tech-detector.utils';
import { AdminTemplatesService, TemplateComponent } from '@features/admin/services/admin-templates.service';
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
    LucideLayers,
    LucideLaptop,
    ComboboxComponent,
    TechLogoComponent
  ],
  templateUrl: './ide-template-selector.component.html',
  styleUrl: './ide-template-selector.component.scss'
})
export class IdeTemplateSelectorComponent implements OnInit {
  private templatesService = inject(AdminTemplatesService);

  approvedTemplates = input<ApprovedTemplate[]>([]);
  selectedTemplate = input<ApprovedTemplate | null>(null);
  templateOptions = input<ComboboxOption[]>([]);

  templateSelected = output<string>();
  requestTemplate = output<void>();

  techCatalog = signal<TemplateComponent[]>([]);

  ngOnInit(): void {
    this.templatesService.getTechCatalog().subscribe({
      next: list => this.techCatalog.set(list || []),
      error: () => this.techCatalog.set([])
    });
  }

  onTemplateOptionSelected(opt: ComboboxOption): void {
    if (opt?.value) {
      this.templateSelected.emit(opt.value);
    }
  }

  getPrimaryTechIcon(tpl: ApprovedTemplate): TechIconKey {
    return detectTemplateTech(tpl).iconType;
  }

  getTechBadges(tpl: ApprovedTemplate): TechBadgeItem[] {
    const catalog = this.techCatalog();
    if (catalog.length > 0 && tpl?.stack_config) {
      const resolved = resolveTemplateStack({ stack_config: tpl.stack_config }, catalog);
      if (resolved.length > 0) {
        const badges: TechBadgeItem[] = resolved.map(item => ({
          label: item.name,
          icon: item.iconKey as TechIconKey
        }));

        const dbEngine = this.databaseEngine(tpl);
        if (dbEngine && !badges.some(b => this.isDatabaseTech(b.icon) || b.label.toLowerCase().includes(dbEngine.toLowerCase()))) {
          badges.push({
            label: dbEngine,
            icon: this.databaseIconKey(dbEngine)
          });
        }
        return badges;
      }
    }

    const inferred = inferTemplateTechnologies(tpl);
    const badges: TechBadgeItem[] = inferred.map(item => ({
      label: item.name,
      icon: item.iconType
    }));

    const dbEngine = this.databaseEngine(tpl);
    if (dbEngine && !badges.some(b => this.isDatabaseTech(b.icon) || b.label.toLowerCase().includes(dbEngine.toLowerCase()))) {
      badges.push({
        label: dbEngine,
        icon: this.databaseIconKey(dbEngine)
      });
    }

    return badges;
  }

  isDatabaseTech(techId: string): boolean {
    const lower = (techId || '').toLowerCase();
    return ['postgres', 'mysql', 'mariadb', 'mongodb', 'redis', 'sqlite', 'db'].includes(lower);
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

  databaseIconKey(engine: string): TechIconKey {
    const lower = (engine || '').toLowerCase();
    if (lower.includes('postgres')) return 'postgres';
    if (lower.includes('mysql')) return 'mysql';
    if (lower.includes('mariadb')) return 'mariadb';
    if (lower.includes('mongo')) return 'mongodb';
    if (lower.includes('sqlite')) return 'sqlite';
    if (lower.includes('redis')) return 'redis';
    return 'db';
  }
}
