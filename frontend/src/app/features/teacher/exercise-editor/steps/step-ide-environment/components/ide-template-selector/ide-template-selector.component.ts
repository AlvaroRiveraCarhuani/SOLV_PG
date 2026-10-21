import { Component, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import {
  LucideServer,
  LucidePlus,
  LucideDatabase
} from '@lucide/angular';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { ApprovedTemplate } from '../../step-ide-environment.component';

@Component({
  selector: 'ide-template-selector',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideServer,
    LucidePlus,
    LucideDatabase,
    ComboboxComponent,
    MachineDataDirective
  ],
  templateUrl: './ide-template-selector.component.html',
  styleUrl: './ide-template-selector.component.scss'
})
export class IdeTemplateSelectorComponent {
  approvedTemplates = input<ApprovedTemplate[]>([]);
  selectedTemplate = input<ApprovedTemplate | null>(null);
  templateOptions = input<ComboboxOption[]>([]);

  templateSelected = output<string>();
  requestTemplate = output<void>();

  onTemplateOptionSelected(opt: ComboboxOption): void {
    if (opt?.value) {
      this.templateSelected.emit(opt.value);
    }
  }
}
