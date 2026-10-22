import { Component, ChangeDetectionStrategy, input } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'form-field',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './form-field.component.html',
  styleUrl: './form-field.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class FormFieldComponent {
  readonly label = input.required<string>();
  // El alias 'for' es parte de la API pública: cambia el nombre y rompes los templates consumidores.
  // eslint-disable-next-line @angular-eslint/no-input-rename
  readonly fieldId = input<string>('', { alias: 'for' });
  readonly required = input<boolean>(false);
  readonly hint = input<string>();
  readonly error = input<string>();
}
