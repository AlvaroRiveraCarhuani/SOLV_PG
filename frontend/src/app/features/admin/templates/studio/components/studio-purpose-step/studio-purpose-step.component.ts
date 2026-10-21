import { Component, input, output } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { TargetEnvironment } from '@features/admin/models/admin-templates.models';

@Component({
  selector: 'studio-purpose-step',
  imports: [FormsModule, ComboboxComponent],
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
}
