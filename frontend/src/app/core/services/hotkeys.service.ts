import { Injectable, signal, computed } from '@angular/core';

export interface HotkeyConfig {
  key: string; // e.g. 'j', 'k', 'a', 'c', 'ctrl+enter', '?'
  description: string;
  category: 'SpeedGrader' | 'Navegación' | 'Acciones' | 'General';
  scope?: string; // e.g. 'speed-grader', 'global'
  allowInInput?: boolean;
  action: (event: KeyboardEvent) => void;
}

@Injectable({
  providedIn: 'root'
})
export class HotkeysService {
  private registeredHotkeys = signal<HotkeyConfig[]>([]);
  readonly isHelpModalOpen = signal<boolean>(false);
  readonly currentScope = signal<string>('global');

  readonly activeHotkeys = computed(() => {
    const scope = this.currentScope();
    return this.registeredHotkeys().filter(h => !h.scope || h.scope === 'global' || h.scope === scope);
  });

  readonly hotkeysByCategory = computed(() => {
    const map = new Map<string, HotkeyConfig[]>();
    for (const h of this.activeHotkeys()) {
      const list = map.get(h.category) || [];
      list.push(h);
      map.set(h.category, list);
    }
    return map;
  });

  constructor() {
    this.initGlobalListener();
    // Default help shortcut
    this.register({
      key: '?',
      description: 'Mostrar atajos de teclado',
      category: 'General',
      action: () => this.toggleHelpModal()
    });
  }

  setScope(scope: string): void {
    this.currentScope.set(scope);
  }

  register(config: HotkeyConfig): () => void {
    const normalizedKey = config.key.toLowerCase();
    const item: HotkeyConfig = { ...config, key: normalizedKey };
    
    this.registeredHotkeys.update(list => [...list, item]);

    return () => {
      this.registeredHotkeys.update(list => list.filter(h => h !== item));
    };
  }

  toggleHelpModal(): void {
    this.isHelpModalOpen.update(v => !v);
  }

  openHelpModal(): void {
    this.isHelpModalOpen.set(true);
  }

  closeHelpModal(): void {
    this.isHelpModalOpen.set(false);
  }

  private initGlobalListener(): void {
    window.addEventListener('keydown', (event: KeyboardEvent) => {
      const target = event.target as HTMLElement | null;
      const isInputFocused = target && (
        target.tagName === 'INPUT' ||
        target.tagName === 'TEXTAREA' ||
        target.tagName === 'SELECT' ||
        target.isContentEditable
      );

      const isCtrl = event.ctrlKey || event.metaKey;
      const keyName = event.key.toLowerCase();
      let combo = '';

      if (isCtrl && keyName === 'enter') {
        combo = 'ctrl+enter';
      } else if (event.key === '?' || (event.shiftKey && keyName === '/')) {
        combo = '?';
      } else if (!isCtrl && !event.altKey) {
        combo = keyName;
      }

      if (!combo) return;

      const matched = this.activeHotkeys().find(h => h.key === combo);
      if (!matched) return;

      // Allow shortcut inside inputs only if explicitly enabled (like Ctrl+Enter)
      if (isInputFocused && !matched.allowInInput && combo !== 'ctrl+enter' && combo !== 'escape') {
        return;
      }

      event.preventDefault();
      matched.action(event);
    });
  }
}
