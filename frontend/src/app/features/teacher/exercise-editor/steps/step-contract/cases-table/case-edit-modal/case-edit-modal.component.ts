import { Component, EventEmitter, Input, Output, computed, signal, OnChanges, SimpleChanges } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { LucideX, LucideCheck, LucideFileText } from '@lucide/angular';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';

@Component({
  selector: 'case-edit-modal',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideX,
    LucideCheck,
    LucideFileText,
    MachineDataDirective
  ],
  templateUrl: './case-edit-modal.component.html',
  styleUrl: './case-edit-modal.component.scss'
})
export class CaseEditModalComponent implements OnChanges {
  @Input() isOpen = false;
  @Input() title = 'Editar contenido';
  @Input() subtitle = '';
  @Input() value = '';
  @Input() placeholder = 'Escribe o pega el contenido multilínea...';

  @Output() save = new EventEmitter<string>();
  @Output() cancel = new EventEmitter<void>();

  readonly currentValue = signal<string>('');

  readonly lineCount = computed(() => {
    const val = this.currentValue();
    if (!val) return 0;
    return val.split('\n').length;
  });

  readonly charCount = computed(() => this.currentValue().length);

  ngOnChanges(changes: SimpleChanges): void {
    if (changes['value'] && this.isOpen) {
      this.currentValue.set(this.value || '');
    }
    if (changes['isOpen'] && this.isOpen) {
      this.currentValue.set(this.value || '');
    }
  }

  onTextChange(val: string): void {
    this.currentValue.set(val);
  }

  onSave(): void {
    this.save.emit(this.currentValue());
  }

  onCancel(): void {
    this.cancel.emit();
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
