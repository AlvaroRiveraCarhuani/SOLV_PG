import { Component, ChangeDetectionStrategy, model, input, output, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { 
  LucideCode, 
  LucideLaptop, 
  LucideRadio, 
  LucideShield, 
  LucideEdit3, 
  LucideEye, 
  LucideClock,
  LucideLock 
} from '@lucide/angular';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import { marked } from 'marked';

@Component({
  selector: 'step-general',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideCode,
    LucideLaptop,
    LucideRadio,
    LucideShield,
    LucideEdit3,
    LucideEye,
    LucideClock,
    LucideLock,
    FormFieldComponent
  ],
  templateUrl: './step-general.component.html',
  styleUrl: './step-general.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class StepGeneralComponent {
  title = model.required<string>();
  titleError = input<string | null>(null);
  dueDate = model.required<string>();
  hasDueDate = model.required<boolean>();
  labType = model<'ALGORITMO' | 'IDE_PERSISTENTE' | null>(null);
  labTypeError = input<string | null>(null);
  lockLabType = input<boolean>(false);
  lockedTemplateName = input<string>('');
  pedagogicalPurpose = model<'PRACTICE' | 'EXAM' | null>(null);
  purposeError = input<string | null>(null);
  allowBroadcast = model.required<boolean>();
  description = model.required<string>();

  titleChanged = output<string>();

  activeDescTab = signal<'edit' | 'preview'>('edit');

  renderedDescription = computed(() => {
    const raw = this.description();
    if (!raw.trim()) return '<p class="empty-preview">Sin enunciado especificado.</p>';
    try {
      return marked.parse(raw, { async: false }) as string;
    } catch {
      return raw;
    }
  });

  onTitleInput(val: string): void {
    this.title.set(val);
    this.titleChanged.emit(val);
  }

  toggleNoDueDate(noDeadline: boolean): void {
    this.hasDueDate.set(!noDeadline);
    if (noDeadline) {
      this.dueDate.set('');
    }
  }

  setLabType(type: 'ALGORITMO' | 'IDE_PERSISTENTE'): void {
    if (this.lockLabType()) return;
    this.labType.set(type);
  }

  setPedagogicalPurpose(purpose: 'PRACTICE' | 'EXAM'): void {
    this.pedagogicalPurpose.set(purpose);
    this.allowBroadcast.set(purpose === 'PRACTICE');
  }
}
