import { 
  Component, 
  input, 
  output, 
  viewChild, 
  ElementRef, 
  DestroyRef, 
  inject 
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
  private debounceTimer: ReturnType<typeof setTimeout> | null = null;

  inputEl = viewChild<ElementRef<HTMLInputElement>>('inputEl');

  query = input<string>('');
  placeholder = input<string>('Buscar...');
  debounceMs = input<number>(0);
  compact = input<boolean>(true);
  ariaLabel = input<string>('Buscar en la lista');

  queryChange = output<string>();
  cleared = output<void>();

  constructor() {
    this.destroyRef.onDestroy(() => {
      if (this.debounceTimer) {
        clearTimeout(this.debounceTimer);
      }
    });
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
