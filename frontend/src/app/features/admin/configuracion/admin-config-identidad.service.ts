import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, tap } from 'rxjs';
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
    return this.http.post<{ logo_url: string; file: string }>('/api/v1/tenants/logo', form).pipe(
      tap({
        next: (res) => {
          this.isSaving.set(false);
          const cfg = this.currentConfig();
          if (cfg) {
            const updated: TenantConfig = { ...cfg, logo_url: res.logo_url };
            this.tenantService.config.set(updated);
          }
        },
        error: () => this.isSaving.set(false)
      })
    );
  }

  saveBranding(payload: BrandingPayload): Observable<unknown> {
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
            const updated: TenantConfig = {
              ...cfg,
              institution_name: payload.institution_name || cfg.institution_name,
              logo_url: payload.logo_url || cfg.logo_url,
              tenant_primary_color: payload.tenant_primary_color || cfg.tenant_primary_color,
              support_email: payload.support_email || cfg.support_email
            };
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
    const e = err as { error?: { message?: string; error?: string } };
    return e?.error?.message || e?.error?.error || 'No se pudo guardar la identidad institucional. Intenta nuevamente.';
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
    if (logo && !/^https:\/\/.+/.test(logo)) {
      result.valid = false;
      result.logoError = 'La URL del logo debe iniciar con https://.';
    }
    if (email && !/^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(email)) {
      result.valid = false;
      result.emailError = 'El correo de soporte no tiene un formato válido.';
    }
    return result;
  }
}
