import { Component, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { LucideCalendar } from '@lucide/angular';

@Component({
  selector: 'ide-delivery-config',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideCalendar
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

  onAllowResubmit(event: Event): void {
    const checked = (event.target as HTMLInputElement).checked;
    this.allowResubmitChange.emit(checked);
  }

  onAutoCheckpoint(event: Event): void {
    const checked = (event.target as HTMLInputElement).checked;
    this.autoCheckpointChange.emit(checked);
  }
}
