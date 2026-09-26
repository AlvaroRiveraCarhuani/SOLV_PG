import { Component, EventEmitter, Input, Output, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { LucideAlertTriangle, LucideX } from '@lucide/angular';
import { TemplateCategory, AdminTemplatesService } from '../../../../../services/admin-templates.service';

@Component({
  selector: 'category-delete-modal',
  standalone: true,
  imports: [
    CommonModule,
    LucideAlertTriangle,
    LucideX
  ],
  templateUrl: './category-delete-modal.component.html',
  styleUrls: ['./category-delete-modal.component.scss']
})
export class CategoryDeleteModalComponent {
  private readonly templatesService = inject(AdminTemplatesService);

  @Input({ required: true }) category!: TemplateCategory;

  @Output() cancelled = new EventEmitter<void>();
  @Output() deleted = new EventEmitter<TemplateCategory>();

  readonly isSubmitting = signal<boolean>(false);
  readonly errorMessage = signal<string | null>(null);

  onCancel(): void {
    this.cancelled.emit();
  }

  onConfirm(): void {
    if (!this.category) return;

    this.isSubmitting.set(true);
    this.errorMessage.set(null);

    this.templatesService.deleteCategory(this.category.id).subscribe({
      next: () => {
        this.isSubmitting.set(false);
        this.deleted.emit(this.category);
      },
      error: (err) => {
        this.isSubmitting.set(false);
        let msg = 'Error al eliminar categoría.';
        if (err.status === 409 || err.error?.error?.code === 'category_in_use') {
          msg = 'No se puede eliminar la categoría: tiene modelos o plantillas asociadas.';
        } else if (err.error?.error?.message) {
          msg = err.error.error.message;
        }
        this.errorMessage.set(msg);
      }
    });
  }
}
