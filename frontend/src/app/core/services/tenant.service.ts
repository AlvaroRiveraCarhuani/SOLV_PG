import { Injectable, inject, signal, computed } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { TenantConfig, TenantBrandingHSL } from '@core/models/tenant.model';

@Injectable({
  providedIn: 'root'
})
export class TenantService {
  private readonly http = inject(HttpClient);

  // Signals reactivos para el estado del Tenant
  readonly config = signal<TenantConfig | null>(null);
  readonly loading = signal<boolean>(true);
  readonly error = signal<string | null>(null);

  // Iniciales institucionales para el shield cuando no hay logo SVG/PNG
  readonly tenantInitials = computed<string>(() => {
    const name = this.config()?.institution_name || 'SL';
    const parts = name.trim().split(/\s+/);
    if (parts.length >= 2) {
      return (parts[0][0] + parts[1][0]).toUpperCase();
    }
    return name.slice(0, 2).toUpperCase();
  });

  // Clave para caché local instantánea (Stale-While-Revalidate)
  private readonly CACHE_KEY_PREFIX = 'solv_tenant_config_';

  /**
   * Inicializador de aplicación (APP_INITIALIZER).
   * Se ejecuta antes de que Angular renderice el primer frame,
   * eliminando el FOUC (Flash of Unstyled Content).
   */
  async init(): Promise<void> {
    const slug = this.resolveCurrentSlug();
    const cacheKey = `${this.CACHE_KEY_PREFIX}${slug}`;

    // 1. Carga inmediata desde caché local para 0ms de parpadeo
    const cached = this.loadFromStorage(cacheKey);
    if (cached) {
      this.config.set(cached);
      this.applyBranding(cached);
    }

    // 2. Revalidación contra el backend
    try {
      const url = slug ? `/api/v1/config/public?slug=${encodeURIComponent(slug)}` : '/api/v1/config/public';
      const remoteConfig = await firstValueFrom(this.http.get<TenantConfig>(url));

      this.config.set(remoteConfig);
      this.applyBranding(remoteConfig);
      this.saveToStorage(cacheKey, remoteConfig);
      this.error.set(null);
    } catch (err: any) {
      // Si la red falla pero teníamos caché, se mantiene el branding existente
      if (!this.config()) {
        const fallbackConfig: TenantConfig = {
          tenant_id: '00000000-0000-0000-0000-000000000000',
          slug: slug || 'default',
          institution_name: 'Plataforma SOLV',
          logo_url: '',
          tenant_primary_color: '#2563EB',
          support_email: 'soporte@solv.edu.bo'
        };
        this.config.set(fallbackConfig);
        this.applyBranding(fallbackConfig);
        this.error.set('No se pudo contactar el servidor institucional. Usando configuración base.');
      }
    } finally {
      this.loading.set(false);
    }
  }

  /**
   * Resuelve el slug institucional a partir del subdominio del host.
   * Soporta entornos locales de desarrollo mediante parámetro '?tenant=...' o '?slug=...'.
   */
  resolveCurrentSlug(): string {
    // 1. Parámetro explícito de query en desarrollo local (ej. localhost:4200/?tenant=umsa)
    if (typeof window !== 'undefined' && window.location) {
      const params = new URLSearchParams(window.location.search);
      const queryTenant = params.get('tenant') || params.get('slug');
      if (queryTenant) {
        return queryTenant.trim().toLowerCase();
      }

      // 2. Extracción de subdominio (ej. "uab" de "uab.solv.uab.edu.bo" o "umsa.solv.umsa.edu.bo")
      const hostname = window.location.hostname;
      if (hostname && hostname !== 'localhost' && hostname !== '127.0.0.1') {
        const parts = hostname.split('.');
        if (parts.length >= 2) {
          return parts[0].toLowerCase();
        }
      }
    }

    return ''; // Sin slug forzado: consulta la configuración raíz del backend/host
  }

  /**
   * Aplica los tokens de diseño al DOM mediante CSS Custom Properties
   * con cálculo matemático de HSL y contraste WCAG 2.1 AA.
   */
  applyBranding(config: TenantConfig): void {
    if (typeof document === 'undefined') return;

    const root = document.documentElement;
    const hex = config.tenant_primary_color || '#2563EB';
    const hsl = this.hexToHSL(hex);

    // 1. Inyectar canales HSL en el :root
    root.style.setProperty('--tenant-primary-h', hsl.h.toString());
    root.style.setProperty('--tenant-primary-s', hsl.s);
    root.style.setProperty('--tenant-primary-l', hsl.l);

    // 2. Determinar contraste óptimo (WCAG 2.1 AA) para el texto sobre el botón primario
    const lightnessNum = parseInt(hsl.l.replace('%', ''), 10);
    const contrastTextColor = lightnessNum > 60 ? '#0F172A' : '#FFFFFF';
    root.style.setProperty('--tenant-primary-text', contrastTextColor);

    // 3. Título de pestaña del navegador
    if (config.institution_name) {
      document.title = `${config.institution_name} — SOLV`;
    }
  }

  /**
   * Convierte un color HEX (#RRGGBB) a canales HSL matemáticos puros.
   * Si el hex es inválido, cae elegantemente a HSL(221, 83%, 53%) (Azul SOLV).
   */
  hexToHSL(hex: string): TenantBrandingHSL {
    const defaultHSL: TenantBrandingHSL = { h: 221, s: '83%', l: '53%' };

    if (!hex || !/^#([A-Fa-f0-9]{6})$/.test(hex)) {
      return defaultHSL;
    }

    let r = parseInt(hex.substring(1, 3), 16) / 255;
    let g = parseInt(hex.substring(3, 5), 16) / 255;
    let b = parseInt(hex.substring(5, 7), 16) / 255;

    const max = Math.max(r, g, b);
    const min = Math.min(r, g, b);
    let h = 0;
    let s = 0;
    let l = (max + min) / 2;

    if (max !== min) {
      const d = max - min;
      s = l > 0.5 ? d / (2 - max - min) : d / (max + min);

      switch (max) {
        case r: h = (g - b) / d + (g < b ? 6 : 0); break;
        case g: h = (b - r) / d + 2; break;
        case b: h = (r - g) / d + 4; break;
      }
      h /= 6;
    }

    return {
      h: Math.round(h * 360),
      s: `${Math.round(s * 100)}%`,
      l: `${Math.round(l * 100)}%`
    };
  }

  private loadFromStorage(key: string): TenantConfig | null {
    try {
      if (typeof window !== 'undefined' && window.sessionStorage) {
        const raw = sessionStorage.getItem(key);
        return raw ? JSON.parse(raw) : null;
      }
    } catch {
      // Ignorar restricciones de cookies/storage en navegadores privados
    }
    return null;
  }

  private saveToStorage(key: string, data: TenantConfig): void {
    try {
      if (typeof window !== 'undefined' && window.sessionStorage) {
        sessionStorage.setItem(key, JSON.stringify(data));
      }
    } catch {
      // Ignorar excepciones de cuota de storage
    }
  }
}
