import { 
  Component, 
  input, 
  output, 
  signal, 
  computed, 
  ElementRef, 
  viewChild, 
  HostListener, 
  effect 
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { LucideSearch, LucideChevronDown, LucideCheck } from '@lucide/angular';

export interface ComboboxOption<T = any> {
  id: string;
  label: string;
  value: T;
  group?: string;
  badge?: string;
  badgeVariant?: 'local' | 'official' | 'default';
  meta?: string;
  description?: string;
}

export interface ComboboxGroup {
  name: string;
  options: ComboboxOption[];
}

@Component({
  selector: 'solv-combobox',
  standalone: true,
  imports: [CommonModule, LucideSearch, LucideChevronDown, LucideCheck],
  template: `
    <div 
      class="combobox-wrapper" 
      [class.open]="isOpen()" 
      [class.disabled]="disabled()"
    >
      <div class="input-container">
        <svg lucideSearch class="search-icon" aria-hidden="true"></svg>
        <input
          #inputEl
          type="text"
          class="combobox-input"
          [placeholder]="placeholder()"
          [value]="value()"
          [disabled]="disabled()"
          role="combobox"
          [attr.aria-expanded]="isOpen()"
          [attr.aria-haspopup]="'listbox'"
          [attr.aria-controls]="listboxId"
          [attr.aria-label]="inputAriaLabel()"
          [attr.aria-activedescendant]="activeDescendantId()"
          (input)="onInputChange($event)"
          (focus)="onInputFocus()"
          (keydown)="onKeydown($event)"
        />
        <button 
          type="button" 
          class="toggle-btn" 
          tabindex="-1"
          (click)="toggleDropdown()"
          [attr.aria-label]="isOpen() ? 'Cerrar opciones' : 'Abrir opciones'"
        >
          <svg lucideChevronDown class="chevron-icon" [class.rotated]="isOpen()"></svg>
        </button>
      </div>

      @if (isOpen() && !suppressListbox()) {
        <div 
          class="combobox-overlay" 
          [id]="listboxId" 
          role="listbox"
          tabindex="-1"
        >
          <div class="combobox-header">
            <span class="header-count">{{ headerCountText() }}</span>
          </div>

          <div class="options-scroll-container">
            @if (groupedOptions().length === 0) {
              <div class="no-options-message">
                No se encontraron opciones
              </div>
            } @else {
              @for (grp of groupedOptions(); track grp.name) {
                @if (grp.name) {
                  <div class="group-header" role="presentation">
                    {{ grp.name }}
                  </div>
                }
                @for (opt of grp.options; track opt.id) {
                  <div
                    class="option-item"
                    [id]="'opt-' + opt.id"
                    role="option"
                    [attr.aria-selected]="isOptionSelected(opt)"
                    [class.active]="isOptionActive(opt)"
                    [class.selected]="isOptionSelected(opt)"
                    (click)="selectOption(opt)"
                    (mouseenter)="setActiveOption(opt)"
                  >
                    <div class="option-content">
                      <span class="option-label">{{ opt.label }}</span>
                      @if (opt.badge) {
                        <span 
                          class="option-badge" 
                          [class]="'badge-' + (opt.badgeVariant || 'default')"
                        >
                          {{ opt.badge }}
                        </span>
                      }
                      @if (opt.meta) {
                        <span class="option-meta">{{ opt.meta }}</span>
                      }
                    </div>

                    @if (opt.description) {
                      <div class="option-description">{{ opt.description }}</div>
                    }

                    @if (isOptionSelected(opt)) {
                      <svg lucideCheck class="selected-check" aria-hidden="true"></svg>
                    }
                  </div>
                }
              }
            }
          </div>
        </div>
      }
    </div>
  `,
  styleUrls: ['./combobox.component.scss']
})
export class SolvComboboxComponent {
  inputEl = viewChild<ElementRef<HTMLInputElement>>('inputEl');

  value = input<string>('');
  options = input<ComboboxOption[]>([]);
  placeholder = input<string>('Escriba para filtrar...');
  maxSuggestions = input<number>(8);
  totalAvailableCount = input<number | null>(null);
  disabled = input<boolean>(false);
  inputAriaLabel = input<string>('Selector con búsqueda');
  suppressListbox = input<boolean>(false);

  valueChange = output<string>();
  optionSelected = output<ComboboxOption>();
  closed = output<void>();

  isOpen = signal<boolean>(false);
  activeFlatIndex = signal<number>(-1);
  listboxId = 'combobox-listbox-' + Math.random().toString(36).substring(2, 9);

  // Filtra y acota a un máximo de elementos
  filteredOptions = computed<ComboboxOption[]>(() => {
    const query = this.value().trim().toLowerCase();
    const all = this.options();
    const filtered = query 
      ? all.filter(o => 
          o.label.toLowerCase().includes(query) || 
          (o.meta && o.meta.toLowerCase().includes(query)) ||
          (o.description && o.description.toLowerCase().includes(query))
        )
      : all;
    return filtered.slice(0, this.maxSuggestions());
  });

  groupedOptions = computed<ComboboxGroup[]>(() => {
    const list = this.filteredOptions();
    const groupsMap = new Map<string, ComboboxOption[]>();

    for (const opt of list) {
      const gName = opt.group || '';
      if (!groupsMap.has(gName)) {
        groupsMap.set(gName, []);
      }
      groupsMap.get(gName)!.push(opt);
    }

    const result: ComboboxGroup[] = [];
    for (const [name, opts] of groupsMap.entries()) {
      result.push({ name, options: opts });
    }
    return result;
  });

  flatVisibleOptions = computed<ComboboxOption[]>(() => {
    return this.filteredOptions();
  });

  headerCountText = computed<string>(() => {
    const total = this.totalAvailableCount() ?? this.options().length;
    const current = this.filteredOptions().length;
    if (this.value().trim()) {
      return `${current} de ${total} disponibles; escriba para filtrar`;
    }
    return `${total} disponibles; escriba para filtrar`;
  });

  activeDescendantId = computed<string | null>(() => {
    const idx = this.activeFlatIndex();
    const opts = this.flatVisibleOptions();
    if (idx >= 0 && idx < opts.length) {
      return 'opt-' + opts[idx].id;
    }
    return null;
  });

  onInputChange(event: Event): void {
    const val = (event.target as HTMLInputElement).value;
    this.valueChange.emit(val);
    if (!this.isOpen()) {
      this.isOpen.set(true);
    }
    this.activeFlatIndex.set(-1);
  }

  onInputFocus(): void {
    if (!this.disabled() && !this.suppressListbox()) {
      this.isOpen.set(true);
    }
  }

  toggleDropdown(): void {
    if (this.disabled()) return;
    if (this.isOpen()) {
      this.close();
    } else {
      this.isOpen.set(true);
      this.inputEl()?.nativeElement.focus();
    }
  }

  close(): void {
    this.isOpen.set(false);
    this.activeFlatIndex.set(-1);
    this.closed.emit();
  }

  selectOption(option: ComboboxOption): void {
    this.optionSelected.emit(option);
    this.close();
  }

  isOptionSelected(option: ComboboxOption): boolean {
    return this.value() === option.label;
  }

  isOptionActive(option: ComboboxOption): boolean {
    const active = this.activeDescendantId();
    return active === 'opt-' + option.id;
  }

  setActiveOption(option: ComboboxOption): void {
    const idx = this.flatVisibleOptions().findIndex(o => o.id === option.id);
    if (idx !== -1) {
      this.activeFlatIndex.set(idx);
    }
  }

  onKeydown(event: KeyboardEvent): void {
    const opts = this.flatVisibleOptions();

    switch (event.key) {
      case 'ArrowDown':
        event.preventDefault();
        if (!this.isOpen()) {
          this.isOpen.set(true);
          this.activeFlatIndex.set(0);
        } else {
          const next = (this.activeFlatIndex() + 1) % opts.length;
          this.activeFlatIndex.set(next);
        }
        break;

      case 'ArrowUp':
        event.preventDefault();
        if (!this.isOpen()) {
          this.isOpen.set(true);
          this.activeFlatIndex.set(opts.length - 1);
        } else {
          const prev = (this.activeFlatIndex() - 1 + opts.length) % opts.length;
          this.activeFlatIndex.set(prev);
        }
        break;

      case 'Enter':
        if (this.isOpen() && this.activeFlatIndex() >= 0 && this.activeFlatIndex() < opts.length) {
          event.preventDefault();
          this.selectOption(opts[this.activeFlatIndex()]);
        }
        break;

      case 'Escape':
        if (this.isOpen()) {
          event.preventDefault();
          event.stopPropagation();
          this.close();
        }
        break;

      case 'Tab':
        if (this.isOpen()) {
          this.close();
        }
        break;
    }
  }

  @HostListener('document:click', ['$event'])
  onDocumentClick(event: MouseEvent): void {
    if (!this.inputEl()) return;
    const clickedInside = (event.target as HTMLElement).closest('.combobox-wrapper');
    if (!clickedInside && this.isOpen()) {
      this.close();
    }
  }
}
