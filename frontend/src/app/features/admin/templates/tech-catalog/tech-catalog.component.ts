import { Component, OnInit, OnDestroy, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import {
  AdminTemplatesService,
  TemplateComponent,
  ComponentLayer,
  UpdateTemplateComponentDTO
} from '../../services/admin-templates.service';
import { StatusTabsComponent, StatusTabItem } from '@shared/components/status-tabs/status-tabs.component';
import { FilterPopoverComponent } from '@shared/components/filter-popover/filter-popover.component';
import { PaginationBarComponent } from '@shared/components/pagination-bar/pagination-bar.component';
import { SearchBarComponent } from '@shared/components/search-bar/search-bar.component';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';
import { ConfirmModalComponent } from '@shared/components/confirm-modal/confirm-modal.component';
import { ViewSwitcherComponent } from '@shared/components/view-switcher/view-switcher.component';
import { ToastService } from '@shared/services/toast.service';
import { ComponentFormModalComponent } from '../components/model-library/components/component-form-modal/component-form-modal.component';
import { TechLogoComponent } from '../components/tech-logo/tech-logo.component';
import { ContextHelpService } from '@shared/help/context-help.service';
import { ProductTourService } from '@shared/help/product-tour.service';
import { InteractiveTutorialService } from '@shared/help/interactive-tutorial.service';
import {
  buildTechCatalogTour,
  buildTechCatalogTutorial
} from './tech-catalog-help.definitions';
import {
  LucidePlus,
  LucidePencil,
  LucideTrash2,
  LucideCheck,
  LucideAlertTriangle,
  LucideBoxes,
  LucideArrowUp,
  LucideArrowDown
} from '@lucide/angular';
import {
  computeComponentMemoryBadge,
  getAddonEcosystemLabel,
  getCompatibleAddonsCountForRuntime,
  getComponentProfileLabel,
  isManageableTechComponent,
  isOrphanAddonInCatalog
} from '../components/model-library/model-library.utils';

export const TECH_CATALOG_HELP_KEY = 'tech-catalog';

export type DeleteDialog = {
  title: string;
  message: string;
  confirmText: string;
  type: 'danger' | 'primary';
  action: 'delete' | 'deactivate' | 'close';
};

export type TechSortField = 'name' | 'profile' | 'usage';

@Component({
  selector: 'tech-catalog',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    MachineDataDirective,
    StatusTabsComponent,
    FilterPopoverComponent,
    PaginationBarComponent,
    SearchBarComponent,
    SkeletonLoaderComponent,
    ConfirmModalComponent,
    ViewSwitcherComponent,
    ComponentFormModalComponent,
    TechLogoComponent,
    LucidePlus,
    LucidePencil,
    LucideTrash2,
    LucideCheck,
    LucideAlertTriangle,
    LucideBoxes,
    LucideArrowUp,
    LucideArrowDown
  ],
  templateUrl: './tech-catalog.component.html',
  styleUrls: ['./tech-catalog.component.scss']
})
export class TechCatalogComponent implements OnInit, OnDestroy {
  private readonly templatesService = inject(AdminTemplatesService);
  private readonly help = inject(ContextHelpService);
  private readonly productTour = inject(ProductTourService);
  private readonly tutorial = inject(InteractiveTutorialService);
  private readonly toast = inject(ToastService);

  readonly viewMode = signal<'table' | 'cards'>('table');
  readonly isLoading = signal<boolean>(false);

  readonly techComponents = signal<TemplateComponent[]>([]);
  readonly showComponentModal = signal<boolean>(false);
  readonly defaultComponentLayer = signal<ComponentLayer>('addon');
  readonly defaultComponentSubCategory = signal<'backend' | 'web'>('backend');
  readonly defaultCompatibleRuntime = signal<string>('');
  readonly editingComponent = signal<TemplateComponent | null>(null);

  readonly componentSearchQuery = signal<string>('');
  readonly selectedProfileTab = signal<'ALL' | 'BACKEND' | 'WEB' | 'ADDON' | 'DATABASE'>('ALL');
  readonly statusFilter = signal<'ALL' | 'ACTIVE' | 'INACTIVE'>('ALL');
  readonly originFilter = signal<'ALL' | 'OFFICIAL' | 'CUSTOM'>('ALL');

  // Alias para mantener compatibilidad con tests y consumers existentes
  get componentProfileFilter() {
    return this.selectedProfileTab;
  }
  get componentStatusFilter() {
    return this.statusFilter;
  }

  readonly sortField = signal<TechSortField>('name');
  readonly sortDirection = signal<'asc' | 'desc'>('asc');

  readonly pageSize = signal<number>(12);
  readonly currentPage = signal<number>(1);

  readonly totalComponentsCount = computed(() => this.techComponents().length);
  readonly activeComponentsCount = computed(
    () => this.techComponents().filter(c => c.is_active !== false).length
  );
  readonly backendRuntimesCount = computed(
    () => this.techComponents().filter(c => c.layer === 'runtime' && c.sub_category !== 'web').length
  );
  readonly webRuntimesCount = computed(
    () => this.techComponents().filter(c => c.layer === 'runtime' && c.sub_category === 'web').length
  );
  readonly addonsCount = computed(
    () => this.techComponents().filter(c => c.layer === 'addon').length
  );
  readonly databasesCount = computed(
    () => this.techComponents().filter(c => c.layer === 'database').length
  );

  readonly profileTabs = computed<StatusTabItem[]>(() => [
    { id: 'ALL', label: 'Todas', count: this.totalComponentsCount(), badgeVariant: 'neutral' },
    { id: 'BACKEND', label: 'Lenguajes', count: this.backendRuntimesCount(), badgeVariant: 'neutral' },
    { id: 'WEB', label: 'Interfaz Web', count: this.webRuntimesCount(), badgeVariant: 'neutral' },
    { id: 'ADDON', label: 'Complementos', count: this.addonsCount(), badgeVariant: 'neutral' },
    { id: 'DATABASE', label: 'Bases de datos', count: this.databasesCount(), badgeVariant: 'neutral' }
  ]);

  readonly activeFilterCount = computed<number>(() => {
    let count = 0;
    if (this.statusFilter() !== 'ALL') count++;
    if (this.originFilter() !== 'ALL') count++;
    return count;
  });

  readonly filteredTechComponents = computed(() => {
    const query = this.componentSearchQuery().trim().toLowerCase();
    const profile = this.selectedProfileTab();
    const status = this.statusFilter();
    const origin = this.originFilter();
    const sField = this.sortField();
    const sDir = this.sortDirection() === 'asc' ? 1 : -1;

    const items = this.techComponents().filter(c => {
      if (!isManageableTechComponent(c)) return false;

      if (profile === 'BACKEND' && (c.layer !== 'runtime' || c.sub_category === 'web')) return false;
      if (profile === 'WEB' && (c.layer !== 'runtime' || c.sub_category !== 'web')) return false;
      if (profile === 'ADDON' && c.layer !== 'addon') return false;
      if (profile === 'DATABASE' && c.layer !== 'database') return false;

      if (status === 'ACTIVE' && c.is_active === false) return false;
      if (status === 'INACTIVE' && c.is_active !== false) return false;

      if (origin === 'OFFICIAL' && !c.is_system) return false;
      if (origin === 'CUSTOM' && c.is_system) return false;

      if (query) {
        const matchesName = (c.name || '').toLowerCase().includes(query);
        const matchesSlug = (c.slug || '').toLowerCase().includes(query);
        const matchesDesc = (c.description || '').toLowerCase().includes(query);
        const matchesImage = (c.base_image || '').toLowerCase().includes(query);
        const matchesTools = (c.tools_provided || []).some(t => t.toLowerCase().includes(query));
        const matchesPkgs = (c.package_names || []).some(p => p.toLowerCase().includes(query));
        if (!matchesName && !matchesSlug && !matchesDesc && !matchesImage && !matchesTools && !matchesPkgs) {
          return false;
        }
      }

      return true;
    });

    return items.sort((a, b) => {
      if (sField === 'usage') {
        const uA = a.usage_count ?? 0;
        const uB = b.usage_count ?? 0;
        if (uA !== uB) return (uA - uB) * sDir;
        return (a.name || '').localeCompare(b.name || '');
      }

      if (sField === 'profile') {
        const pA = this.getProfileDisplayLabel(a);
        const pB = this.getProfileDisplayLabel(b);
        const cmp = pA.localeCompare(pB);
        if (cmp !== 0) return cmp * sDir;
        return (a.name || '').localeCompare(b.name || '');
      }

      return (a.name || '').localeCompare(b.name || '') * sDir;
    });
  });

  readonly totalFilteredItems = computed(() => this.filteredTechComponents().length);
  readonly totalPages = computed(() => Math.max(1, Math.ceil(this.totalFilteredItems() / this.pageSize())));

  readonly paginatedComponents = computed(() => {
    const start = (this.currentPage() - 1) * this.pageSize();
    return this.filteredTechComponents().slice(start, start + this.pageSize());
  });

  readonly paginationFrom = computed(() =>
    this.totalFilteredItems() === 0 ? 0 : (this.currentPage() - 1) * this.pageSize() + 1
  );
  readonly paginationTo = computed(() =>
    Math.min(this.currentPage() * this.pageSize(), this.totalFilteredItems())
  );

  readonly pendingDelete = signal<TemplateComponent | null>(null);
  readonly isDeleting = signal<boolean>(false);

  readonly deleteDialog = computed<DeleteDialog | null>(() => {
    const comp = this.pendingDelete();
    if (!comp) return null;
    const usage = comp.usage_count ?? 0;
    if (usage > 0) {
      if (comp.is_active === false) {
        return {
          title: 'Tecnología en uso',
          message: `«${comp.name}» está inactiva, pero ${usage} entorno(s) todavía la referencian. No se puede eliminar mientras la usen.`,
          confirmText: 'Entendido',
          type: 'primary',
          action: 'close'
        };
      }
      return {
        title: 'No se puede eliminar',
        message: `«${comp.name}» la usan ${usage} entorno(s). Desactívela para retirarla del constructor sin alterar esos entornos.`,
        confirmText: 'Desactivar tecnología',
        type: 'primary',
        action: 'deactivate'
      };
    }
    return {
      title: '¿Eliminar tecnología?',
      message: `«${comp.name}» no la usa ningún entorno. Esta acción no se puede deshacer.`,
      confirmText: 'Eliminar',
      type: 'danger',
      action: 'delete'
    };
  });

  ngOnInit(): void {
    this.help.registerScreen(TECH_CATALOG_HELP_KEY);
    this.help.registerGuides(TECH_CATALOG_HELP_KEY, {
      tour: {
        label: 'Recorrido visual',
        hint: 'Señala pestañas, filtros, tabla y acciones de baja.',
        run: () => void this.productTour.start(buildTechCatalogTour())
      },
      tutorial: {
        label: 'Tutorial guiado',
        hint: 'Enseña a retirar una tecnología que está en uso sin eliminar historial.',
        run: () => this.startTechCatalogTutorial()
      }
    });
    this.loadData();
  }

  ngOnDestroy(): void {
    this.help.unregisterGuides(TECH_CATALOG_HELP_KEY);
    this.help.unregisterScreen(TECH_CATALOG_HELP_KEY);
  }

  startTechCatalogTutorial(): void {
    this.tutorial.start(
      buildTechCatalogTutorial({
        hasSearchQuery: () => this.componentSearchQuery().trim().length > 0
      })
    );
  }

  loadData(): void {
    this.isLoading.set(true);
    if (typeof this.templatesService.getTechCatalog !== 'function') {
      this.isLoading.set(false);
      return;
    }
    this.templatesService.getTechCatalog(undefined, true).subscribe({
      next: items => {
        this.techComponents.set((items || []).filter(isManageableTechComponent));
        this.isLoading.set(false);
      },
      error: () => {
        this.isLoading.set(false);
        this.toast.show('Error al consultar el catálogo de tecnologías', 'error');
      }
    });
  }

  onTabChange(tabId: string): void {
    this.selectedProfileTab.set(tabId as 'ALL' | 'BACKEND' | 'WEB' | 'ADDON' | 'DATABASE');
    this.currentPage.set(1);
  }

  onSearchChange(query: string): void {
    this.componentSearchQuery.set(query);
    this.currentPage.set(1);
  }

  setStatusFilter(status: 'ALL' | 'ACTIVE' | 'INACTIVE'): void {
    this.statusFilter.set(status);
    this.currentPage.set(1);
  }

  setOriginFilter(origin: 'ALL' | 'OFFICIAL' | 'CUSTOM'): void {
    this.originFilter.set(origin);
    this.currentPage.set(1);
  }

  clearFilters(): void {
    this.statusFilter.set('ALL');
    this.originFilter.set('ALL');
    this.currentPage.set(1);
  }

  resetComponentFilters(): void {
    this.componentSearchQuery.set('');
    this.selectedProfileTab.set('ALL');
    this.statusFilter.set('ALL');
    this.originFilter.set('ALL');
    this.currentPage.set(1);
  }

  toggleSort(field: TechSortField): void {
    if (this.sortField() === field) {
      this.sortDirection.update(d => (d === 'asc' ? 'desc' : 'asc'));
    } else {
      this.sortField.set(field);
      this.sortDirection.set(field === 'usage' ? 'desc' : 'asc');
    }
    this.currentPage.set(1);
  }

  openNewComponentModal(
    layer?: ComponentLayer,
    subCategory?: 'backend' | 'web',
    compatibleRuntime: string = ''
  ): void {
    let resolvedLayer = layer;
    let resolvedSubCategory = subCategory;

    if (!resolvedLayer) {
      const profile = this.selectedProfileTab();
      if (profile === 'WEB') {
        resolvedLayer = 'runtime';
        resolvedSubCategory = 'web';
      } else if (profile === 'ADDON') {
        resolvedLayer = 'addon';
        resolvedSubCategory = 'backend';
      } else if (profile === 'DATABASE') {
        resolvedLayer = 'database';
        resolvedSubCategory = 'backend';
      } else {
        resolvedLayer = 'runtime';
        resolvedSubCategory = 'backend';
      }
    }

    this.editingComponent.set(null);
    this.defaultComponentLayer.set(resolvedLayer);
    this.defaultComponentSubCategory.set(resolvedSubCategory || 'backend');
    this.defaultCompatibleRuntime.set(compatibleRuntime);
    this.showComponentModal.set(true);
  }

  openEditComponentModal(comp: TemplateComponent): void {
    this.editingComponent.set(comp);
    this.defaultComponentLayer.set(comp.layer);
    this.defaultComponentSubCategory.set((comp.sub_category as 'backend' | 'web') || 'backend');
    this.defaultCompatibleRuntime.set(comp.compatible_runtimes?.[0] || '');
    this.showComponentModal.set(true);
  }

  openAddAddonForRuntime(rt: TemplateComponent): void {
    const key = (rt.icon_key && rt.icon_key !== 'docker' ? rt.icon_key : rt.slug || rt.name).toLowerCase();
    this.openNewComponentModal('addon', 'backend', key);
  }

  closeComponentModal(): void {
    this.editingComponent.set(null);
    this.showComponentModal.set(false);
  }

  toggleComponentActive(comp: TemplateComponent): void {
    const nextActive = comp.is_active === false;
    this.techComponents.update(list => list.map(c => (c.id === comp.id ? { ...c, is_active: nextActive } : c)));

    const dto: UpdateTemplateComponentDTO = {
      name: comp.name,
      sub_category: comp.sub_category,
      description: comp.description || '',
      icon_key: comp.icon_key,
      base_image: comp.base_image,
      compatible_runtimes: comp.compatible_runtimes || [],
      install_mode: comp.install_mode,
      package_names: comp.package_names || [],
      build_layer_snippet: comp.build_layer_snippet || '',
      setup_snippet: comp.setup_snippet || '',
      judge_entrypoint: comp.judge_entrypoint || '',
      tools_provided: comp.tools_provided || [],
      service_engine: comp.service_engine || '',
      ram_delta_mb: comp.ram_delta_mb || 0,
      sort_order: comp.sort_order || 0,
      is_active: nextActive
    };

    this.templatesService.updateTechCatalogItem(comp.id, dto).subscribe({
      next: updated => {
        this.techComponents.update(list => list.map(c => (c.id === comp.id ? updated : c)));
        this.toast.show(`Tecnología "${comp.name}" ${nextActive ? 'activada' : 'desactivada'}.`, 'success');
      },
      error: err => {
        this.techComponents.update(list => list.map(c => (c.id === comp.id ? comp : c)));
        this.toast.show(err.error?.message || 'Error al cambiar estado de tecnología', 'error');
      }
    });
  }

  getComponentProfileLabel(comp: TemplateComponent): string {
    return getComponentProfileLabel(comp);
  }

  getProfileDisplayLabel(comp: TemplateComponent): string {
    if (comp.layer === 'runtime') {
      return comp.sub_category === 'web' ? 'Interfaz Web' : 'Lenguaje';
    }
    if (comp.layer === 'addon') {
      const eco = getAddonEcosystemLabel(comp);
      return eco && eco !== 'Universal' ? `Complemento · ${eco}` : 'Complemento';
    }
    return 'Base de datos';
  }

  getAddonEcosystemLabel(comp: TemplateComponent): string {
    return getAddonEcosystemLabel(comp);
  }

  isOrphanAddon(comp: TemplateComponent): boolean {
    return isOrphanAddonInCatalog(comp, this.techComponents());
  }

  getCompatibleAddonsCount(runtimeComp: TemplateComponent): number {
    return getCompatibleAddonsCountForRuntime(runtimeComp, this.techComponents());
  }

  /** Concordancia del uso: «entorno» para 1, «entornos» para el resto. */
  getUsageNoun(comp: TemplateComponent): string {
    return (comp.usage_count ?? 0) === 1 ? 'entorno' : 'entornos';
  }

  getUsageTooltip(comp: TemplateComponent): string {
    const usage = comp.usage_count ?? 0;
    return usage === 1 ? 'La referencia 1 entorno' : `La referencian ${usage} entornos`;
  }

  /**
   * Memoria RAM a mostrar. Los lenguajes base (runtime que no es interfaz web)
   * declaran su memoria base sin «+»; las capas que suman RAM encima
   * (interfaz web, complementos y bases de datos) la muestran como delta.
   * Devuelve cadena vacía cuando no hay cuota que mostrar.
   */
  getRamLabel(comp: TemplateComponent): string {
    const mb = comp.ram_delta_mb ?? 0;
    if (mb <= 0) return '';
    const isAdditiveLayer = !(comp.layer === 'runtime' && comp.sub_category !== 'web');
    return isAdditiveLayer ? `+${mb} MB` : `${mb} MB`;
  }

  getComponentMemoryBadge(
    comp: TemplateComponent
  ): { label: string; detail: string; variant: 'light' | 'standard' | 'heavy' } {
    return computeComponentMemoryBadge(comp);
  }

  handleComponentSaved(savedComp: TemplateComponent): void {
    const isEdit = !!this.editingComponent();
    this.editingComponent.set(null);
    this.showComponentModal.set(false);

    if (isEdit) {
      this.techComponents.update(list => list.map(c => (c.id === savedComp.id ? savedComp : c)));
      this.toast.show(`Tecnología "${savedComp.name}" actualizada.`, 'success');
    } else {
      this.toast.show(`Tecnología "${savedComp.name}" registrada en el catálogo de tecnologías.`, 'success');
      this.loadData();
    }
  }

  openDeleteConfirm(item: TemplateComponent): void {
    if (item.is_system) {
      this.toast.show('Las tecnologías base del sistema no se pueden eliminar.', 'error');
      return;
    }
    this.pendingDelete.set(item);
  }

  cancelDelete(): void {
    this.pendingDelete.set(null);
  }

  onDeleteConfirmed(): void {
    const dialog = this.deleteDialog();
    const comp = this.pendingDelete();
    if (!dialog || !comp) return;

    if (dialog.action === 'close') {
      this.pendingDelete.set(null);
      return;
    }
    if (dialog.action === 'deactivate') {
      this.pendingDelete.set(null);
      this.toggleComponentActive(comp);
      return;
    }

    this.isDeleting.set(true);
    this.templatesService.deleteTechCatalogItem(comp.id).subscribe({
      next: () => {
        this.isDeleting.set(false);
        this.pendingDelete.set(null);
        this.toast.show(`Tecnología "${comp.name}" eliminada.`, 'success');
        this.loadData();
      },
      error: err => {
        this.isDeleting.set(false);
        this.pendingDelete.set(null);
        this.toast.show(err.error?.message || 'Error al eliminar tecnología', 'error');
      }
    });
  }
}
