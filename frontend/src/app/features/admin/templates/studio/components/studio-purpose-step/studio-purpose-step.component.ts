import { Component, input, output } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { TargetEnvironment } from '@features/admin/models/admin-templates.models';

import { EnvironmentLabelPipe } from '@shared/pipes/environment-label.pipe';
import { LucideCheck, LucideCode, LucideLayers, LucideTerminal } from '@lucide/angular';

@Component({
  selector: 'studio-purpose-step',
  imports: [EnvironmentLabelPipe, FormsModule, ComboboxComponent, LucideCheck, LucideCode, LucideLayers, LucideTerminal],
  templateUrl: './studio-purpose-step.component.html',
  styleUrls: ['./studio-purpose-step.component.scss']
})
export class StudioPurposeStepComponent {
  readonly targetEnvironment = input.required<TargetEnvironment>();
  readonly name = input.required<string>();
  readonly description = input.required<string>();
  readonly selectedCategoryId = input<string | null>(null);
  readonly categoryOptions = input<ComboboxOption<string>[]>([]);

  readonly targetEnvironmentChange = output<TargetEnvironment>();
  readonly nameChange = output<string>();
  readonly descriptionChange = output<string>();
  readonly categoryChange = output<string | null>();

  selectModality(mode: 'IDE_PERSISTENTE' | 'JUEZ_EFIMERO' | 'AMBOS'): void {
    this.targetEnvironmentChange.emit(mode);
  }
}
