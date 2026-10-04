import { Component, ChangeDetectionStrategy, model, input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { LucideShield, LucideCode } from '@lucide/angular';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';

@Component({
  selector: 'step-rules',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideShield,
    LucideCode,
    FormFieldComponent,
    MachineDataDirective
  ],
  templateUrl: './step-rules.component.html',
  styleUrl: './step-rules.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class StepRulesComponent {
  blockNativeSort = model.required<boolean>();
  blockSystemModules = model.required<boolean>();
  boilerplate = model.required<string>();
  language = input.required<string>();
}
