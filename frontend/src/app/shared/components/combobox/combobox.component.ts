import { 
  Component, 
  input, 
  output, 
  signal, 
  computed, 
  ElementRef, 
  viewChild, 
  inject,
  DestroyRef
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
  selector: 'combobox',
  standalone: true,
  imports: [CommonModule, LucideSearch, LucideChevronDown, LucideCheck],
  template: `
    <div 
      class="combobox-wrapper" 
      [class.open]="isOpen()" 
      [class.disabled]="disabled()"
      [class.compact]="compact()"
      [class.no-search-icon]="!showSearchIcon()"
    >
      <div class="input-container">
        @if (showSearchIcon()) {
          <svg lucideSearch class="search-icon" aria-hidden="true"></svg>
        }
        <input
          #inputEl
          type="text"
          class="combobox-input"
          [placeholder]="placeholder()"
          [value]="displayValue()"
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
            @if (availableGroups().length > 1) {
              <div class="combobox-group-chips">
                <button 
                  type="button" 
                  class="group-chip"
                  [class.active]="selectedGroupFilter() === null"
                  (click)="selectGroupFilter(null, $event)"
                >
                  Todas ({{ totalOptionsCount() }})
                </button>
                @for (grp of availableGroups(); track grp.name) {
                  <button 
                    type="button" 
                    class="group-chip"
                    [class.active]="selectedGroupFilter() === grp.name"
                    (click)="selectGroupFilter(grp.name, $event)"
                  >
                    {{ grp.name }} ({{ grp.count }})
                  </button>
                }
              </div>
            }
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

              @if (hasMoreOptions()) {
                <div class="combobox-more-row">
                  <button 
                    type="button" 
                    class="btn-combobox-more"
                    (click)="showMoreOptions($event)"
                  >
                    Mostrar más ({{ remainingOptionsCount() }} restantes)
                  </button>
                </div>
              }
            }
          </div>
        </div>
      }
    </div>
  `,
  styleUrls: ['./combobox.component.scss']
})
export class ComboboxComponent {
  private readonly elementRef = inject(ElementRef);
  private readonly destroyRef = inject(DestroyRef);

  inputEl = viewChild<ElementRef<HTMLInputElement>>('inputEl');

  value = input<string>('');
  options = input<ComboboxOption[]>([]);
  placeholder = input<string>('Escriba para filtrar...');
  maxSuggestions = input<number>(8);
  totalAvailableCount = input<number | null>(null);
  disabled = input<boolean>(false);
  compact = input<boolean>(false);
  showSearchIcon = input<boolean>(true);
  inputAriaLabel = input<string>('Selector con búsqueda');
  suppressListbox = input<boolean>(false);

  valueChange = output<string>();
  optionSelected = output<ComboboxOption>();
  closed = output<void>();

  isOpen = signal<boolean>(false);
  searchQuery = signal<string | null>(null);
  activeFlatIndex = signal<number>(-1);
  selectedGroupFilter = signal<string | null>(null);
  visibleLimit = signal<number>(8);
  listboxId = 'combobox-listbox-' + Math.random().toString(36).substring(2, 9);

  displayValue = computed<string>(() => {
    const typed = this.searchQuery();
    if (typed !== null) return typed;
    return this.value();
  });

  effectiveQuery = computed<string>(() => {
    const typed = this.searchQuery();
    if (typed !== null) return typed;
    const val = this.value();
    const isExactOption = this.options().some(o => o.label === val);
    if (isExactOption) return '';
    return val;
  });

  constructor() {
    if (typeof document !== 'undefined') {
      const listener = (event: MouseEvent) => {
        if (!this.isOpen()) return;
        const target = event.target as Node;
        if (!this.elementRef.nativeElement.contains(target)) {
          this.close();
        }
      };
      document.addEventListener('click', listener, { capture: true });
      this.destroyRef.onDestroy(() => {
        document.removeEventListener('click', listener, { capture: true });
      });
    }
  }

  totalOptionsCount = computed<number>(() => {
    return this.totalAvailableCount() ?? this.options().length;
  });

  availableGroups = computed<{ name: string; count: number }[]>(() => {
    const all = this.options();
    const map = new Map<string, number>();
    for (const opt of all) {
      if (opt.group) {
        map.set(opt.group, (map.get(opt.group) || 0) + 1);
      }
    }
    if (map.size <= 1) return [];
    return Array.from(map.entries()).map(([name, count]) => ({ name, count }));
  });

  allMatchingOptions = computed<ComboboxOption[]>(() => {
    const query = this.effectiveQuery().trim().toLowerCase();
    const grpFilter = this.selectedGroupFilter();
    let list = this.options();

    if (grpFilter !== null) {
      list = list.filter(o => o.group === grpFilter);
    }

    if (query) {
      list = list.filter(o => 
        o.label.toLowerCase().includes(query) || 
        (o.meta && o.meta.toLowerCase().includes(query)) ||
        (o.description && o.description.toLowerCase().includes(query))
      );
    }
    return list;
  });

  filteredOptions = computed<ComboboxOption[]>(() => {
    const lim = this.visibleLimit();
    return this.allMatchingOptions().slice(0, lim);
  });

  hasMoreOptions = computed<boolean>(() => {
    return this.allMatchingOptions().length > this.visibleLimit();
  });

  remainingOptionsCount = computed<number>(() => {
    const total = this.allMatchingOptions().length;
    const current = this.visibleLimit();
    return Math.max(0, total - current);
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
    const total = this.allMatchingOptions().length;
    const current = this.filteredOptions().length;
    if (this.effectiveQuery().trim() || this.selectedGroupFilter() !== null) {
      return `${current} de ${total} mostradas; escriba para filtrar`;
    }
    return `${current} de ${total} disponibles; escriba para filtrar`;
  });

  activeDescendantId = computed<string | null>(() => {
    const idx = this.activeFlatIndex();
    const opts = this.flatVisibleOptions();
    if (idx >= 0 && idx < opts.length) {
      return 'opt-' + opts[idx].id;
    }
    return null;
  });

  showMoreOptions(event: MouseEvent): void {
    event.stopPropagation();
    event.preventDefault();
    this.visibleLimit.update(v => v + (this.maxSuggestions() || 8));
  }

  selectGroupFilter(groupName: string | null, event: MouseEvent): void {
    event.stopPropagation();
    event.preventDefault();
    this.selectedGroupFilter.set(groupName);
    this.visibleLimit.set(this.maxSuggestions() || 8);
    this.activeFlatIndex.set(-1);
  }

  onInputChange(event: Event): void {
    const val = (event.target as HTMLInputElement).value;
    this.searchQuery.set(val);
    this.valueChange.emit(val);
    if (!this.isOpen()) {
      this.isOpen.set(true);
    }
    this.visibleLimit.set(this.maxSuggestions() || 8);
    this.activeFlatIndex.set(-1);
  }

  onInputFocus(): void {
    if (!this.disabled() && !this.suppressListbox()) {
      this.isOpen.set(true);
      this.inputEl()?.nativeElement.select();
    }
  }

  toggleDropdown(): void {
    if (this.disabled()) return;
    if (this.isOpen()) {
      this.close();
    } else {
      this.isOpen.set(true);
      const el = this.inputEl()?.nativeElement;
      el?.focus();
      el?.select();
    }
  }

  close(): void {
    this.isOpen.set(false);
    this.searchQuery.set(null);
    this.activeFlatIndex.set(-1);
    this.closed.emit();
  }

  selectOption(option: ComboboxOption): void {
    this.searchQuery.set(null);
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
        } else if (opts.length > 0) {
          const next = (this.activeFlatIndex() + 1) % opts.length;
          this.activeFlatIndex.set(next);
        }
        break;

      case 'ArrowUp':
        event.preventDefault();
        if (!this.isOpen()) {
          this.isOpen.set(true);
          this.activeFlatIndex.set(opts.length - 1);
        } else if (opts.length > 0) {
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
}

export { ComboboxComponent as SolvComboboxComponent };
