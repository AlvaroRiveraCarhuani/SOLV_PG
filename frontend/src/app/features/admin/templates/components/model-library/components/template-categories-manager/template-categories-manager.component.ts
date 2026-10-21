import { Component, input, output } from '@angular/core';
import {
  LucideChevronDown,
  LucideChevronUp,
  LucideGripVertical,
  LucidePencil,
  LucideTag,
  LucideTrash2
} from '@lucide/angular';
import {
  TemplateCategory,
  TemplateModelItem
} from '@features/admin/models/admin-templates.models';

@Component({
  selector: 'template-categories-manager',
  imports: [
    LucideChevronDown,
    LucideChevronUp,
    LucideGripVertical,
    LucidePencil,
    LucideTag,
    LucideTrash2
  ],
  templateUrl: './template-categories-manager.component.html',
  styleUrls: ['./template-categories-manager.component.scss']
})
export class TemplateCategoriesManagerComponent {
  readonly categories = input.required<TemplateCategory[]>();
  readonly models = input<TemplateModelItem[]>([]);

  readonly moveUp = output<number>();
  readonly moveDown = output<number>();
  readonly dragStartIndex = output<number>();
  readonly dropIndex = output<number>();
  readonly editCategory = output<TemplateCategory>();
  readonly deleteCategory = output<TemplateCategory>();

  getCategoryModelCount(categoryId: string): number {
    return this.models().filter(m => m.category_id === categoryId).length;
  }

  onDragOver(event: DragEvent): void {
    event.preventDefault();
  }
}
