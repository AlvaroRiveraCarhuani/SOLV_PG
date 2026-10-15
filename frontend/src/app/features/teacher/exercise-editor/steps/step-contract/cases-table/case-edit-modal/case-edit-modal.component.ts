import { Component, EventEmitter, Input, Output, computed, signal, OnChanges, SimpleChanges } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { 
  LucideX, 
  LucideCheck, 
  LucideFileText, 
  LucidePlay, 
  LucideLoader2
} from '@lucide/angular';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';

export interface CaseEditSavePayload {
  input: string;
  expected_output: string;
  description?: string;
}

@Component({
  selector: 'case-edit-modal',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideX,
    LucideCheck,
    LucideFileText,
    LucidePlay,
    LucideLoader2,
    MachineDataDirective
  ],
  templateUrl: './case-edit-modal.component.html',
  styleUrl: './case-edit-modal.component.scss'
})
export class CaseEditModalComponent implements OnChanges {
  @Input() isOpen = false;
  @Input() caseNumber = 1;
  @Input() inputVal = '';
  @Input() outputVal = '';
  @Input() descriptionVal = '';
  @Input() hasReferenceSolution = false;
  @Input() isCalculating = false;

  @Output() save = new EventEmitter<CaseEditSavePayload>();
  @Output() cancel = new EventEmitter<void>();
  @Output() calculateOutput = new EventEmitter<string>();

  readonly currentInput = signal<string>('');
  readonly currentOutput = signal<string>('');
  readonly currentDescription = signal<string>('');

  readonly inputLines = computed(() => {
    const val = this.currentInput();
    return val ? val.split('\n').length : 0;
  });

  readonly outputLines = computed(() => {
    const val = this.currentOutput();
    return val ? val.split('\n').length : 0;
  });

  ngOnChanges(changes: SimpleChanges): void {
    if (changes['isOpen'] && this.isOpen) {
      this.currentInput.set(this.normalizePosix(this.inputVal || ''));
      this.currentOutput.set(this.normalizePosix(this.outputVal || ''));
      this.currentDescription.set((this.descriptionVal || '').trim());
    } else {
      if (changes['inputVal'] && this.inputVal !== undefined) {
        this.currentInput.set(this.normalizePosix(this.inputVal));
      }
      if (changes['outputVal'] && this.outputVal !== undefined) {
        this.currentOutput.set(this.normalizePosix(this.outputVal));
      }
      if (changes['descriptionVal'] && this.descriptionVal !== undefined) {
        this.currentDescription.set((this.descriptionVal || '').trim());
      }
    }
  }

  onInputChange(val: string): void {
    this.currentInput.set(val);
  }

  onOutputChange(val: string): void {
    this.currentOutput.set(val);
  }

  onTriggerCalculate(): void {
    this.calculateOutput.emit(this.normalizePosix(this.currentInput()));
  }

  onSave(): void {
    this.save.emit({
      input: this.normalizePosix(this.currentInput()),
      expected_output: this.normalizePosix(this.currentOutput()),
      description: this.currentDescription().trim()
    });
  }

  onCancel(): void {
    this.cancel.emit();
  }

  private normalizePosix(text: string): string {
    if (!text) return '';
    return text.replace(/\r\n/g, '\n').replace(/\r/g, '\n');
  }

  onBackdropClick(event: MouseEvent): void {
    if ((event.target as HTMLElement).classList.contains('modal-backdrop')) {
      this.onCancel();
    }
  }

  onKeyDown(event: KeyboardEvent): void {
    if (event.key === 'Escape') {
      this.onCancel();
    }
  }
}
