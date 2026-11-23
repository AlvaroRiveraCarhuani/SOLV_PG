import { Injectable, signal, computed, effect } from '@angular/core';

export type ThemeMode = 'light' | 'dark' | 'system';

@Injectable({
  providedIn: 'root'
})
export class ThemeService {
  private readonly storageKey = 'solv_theme_preference';

  readonly preference = signal<ThemeMode>('system');

  private mediaQuery: MediaQueryList | null =
    typeof window !== 'undefined' && typeof window.matchMedia === 'function'
      ? window.matchMedia('(prefers-color-scheme: dark)')
      : null;

  private systemIsDark = signal<boolean>(
    this.mediaQuery ? this.mediaQuery.matches : false
  );

  readonly isDark = computed<boolean>(() => {
    const pref = this.preference();
    if (pref === 'dark') return true;
    if (pref === 'light') return false;
    return this.systemIsDark();
  });

  constructor() {
    this.loadFromStorage();
    this.applyTheme(this.isDark());

    if (this.mediaQuery) {
      this.mediaQuery.addEventListener('change', (e) => {
        this.systemIsDark.set(e.matches);
        if (this.preference() === 'system') {
          this.applyTheme(this.isDark());
        }
      });
    }

    effect(() => {
      const dark = this.isDark();
      this.applyTheme(dark);
    });
  }

  toggleTheme(): void {
    const next: ThemeMode = this.isDark() ? 'light' : 'dark';
    this.setTheme(next);
  }

  setTheme(theme: ThemeMode): void {
    this.preference.set(theme);
    if (typeof localStorage !== 'undefined') {
      try {
        localStorage.setItem(this.storageKey, theme);
      } catch {
        // Ignorar errores de acceso a almacenamiento local
      }
    }
    this.applyTheme(this.isDark());
  }

  private loadFromStorage(): void {
    if (typeof localStorage !== 'undefined') {
      try {
        const stored = localStorage.getItem(this.storageKey) as ThemeMode | null;
        if (stored === 'light' || stored === 'dark' || stored === 'system') {
          this.preference.set(stored);
        }
      } catch {
        // Ignorar
      }
    }
  }

  private applyTheme(isDark: boolean): void {
    if (typeof document === 'undefined') return;
    const root = document.documentElement;
    if (isDark) {
      root.setAttribute('data-theme', 'dark');
      root.style.colorScheme = 'dark';
    } else {
      root.removeAttribute('data-theme');
      root.style.colorScheme = 'light';
    }
  }
}
