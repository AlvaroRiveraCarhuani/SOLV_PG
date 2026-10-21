import { Component, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import {
  LucideBoxes,
  LucideCheck,
  LucidePencil,
  LucidePower,
  LucideTag
} from '@lucide/angular';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { SearchBarComponent } from '@shared/components/search-bar/search-bar.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import {
  TemplateCategory,
  TemplateModelItem
} from '@features/admin/models/admin-templates.models';

@Component({
  selector: 'template-models-section',
  imports: [
    CommonModule,
    ComboboxComponent,
    SearchBarComponent,
    MachineDataDirective,
    LucideBoxes,
    LucideCheck,
    LucidePencil,
    LucidePower,
    LucideTag
  ],
  templateUrl: './template-models-section.component.html',
  styleUrls: ['./template-models-section.component.scss']
})
export class TemplateModelsSectionComponent {
  readonly viewMode = input.required<'cards' | 'table'>();
  readonly filteredModels = input.required<TemplateModelItem[]>();
  readonly categories = input<TemplateCategory[]>([]);
  readonly searchQuery = input<string>('');
  readonly selectedPurposeFilterLabel = input<string>('Todos los propósitos');
  readonly purposeFilterOptions = input<ComboboxOption[]>([]);
  readonly selectedCategoryFilterLabel = input<string>('Todas las categorías');
  readonly categoryFilterOptions = input<ComboboxOption[]>([]);
  readonly selectedStatusFilterLabel = input<string>('Todos');
  readonly statusFilterOptions = input<ComboboxOption[]>([]);

  readonly searchQueryChange = output<string>();
  readonly purposeFilterSelect = output<ComboboxOption>();
  readonly categoryFilterSelect = output<ComboboxOption>();
  readonly statusFilterSelect = output<ComboboxOption>();
  readonly editModel = output<TemplateModelItem>();
  readonly toggleModelActive = output<TemplateModelItem>();

  getCategoryName(categoryId?: string | null): string {
    if (!categoryId) return 'Sin categoría';
    const found = this.categories().find(c => c.id === categoryId);
    return found ? found.name : 'Sin categoría';
  }
}
