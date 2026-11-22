import {
  ChangeDetectionStrategy,
  Component,
  ElementRef,
  HostListener,
  contentChild,
  effect,
  input,
  output,
  viewChild
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { LucideX } from '@lucide/angular';

@Component({
  selector: 'drawer',
  standalone: true,
  imports: [CommonModule, LucideX],
  templateUrl: './drawer.component.html',
  styleUrl: './drawer.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class DrawerComponent {
  readonly open = input<boolean>(false);
  readonly title = input.required<string>();
  readonly subtitle = input<string>('');
  readonly width = input<string>('380px');
  readonly closeOnBackdrop = input<boolean>(true);
  readonly closeLabel = input<string>('Cerrar panel');
  readonly closed = output<void>();

  private readonly panel = viewChild<ElementRef<HTMLElement>>('panel');
  private wasOpen = false;

  /** El pie solo se renderiza cuando el consumidor proyecta contenido para él. */
  readonly footer = contentChild<ElementRef>('[drawer-footer]');

  constructor() {
    effect(() => {
      const open = this.open();
      if (open && !this.wasOpen) {
        setTimeout(() => this.panel()?.nativeElement.focus(), 0);
      }
      this.wasOpen = open;
    });
  }

  close(): void {
    this.closed.emit();
  }

  onBackdrop(): void {
    if (this.closeOnBackdrop()) {
      this.close();
    }
  }

  @HostListener('document:keydown.escape', ['$event'])
  onKeydown(event: Event): void {
    if (this.open()) {
      event.stopPropagation();
      this.close();
    }
  }
}
