import { Component, OnInit, OnDestroy, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { SearchBarComponent } from '@shared/components/search-bar/search-bar.component';
import {
  AdminTemplatesService,
  TemplateCategory,
  ReorderCategoryItem,
  TemplateModelItem
} from '../services/admin-templates.service';
import { CategoryFormModalComponent } from '../templates/components/model-library/components/category-form-modal/category-form-modal.component';
import { CategoryDeleteModalComponent } from '../templates/components/model-library/components/category-delete-modal/category-delete-modal.component';
import { ContextHelpService } from '@shared/help/context-help.service';
import { ProductTourService } from '@shared/help/product-tour.service';
import { buildCategoriesTour } from './categories-help.definitions';
import {
  LucidePlus,
  LucideAlertTriangle,
  LucideCheckCircle2,
  LucideGripVertical,
  LucideChevronUp,
  LucideChevronDown,
  LucidePencil,
  LucideTrash2,
  LucideArrowRight,
  LucideInfo
} from '@lucide/angular';

export const ACADEMIC_CATEGORIES_HELP_KEY = 'academic-categories';

@Component({
  selector: 'academic-categories',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    RouterLink,
    MachineDataDirective,
    SearchBarComponent,
    CategoryFormModalComponent,
    CategoryDeleteModalComponent,
    LucidePlus,
    LucideAlertTriangle,
    LucideCheckCircle2,
    LucideGripVertical,
    LucideChevronUp,
    LucideChevronDown,
    LucidePencil,
    LucideTrash2,
    LucideArrowRight,
    LucideInfo
  ],
  templateUrl: './academic-categories.component.html',
  styleUrls: ['./academic-categories.component.scss']
})
export class AcademicCategoriesComponent implements OnInit, OnDestroy {
  private readonly templatesService = inject(AdminTemplatesService);
  private readonly help = inject(ContextHelpService);
  private readonly productTour = inject(ProductTourService);

  readonly isLoading = signal<boolean>(false);
  readonly categories = signal<TemplateCategory[]>([]);
  readonly models = signal<TemplateModelItem[]>([]);
  readonly searchQuery = signal<string>('');

  readonly totalCategoriesCount = computed(() => this.categories().length);
  readonly activeCategoriesCount = computed(
    () => this.categories().filter(c => c.is_active !== false).length
  );

  readonly isFiltered = computed(() => this.searchQuery().trim().length > 0);

  readonly filteredCategories = computed(() => {
    const q = this.searchQuery().trim().toLowerCase();
    if (!q) return this.categories();
    return this.categories().filter(c =>
      c.name.toLowerCase().includes(q) ||
      (c.description && c.description.toLowerCase().includes(q))
    );
  });

  readonly showCategoryModal = signal<boolean>(false);
  readonly editingCategory = signal<TemplateCategory | null>(null);

  readonly categoryToDelete = signal<TemplateCategory | null>(null);

  readonly toastMessage = signal<string | null>(null);
  readonly toastType = signal<'success' | 'error'>('success');
  private toastTimer: ReturnType<typeof setTimeout> | null = null;
  draggedCategoryIndex: number | null = null;

  ngOnInit(): void {
    this.help.registerScreen(ACADEMIC_CATEGORIES_HELP_KEY);
    this.help.registerGuides(ACADEMIC_CATEGORIES_HELP_KEY, {
      tour: {
        label: 'Recorrido visual',
        hint: 'Señala el resumen, el alta y las reglas de la lista.',
        run: () => void this.productTour.start(buildCategoriesTour())
      }
    });
    this.loadData();
  }

  ngOnDestroy(): void {
    this.help.unregisterGuides(ACADEMIC_CATEGORIES_HELP_KEY);
    this.help.unregisterScreen(ACADEMIC_CATEGORIES_HELP_KEY);
    if (this.toastTimer) {
      clearTimeout(this.toastTimer);
    }
  }

  loadData(): void {
    this.isLoading.set(true);
    this.templatesService.getCategories().subscribe({
      next: cats => {
        this.categories.set(cats);
        this.loadModelsForUsage();
      },
      error: err => {
        this.isLoading.set(false);
        this.showToast('Error al cargar categorías: ' + (err.message || 'Error de conexión'), 'error');
      }
    });
  }

  private loadModelsForUsage(): void {
    this.templatesService.getModels(undefined, undefined, true).subscribe({
      next: models => {
        this.models.set(models);
        this.isLoading.set(false);
      },
      error: () => {
        this.isLoading.set(false);
      }
    });
  }

  onSearchChange(query: string): void {
    this.searchQuery.set(query);
  }

  onSearchClear(): void {
    this.searchQuery.set('');
  }

  getCategoryModelCount(categoryId: string): number {
    return this.models().filter(m => m.category_id === categoryId).length;
  }

  getCategoryIndex(cat: TemplateCategory): number {
    return this.categories().findIndex(c => c.id === cat.id);
  }

  formatOrder(order: number): string {
    const val = Math.floor(order);
    return val < 10 ? `0${val}` : `${val}`;
  }

  onRowClick(cat: TemplateCategory, event: MouseEvent): void {
    const target = event.target as HTMLElement | null;
    if (target?.closest('button') || target?.closest('a') || target?.closest('.order-controls')) {
      return;
    }
    this.openEditCategory(cat);
  }

  openCreateCategory(): void {
    this.editingCategory.set(null);
    this.showCategoryModal.set(true);
  }

  openEditCategory(cat: TemplateCategory): void {
    this.editingCategory.set(cat);
    this.showCategoryModal.set(true);
  }

  closeCategoryModal(): void {
    this.showCategoryModal.set(false);
    this.editingCategory.set(null);
  }

  handleCategorySaved(_event: { id: string; name: string }): void {
    const isEdit = !!this.editingCategory();
    this.closeCategoryModal();
    this.showToast(isEdit ? 'Categoría actualizada.' : 'Categoría creada exitosamente.', 'success');
    this.loadData();
  }

  confirmDeleteCategory(cat: TemplateCategory): void {
    this.categoryToDelete.set(cat);
  }

  cancelDeleteCategory(): void {
    this.categoryToDelete.set(null);
  }

  handleCategoryDeleted(cat: TemplateCategory): void {
    this.cancelDeleteCategory();
    this.showToast(`Categoría "${cat.name}" eliminada.`, 'success');
    this.loadData();
  }

  moveCategoryUp(index: number): void {
    if (this.isFiltered() || index <= 0) return;
    const cats = [...this.categories()];
    [cats[index - 1], cats[index]] = [cats[index], cats[index - 1]];
    this.applyCategoryOrder(cats);
  }

  moveCategoryDown(index: number): void {
    if (this.isFiltered()) return;
    const cats = [...this.categories()];
    if (index >= cats.length - 1) return;
    [cats[index], cats[index + 1]] = [cats[index + 1], cats[index]];
    this.applyCategoryOrder(cats);
  }

  onDragStart(index: number): void {
    if (this.isFiltered()) return;
    this.draggedCategoryIndex = index;
  }

  onDragOver(event: DragEvent): void {
    if (this.isFiltered()) return;
    event.preventDefault();
  }

  onDrop(targetIndex: number): void {
    if (this.isFiltered() || this.draggedCategoryIndex === null || this.draggedCategoryIndex === targetIndex) {
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
        this.loadData();
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
