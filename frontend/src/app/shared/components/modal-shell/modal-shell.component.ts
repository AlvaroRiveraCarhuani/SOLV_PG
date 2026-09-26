import { Component, ChangeDetectionStrategy, input, output, HostListener } from '@angular/core';
import { CommonModule } from '@angular/common';
import { LucideX } from '@lucide/angular';

export type ModalIntent = 'info' | 'success' | 'warning' | 'error';

@Component({
  selector: 'modal-shell',
  standalone: true,
  imports: [CommonModule, LucideX],
  templateUrl: './modal-shell.component.html',
  styleUrl: './modal-shell.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
  // El atributo estatico title="..." quedaria en el host y el navegador
  // mostraria un tooltip nativo con el titulo. Se elimina del DOM.
  host: { '[attr.title]': 'null' }
})
export class ModalShellComponent {
  readonly title = input.required<string>();
  readonly subtitle = input<string>();
  readonly intent = input<ModalIntent>('info');
  readonly maxWidth = input<string>('540px');
  readonly showClose = input<boolean>(true);
  readonly showIcon = input<boolean>(true);
  readonly disableClose = input<boolean>(false);

  readonly close = output<void>();

  @HostListener('document:keydown.escape')
  handleEscape(): void {
    if (!this.disableClose()) {
      this.close.emit();
    }
  }

  onBackdrop(): void {
    if (!this.disableClose()) {
      this.close.emit();
    }
  }
}
