import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router, RouterModule } from '@angular/router';
import {
  AdminTemplatesService,
  TemplateModelItem,
  TemplateCategory,
  AdminTemplateItem,
  UpdateTemplateModelDTO,
  CreateCategoryDTO,
  UpdateCategoryDTO,
  ReorderCategoryItem
} from '../../../services/admin-templates.service';
import { ComboboxComponent, ComboboxOption } from '../../../../../shared/components/combobox/combobox.component';
import { SearchBarComponent } from '../../../../../shared/components/search-bar/search-bar.component';
import { KpiCardComponent, KpiGridComponent } from '../../../../../shared/components/kpi-card/kpi-card.component';
import {
  LucideBoxes,
  LucideTag,
  LucidePlus,
  LucidePencil,
  LucideTrash2,
  LucideX,
  LucideCheck,
  LucidePower,
  LucideAlertTriangle,
  LucideGripVertical,
  LucideChevronUp,
  LucideChevronDown,
  LucideLayoutGrid,
  LucideTable,
  LucideCheckCircle2,
  LucideCode,
  LucideTerminal,
  LucideSparkles
} from '@lucide/angular';

@Component({
  selector: 'solv-model-library',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    RouterModule,
    ComboboxComponent,
    SearchBarComponent,
    KpiCardComponent,
    KpiGridComponent,
    LucideBoxes,
    LucideTag,
    LucidePlus,
    LucidePencil,
    LucideTrash2,
    LucideX,
    LucideCheck,
    LucidePower,
    LucideAlertTriangle,
    LucideGripVertical,
    LucideChevronUp,
    LucideChevronDown,
    LucideLayoutGrid,
    LucideTable,
    LucideCheckCircle2,
    LucideCode,
    LucideTerminal,
    LucideSparkles
  ],
  templateUrl: './model-library.component.html',
  styleUrls: ['./model-library.component.scss']
})
export class ModelLibraryComponent implements OnInit {
  private readonly templatesService = inject(AdminTemplatesService);
  private readonly router = inject(Router);

  // Tabs y Vista
  readonly activeTab = signal<'models' | 'categories'>('models');
  readonly viewMode = signal<'cards' | 'table'>('table');
  readonly isLoading = signal<boolean>(false);

  // Data
  readonly models = signal<TemplateModelItem[]>([]);
  readonly categories = signal<TemplateCategory[]>([]);

  // KPIs
  readonly totalModelsCount = computed(() => this.models().length);
  readonly activeModelsCount = computed(() => this.models().filter(m => m.is_active !== false).length);
  readonly ideModelsCount = computed(() => this.models().filter(m => m.target_environment === 'IDE_PERSISTENTE').length);
  readonly judgeModelsCount = computed(() => this.models().filter(m => m.target_environment === 'JUEZ_VIRTUAL' || m.target_environment === 'JUEZ_EFIMERO').length);
  readonly categoriesCount = computed(() => this.categories().length);

  // Filtros Modelos
  readonly searchQuery = signal<string>('');
  readonly filterPurpose = signal<string>('ALL');
  readonly filterCategory = signal<string>('ALL');
  readonly filterStatus = signal<'ALL' | 'ACTIVE' | 'INACTIVE'>('ALL');

  // Modales Modelos (Edición)
  readonly showModelModal = signal<boolean>(false);
  readonly editingModel = signal<TemplateModelItem | null>(null);
  readonly modelTitle = signal<string>('');
  readonly modelDescription = signal<string>('');
  readonly modelCategoryId = signal<string>('');
  readonly isModelSaving = signal<boolean>(false);

  // Modal Nuevo Modelo Oficial (Promoción / Creación)
  readonly showNewModelModal = signal<boolean>(false);
  readonly approvedTemplates = signal<AdminTemplateItem[]>([]);
  readonly selectedTemplateId = signal<string>('');
  readonly newModelTitle = signal<string>('');
  readonly newModelCategoryId = signal<string>('');
  readonly newModelDescription = signal<string>('');
  readonly isNewModelPromoting = signal<boolean>(false);
  readonly newModelError = signal<string | null>(null);

  readonly approvedTemplateComboboxOptions = computed<ComboboxOption[]>(() => {
    return this.approvedTemplates().map(tpl => ({
      id: tpl.id,
      label: tpl.name,
      value: tpl.id,
      meta: `${tpl.docker_image} · ${tpl.target_environment || 'IDE'}`
    }));
  });

  readonly selectedTemplateToPromoteLabel = computed<string>(() => {
    const id = this.selectedTemplateId();
    if (!id) return '';
    const match = this.approvedTemplates().find(t => t.id === id);
    return match ? match.name : '';
  });

  onTemplateToPromoteSelected(opt: ComboboxOption): void {
    this.onSelectTemplateToPromote(opt.value || '');
  }

  readonly categoryComboboxOptions = computed<ComboboxOption[]>(() => {
    return [
      { id: '', label: '-- Sin categoría asignada --', value: '' },
      ...this.categories().map(c => ({
        id: c.id,
        label: c.name,
        value: c.id
      }))
    ];
  });

  readonly selectedNewModelCategoryLabel = computed<string>(() => {
    const id = this.newModelCategoryId();
    if (!id) return '-- Sin categoría asignada --';
    const match = this.categories().find(c => c.id === id);
    return match ? match.name : '-- Sin categoría asignada --';
  });

  onNewModelCategorySelected(opt: ComboboxOption): void {
    this.newModelCategoryId.set(opt.value || '');
  }

  readonly selectedEditModelCategoryLabel = computed<string>(() => {
    const id = this.modelCategoryId();
    if (!id) return 'Seleccione una categoría';
    const match = this.categories().find(c => c.id === id);
    return match ? match.name : 'Seleccione una categoría';
  });

  onEditModelCategorySelected(opt: ComboboxOption): void {
    this.modelCategoryId.set(opt.value || '');
  }

  readonly purposeFilterComboboxOptions = computed<ComboboxOption[]>(() => [
    { id: 'ALL', label: 'Todos los propósitos', value: 'ALL' },
    { id: 'IDE_PERSISTENTE', label: 'IDE Persistente', value: 'IDE_PERSISTENTE' },
    { id: 'JUEZ_VIRTUAL', label: 'Juez Virtual', value: 'JUEZ_VIRTUAL' }
  ]);

  readonly selectedPurposeFilterLabel = computed<string>(() => {
    const val = this.filterPurpose();
    const match = this.purposeFilterComboboxOptions().find(o => o.value === val);
    return match ? match.label : 'Todos los propósitos';
  });

  onPurposeFilterSelected(opt: ComboboxOption): void {
    this.filterPurpose.set(opt.value || 'ALL');
  }

  readonly categoryFilterComboboxOptions = computed<ComboboxOption[]>(() => [
    { id: 'ALL', label: 'Todas las categorías', value: 'ALL' },
    ...this.categories().map(c => ({
      id: c.id,
      label: c.name,
      value: c.id
    }))
  ]);

  readonly selectedCategoryFilterLabel = computed<string>(() => {
    const val = this.filterCategory();
    if (val === 'ALL') return 'Todas las categorías';
    const match = this.categories().find(c => c.id === val);
    return match ? match.name : 'Todas las categorías';
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
    const match = this.statusFilterComboboxOptions().find(o => o.value === val);
    return match ? match.label : 'Todos';
  });

  onStatusFilterSelected(opt: ComboboxOption): void {
    this.filterStatus.set(opt.value || 'ALL');
  }

  // Modales Categorías
  readonly showCategoryModal = signal<boolean>(false);
  readonly editingCategory = signal<TemplateCategory | null>(null);
  readonly categoryName = signal<string>('');
  readonly categoryDescription = signal<string>('');
  readonly isCategorySaving = signal<boolean>(false);

  // Modal Confirmar Eliminar Categoría
  readonly categoryToDelete = signal<TemplateCategory | null>(null);
  readonly isCategoryDeleting = signal<boolean>(false);

  // Feedback Toast
  readonly toastMessage = signal<string | null>(null);
  readonly toastType = signal<'success' | 'error'>('success');
  private toastTimer: any = null;

  // Drag and drop state
  draggedCategoryIndex: number | null = null;

  // Modelos filtrados
  readonly filteredModels = computed(() => {
    const list = this.models();
    const query = this.searchQuery().trim().toLowerCase();
    const purpose = this.filterPurpose();
    const catId = this.filterCategory();
    const status = this.filterStatus();

    return list.filter(m => {
      // Búsqueda por nombre o descripción o imagen
      if (query) {
        const matchesName = (m.name || m.title || '').toLowerCase().includes(query);
        const matchesDesc = (m.description || '').toLowerCase().includes(query);
        const matchesImage = (m.docker_image || '').toLowerCase().includes(query);
        if (!matchesName && !matchesDesc && !matchesImage) return false;
      }

      // Propósito
      if (purpose !== 'ALL') {
        if (purpose === 'JUEZ_VIRTUAL') {
          if (m.target_environment !== 'JUEZ_VIRTUAL' && m.target_environment !== 'JUEZ_EFIMERO') {
            return false;
          }
        } else if (m.target_environment !== purpose) {
          return false;
        }
      }

      // Categoría
      if (catId !== 'ALL' && m.category_id !== catId) {
        return false;
      }

      // Estado activo/inactivo
      if (status === 'ACTIVE' && m.is_active === false) {
        return false;
      }
      if (status === 'INACTIVE' && m.is_active !== false) {
        return false;
      }

      return true;
    });
  });

  ngOnInit(): void {
    this.loadData();
  }

  loadData(): void {
    this.isLoading.set(true);
    // Cargar modelos con include_inactive=true para gestión completa
    this.templatesService.getModels(undefined, undefined, true).subscribe({
      next: (models) => {
        this.models.set(models);
        this.loadCategories();
      },
      error: (err) => {
        this.isLoading.set(false);
        this.showToast('Error al cargar la lista de modelos: ' + (err.message || 'Error de conexión'), 'error');
      }
    });
  }

  loadCategories(): void {
    this.templatesService.getCategories().subscribe({
      next: (cats) => {
        this.categories.set(cats);
        this.isLoading.set(false);
      },
      error: (err) => {
        this.isLoading.set(false);
        this.showToast('Error al cargar categorías: ' + (err.message || 'Error de conexión'), 'error');
      }
    });
  }

  getCategoryModelCount(categoryId: string): number {
    return this.models().filter(m => m.category_id === categoryId).length;
  }

  getCategoryName(categoryId?: string | null): string {
    if (!categoryId) return 'Sin categoría';
    const found = this.categories().find(c => c.id === categoryId);
    return found ? found.name : 'Sin categoría';
  }

  // --- GESTIÓN DE MODELOS ---

  openNewModelModal(): void {
    this.showNewModelModal.set(true);
    this.selectedTemplateId.set('');
    this.newModelTitle.set('');
    this.newModelCategoryId.set('');
    this.newModelDescription.set('');
    this.newModelError.set(null);
    this.templatesService.getTemplates('approved').subscribe({
      next: (tpls) => this.approvedTemplates.set(tpls || []),
      error: () => this.approvedTemplates.set([])
    });
  }

  closeNewModelModal(): void {
    this.showNewModelModal.set(false);
    this.newModelError.set(null);
  }

  onSelectTemplateToPromote(tplId: string): void {
    this.selectedTemplateId.set(tplId);
    const tpl = this.approvedTemplates().find(t => t.id === tplId);
    if (tpl) {
      this.newModelTitle.set(tpl.name);
      this.newModelCategoryId.set(tpl.category_id || '');
      this.newModelDescription.set(tpl.description || '');
    }
  }

  confirmPromoteToModel(): void {
    const tplId = this.selectedTemplateId();
    const title = this.newModelTitle().trim();
    if (!tplId) {
      this.newModelError.set('Debe seleccionar una plantilla aprobada como base.');
      return;
    }
    if (!title) {
      this.newModelError.set('El título del modelo es obligatorio.');
      return;
    }

    this.isNewModelPromoting.set(true);
    this.newModelError.set(null);

    this.templatesService.promoteToModel(tplId, {
      name: title,
      category_id: this.newModelCategoryId() || undefined,
      description: this.newModelDescription().trim()
    }).subscribe({
      next: () => {
        this.isNewModelPromoting.set(false);
        this.closeNewModelModal();
        this.showToast('Plantilla promovida a modelo oficial exitosamente.', 'success');
        this.loadData();
      },
      error: (err) => {
        this.isNewModelPromoting.set(false);
        const msg = err.error?.message || err.error?.error || 'No se pudo promover la plantilla a modelo.';
        this.newModelError.set(msg);
      }
    });
  }

  navigateToTemplatesWizard(): void {
    this.closeNewModelModal();
    this.router.navigate(['/admin/plantillas']);
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
      error: (err) => {
        this.isModelSaving.set(false);
        const msg = err.error?.error?.message || err.message || 'Error al guardar modelo';
        this.showToast(msg, 'error');
      }
    });
  }

  toggleModelActive(model: TemplateModelItem): void {
    const isCurrentlyActive = model.is_active !== false;
    this.isLoading.set(true);

    if (isCurrentlyActive) {
      this.templatesService.deactivateModel(model.id).subscribe({
        next: () => {
          this.showToast(`Modelo "${model.name || model.title}" desactivado del catálogo.`, 'success');
          this.loadData();
        },
        error: (err) => {
          this.isLoading.set(false);
          const msg = err.error?.error?.message || err.message || 'Error al desactivar';
          this.showToast(msg, 'error');
        }
      });
    } else {
      this.templatesService.reactivateModel(model.id).subscribe({
        next: () => {
          this.showToast(`Modelo "${model.name || model.title}" reactivado en el catálogo.`, 'success');
          this.loadData();
        },
        error: (err) => {
          this.isLoading.set(false);
          const msg = err.error?.error?.message || err.message || 'Error al reactivar';
          this.showToast(msg, 'error');
        }
      });
    }
  }

  // --- GESTIÓN DE CATEGORÍAS ---

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

    if (editing) {
      const dto: UpdateCategoryDTO = {
        name,
        description: this.categoryDescription().trim()
      };
      this.templatesService.updateCategory(editing.id, dto).subscribe({
        next: () => {
          this.isCategorySaving.set(false);
          this.closeCategoryModal();
          this.showToast('Categoría actualizada.', 'success');
          this.loadCategories();
        },
        error: (err) => {
          this.isCategorySaving.set(false);
          const msg = err.error?.error?.message || err.message || 'Error al actualizar categoría';
          this.showToast(msg, 'error');
        }
      });
    } else {
      const dto: CreateCategoryDTO = {
        name,
        description: this.categoryDescription().trim()
      };
      this.templatesService.createCategory(dto).subscribe({
        next: () => {
          this.isCategorySaving.set(false);
          this.closeCategoryModal();
          this.showToast('Categoría creada exitosamente.', 'success');
          this.loadCategories();
        },
        error: (err) => {
          this.isCategorySaving.set(false);
          const msg = err.error?.error?.message || err.message || 'Error al crear categoría';
          this.showToast(msg, 'error');
        }
      });
    }
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
      error: (err) => {
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

  // --- REORDENAMIENTO DE CATEGORÍAS ---

  moveCategoryUp(index: number): void {
    if (index <= 0) return;
    const cats = [...this.categories()];
    const temp = cats[index];
    cats[index] = cats[index - 1];
    cats[index - 1] = temp;
    this.applyCategoryOrder(cats);
  }

  moveCategoryDown(index: number): void {
    const cats = [...this.categories()];
    if (index >= cats.length - 1) return;
    const temp = cats[index];
    cats[index] = cats[index + 1];
    cats[index + 1] = temp;
    this.applyCategoryOrder(cats);
  }

  onDragStart(index: number): void {
    this.draggedCategoryIndex = index;
  }

  onDragOver(event: DragEvent, index: number): void {
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
      next: () => {
        this.showToast('Orden de categorías actualizado.', 'success');
      },
      error: () => {
        this.showToast('Error al persistir el nuevo orden.', 'error');
        this.loadCategories();
      }
    });
  }

  // --- TOAST NOTIFICATIONS ---

  showToast(message: string, type: 'success' | 'error' = 'success'): void {
    if (this.toastTimer) {
      clearTimeout(this.toastTimer);
    }
    this.toastMessage.set(message);
    this.toastType.set(type);
    this.toastTimer = setTimeout(() => {
      this.toastMessage.set(null);
    }, 4000);
  }
}
