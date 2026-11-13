import { Component, OnInit, OnDestroy, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import {
  AdminTemplatesService,
  TemplateCategory,
  ReorderCategoryItem,
  TemplateModelItem
} from '../services/admin-templates.service';
import { KpiCardComponent, KpiGridComponent } from '@shared/components/kpi-card/kpi-card.component';
import { TemplateCategoriesManagerComponent } from '../templates/components/model-library/components/template-categories-manager/template-categories-manager.component';
import { CategoryFormModalComponent } from '../templates/components/model-library/components/category-form-modal/category-form-modal.component';
import { CategoryDeleteModalComponent } from '../templates/components/model-library/components/category-delete-modal/category-delete-modal.component';
import { ContextHelpService } from '@shared/help/context-help.service';
import {
  LucideTag,
  LucidePlus,
  LucideAlertTriangle,
  LucideCheckCircle2
} from '@lucide/angular';

export const ACADEMIC_CATEGORIES_HELP_KEY = 'academic-categories';

@Component({
  selector: 'academic-categories',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    KpiCardComponent,
    KpiGridComponent,
    TemplateCategoriesManagerComponent,
    CategoryFormModalComponent,
    CategoryDeleteModalComponent,
    LucideTag,
    LucidePlus,
    LucideAlertTriangle,
    LucideCheckCircle2
  ],
  templateUrl: './academic-categories.component.html',
  styleUrls: ['./academic-categories.component.scss']
})
export class AcademicCategoriesComponent implements OnInit, OnDestroy {
  private readonly templatesService = inject(AdminTemplatesService);
  private readonly help = inject(ContextHelpService);

  readonly isLoading = signal<boolean>(false);
  readonly categories = signal<TemplateCategory[]>([]);
  readonly models = signal<TemplateModelItem[]>([]);

  readonly totalCategoriesCount = computed(() => this.categories().length);
  readonly activeCategoriesCount = computed(
    () => this.categories().filter(c => c.is_active !== false).length
  );

  readonly showCategoryModal = signal<boolean>(false);
  readonly editingCategory = signal<TemplateCategory | null>(null);

  readonly categoryToDelete = signal<TemplateCategory | null>(null);

  readonly toastMessage = signal<string | null>(null);
  readonly toastType = signal<'success' | 'error'>('success');
  private toastTimer: ReturnType<typeof setTimeout> | null = null;
  draggedCategoryIndex: number | null = null;

  ngOnInit(): void {
    this.help.registerScreen(ACADEMIC_CATEGORIES_HELP_KEY);
    this.loadData();
  }

  ngOnDestroy(): void {
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
