import { Component, ChangeDetectionStrategy, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { LucideKeyboard, LucideX } from '@lucide/angular';
import { HotkeysService } from '@core/services/hotkeys.service';

@Component({
  selector: 'hotkeys-modal',
  standalone: true,
  imports: [CommonModule, LucideKeyboard, LucideX],
  templateUrl: './hotkeys-modal.component.html',
  styleUrl: './hotkeys-modal.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class HotkeysModalComponent {
  hotkeysService = inject(HotkeysService);
  isOpen = this.hotkeysService.isHelpModalOpen;
  hotkeysByCategory = this.hotkeysService.hotkeysByCategory;

  close(): void {
    this.hotkeysService.closeHelpModal();
  }
}
