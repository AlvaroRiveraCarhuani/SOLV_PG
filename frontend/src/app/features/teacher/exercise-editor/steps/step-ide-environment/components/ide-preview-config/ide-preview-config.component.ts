import { Component, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { LucideMonitorPlay } from '@lucide/angular';

@Component({
  selector: 'ide-preview-config',
  standalone: true,
  imports: [CommonModule, FormsModule, LucideMonitorPlay],
  templateUrl: './ide-preview-config.component.html',
  styleUrl: './ide-preview-config.component.scss'
})
export class IdePreviewConfigComponent {
  appRunCommand = input<string>('');
  appPort = input<number>(0);

  appRunCommandChange = output<string>();
  appPortChange = output<number>();

  onCommandChange(value: string): void {
    this.appRunCommandChange.emit(value);
  }

  onPortChange(value: string | number | null): void {
    if (value === null || value === undefined || value === '') {
      this.appPortChange.emit(0);
      return;
    }
    const parsed = typeof value === 'number' ? value : Number.parseInt(value, 10);
    this.appPortChange.emit(Number.isFinite(parsed) ? parsed : 0);
  }
}
