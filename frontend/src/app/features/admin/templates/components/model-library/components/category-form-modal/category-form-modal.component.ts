import { Component, EventEmitter, Input, OnInit, Output, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { LucideTag, LucideAlertTriangle } from '@lucide/angular';
import {
  TemplateCategory,
  CreateCategoryDTO,
  UpdateCategoryDTO,
  AdminTemplatesService
} from '../../../../../services/admin-templates.service';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';

@Component({
  selector: 'category-form-modal',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ModalShellComponent,
    FormFieldComponent,
    LucideTag,
    LucideAlertTriangle
  ],
  templateUrl: './category-form-modal.component.html',
  styleUrls: ['./category-form-modal.component.scss']
})
export class CategoryFormModalComponent implements OnInit {
  private readonly templatesService = inject(AdminTemplatesService);

  @Input() category: TemplateCategory | null = null;

  @Output() closed = new EventEmitter<void>();
  @Output() saved = new EventEmitter<{ id: string; name: string }>();

  readonly name = signal<string>('');
  readonly description = signal<string>('');
  readonly isSubmitting = signal<boolean>(false);
  readonly errorMessage = signal<string | null>(null);

  ngOnInit(): void {
    if (this.category) {
      this.name.set(this.category.name);
      this.description.set(this.category.description || '');
    }
  }

  onClose(): void {
    this.closed.emit();
  }

  onSave(): void {
    const nameVal = this.name().trim();
    if (!nameVal) {
      this.errorMessage.set('El nombre de la categoría es obligatorio.');
      return;
    }

    this.isSubmitting.set(true);
    this.errorMessage.set(null);

    if (this.category) {
      const dto: UpdateCategoryDTO = {
        name: nameVal,
        description: this.description().trim()
      };
      this.templatesService.updateCategory(this.category.id, dto).subscribe({
        next: () => {
          this.isSubmitting.set(false);
          this.saved.emit({ id: this.category!.id, name: nameVal });
        },
        error: (err) => {
          this.isSubmitting.set(false);
          const msg = err.error?.error?.message || err.message || 'Error al actualizar categoría';
          this.errorMessage.set(msg);
        }
      });
    } else {
      const dto: CreateCategoryDTO = {
        name: nameVal,
        description: this.description().trim()
      };
      this.templatesService.createCategory(dto).subscribe({
        next: (created: any) => {
          this.isSubmitting.set(false);
          this.saved.emit({ id: created?.id || '', name: nameVal });
        },
        error: (err) => {
          this.isSubmitting.set(false);
          const msg = err.error?.error?.message || err.message || 'Error al crear categoría';
          this.errorMessage.set(msg);
        }
      });
    }
  }
}
