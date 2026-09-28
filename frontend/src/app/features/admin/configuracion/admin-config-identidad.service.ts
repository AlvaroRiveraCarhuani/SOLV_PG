import { Injectable, inject, signal } from '@angular/core';
import { HttpClient, HttpErrorResponse } from '@angular/common/http';
import { Observable, map, tap } from 'rxjs';
import { TenantService } from '@core/services/tenant.service';
import { TenantConfig } from '@core/models/tenant.model';

export interface BrandingPayload {
  institution_name: string;
  logo_url: string;
  tenant_primary_color: string;
  support_email: string;
}

export interface BrandingValidation {
  valid: boolean;
  nameError?: string;
  colorError?: string;
  logoError?: string;
  emailError?: string;
  fontSansError?: string;
  fontMonoError?: string;
}

/** Ranura inline donde se muestra un rechazo tipográfico del backend. */
export type FontErrorSlot = 'sans' | 'mono' | 'general';

export interface FontErrorMapping {
  slot: FontErrorSlot;
  message: string;
}

/**
 * Merge puro del payload de branding sobre el config vigente (contrato PUT
 * /admin/branding: campos vacíos no pisan). Preserva las fuentes guardadas
 * salvo que el payload traiga valores tipográficos explícitos.
 */
export function mergeBrandingIntoConfig(
  cfg: TenantConfig,
  payload: BrandingPayload & { font_sans_family?: string; font_mono_family?: string }
): TenantConfig {
  return {
    ...cfg,
    institution_name: payload.institution_name || cfg.institution_name,
    logo_url: payload.logo_url || cfg.logo_url,
    tenant_primary_color: payload.tenant_primary_color || cfg.tenant_primary_color,
    support_email: payload.support_email || cfg.support_email,
    ...(payload.font_sans_family ? { font_sans_family: payload.font_sans_family } : {}),
    ...(payload.font_mono_family ? { font_mono_family: payload.font_mono_family } : {})
  };
}
/**
 * Traduce códigos 422 de tipografía a ranura + mensaje inline.
 * Sin contexto de ranura en la respuesta, todo mapea a `general` y el
 * componente lo muestra en el banner tipográfico con copia parcial.
 */
export function mapFontErrorCode(code: string): FontErrorMapping {
  switch (code) {
    case 'font_slug_unknown':
      return { slot: 'general', message: 'La fuente elegida no pertenece al catálogo curado.' };
    case 'font_kind_mismatch':
      return { slot: 'general', message: 'La fuente elegida no corresponde a ese uso (interfaz o datos de máquina).' };
    case 'font_format_invalid':
      return { slot: 'general', message: 'Valor de fuente inválido: usa el catálogo o una URL https:// válida.' };
    case 'font_url_host_not_allowed':
      return { slot: 'general', message: 'Solo se permiten hojas CSS de fonts.googleapis.com.' };
    case 'font_url_unreachable':
      return { slot: 'general', message: 'No se pudo alcanzar la hoja CSS de la fuente. Verifica la URL e intenta nuevamente.' };
    default:
      return { slot: 'general', message: 'La tipografía fue rechazada por el servidor. Revisa las fuentes elegidas.' };
  }
}

/**
 * Persistencia de identidad institucional (white-label).
 *
 * Contrato backend: PUT /api/v1/admin/branding hace MERGE PARCIAL: los campos
 * enviados vacios NO sobreescriben el valor existente. La propagacion a otros
 * usuarios va por GET /api/v1/config/public con caché servidor de 5 min; para
 * la sesion del admin que guarda, el branding se aplica localmente en el acto
 * via TenantService.applyBranding (misma vía que el anti-FOUC del initializer).
 */
@Injectable({ providedIn: 'root' })
export class AdminConfigIdentidadService {
  private readonly http = inject(HttpClient);
  private readonly tenantService = inject(TenantService);

  readonly isSaving = signal(false);
  readonly lastSavedAt = signal<Date | null>(null);

  currentConfig(): TenantConfig | null {
    return this.tenantService.config();
  }

  /** Valores vigentes para inicializar el formulario. */
  initialValues(): BrandingPayload {
    const cfg = this.currentConfig();
    return {
      institution_name: cfg?.institution_name ?? '',
      logo_url: cfg?.logo_url ?? '',
      tenant_primary_color: cfg?.tenant_primary_color ?? '#2563EB',
      support_email: cfg?.support_email ?? ''
    };
  }

  /**
   * Subida real del imagotipo (multipart) a POST /api/v1/tenants/logo.
   * El backend persiste logo_url en tenants.config y sirve el archivo por
   * /api/v1/public/branding/logo/{tenantId}{ext} con caché de 5 min.
   */
  uploadLogo(file: File): Observable<{ logo_url: string; file: string }> {
    this.isSaving.set(true);
    const form = new FormData();
    form.append('logo', file, file.name);
    // El backend responde con el sobre GlobalResponse { data, error, message };
    // se desenvuelve data con fallback al objeto plano.
    return this.http.post<{ logo_url: string; file: string } | { data: { logo_url: string; file: string } }>('/api/v1/tenants/logo', form).pipe(
      map((res) => ('data' in res && res.data ? res.data : (res as { logo_url: string; file: string }))),
      tap({
        next: (res) => {
          this.isSaving.set(false);
          const cfg = this.currentConfig();
          if (cfg) {
            // Cache-buster para que topbar/preview muestren el archivo nuevo
            // sin F5 (el backend sirve la misma URL con Cache-Control 5 min).
            const sep = res.logo_url.includes('?') ? '&' : '?';
            const busted = `${res.logo_url}${sep}t=${Date.now()}`;
            const updated: TenantConfig = { ...cfg, logo_url: busted };
            this.tenantService.config.set(updated);
            this.tenantService.applyBranding(updated);
          }
        },
        error: () => this.isSaving.set(false)
      })
    );
  }

  saveBranding(payload: BrandingPayload & { font_sans_family?: string; font_mono_family?: string }): Observable<unknown> {
    this.isSaving.set(true);
    return this.http.put('/api/v1/admin/branding', payload).pipe(
      tap({
        next: () => {
          this.isSaving.set(false);
          this.lastSavedAt.set(new Date());
          // Aplicación inmediata en esta sesión (el resto de usuarios la recibe
          // cuando expira la caché de /config/public, max 5 min).
          const cfg = this.currentConfig();
          if (cfg) {
            const updated = mergeBrandingIntoConfig(cfg, payload);
            this.tenantService.config.set(updated);
            this.tenantService.applyBranding(updated);
          }
        },
        error: () => this.isSaving.set(false)
      })
    );
  }

  /** Traduce errores HTTP a mensajes accionables. */
  resolveError(err: unknown): string {
    if (err instanceof HttpErrorResponse && (err.status === 0 || !err.error)) {
      return 'Sin conexión con el servidor. Revisa tu red e intenta nuevamente.';
    }
    const e = err as { error?: { message?: string; error?: string }; message?: string };
    return (
      e?.error?.message ||
      e?.error?.error ||
      (typeof e?.error === 'string' ? e.error : undefined) ||
      e?.message ||
      'No se pudo guardar la identidad institucional. Intenta nuevamente.'
    );
  }

  /** Validación de formulario alineada al backend (HEX #RRGGBB obligatorio). */
  validate(payload: BrandingPayload): BrandingValidation {
    const name = payload.institution_name.trim();
    const color = payload.tenant_primary_color.trim();
    const logo = payload.logo_url.trim();
    const email = payload.support_email.trim();

    const result: BrandingValidation = { valid: true };

    if (!name) {
      result.valid = false;
      result.nameError = 'El nombre de la universidad es obligatorio.';
    }
    if (!/^#([A-Fa-f0-9]{6})$/.test(color)) {
      result.valid = false;
      result.colorError = 'Formato inválido. Usa el formato #RRGGBB (ej. #2563EB).';
    }
    // El backend genera URLs relativas (/api/v1/public/branding/logo/<id>.<ext>)
    // al subir por multipart; se aceptan junto a las absolutas https://.
    if (logo && !/^https:\/\/.+/.test(logo) && !logo.startsWith('/')) {
      result.valid = false;
      result.logoError = 'La URL del logo debe iniciar con https:// o ser una ruta interna (/api/...).';
    }
    if (email && !/^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(email)) {
      result.valid = false;
      result.emailError = 'El correo de soporte no tiene un formato válido.';
    }
    return result;
  }
}
