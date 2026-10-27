import { Component, ChangeDetectionStrategy, computed, input, signal } from '@angular/core';
import { TechLogoComponent } from '../../../tech-logo/tech-logo.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { TemplateComponent } from '@features/admin/models/admin-templates.models';
import { isOrphanAddonInCatalog } from '../../model-library.utils';
import {
  LucideNetwork,
  LucideCode,
  LucideGlobe,
  LucidePuzzle,
  LucideDatabase,
  LucideChevronRight,
  LucideCheckCircle2,
  LucidePower,
  LucideAlertTriangle
} from '@lucide/angular';

export type CatalogGroupId = 'languages' | 'web' | 'addons' | 'databases';

export interface CatalogTreeGroup {
  id: CatalogGroupId;
  label: string;
  hint: string;
  components: TemplateComponent[];
  activeCount: number;
}

/**
 * Mapa del Sistema: árbol de solo lectura con todas las tecnologías del
 * catálogo, agrupadas por rol. Es la vista panorámica para el administrador;
 * la edición sigue viva en la pestaña «Catálogo de Tecnologías».
 */
@Component({
  selector: 'catalog-tree',
  standalone: true,
  imports: [
    TechLogoComponent,
    MachineDataDirective,
    LucideNetwork,
    LucideCode,
    LucideGlobe,
    LucidePuzzle,
    LucideDatabase,
    LucideChevronRight,
    LucideCheckCircle2,
    LucidePower,
    LucideAlertTriangle
  ],
  templateUrl: './catalog-tree.component.html',
  styleUrl: './catalog-tree.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class CatalogTreeComponent {
  readonly components = input<TemplateComponent[]>([]);

  readonly total = computed(() => this.components().length);
  readonly activeTotal = computed(
    () => this.components().filter((c) => c.is_active !== false).length
  );
  readonly inactiveTotal = computed(() => this.total() - this.activeTotal());

  private readonly expandedState = signal<Record<CatalogGroupId, boolean>>({
    languages: true,
    web: true,
    addons: true,
    databases: true
  });

  readonly expanded = this.expandedState.asReadonly();

  toggleGroup(id: CatalogGroupId): void {
    this.expandedState.update((value) => ({ ...value, [id]: !value[id] }));
  }

  readonly groups = computed<CatalogTreeGroup[]>(() => {
    const all = this.components();

    const sortLeaves = (a: TemplateComponent, b: TemplateComponent): number => {
      const activeDiff =
        (a.is_active !== false ? 0 : 1) - (b.is_active !== false ? 0 : 1);
      if (activeDiff !== 0) return activeDiff;
      const orderDiff = (a.sort_order ?? 0) - (b.sort_order ?? 0);
      if (orderDiff !== 0) return orderDiff;
      return (a.name || '').localeCompare(b.name || '');
    };

    const pick = (
      match: (comp: TemplateComponent) => boolean
    ): { components: TemplateComponent[]; activeCount: number } => {
      const list = all.filter(match).sort(sortLeaves);
      return {
        components: list,
        activeCount: list.filter((c) => c.is_active !== false).length
      };
    };

    return [
      {
        id: 'languages',
        label: 'Lenguajes Principales',
        hint: 'Runtime base sobre los que se construyen los entornos',
        ...pick((c) => c.layer === 'runtime' && c.sub_category !== 'web')
      },
      {
        id: 'web',
        label: 'Interfaces Web',
        hint: 'Plataformas disponibles en el IDE persistente',
        ...pick((c) => c.layer === 'runtime' && c.sub_category === 'web')
      },
      {
        id: 'addons',
        label: 'Complementos y Librerías',
        hint: 'Herramientas instalables sobre un lenguaje',
        ...pick((c) => c.layer === 'addon')
      },
      {
        id: 'databases',
        label: 'Bases de Datos',
        hint: 'Servicios de datos que acompañan al entorno',
        ...pick((c) => c.layer === 'database')
      }
    ];
  });

  isActive(comp: TemplateComponent): boolean {
    return comp.is_active !== false;
  }

  isOrphan(comp: TemplateComponent): boolean {
    return isOrphanAddonInCatalog(comp, this.components());
  }
}
