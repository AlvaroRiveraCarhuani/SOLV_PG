import { Component, input, output } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import { CuratedIconMeta, TechLogoComponent } from '../../../../tech-logo/tech-logo.component';

@Component({
  selector: 'smart-icon-picker',
  imports: [FormsModule, FormFieldComponent, TechLogoComponent],
  templateUrl: './smart-icon-picker.component.html',
  styleUrls: ['./smart-icon-picker.component.scss']
})
export class SmartIconPickerComponent {
  readonly effectiveIconKey = input.required<string>();
  readonly activeIconLabel = input.required<string>();
  readonly iconResolutionSource = input.required<string>();

  readonly showIconPickerGrid = input<boolean>(false);
  readonly showCustomIconInput = input<boolean>(false);
  readonly iconSearchQuery = input<string>('');
  readonly filteredCuratedIcons = input<CuratedIconMeta[]>([]);
  readonly iconKey = input<string>('docker');
  readonly customIconUrl = input<string>('');

  readonly toggleGrid = output<void>();
  readonly toggleCustomUrl = output<void>();
  readonly searchQueryChange = output<string>();
  readonly selectIcon = output<string>();
  readonly customUrlChange = output<string>();
}
