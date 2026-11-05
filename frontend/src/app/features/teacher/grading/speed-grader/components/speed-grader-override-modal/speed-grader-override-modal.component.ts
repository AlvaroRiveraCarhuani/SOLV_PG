import { Component, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { LucideEdit3, LucideAlertTriangle } from '@lucide/angular';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';

@Component({
  selector: 'speed-grader-override-modal',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideEdit3,
    LucideAlertTriangle,
    ModalShellComponent,
    ComboboxComponent
  ],
  templateUrl: './speed-grader-override-modal.component.html',
  styleUrl: './speed-grader-override-modal.component.scss'
})
export class SpeedGraderOverrideModalComponent {
  /** En modo proyecto no hay veredicto: solo se ajusta la nota. */
  readonly mode = input<'judge' | 'project'>('judge');
  readonly overrideVerdict = input<string>('AC');
  readonly overrideScore = input<number>(100);
  readonly overrideReason = input<string>('');
  readonly notifyStudent = input<boolean>(true);
  readonly overrideError = input<string | null>(null);
  readonly isSubmittingOverride = input<boolean>(false);
  readonly verdictOptions = input<ComboboxOption[]>([]);

  readonly dismiss = output<void>();
  readonly verdictChange = output<string>();
  readonly scoreChange = output<number>();
  readonly reasonChange = output<string>();
  readonly notifyStudentChange = output<boolean>();
  readonly submitOverride = output<void>();
}
