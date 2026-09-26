import { Component, EventEmitter, Input, Output, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { LucideSparkles, LucideAlertTriangle } from '@lucide/angular';
import { 
  AdminTemplateItem, 
  TemplateCategory, 
  AdminTemplatesService 
} from '../../../../../services/admin-templates.service';
import { ComboboxComponent, ComboboxOption } from '../../../../../../../shared/components/combobox/combobox.component';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';

@Component({
  selector: 'model-new-modal',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ComboboxComponent,
    ModalShellComponent,
    FormFieldComponent,
    LucideSparkles,
    LucideAlertTriangle
  ],
  templateUrl: './model-new-modal.component.html',
  styleUrls: ['./model-new-modal.component.scss']
})
export class ModelNewModalComponent {
  private readonly templatesService = inject(AdminTemplatesService);

  @Input() approvedTemplates: AdminTemplateItem[] = [];
  @Input() categories: TemplateCategory[] = [];

  @Output() closed = new EventEmitter<void>();
  @Output() promoted = new EventEmitter<{ id: string; name: string }>();
  @Output() openWizard = new EventEmitter<void>();

  readonly selectedTemplateId = signal<string>('');
  readonly modelTitle = signal<string>('');
  readonly modelCategoryId = signal<string>('');
  readonly modelDescription = signal<string>('');
  readonly isSubmitting = signal<boolean>(false);
  readonly errorMessage = signal<string | null>(null);

  readonly templateOptions = computed<ComboboxOption[]>(() => {
    return this.approvedTemplates.map(tpl => ({
      id: tpl.id,
      label: tpl.name,
      value: tpl.id,
      meta: `${tpl.docker_image} · ${tpl.target_environment || 'IDE'}`
    }));
  });

  readonly selectedTemplateLabel = computed<string>(() => {
    const id = this.selectedTemplateId();
    if (!id) return '';
    const match = this.approvedTemplates.find(t => t.id === id);
    return match ? match.name : '';
  });

  readonly categoryOptions = computed<ComboboxOption[]>(() => {
    return this.categories.map(c => ({
      id: c.id,
      label: c.name,
      value: c.id
    }));
  });

  readonly selectedCategoryLabel = computed<string>(() => {
    const id = this.modelCategoryId();
    if (!id) return '';
    const match = this.categories.find(c => c.id === id);
    return match ? match.name : '';
  });

  onTemplateSelected(opt: ComboboxOption): void {
    const id = opt.value || '';
    this.selectedTemplateId.set(id);
    const tpl = this.approvedTemplates.find(t => t.id === id);
    if (tpl) {
      this.modelTitle.set(tpl.name);
      this.modelCategoryId.set(tpl.category_id || '');
      this.modelDescription.set(tpl.description || '');
    }
  }

  onCategorySelected(opt: ComboboxOption): void {
    this.modelCategoryId.set(opt.value || '');
  }

  onClose(): void {
    this.closed.emit();
  }

  onOpenWizard(): void {
    this.openWizard.emit();
  }

  onConfirm(): void {
    const tplId = this.selectedTemplateId();
    const title = this.modelTitle().trim();
    if (!tplId) {
      this.errorMessage.set('Debe seleccionar una plantilla aprobada como base.');
      return;
    }
    if (!title) {
      this.errorMessage.set('El título del modelo es obligatorio.');
      return;
    }
    const categoryId = this.modelCategoryId();
    if (!categoryId) {
      this.errorMessage.set('La categoría es obligatoria');
      return;
    }

    this.isSubmitting.set(true);
    this.errorMessage.set(null);

    this.templatesService.promoteToModel(tplId, {
      name: title,
      category_id: categoryId,
      description: this.modelDescription().trim()
    }).subscribe({
      next: (res: any) => {
        this.isSubmitting.set(false);
        this.promoted.emit({ id: res?.id || tplId, name: title });
      },
      error: (err) => {
        this.isSubmitting.set(false);
        const msg = err.error?.message || err.error?.error || 'No se pudo promover la plantilla a modelo.';
        this.errorMessage.set(msg);
      }
    });
  }
}
