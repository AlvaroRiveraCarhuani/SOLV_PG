import { Injectable, inject, signal, effect } from '@angular/core';
import { TenantService } from './tenant.service';
import {
  CURATED_FONTS,
  CuratedFont,
  findCuratedFont,
  curatedFontCSSUrl,
  DEFAULT_SANS_SLUG,
  DEFAULT_MONO_SLUG
} from '@shared/curated-fonts';

export interface UserTypographyPreference {
  sansSlug: string | null; // null = use institutional default
  monoSlug: string | null; // null = use institutional default
}

@Injectable({
  providedIn: 'root'
})
export class UserTypographyService {
  private tenantService = inject(TenantService);
  private readonly storageKey = 'solv_user_typography_pref';

  readonly sansPreference = signal<string | null>(null);
  readonly monoPreference = signal<string | null>(null);

  readonly sansOptions: CuratedFont[] = CURATED_FONTS.filter(f => f.kind === 'sans');
  readonly monoOptions: CuratedFont[] = CURATED_FONTS.filter(f => f.kind === 'mono');

  constructor() {
    this.loadFromStorage();

    // Re-apply whenever preference signals change
    effect(() => {
      const sans = this.sansPreference();
      const mono = this.monoPreference();
      this.applyTypography(sans, mono);
    });
  }

  setSansPreference(slug: string | null): void {
    this.sansPreference.set(slug);
    this.saveToStorage();
    this.applyTypography(slug, this.monoPreference());
  }

  setMonoPreference(slug: string | null): void {
    this.monoPreference.set(slug);
    this.saveToStorage();
    this.applyTypography(this.sansPreference(), slug);
  }

  resetToInstitutional(): void {
    this.sansPreference.set(null);
    this.monoPreference.set(null);
    this.saveToStorage();
    this.applyTypography(null, null);
  }

  applyTypography(sansSlug: string | null, monoSlug: string | null): void {
    if (typeof document === 'undefined') return;

    const tenantConfig = this.tenantService.config();

    // Resolve effective sans
    const effectiveSansSlug = sansSlug || this.extractSlug(tenantConfig?.font_sans_family) || DEFAULT_SANS_SLUG;
    const effectiveMonoSlug = monoSlug || this.extractSlug(tenantConfig?.font_mono_family) || DEFAULT_MONO_SLUG;

    const sansFont = findCuratedFont(effectiveSansSlug);
    const monoFont = findCuratedFont(effectiveMonoSlug);

    const cssUrls = new Set<string>();
    if (sansFont) cssUrls.add(curatedFontCSSUrl(sansFont));
    if (monoFont) cssUrls.add(curatedFontCSSUrl(monoFont));

    for (const url of cssUrls) {
      this.ensureStylesheet(url);
    }

    const sansFamily = sansFont?.name ?? 'Inter';
    const monoFamily = monoFont?.name ?? 'JetBrains Mono';

    const root = document.documentElement;
    root.style.setProperty('--font-sans', `'${sansFamily}', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif`);
    root.style.setProperty('--font-mono', `'${monoFamily}', 'Fira Code', Menlo, Monaco, Consolas, monospace`);
  }

  private extractSlug(value?: string): string | null {
    if (!value) return null;
    if (value.startsWith('cat:')) return value.slice(4);
    return null;
  }

  private ensureStylesheet(url: string): void {
    const linkId = `user-font-${this.hashString(url)}`;
    if (document.getElementById(linkId)) return;

    const link = document.createElement('link');
    link.id = linkId;
    link.rel = 'stylesheet';
    link.href = url;
    document.head.appendChild(link);
  }

  private hashString(value: string): string {
    let hash = 0;
    for (let i = 0; i < value.length; i++) {
      hash = (hash * 31 + value.charCodeAt(i)) | 0;
    }
    return Math.abs(hash).toString(36);
  }

  private loadFromStorage(): void {
    try {
      const raw = localStorage.getItem(this.storageKey);
      if (raw) {
        const parsed: UserTypographyPreference = JSON.parse(raw);
        if (parsed) {
          this.sansPreference.set(parsed.sansSlug ?? null);
          this.monoPreference.set(parsed.monoSlug ?? null);
        }
      }
    } catch {
      // Ignore in non-browser/restricted storage
    }
  }

  private saveToStorage(): void {
    try {
      const pref: UserTypographyPreference = {
        sansSlug: this.sansPreference(),
        monoSlug: this.monoPreference()
      };
      localStorage.setItem(this.storageKey, JSON.stringify(pref));
    } catch {
      // Ignore storage errors
    }
  }
}
