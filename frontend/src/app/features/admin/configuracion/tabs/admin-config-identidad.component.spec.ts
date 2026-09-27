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
  let mockService: any;
  let configSignal: ReturnType<typeof signal<TenantConfig | null>>;

  const setup = async (): Promise<void> => {
    configSignal = signal<TenantConfig | null>(makeConfig());
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
        if (payload.logo_url.trim() && !/^https:\/\/.+/.test(payload.logo_url.trim())) {
          result.valid = false;
          result.logoError = 'La URL del logo debe iniciar con https://.';
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
        { provide: TenantService, useValue: { config: configSignal, applyBranding: vi.fn() } }
      ]
    }).compileComponents();

    const fixture = TestBed.createComponent(AdminConfigIdentidadComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
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
});
