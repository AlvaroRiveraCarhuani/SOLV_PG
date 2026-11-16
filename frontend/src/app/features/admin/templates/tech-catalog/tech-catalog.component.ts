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
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { SearchBarComponent } from '@shared/components/search-bar/search-bar.component';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';
import { ConfirmModalComponent } from '@shared/components/confirm-modal/confirm-modal.component';
import { ViewSwitcherComponent } from '@shared/components/view-switcher/view-switcher.component';
import { ToastService } from '@shared/services/toast.service';
import { ComponentFormModalComponent } from '../components/model-library/components/component-form-modal/component-form-modal.component';
import { CatalogTreeComponent } from '../components/model-library/components/catalog-tree/catalog-tree.component';
import { TechLogoComponent } from '../components/tech-logo/tech-logo.component';
import { ContextHelpService } from '@shared/help/context-help.service';
import { ProductTourService } from '@shared/help/product-tour.service';
import { InteractiveTutorialService } from '@shared/help/interactive-tutorial.service';
import {
  buildTechCatalogTour,
  buildTechCatalogTutorial
} from './tech-catalog-help.definitions';
import {
  LucideCpu,
  LucidePlus,
  LucidePencil,
  LucideTrash2,
  LucideCheck,
  LucidePower,
  LucideAlertTriangle,
  LucideNetwork,
  LucideBoxes
} from '@lucide/angular';
import {
  computeComponentMemoryBadge,
  filterAndSortTechComponentsForLibrary,
  getAddonEcosystemLabel,
  getCompatibleAddonsCountForRuntime,
  getComponentProfileLabel,
  isManageableTechComponent,
  isOrphanAddonInCatalog
} from '../components/model-library/model-library.utils';

export const TECH_CATALOG_HELP_KEY = 'tech-catalog';

type DeleteDialog = {
  title: string;
  message: string;
  confirmText: string;
  type: 'danger' | 'primary';
  action: 'delete' | 'deactivate' | 'close';
};

@Component({
  selector: 'tech-catalog',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    MachineDataDirective,
    ComboboxComponent,
    SearchBarComponent,
    SkeletonLoaderComponent,
    ConfirmModalComponent,
    ViewSwitcherComponent,
    ComponentFormModalComponent,
    CatalogTreeComponent,
    TechLogoComponent,
    LucideCpu,
    LucidePlus,
    LucidePencil,
    LucideTrash2,
    LucideCheck,
    LucidePower,
    LucideAlertTriangle,
    LucideNetwork,
    LucideBoxes
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

  readonly activeTab = signal<'components' | 'map'>('components');
  readonly viewMode = signal<'cards' | 'table'>('cards');
  readonly isLoading = signal<boolean>(false);

  readonly techComponents = signal<TemplateComponent[]>([]);
  readonly showComponentModal = signal<boolean>(false);
  readonly defaultComponentLayer = signal<ComponentLayer>('addon');
  readonly defaultComponentSubCategory = signal<'backend' | 'web'>('backend');
  readonly defaultCompatibleRuntime = signal<string>('');
  readonly editingComponent = signal<TemplateComponent | null>(null);

  readonly componentSearchQuery = signal<string>('');
  readonly componentProfileFilter = signal<'ALL' | 'BACKEND' | 'WEB' | 'ADDON' | 'DATABASE'>('ALL');
  readonly componentStatusFilter = signal<'ALL' | 'ACTIVE' | 'INACTIVE'>('ALL');

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

  // Los conteos viven donde se filtra: la opción declara cuántos elementos
  // selecciona. Antes ocupaban cinco tarjetas KPI que empujaban el catálogo
  // bajo el pliegue sin aportar una decisión.
  readonly componentProfileFilterOptions = computed<ComboboxOption[]>(() => [
    { id: 'ALL', label: `Todos los perfiles (${this.totalComponentsCount()})`, value: 'ALL' },
    { id: 'BACKEND', label: `Lenguaje Principal (${this.backendRuntimesCount()})`, value: 'BACKEND' },
    { id: 'WEB', label: `Interfaz Web / Frontend (${this.webRuntimesCount()})`, value: 'WEB' },
    { id: 'ADDON', label: `Complemento / Librería (${this.addonsCount()})`, value: 'ADDON' },
    { id: 'DATABASE', label: `Base de Datos (${this.databasesCount()})`, value: 'DATABASE' }
  ]);

  readonly selectedComponentProfileFilterLabel = computed<string>(() => {
    const val = this.componentProfileFilter();
    return this.componentProfileFilterOptions().find(o => o.value === val)?.label || 'Todos los perfiles';
  });

  readonly componentStatusFilterOptions = computed<ComboboxOption[]>(() => [
    { id: 'ALL', label: 'Todos los estados', value: 'ALL' },
    { id: 'ACTIVE', label: 'Solo Activos', value: 'ACTIVE' },
    { id: 'INACTIVE', label: 'Solo Inactivos', value: 'INACTIVE' }
  ]);

  readonly selectedComponentStatusFilterLabel = computed<string>(() => {
    const val = this.componentStatusFilter();
    return this.componentStatusFilterOptions().find(o => o.value === val)?.label || 'Todos los estados';
  });

  readonly filteredTechComponents = computed(() =>
    filterAndSortTechComponentsForLibrary({
      components: this.techComponents(),
      searchQuery: this.componentSearchQuery(),
      profileFilter: this.componentProfileFilter(),
      statusFilter: this.componentStatusFilter()
    })
  );

  readonly pendingDelete = signal<TemplateComponent | null>(null);
  readonly isDeleting = signal<boolean>(false);

  readonly deleteDialog = computed<DeleteDialog | null>(() => {
    const comp = this.pendingDelete();
    if (!comp) {
      return null;
    }
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
        hint: 'Señala pestañas, filtros, tarjetas y acciones de baja.',
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

  /** Tutorial de retiro seguro: no exige un botón que el catálogo oficial no tiene. */
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

  onComponentProfileSelected(opt: ComboboxOption): void {
    this.componentProfileFilter.set((opt.value as 'ALL' | 'BACKEND' | 'WEB' | 'ADDON' | 'DATABASE') || 'ALL');
  }

  onComponentStatusSelected(opt: ComboboxOption): void {
    this.componentStatusFilter.set((opt.value as 'ALL' | 'ACTIVE' | 'INACTIVE') || 'ALL');
  }

  resetComponentFilters(): void {
    this.componentSearchQuery.set('');
    this.componentProfileFilter.set('ALL');
    this.componentStatusFilter.set('ALL');
  }

  openNewComponentModal(
    layer?: ComponentLayer,
    subCategory?: 'backend' | 'web',
    compatibleRuntime: string = ''
  ): void {
    let resolvedLayer = layer;
    let resolvedSubCategory = subCategory;

    if (!resolvedLayer) {
      const profile = this.componentProfileFilter();
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

  getAddonEcosystemLabel(comp: TemplateComponent): string {
    return getAddonEcosystemLabel(comp);
  }

  isOrphanAddon(comp: TemplateComponent): boolean {
    return isOrphanAddonInCatalog(comp, this.techComponents());
  }

  getCompatibleAddonsCount(runtimeComp: TemplateComponent): number {
    return getCompatibleAddonsCountForRuntime(runtimeComp, this.techComponents());
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
    if (!dialog || !comp) {
      return;
    }
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
