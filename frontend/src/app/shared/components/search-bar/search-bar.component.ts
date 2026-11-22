import {
  Component,
  DestroyRef,
  ElementRef,
  HostListener,
  inject,
  input,
  output,
  viewChild
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { LucideSearch, LucideX } from '@lucide/angular';

@Component({
  selector: 'search-bar',
  standalone: true,
  imports: [CommonModule, LucideSearch, LucideX],
  templateUrl: './search-bar.component.html',
  styleUrls: ['./search-bar.component.scss']
})
export class SearchBarComponent {
  private readonly destroyRef = inject(DestroyRef);
  private readonly host = inject(ElementRef) as ElementRef<HTMLElement>;
  private debounceTimer: ReturnType<typeof setTimeout> | null = null;

  inputEl = viewChild<ElementRef<HTMLInputElement>>('inputEl');

  query = input<string>('');
  placeholder = input<string>('Buscar...');
  debounceMs = input<number>(0);
  compact = input<boolean>(false);
  ariaLabel = input<string>('Buscar en la lista');
  /** Tecla que enfoca la barra. `null` desactiva el atajo en instancias secundarias. */
  shortcut = input<string | null>('/');

  queryChange = output<string>();
  cleared = output<void>();

  constructor() {
    this.destroyRef.onDestroy(() => {
      if (this.debounceTimer) {
        clearTimeout(this.debounceTimer);
      }
    });
  }

  /**
   * Solo la primera barra de la página atiende el atajo global: así una vista
   * con búsqueda principal y una secundaria no compiten por el mismo foco.
   */
  @HostListener('document:keydown', ['$event'])
  onDocumentKeydown(event: KeyboardEvent): void {
    const shortcut = this.shortcut();
    if (!shortcut || event.defaultPrevented || event.ctrlKey || event.metaKey || event.altKey) {
      return;
    }
    if (event.key !== shortcut) {
      return;
    }

    const target = event.target as HTMLElement | null;
    if (target && (target.isContentEditable || ['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName))) {
      return;
    }

    if (typeof document !== 'undefined' && document.querySelectorAll('search-bar')[0] !== this.host.nativeElement) {
      return;
    }

    event.preventDefault();
    this.focus();
  }

  focus(): void {
    const el = this.inputEl()?.nativeElement;
    if (!el) {
      return;
    }
    el.focus();
    el.select();
  }

  onInput(event: Event): void {
    const val = (event.target as HTMLInputElement).value;
    const ms = this.debounceMs();

    if (ms > 0) {
      if (this.debounceTimer) {
        clearTimeout(this.debounceTimer);
      }
      this.debounceTimer = setTimeout(() => {
        this.queryChange.emit(val);
      }, ms);
    } else {
      this.queryChange.emit(val);
    }
  }

  clear(): void {
    if (this.debounceTimer) {
      clearTimeout(this.debounceTimer);
    }
    const el = this.inputEl()?.nativeElement;
    if (el) {
      el.value = '';
      el.focus();
    }
    this.queryChange.emit('');
    this.cleared.emit();
  }
}
