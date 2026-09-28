import '@angular/compiler';
import '@angular/localize/init';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { of, throwError } from 'rxjs';
import { AdminConfigIdentidadComponent } from './admin-config-identidad.component';
import { AdminConfigIdentidadService, BrandingPayload } from '../admin-config-identidad.service';
import { TenantService } from '@core/services/tenant.service';
import { TenantConfig } from '@core/models/tenant.model';

const makeConfig = (overrides: Partial<TenantConfig> = {}): TenantConfig => ({
  tenant_id: 't1',
  slug: 'uab',
  institution_name: 'Universidad Adventista de Bolivia',
  logo_url: '',
  tenant_primary_color: '#2563EB',
  support_email: 'soporte@solv.edu.bo',
  ...overrides
});

describe('AdminConfigIdentidadComponent', () => {
  let component: AdminConfigIdentidadComponent;
  let fixture: import('@angular/core/testing').ComponentFixture<AdminConfigIdentidadComponent>;
  let mockService: any;
  let tenantMock: any;
  let configSignal: ReturnType<typeof signal<TenantConfig | null>>;

  const setup = async (): Promise<void> => {
    configSignal = signal<TenantConfig | null>(makeConfig());
    tenantMock = { config: configSignal, applyBranding: vi.fn(), applyTenantFonts: vi.fn() };
    mockService = {
      isSaving: signal(false),
      currentConfig: () => configSignal(),
      initialValues: (): BrandingPayload => ({
        institution_name: configSignal()?.institution_name ?? '',
        logo_url: configSignal()?.logo_url ?? '',
        tenant_primary_color: configSignal()?.tenant_primary_color ?? '#2563EB',
        support_email: configSignal()?.support_email ?? ''
      }),
      saveBranding: vi.fn().mockImplementation((payload: BrandingPayload) => {
        configSignal.set(makeConfig(payload));
        return of({ status: 'updated' });
      }),
      uploadLogo: vi.fn().mockImplementation(() => of({ logo_url: '/api/v1/public/branding/logo/t1.png', file: 'logo_t1.png' })),
      resolveError: vi.fn().mockReturnValue('No se pudo guardar la identidad institucional.'),
      // Validación real (misma lógica del servicio) para los tests de formato
      validate: (payload: BrandingPayload) => {
        const result: any = { valid: true };
        if (!payload.institution_name.trim()) {
          result.valid = false;
          result.nameError = 'El nombre de la universidad es obligatorio.';
        }
        if (!/^#([A-Fa-f0-9]{6})$/.test(payload.tenant_primary_color.trim())) {
          result.valid = false;
          result.colorError = 'Formato inválido. Usa el formato #RRGGBB (ej. #2563EB).';
        }
        const logo = payload.logo_url.trim();
        if (logo && !/^https:\/\/.+/.test(logo) && !logo.startsWith('/')) {
          result.valid = false;
          result.logoError = 'La URL del logo debe iniciar con https:// o ser una ruta interna (/api/...).';
        }
        if (payload.support_email.trim() && !/^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(payload.support_email.trim())) {
          result.valid = false;
          result.emailError = 'El correo de soporte no tiene un formato válido.';
        }
        return result;
      }
    };

    await TestBed.configureTestingModule({
      imports: [AdminConfigIdentidadComponent],
      providers: [
        { provide: AdminConfigIdentidadService, useValue: mockService },
        { provide: TenantService, useValue: tenantMock }
      ]
    }).compileComponents();

    const fixtureRef = TestBed.createComponent(AdminConfigIdentidadComponent);
    fixture = fixtureRef;
    component = fixtureRef.componentInstance;
    fixtureRef.detectChanges();
  };

  it('debe inicializar el formulario con los valores vigentes del tenant', async () => {
    await setup();

    expect(component.institutionName()).toBe('Universidad Adventista de Bolivia');
    expect(component.primaryColor()).toBe('#2563EB');
    expect(component.isDirty()).toBe(false);
    expect(component.canSave()).toBe(false);
  });

  it('guardado parcial: enviar solo un color nuevo actualiza y el resto permanece', async () => {
    await setup();

    component.primaryColor.set('#16A34A');
    component.save();

    expect(mockService.saveBranding).toHaveBeenCalledWith(
      expect.objectContaining({
        institution_name: 'Universidad Adventista de Bolivia',
        tenant_primary_color: '#16A34A'
      })
    );
    expect(component.toast()?.type).toBe('success');
    expect(component.isDirty()).toBe(false);
  });

  it('hex inválido marca error de formato y deshabilita Guardar sin llamar al backend', async () => {
    await setup();

    component.primaryColor.set('2563EB');
    expect(component.validation().colorError).toContain('#RRGGBB');
    expect(component.canSave()).toBe(false);

    component.primaryColor.set('#ZZZZZZ');
    expect(component.validation().colorError).toBeDefined();

    expect(mockService.saveBranding).not.toHaveBeenCalled();
  });

  it('sin cambios el botón Guardar permanece deshabilitado (isDirty gate)', async () => {
    await setup();

    expect(component.isDirty()).toBe(false);
    expect(component.canSave()).toBe(false);

    // Tocar y volver al valor inicial no habilita el guardado
    component.institutionName.set('Otra U');
    component.institutionName.set('Universidad Adventista de Bolivia');
    expect(component.canSave()).toBe(false);
    expect(mockService.saveBranding).not.toHaveBeenCalled();
  });

  it('preview HSL reacciona al color sin recargar y resuelve texto según luminosidad', async () => {
    await setup();

    expect(component.previewHsl()).toEqual({ h: 221, s: '83%', l: '53%' });
    expect(component.previewTextColor()).toBe('#FFFFFF');

    component.primaryColor.set('#FDE68A'); // ámbar muy claro (l≈77% > 60)
    expect(component.previewTextColor()).toBe('#0F172A');

    component.primaryColor.set('#1E3A8A'); // azul muy oscuro
    expect(component.previewTextColor()).toBe('#FFFFFF');
  });

  it('cancelar restaura los valores vigentes sin llamar al backend', async () => {
    await setup();

    component.institutionName.set('Nombre Editado');
    component.logoUrl.set('https://x.edu.bo/logo.png');
    component.resetForm();

    expect(component.institutionName()).toBe('Universidad Adventista de Bolivia');
    expect(component.logoUrl()).toBe('');
    expect(component.isDirty()).toBe(false);
    expect(mockService.saveBranding).not.toHaveBeenCalled();
  });

  it('URL de logo sin https se rechaza en la validación', async () => {
    await setup();

    component.logoUrl.set('http://cdn.instituto.edu/logo.png');
    expect(component.validation().logoError).toContain('https://');

    component.logoUrl.set('');
    expect(component.validation().logoError).toBeUndefined();
  });

  it('URL relativa del backend (/api/...) se acepta como logo válido', async () => {
    await setup();

    component.logoUrl.set('/api/v1/public/branding/logo/abc123.png');
    expect(component.validation().logoError).toBeUndefined();
    expect(component.validation().valid).toBe(true);
  });

  it('toggle de soporte apagado vacía el email y el input', async () => {
    await setup();

    expect(component.supportEmailEnabled()).toBe(true);
    component.onSupportEmailToggle(false);
    expect(component.supportEmailEnabled()).toBe(false);
    expect(component.supportEmail()).toBe('');
    expect(component.current().support_email).toBe('');
    expect(component.validation().emailError).toBeUndefined();
  });

  it('el input del logo queda vacío para URLs internas (/api/...) y muestra las https', async () => {
    await setup();

    component.logoUrl.set('/api/v1/public/branding/logo/abc123.png?t=123');
    expect(component.logoInputValue()).toBe('');

    component.onLogoUrlChange('https://x.edu.bo/logo.png');
    expect(component.logoUrl()).toBe('https://x.edu.bo/logo.png');
    expect(component.logoInputValue()).toBe('https://x.edu.bo/logo.png');
  });

  it('durante la subida el guardado queda bloqueado', async () => {
    await setup();

    component.primaryColor.set('#16A34A');
    component.isUploadingLogo.set(true);
    expect(component.canSave()).toBe(false);
    component.isUploadingLogo.set(false);
    expect(component.canSave()).toBe(true);
  });

  it('catalog pick stages tokens through applyTenantFonts', async () => {
    await setup();

    component.onSansCatalogChange('lato');

    expect(component.fontSansValue()).toBe('cat:lato');
    expect(tenantMock.applyTenantFonts).toHaveBeenCalledWith(
      expect.objectContaining({ font_sans_family: 'cat:lato' })
    );
  });

  it('invalid custom URL exposes inline error and never stages', async () => {
    await setup();
    const before = component.fontSansValue();

    component.customSansUrl.set('http://fonts.googleapis.com/css2?family=Inter');
    expect(component.customSansUrlError()).toContain('https');

    component.applyCustomSansUrl();
    expect(component.fontSansValue()).toBe(before);
    expect(tenantMock.applyTenantFonts).not.toHaveBeenCalled();
  });

  it('foreign host custom URL exposes inline host error', async () => {
    await setup();

    component.customMonoUrl.set('https://cdn.evil.example.com/font.css?family=X');
    expect(component.customMonoUrlError()).toContain('fonts.googleapis.com');

    component.applyCustomMonoUrl();
    expect(component.fontMonoValue()).toBe('cat:jetbrains-mono');
  });

  it('valid custom URL stages preview through applyTenantFonts', async () => {
    await setup();

    component.customSansUrl.set('https://fonts.googleapis.com/css2?family=Public+Sans:wght@400;600;700');
    expect(component.customSansUrlError()).toBeNull();

    component.applyCustomSansUrl();
    expect(component.fontSansValue()).toBe('url:https://fonts.googleapis.com/css2?family=Public+Sans:wght@400;600;700');
    expect(tenantMock.applyTenantFonts).toHaveBeenCalledWith(
      expect.objectContaining({ font_sans_family: expect.stringContaining('url:') })
    );
  });

  it('font-only change enables Guardar Marca', async () => {
    await setup();
    expect(component.canSave()).toBe(false);

    component.onSansCatalogChange('lato');

    expect(component.isDirty()).toBe(true);
    expect(component.canSave()).toBe(true);
  });

  it('cancel reverts staged preview tokens to the snapshot', async () => {
    await setup();

    component.onSansCatalogChange('lato');
    component.resetForm();

    expect(component.fontSansValue()).toBe('cat:inter');
    expect(tenantMock.applyTenantFonts).toHaveBeenLastCalledWith(
      expect.objectContaining({ font_sans_family: 'cat:inter' })
    );
  });

  it('save drops unapplied custom input', async () => {
    await setup();

    component.customSansUrl.set('https://fonts.googleapis.com/css2?family=Lato');
    component.primaryColor.set('#16A34A');
    component.save();

    expect(component.customSansUrl()).toBe('');
    expect(component.fontSansValue()).toBe('cat:inter');
    expect(component.toast()?.type).toBe('success');
    expect(component.toast()?.message).toContain('Se descartó la URL sin aplicar');
  });

  it('typography hint disambiguates Aplicar (preview) from Guardar Marca (persist)', async () => {
    await setup();

    const hint: HTMLElement | null = fixture.nativeElement.querySelector('.typography-hint');
    expect(hint).not.toBeNull();
    expect(hint?.textContent).toContain('Aplicar');
    expect(hint?.textContent).toContain('Guardar Marca');
  });

  it('successful save applies fonts to the session without reload', async () => {
    await setup();

    component.onSansCatalogChange('lato');
    component.save();

    expect(configSignal()?.font_sans_family).toBe('cat:lato');
    expect(tenantMock.applyBranding).toHaveBeenCalled();
  });

  it('branding-ok/fonts-422 shows partial copy with the mapped cause', async () => {
    await setup();
    mockService.saveBranding
      .mockReturnValueOnce(of({ status: 'updated' }))
      .mockReturnValueOnce(
        throwError(() => ({ status: 422, error: { error: 'font_url_unreachable', message: 'No alcanzable' } }))
      );

    component.onSansCatalogChange('lato');
    component.save();

    expect(component.fontError()).toContain('Marca guardada');
    expect(component.fontError()).toContain('alcanzar');
    expect(component.toast()).toBeNull();
  });
});
