import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { Router, RouterModule } from '@angular/router';
import {
  AdminTemplatesService,
  TemplateModelItem,
  TemplateCategory,
  AdminTemplateItem,
  UpdateTemplateModelDTO,
  CreateCategoryDTO,
  UpdateCategoryDTO,
  ReorderCategoryItem,
  TemplateComponent,
  ComponentLayer,
  UpdateTemplateComponentDTO
} from '../../../services/admin-templates.service';
import { ComboboxComponent, ComboboxOption } from '../../../../../shared/components/combobox/combobox.component';
import { SearchBarComponent } from '../../../../../shared/components/search-bar/search-bar.component';
import { KpiCardComponent, KpiGridComponent } from '../../../../../shared/components/kpi-card/kpi-card.component';
import { ViewSwitcherComponent } from '../../../../../shared/components/view-switcher/view-switcher.component';
import { ModelNewModalComponent } from './components/model-new-modal/model-new-modal.component';
import { ModelEditModalComponent } from './components/model-edit-modal/model-edit-modal.component';
import { CategoryFormModalComponent } from './components/category-form-modal/category-form-modal.component';
import { CategoryDeleteModalComponent } from './components/category-delete-modal/category-delete-modal.component';
import { ComponentFormModalComponent } from './components/component-form-modal/component-form-modal.component';
import { TemplateModelsSectionComponent } from './components/template-models-section/template-models-section.component';
import { TemplateCategoriesManagerComponent } from './components/template-categories-manager/template-categories-manager.component';
import { TechLogoComponent } from '../tech-logo/tech-logo.component';
import {
  LucideBoxes,
  LucideTag,
  LucidePlus,
  LucidePencil,
  LucideTrash2,
  LucideCheck,
  LucidePower,
  LucideAlertTriangle,
  LucideCheckCircle2,
  LucideCode,
  LucideTerminal
} from '@lucide/angular';
import {
  computeComponentMemoryBadge,
  filterAndSortTechComponentsForLibrary,
  filterModelsForLibrary,
  getAddonEcosystemLabel,
  getCompatibleAddonsCountForRuntime,
  getComponentProfileLabel,
  isOrphanAddonInCatalog
} from './model-library.utils';

@Component({
  selector: 'model-library',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    RouterModule,
    MachineDataDirective,
    ComboboxComponent,
    SearchBarComponent,
    KpiCardComponent,
    KpiGridComponent,
    ViewSwitcherComponent,
    ModelNewModalComponent,
    ModelEditModalComponent,
    CategoryFormModalComponent,
    CategoryDeleteModalComponent,
    ComponentFormModalComponent,
    TemplateModelsSectionComponent,
    TemplateCategoriesManagerComponent,
    TechLogoComponent,
    LucideBoxes,
    LucideTag,
    LucidePlus,
    LucidePencil,
    LucideTrash2,
    LucideCheck,
    LucidePower,
    LucideAlertTriangle,
    LucideCheckCircle2,
    LucideCode,
    LucideTerminal
  ],
  templateUrl: './model-library.component.html',
  styleUrls: ['./model-library.component.scss']
})
export class ModelLibraryComponent implements OnInit {
  private readonly templatesService = inject(AdminTemplatesService);
  private readonly router = inject(Router);

  readonly activeTab = signal<'models' | 'components' | 'categories'>('models');
  readonly viewMode = signal<'cards' | 'table'>('table');
  readonly isLoading = signal<boolean>(false);

  readonly models = signal<TemplateModelItem[]>([]);
  readonly categories = signal<TemplateCategory[]>([]);
  readonly techComponents = signal<TemplateComponent[]>([]);
  readonly showComponentModal = signal<boolean>(false);
  readonly defaultComponentLayer = signal<ComponentLayer>('addon');
  readonly defaultComponentSubCategory = signal<'backend' | 'web'>('backend');
  readonly defaultCompatibleRuntime = signal<string>('');
  readonly editingComponent = signal<TemplateComponent | null>(null);

  readonly componentSearchQuery = signal<string>('');
  readonly componentProfileFilter = signal<'ALL' | 'BACKEND' | 'WEB' | 'ADDON' | 'DATABASE'>('ALL');
  readonly componentStatusFilter = signal<'ALL' | 'ACTIVE' | 'INACTIVE'>('ALL');

  readonly totalModelsCount = computed(() => this.models().length);
  readonly activeModelsCount = computed(() => this.models().filter(m => m.is_active !== false).length);
  readonly ideModelsCount = computed(() =>
    this.models().filter(m => m.target_environment === 'IDE_PERSISTENTE' || m.target_environment === 'AMBOS').length
  );
  readonly judgeModelsCount = computed(() =>
    this.models().filter(
      m =>
        m.target_environment === 'JUEZ_VIRTUAL' ||
        m.target_environment === 'JUEZ_EFIMERO' ||
        m.target_environment === 'AMBOS'
    ).length
  );
  readonly categoriesCount = computed(() => this.categories().length);
  readonly techComponentsCount = computed(() => this.techComponents().length);

  readonly searchQuery = signal<string>('');
  readonly filterPurpose = signal<string>('ALL');
  readonly filterCategory = signal<string>('ALL');
  readonly filterStatus = signal<'ALL' | 'ACTIVE' | 'INACTIVE'>('ALL');

  readonly showModelModal = signal<boolean>(false);
  readonly editingModel = signal<TemplateModelItem | null>(null);
  readonly modelTitle = signal<string>('');
  readonly modelDescription = signal<string>('');
  readonly modelCategoryId = signal<string>('');
  readonly isModelSaving = signal<boolean>(false);

  readonly showNewModelModal = signal<boolean>(false);
  readonly approvedTemplates = signal<AdminTemplateItem[]>([]);

  readonly selectedEditModelCategoryLabel = computed<string>(() => {
    const id = this.modelCategoryId();
    if (!id) return 'Seleccione una categoría';
    return this.categories().find(c => c.id === id)?.name || 'Seleccione una categoría';
  });

  onEditModelCategorySelected(opt: ComboboxOption): void {
    this.modelCategoryId.set(opt.value || '');
  }

  readonly purposeFilterComboboxOptions = computed<ComboboxOption[]>(() => [
    { id: 'ALL', label: 'Todos los propósitos', value: 'ALL' },
    { id: 'IDE_PERSISTENTE', label: 'IDE Persistente', value: 'IDE_PERSISTENTE' },
    { id: 'JUEZ_VIRTUAL', label: 'Juez Virtual', value: 'JUEZ_VIRTUAL' },
    { id: 'AMBOS', label: 'Laboratorio + Evaluador', value: 'AMBOS' }
  ]);

  readonly selectedPurposeFilterLabel = computed<string>(() => {
    const val = this.filterPurpose();
    return this.purposeFilterComboboxOptions().find(o => o.value === val)?.label || 'Todos los propósitos';
  });

  onPurposeFilterSelected(opt: ComboboxOption): void {
    this.filterPurpose.set(opt.value || 'ALL');
  }

  readonly categoryFilterComboboxOptions = computed<ComboboxOption[]>(() => [
    { id: 'ALL', label: 'Todas las categorías', value: 'ALL' },
    ...this.categories().map(c => ({ id: c.id, label: c.name, value: c.id }))
  ]);

  readonly selectedCategoryFilterLabel = computed<string>(() => {
    const val = this.filterCategory();
    if (val === 'ALL') return 'Todas las categorías';
    return this.categories().find(c => c.id === val)?.name || 'Todas las categorías';
  });

  onCategoryFilterSelected(opt: ComboboxOption): void {
    this.filterCategory.set(opt.value || 'ALL');
  }

  readonly statusFilterComboboxOptions = computed<ComboboxOption[]>(() => [
    { id: 'ALL', label: 'Todos', value: 'ALL' },
    { id: 'ACTIVE', label: 'Solo Activos', value: 'ACTIVE' },
    { id: 'INACTIVE', label: 'Solo Inactivos', value: 'INACTIVE' }
  ]);

  readonly selectedStatusFilterLabel = computed<string>(() => {
    const val = this.filterStatus();
    return this.statusFilterComboboxOptions().find(o => o.value === val)?.label || 'Todos';
  });

  onStatusFilterSelected(opt: ComboboxOption): void {
    this.filterStatus.set((opt.value as 'ALL' | 'ACTIVE' | 'INACTIVE') || 'ALL');
  }

  readonly componentProfileFilterOptions = computed<ComboboxOption[]>(() => [
    { id: 'ALL', label: 'Todos los perfiles', value: 'ALL' },
    { id: 'BACKEND', label: 'Lenguaje Principal', value: 'BACKEND' },
    { id: 'WEB', label: 'Interfaz Web / Frontend', value: 'WEB' },
    { id: 'ADDON', label: 'Complemento / Librería', value: 'ADDON' },
    { id: 'DATABASE', label: 'Base de Datos', value: 'DATABASE' }
  ]);

  readonly selectedComponentProfileFilterLabel = computed<string>(() => {
    const val = this.componentProfileFilter();
    return this.componentProfileFilterOptions().find(o => o.value === val)?.label || 'Todos los perfiles';
  });

  onComponentProfileSelected(opt: ComboboxOption): void {
    this.componentProfileFilter.set((opt.value as 'ALL' | 'BACKEND' | 'WEB' | 'ADDON' | 'DATABASE') || 'ALL');
  }

  readonly componentStatusFilterOptions = computed<ComboboxOption[]>(() => [
    { id: 'ALL', label: 'Todos los estados', value: 'ALL' },
    { id: 'ACTIVE', label: 'Solo Activos', value: 'ACTIVE' },
    { id: 'INACTIVE', label: 'Solo Inactivos', value: 'INACTIVE' }
  ]);

  readonly selectedComponentStatusFilterLabel = computed<string>(() => {
    const val = this.componentStatusFilter();
    return this.componentStatusFilterOptions().find(o => o.value === val)?.label || 'Todos los estados';
  });

  onComponentStatusSelected(opt: ComboboxOption): void {
    this.componentStatusFilter.set((opt.value as 'ALL' | 'ACTIVE' | 'INACTIVE') || 'ALL');
  }

  resetComponentFilters(): void {
    this.componentSearchQuery.set('');
    this.componentProfileFilter.set('ALL');
    this.componentStatusFilter.set('ALL');
  }

  readonly showCategoryModal = signal<boolean>(false);
  readonly editingCategory = signal<TemplateCategory | null>(null);
  readonly categoryName = signal<string>('');
  readonly categoryDescription = signal<string>('');
  readonly isCategorySaving = signal<boolean>(false);

  readonly categoryToDelete = signal<TemplateCategory | null>(null);
  readonly isCategoryDeleting = signal<boolean>(false);

  readonly toastMessage = signal<string | null>(null);
  readonly toastType = signal<'success' | 'error'>('success');
  private toastTimer: ReturnType<typeof setTimeout> | null = null;

  draggedCategoryIndex: number | null = null;

  readonly filteredModels = computed(() =>
    filterModelsForLibrary({
      models: this.models(),
      searchQuery: this.searchQuery(),
      filterPurpose: this.filterPurpose(),
      filterCategory: this.filterCategory(),
      filterStatus: this.filterStatus()
    })
  );

  readonly filteredTechComponents = computed(() =>
    filterAndSortTechComponentsForLibrary({
      components: this.techComponents(),
      searchQuery: this.componentSearchQuery(),
      profileFilter: this.componentProfileFilter(),
      statusFilter: this.componentStatusFilter()
    })
  );

  ngOnInit(): void {
    this.loadData();
  }

  loadData(): void {
    this.isLoading.set(true);
    this.templatesService.getModels(undefined, undefined, true).subscribe({
      next: models => {
        this.models.set(models);
        this.loadCategories();
      },
      error: err => {
        this.isLoading.set(false);
        this.showToast('Error al cargar la lista de modelos: ' + (err.message || 'Error de conexión'), 'error');
      }
    });
  }

  loadCategories(): void {
    this.templatesService.getCategories().subscribe({
      next: cats => {
        this.categories.set(cats);
        this.loadTechComponents();
      },
      error: err => {
        this.isLoading.set(false);
        this.showToast('Error al cargar categorías: ' + (err.message || 'Error de conexión'), 'error');
      }
    });
  }

  loadTechComponents(): void {
    if (typeof this.templatesService.getTechCatalog !== 'function') {
      this.isLoading.set(false);
      return;
    }
    this.templatesService.getTechCatalog(undefined, true).subscribe({
      next: items => {
        this.techComponents.set(items);
        this.isLoading.set(false);
      },
      error: () => {
        this.isLoading.set(false);
      }
    });
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
        this.showToast(`Tecnología "${comp.name}" ${nextActive ? 'activada' : 'desactivada'}.`, 'success');
      },
      error: err => {
        this.techComponents.update(list => list.map(c => (c.id === comp.id ? comp : c)));
        this.showToast(err.error?.error?.message || err.message || 'Error al cambiar estado de tecnología', 'error');
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
      this.showToast(`Tecnología "${savedComp.name}" actualizada exitosamente.`, 'success');
    } else {
      this.showToast(`Tecnología "${savedComp.name}" registrada en el catálogo de tecnologías.`, 'success');
      this.loadTechComponents();
    }
  }

  deleteTechComponent(item: TemplateComponent): void {
    if (item.is_system) {
      this.showToast('Las tecnologías base del sistema no se pueden eliminar.', 'error');
      return;
    }
    this.templatesService.deleteTechCatalogItem(item.id).subscribe({
      next: () => {
        this.showToast(`Tecnología "${item.name}" eliminada.`, 'success');
        this.loadTechComponents();
      },
      error: err => {
        this.showToast(err.error?.error?.message || 'Error al eliminar tecnología', 'error');
      }
    });
  }

  getCategoryModelCount(categoryId: string): number {
    return this.models().filter(m => m.category_id === categoryId).length;
  }

  getCategoryName(categoryId?: string | null): string {
    if (!categoryId) return 'Sin categoría';
    return this.categories().find(c => c.id === categoryId)?.name || 'Sin categoría';
  }

  openNewModelModal(): void {
    this.showNewModelModal.set(true);
    this.templatesService.getTemplates('approved').subscribe({
      next: tpls => this.approvedTemplates.set(tpls || []),
      error: () => this.approvedTemplates.set([])
    });
  }

  closeNewModelModal(): void {
    this.showNewModelModal.set(false);
  }

  navigateToTemplatesWizard(): void {
    this.closeNewModelModal();
    this.router.navigate(['/admin/plantillas/nueva']);
  }

  handleModelPromoted(_event: { id: string; name: string }): void {
    this.closeNewModelModal();
    this.showToast('Plantilla promovida a modelo oficial exitosamente.', 'success');
    this.loadData();
  }

  handleModelSaved(_event: { id: string; title: string }): void {
    this.closeModelModal();
    this.showToast('Modelo actualizado exitosamente.', 'success');
    this.loadData();
  }

  handleCategorySaved(_event: { id: string; name: string }): void {
    const isEdit = !!this.editingCategory();
    this.closeCategoryModal();
    this.showToast(isEdit ? 'Categoría actualizada.' : 'Categoría creada exitosamente.', 'success');
    this.loadCategories();
  }

  handleCategoryDeleted(cat: TemplateCategory): void {
    this.cancelDeleteCategory();
    this.showToast(`Categoría "${cat.name}" eliminada.`, 'success');
    this.loadCategories();
  }

  openEditModel(model: TemplateModelItem): void {
    this.editingModel.set(model);
    this.modelTitle.set(model.title || model.name || '');
    this.modelDescription.set(model.description || '');
    this.modelCategoryId.set(model.category_id || '');
    this.showModelModal.set(true);
  }

  closeModelModal(): void {
    this.showModelModal.set(false);
    this.editingModel.set(null);
  }

  saveModel(): void {
    const model = this.editingModel();
    if (!model) return;

    const title = this.modelTitle().trim();
    const catId = this.modelCategoryId().trim();

    if (!title) {
      this.showToast('El título del modelo es obligatorio.', 'error');
      return;
    }
    if (!catId) {
      this.showToast('Debe asignar una categoría académica.', 'error');
      return;
    }

    this.isModelSaving.set(true);
    const dto: UpdateTemplateModelDTO = {
      title,
      description: this.modelDescription().trim(),
      category_id: catId
    };

    this.templatesService.updateModel(model.id, dto).subscribe({
      next: () => {
        this.isModelSaving.set(false);
        this.closeModelModal();
        this.showToast('Modelo actualizado exitosamente.', 'success');
        this.loadData();
      },
      error: err => {
        this.isModelSaving.set(false);
        this.showToast(err.error?.error?.message || err.message || 'Error al guardar modelo', 'error');
      }
    });
  }

  toggleModelActive(model: TemplateModelItem): void {
    const isCurrentlyActive = model.is_active !== false;
    this.isLoading.set(true);

    const obs$ = isCurrentlyActive
      ? this.templatesService.deactivateModel(model.id)
      : this.templatesService.reactivateModel(model.id);

    obs$.subscribe({
      next: () => {
        this.showToast(
          `Modelo "${model.name || model.title}" ${isCurrentlyActive ? 'desactivado del' : 'reactivado en el'} catálogo.`,
          'success'
        );
        this.loadData();
      },
      error: err => {
        this.isLoading.set(false);
        this.showToast(err.error?.error?.message || err.message || 'Error al cambiar estado del modelo', 'error');
      }
    });
  }

  openCreateCategory(): void {
    this.editingCategory.set(null);
    this.categoryName.set('');
    this.categoryDescription.set('');
    this.showCategoryModal.set(true);
  }

  openEditCategory(cat: TemplateCategory): void {
    this.editingCategory.set(cat);
    this.categoryName.set(cat.name);
    this.categoryDescription.set(cat.description || '');
    this.showCategoryModal.set(true);
  }

  closeCategoryModal(): void {
    this.showCategoryModal.set(false);
    this.editingCategory.set(null);
  }

  saveCategory(): void {
    const name = this.categoryName().trim();
    if (!name) {
      this.showToast('El nombre de la categoría es obligatorio.', 'error');
      return;
    }

    this.isCategorySaving.set(true);
    const editing = this.editingCategory();
    const dto: CreateCategoryDTO | UpdateCategoryDTO = {
      name,
      description: this.categoryDescription().trim()
    };

    const obs$ = editing
      ? this.templatesService.updateCategory(editing.id, dto)
      : this.templatesService.createCategory(dto);

    obs$.subscribe({
      next: () => {
        this.isCategorySaving.set(false);
        this.closeCategoryModal();
        this.showToast(editing ? 'Categoría actualizada.' : 'Categoría creada exitosamente.', 'success');
        this.loadCategories();
      },
      error: err => {
        this.isCategorySaving.set(false);
        this.showToast(err.error?.error?.message || err.message || 'Error al guardar categoría', 'error');
      }
    });
  }

  confirmDeleteCategory(cat: TemplateCategory): void {
    this.categoryToDelete.set(cat);
  }

  cancelDeleteCategory(): void {
    this.categoryToDelete.set(null);
  }

  executeDeleteCategory(): void {
    const cat = this.categoryToDelete();
    if (!cat) return;

    this.isCategoryDeleting.set(true);
    this.templatesService.deleteCategory(cat.id).subscribe({
      next: () => {
        this.isCategoryDeleting.set(false);
        this.categoryToDelete.set(null);
        this.showToast('Categoría eliminada.', 'success');
        this.loadCategories();
      },
      error: err => {
        this.isCategoryDeleting.set(false);
        let msg = 'Error al eliminar categoría.';
        if (err.status === 409 || err.error?.error?.code === 'category_in_use') {
          msg = 'No se puede eliminar la categoría: tiene modelos o plantillas asociadas.';
        } else if (err.error?.error?.message) {
          msg = err.error.error.message;
        }
        this.showToast(msg, 'error');
      }
    });
  }

  moveCategoryUp(index: number): void {
    if (index <= 0) return;
    const cats = [...this.categories()];
    [cats[index - 1], cats[index]] = [cats[index], cats[index - 1]];
    this.applyCategoryOrder(cats);
  }

  moveCategoryDown(index: number): void {
    const cats = [...this.categories()];
    if (index >= cats.length - 1) return;
    [cats[index], cats[index + 1]] = [cats[index + 1], cats[index]];
    this.applyCategoryOrder(cats);
  }

  onDragStart(index: number): void {
    this.draggedCategoryIndex = index;
  }

  onDragOver(event: DragEvent, _index: number): void {
    event.preventDefault();
  }

  onDrop(targetIndex: number): void {
    if (this.draggedCategoryIndex === null || this.draggedCategoryIndex === targetIndex) {
      this.draggedCategoryIndex = null;
      return;
    }

    const cats = [...this.categories()];
    const [moved] = cats.splice(this.draggedCategoryIndex, 1);
    cats.splice(targetIndex, 0, moved);
    this.draggedCategoryIndex = null;
    this.applyCategoryOrder(cats);
  }

  private applyCategoryOrder(orderedCats: TemplateCategory[]): void {
    this.categories.set(orderedCats);
    const items: ReorderCategoryItem[] = orderedCats.map((cat, idx) => ({
      id: cat.id,
      sort_order: (idx + 1) * 10
    }));

    this.templatesService.reorderCategories(items).subscribe({
      next: () => this.showToast('Orden de categorías actualizado.', 'success'),
      error: () => {
        this.showToast('Error al persistir el nuevo orden.', 'error');
        this.loadCategories();
      }
    });
  }

  showToast(message: string, type: 'success' | 'error' = 'success'): void {
    if (this.toastTimer) clearTimeout(this.toastTimer);
    this.toastMessage.set(message);
    this.toastType.set(type);
    this.toastTimer = setTimeout(() => {
      this.toastMessage.set(null);
    }, 4000);
  }
}
