import { Component, EventEmitter, Input, OnInit, Output, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { LucideX, LucideAlertTriangle } from '@lucide/angular';
import { 
  TemplateModelItem, 
  TemplateCategory, 
  UpdateTemplateModelDTO, 
  AdminTemplatesService 
} from '../../../../../services/admin-templates.service';
import { ComboboxComponent, ComboboxOption } from '../../../../../../../shared/components/combobox/combobox.component';

@Component({
  selector: 'model-edit-modal',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ComboboxComponent,
    LucideX,
    LucideAlertTriangle
  ],
  templateUrl: './model-edit-modal.component.html',
  styleUrls: ['./model-edit-modal.component.scss']
})
export class ModelEditModalComponent implements OnInit {
  private readonly templatesService = inject(AdminTemplatesService);

  @Input({ required: true }) model!: TemplateModelItem;
  @Input() categories: TemplateCategory[] = [];

  @Output() closed = new EventEmitter<void>();
  @Output() saved = new EventEmitter<{ id: string; title: string }>();

  readonly title = signal<string>('');
  readonly categoryId = signal<string>('');
  readonly description = signal<string>('');
  readonly isSubmitting = signal<boolean>(false);
  readonly errorMessage = signal<string | null>(null);

  ngOnInit(): void {
    if (this.model) {
      this.title.set(this.model.title || this.model.name || '');
      this.categoryId.set(this.model.category_id || '');
      this.description.set(this.model.description || '');
    }
  }

  readonly categoryOptions = computed<ComboboxOption[]>(() => {
    return [
      { id: '', label: 'Seleccione una categoría', value: '' },
      ...this.categories.map(c => ({
        id: c.id,
        label: c.name,
        value: c.id
      }))
    ];
  });

  readonly selectedCategoryLabel = computed<string>(() => {
    const id = this.categoryId();
    if (!id) return 'Seleccione una categoría';
    const match = this.categories.find(c => c.id === id);
    return match ? match.name : 'Seleccione una categoría';
  });

  onCategorySelected(opt: ComboboxOption): void {
    this.categoryId.set(opt.value || '');
  }

  onClose(): void {
    this.closed.emit();
  }

  onSave(): void {
    const titleVal = this.title().trim();
    const catVal = this.categoryId().trim();

    if (!titleVal) {
      this.errorMessage.set('El título del modelo es obligatorio.');
      return;
    }
    if (!catVal) {
      this.errorMessage.set('Debe asignar una categoría académica.');
      return;
    }

    this.isSubmitting.set(true);
    this.errorMessage.set(null);

    const dto: UpdateTemplateModelDTO = {
      title: titleVal,
      description: this.description().trim(),
      category_id: catVal
    };

    this.templatesService.updateModel(this.model.id, dto).subscribe({
      next: () => {
        this.isSubmitting.set(false);
        this.saved.emit({ id: this.model.id, title: titleVal });
      },
      error: (err) => {
        this.isSubmitting.set(false);
        const msg = err.error?.error?.message || err.message || 'Error al guardar modelo';
        this.errorMessage.set(msg);
      }
    });
  }
}
