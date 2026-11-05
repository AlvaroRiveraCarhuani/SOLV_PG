import { Component, input, output, signal, effect } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { LucideCalendar, LucideClock } from '@lucide/angular';

@Component({
  selector: 'ide-delivery-config',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideCalendar,
    LucideClock
  ],
  templateUrl: './ide-delivery-config.component.html',
  styleUrl: './ide-delivery-config.component.scss'
})
export class IdeDeliveryConfigComponent {
  dueDate = input<string>('');
  allowResubmit = input<boolean>(false);
  autoCheckpoint = input<boolean>(false);

  dueDateChange = output<string>();
  allowResubmitChange = output<boolean>();
  autoCheckpointChange = output<boolean>();

  readonly hasDueDate = signal<boolean>(false);

  constructor() {
    effect(() => {
      this.hasDueDate.set(!!this.dueDate());
    });
  }

  setHasDueDate(enabled: boolean): void {
    this.hasDueDate.set(enabled);
    if (!enabled) {
      this.dueDateChange.emit('');
    } else if (!this.dueDate()) {
      this.applyPresetDueDate(7);
    }
  }

  applyPresetDueDate(days: number | 'sunday'): void {
    const d = new Date();
    if (days === 'sunday') {
      const diff = (7 - d.getDay()) % 7 || 7;
      d.setDate(d.getDate() + diff);
    } else {
      d.setDate(d.getDate() + days);
    }
    d.setHours(23, 59, 0, 0);

    const year = d.getFullYear();
    const month = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');
    const hours = String(d.getHours()).padStart(2, '0');
    const mins = String(d.getMinutes()).padStart(2, '0');
    const localIso = `${year}-${month}-${day}T${hours}:${mins}`;

    this.hasDueDate.set(true);
    this.dueDateChange.emit(localIso);
  }

  onDueDateInput(val: string): void {
    this.dueDateChange.emit(val);
  }

  onToggleResubmit(checked: boolean): void {
    this.allowResubmitChange.emit(checked);
  }

  onToggleCheckpoint(checked: boolean): void {
    this.autoCheckpointChange.emit(checked);
  }
}
