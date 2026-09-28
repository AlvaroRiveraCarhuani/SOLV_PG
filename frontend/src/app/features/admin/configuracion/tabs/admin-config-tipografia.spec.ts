/* eslint-disable @typescript-eslint/no-explicit-any */
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
import { CURATED_FONTS, DEFAULT_SANS_SLUG, DEFAULT_MONO_SLUG, fontValueToFamily } from '@shared/curated-fonts';

const makeConfig = (overrides: Partial<TenantConfig> = {}): TenantConfig => ({
  tenant_id: 't1',
  slug: 'uab',
  institution_name: 'Universidad Adventista de Bolivia',
  logo_url: '',
  tenant_primary_color: '#2563EB',
  support_email: 'soporte@solv.edu.bo',
  ...overrides
});

describe('Tipografía white-label (tenant-typography)', () => {
  let component: AdminConfigIdentidadComponent;
  let mockService: any;
  let configSignal: ReturnType<typeof signal<TenantConfig | null>>;
  let applyBrandingMock: ReturnType<typeof vi.fn>;

  const setup = async (configOverrides: Partial<TenantConfig> = {}): Promise<void> => {
    configSignal = signal<TenantConfig | null>(makeConfig(configOverrides));
    applyBrandingMock = vi.fn();    mockService = {
      isSaving: signal(false),
      currentConfig: () => configSignal(),
      initialValues: (): BrandingPayload => ({
        institution_name: configSignal()?.institution_name ?? '',
        logo_url: configSignal()?.logo_url ?? '',
        tenant_primary_color: configSignal()?.tenant_primary_color ?? '#2563EB',
        support_email: configSignal()?.support_email ?? ''
      }),
      saveBranding: vi.fn().mockReturnValue(of({ status: 'updated' })),
      resolveError: vi.fn().mockImplementation((err: unknown) => {
        const e = err as { error?: { message?: string } };
        return e?.error?.message || 'Error de prueba';
      }),
      validate: (): any => ({ valid: true })
    };

    await TestBed.configureTestingModule({
      imports: [AdminConfigIdentidadComponent],
      providers: [
        { provide: AdminConfigIdentidadService, useValue: mockService },
        { provide: TenantService, useValue: { config: configSignal, applyBranding: applyBrandingMock, applyTenantFonts: vi.fn() } }
      ]
    }).compileComponents();

    const fixture = TestBed.createComponent(AdminConfigIdentidadComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  };

  it('tenant sin configuración: defaults Inter/JetBrains Mono', async () => {
    await setup();

    expect(component.fontSansValue()).toBe(`cat:${DEFAULT_SANS_SLUG}`);
    expect(component.fontMonoValue()).toBe(`cat:${DEFAULT_MONO_SLUG}`);
    expect(component.fontSansFamily()).toBe('Inter');
    expect(component.fontMonoFamily()).toBe('JetBrains Mono');
  });

  it('catálogo curado: 6 fuentes sans + 3 mono, todas con pesos del contrato', async () => {
    await setup();

    expect(component.sansCatalog.length).toBe(6);
    expect(component.monoCatalog.length).toBe(3);
    for (const f of CURATED_FONTS) {
      if (f.kind === 'sans') {
        expect(f.weights).toEqual([400, 500, 600, 700]);
      } else {
        expect(f.weights).toEqual([400, 500, 600]);
      }
    }
  });

  it('selección desde catálogo: elegir source-sans-3 actualiza la pareja del preview', async () => {
    await setup();

    component.onSansCatalogChange('source-sans-3');
    expect(component.fontSansValue()).toBe('cat:source-sans-3');
    expect(component.fontSansFamily()).toBe('Source Sans 3');
    expect(component.fontPreviewStack()).toContain("'Source Sans 3'");
  });

  it('URL custom se envía como url:... al backend y el error 422 se muestra sin perder el branding', async () => {
    await setup();
    mockService.saveBranding.mockImplementation((payload: any) => {
      if (payload['font_sans_family']?.startsWith?.('url:')) {
        return throwError(() => ({ error: { message: 'No se pudo alcanzar la URL de la fuente.' } }));
      }
      return of({ status: 'updated' });
    });

    // El guardado exige cambios reales (gate isDirty): tocamos el nombre institucional
    component.institutionName.set('Universidad Adventista de Bolivia (UAB)');
    component.customSansUrl.set('https://fonts.googleapis.com/css2?family=Public+Sans:wght@400;600;700');
    component.applyCustomSansUrl();
    expect(component.fontSansValue()).toBe('url:https://fonts.googleapis.com/css2?family=Public+Sans:wght@400;600;700');

    component.save();

    // El branding pasó (primer save) y el error tipográfico quedó aislado en su banner
    expect(mockService.saveBranding).toHaveBeenCalledTimes(2);
    expect(component.fontError()).toContain('No se pudo alcanzar');
    expect(component.formError()).toBeNull();
  });

  it('preview antes de aplicar: la fuente probada no se persiste hasta guardar', async () => {
    await setup();

    component.onSansCatalogChange('lato');
    component.onMonoCatalogChange('fira-code');

    expect(mockService.saveBranding).not.toHaveBeenCalled();
    expect(configSignal()?.font_sans_family).toBeUndefined();
    expect(component.fontSansFamily()).toBe('Lato');
  });

  it('cancelar restaura las fuentes configuradas sin llamar al backend', async () => {
    await setup({ font_sans_family: 'cat:public-sans', font_mono_family: 'cat:ibm-plex-mono' });

    component.onSansCatalogChange('roboto');
    component.resetForm();

    expect(component.fontSansValue()).toBe('cat:public-sans');
    expect(component.fontMonoValue()).toBe('cat:ibm-plex-mono');
    expect(mockService.saveBranding).not.toHaveBeenCalled();
  });

  it('tenant con configuración previa carga sus fuentes vigentes', async () => {
    await setup({ font_sans_family: 'cat:open-sans', font_mono_family: 'cat:fira-code' });

    expect(component.fontSansValue()).toBe('cat:open-sans');
    expect(component.fontMonoValue()).toBe('cat:fira-code');
    expect(fontValueToFamily('cat:open-sans', DEFAULT_SANS_SLUG)).toBe('Open Sans');
  });

  it('guardado exitoso persiste ambos valores de fuente junto al branding', async () => {
    await setup();

    // El guardado exige cambios reales (gate isDirty): tocamos el nombre institucional
    component.institutionName.set('Universidad Adventista de Bolivia (UAB)');
    component.onSansCatalogChange('roboto');
    component.onMonoCatalogChange('ibm-plex-mono');
    component.save();

    const lastCall = mockService.saveBranding.mock.calls[1][0];
    expect(lastCall['font_sans_family']).toBe('cat:roboto');
    expect(lastCall['font_mono_family']).toBe('cat:ibm-plex-mono');
    expect(component.toast()?.type).toBe('success');
  });

  it('fuentes del catálogo usan display=swap con pesos exactos', async () => {
    const inter = CURATED_FONTS.find((f) => f.slug === 'inter');
    expect(inter).toBeDefined();
    const { curatedFontCSSUrl } = await import('@shared/curated-fonts');
    expect(curatedFontCSSUrl(inter!)).toBe('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap');
  });
});
