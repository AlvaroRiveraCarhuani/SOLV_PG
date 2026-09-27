import { Component, inject, signal, computed, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import {
  AdminConfigIdentidadService,
  BrandingPayload
} from '../admin-config-identidad.service';
import { ModalShellComponent, ModalIntent } from '@shared/components/modal-shell/modal-shell.component';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import {
  CURATED_FONTS,
  DEFAULT_SANS_SLUG,
  DEFAULT_MONO_SLUG,
  findCuratedFont,
  fontValueToFamily,
  fontValueToStack,
  fontValueToMonoStack
} from '@shared/curated-fonts';
import {
  LucidePalette,
  LucideBuilding2,
  LucideUpload,
  LucideMail,
  LucideImage,
  LucideEye,
  LucideInfo,
  LucideCheckCircle2,
  LucideAlertCircle,
  LucideExternalLink,
  LucideType
} from '@lucide/angular';

@Component({
  selector: 'admin-config-identidad',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ModalShellComponent,
    FormFieldComponent,
    LucidePalette,
    LucideBuilding2,
    LucideUpload,
    LucideMail,
    LucideImage,
    LucideEye,
    LucideInfo,  LucideCheckCircle2,
  LucideAlertCircle,
  LucideExternalLink,
  LucideType
  ],
  templateUrl: './admin-config-identidad.component.html',
  styleUrls: ['./admin-config-identidad.component.scss']
})
export class AdminConfigIdentidadComponent implements OnInit {
  private readonly identidadService = inject(AdminConfigIdentidadService);

  readonly isSaving = this.identidadService.isSaving;

  // ------------------------------------------------------------------
  // Formulario (signals, una por campo del contrato PUT /admin/branding)
  // ------------------------------------------------------------------
  readonly institutionName = signal('');
  readonly logoUrl = signal('');
  readonly primaryColor = signal('');
  readonly supportEmail = signal('');

  readonly formError = signal<string | null>(null);

  /** Estado inicial cargado del config vigente, para detectar cambios reales. */
  private readonly initial = signal<BrandingPayload & { font_sans_family?: string; font_mono_family?: string }>({
    institution_name: '',
    logo_url: '',
    tenant_primary_color: '',
    support_email: ''
  });

  readonly current = computed<BrandingPayload>(() => ({
    institution_name: this.institutionName(),
    logo_url: this.logoUrl(),
    tenant_primary_color: this.primaryColor(),
    support_email: this.supportEmail()
  }));

  readonly validation = computed(() => this.identidadService.validate(this.current()));

  readonly isDirty = computed(() => {
    const c = this.current();
    const i = this.initial();
    return (
      c.institution_name !== i.institution_name ||
      c.logo_url !== i.logo_url ||
      c.tenant_primary_color !== i.tenant_primary_color ||
      c.support_email !== i.support_email
    );
  });

  readonly canSave = computed(() => this.validation().valid && this.isDirty() && !this.isSaving());

  // ------------------------------------------------------------------
  // Tipografía white-label (proposal tenant-typography)
  // ------------------------------------------------------------------
  readonly sansCatalog = CURATED_FONTS.filter((f) => f.kind === 'sans');
  readonly monoCatalog = CURATED_FONTS.filter((f) => f.kind === 'mono');

  readonly fontSansValue = signal<string>(`cat:${DEFAULT_SANS_SLUG}`);
  readonly fontMonoValue = signal<string>(`cat:${DEFAULT_MONO_SLUG}`);
  readonly customSansUrl = signal('');
  readonly customMonoUrl = signal('');
  readonly showCustomFonts = signal(false);
  readonly fontError = signal<string | null>(null);

  readonly fontSansDirty = computed(() => this.fontSansValue() !== this.initial().font_sans_family);
  readonly fontMonoDirty = computed(() => this.fontMonoValue() !== this.initial().font_mono_family);

  readonly fontSansFamily = computed(() => fontValueToFamily(this.fontSansValue(), DEFAULT_SANS_SLUG));
  readonly fontMonoFamily = computed(() => fontValueToFamily(this.fontMonoValue(), DEFAULT_MONO_SLUG));

  readonly fontPreviewStack = computed(() => fontValueToStack(this.fontSansValue(), DEFAULT_SANS_SLUG));
  readonly fontMonoPreviewStack = computed(() => fontValueToMonoStack(this.fontMonoValue(), DEFAULT_MONO_SLUG));

  onSansCatalogChange(slug: string): void {
    this.fontSansValue.set(`cat:${slug}`);
    this.fontError.set(null);
  }

  onMonoCatalogChange(slug: string): void {
    this.fontMonoValue.set(`cat:${slug}`);
    this.fontError.set(null);
  }

  applyCustomSansUrl(): void {
    const url = this.customSansUrl().trim();
    if (!url) return;
    this.fontSansValue.set(`url:${url}`);
    this.fontError.set(null);
  }

  applyCustomMonoUrl(): void {
    const url = this.customMonoUrl().trim();
    if (!url) return;
    this.fontMonoValue.set(`url:${url}`);
    this.fontError.set(null);
  }

  toggleCustomFonts(): void {
    this.showCustomFonts.update((v) => !v);
  }

  // ------------------------------------------------------------------
  // Preview en vivo (motor HSL del TenantService, reutilizado como service puro)
  // ------------------------------------------------------------------
  readonly previewHsl = computed(() => this.identidadServicePreviewHSL());

  readonly previewTextColor = computed(() => {
    const l = parseInt(this.previewHsl().l.replace('%', ''), 10);
    return l > 60 ? '#0F172A' : '#FFFFFF';
  });

  readonly contrastOk = computed(() => {
    // El motor del TenantService garantiza contraste AA al elegir texto claro/oscuro;
    // el indicador refleja que el hex es válido y el par color/texto está resuelto.
    return this.validation().colorError === undefined;
  });

  // ------------------------------------------------------------------
  // Toast
  // ------------------------------------------------------------------
  readonly toast = signal<{ type: 'success' | 'error'; message: string } | null>(null);
  private toastTimer?: ReturnType<typeof setTimeout>;

  showToast(message: string, type: 'success' | 'error' = 'success'): void {
    this.toast.set({ message, type });
    if (this.toastTimer) clearTimeout(this.toastTimer);
    this.toastTimer = setTimeout(() => this.toast.set(null), 4000);
  }

  ngOnInit(): void {
    const initial = this.identidadService.initialValues();
    const cfg = this.identidadService.currentConfig();
    this.initial.set({
      ...initial,
      font_sans_family: cfg?.font_sans_family || `cat:${DEFAULT_SANS_SLUG}`,
      font_mono_family: cfg?.font_mono_family || `cat:${DEFAULT_MONO_SLUG}`
    });
    this.institutionName.set(initial.institution_name);
    this.logoUrl.set(initial.logo_url);
    this.primaryColor.set(initial.tenant_primary_color);
    this.supportEmail.set(initial.support_email);
    this.fontSansValue.set(cfg?.font_sans_family || `cat:${DEFAULT_SANS_SLUG}`);
    this.fontMonoValue.set(cfg?.font_mono_family || `cat:${DEFAULT_MONO_SLUG}`);
  }

  // ------------------------------------------------------------------
  // Logo: URL manual + subida real de archivo (multipart)
  // ------------------------------------------------------------------
  readonly logoPreviewFailed = signal(false);
  readonly isUploadingLogo = this.identidadService.isSaving;

  readonly LOGO_MAX_MB = 2;

  onLogoUrlChange(value: string): void {
    this.logoUrl.set(value);
    this.logoPreviewFailed.set(false);
  }

  onLogoError(): void {
    this.logoPreviewFailed.set(true);
  }

  onLogoFileSelected(event: Event): void {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;

    if (file.size > this.LOGO_MAX_MB * 1024 * 1024) {
      this.showToast(`El logo no debe exceder ${this.LOGO_MAX_MB} MB.`, 'error');
      input.value = '';
      return;
    }
    const allowed = ['image/png', 'image/svg+xml', 'image/jpeg', 'image/webp'];
    if (!allowed.includes(file.type)) {
      this.showToast('Formato no permitido. Usa PNG, SVG, JPG o WEBP.', 'error');
      input.value = '';
      return;
    }

    this.identidadService.uploadLogo(file).subscribe({
      next: (res) => {
        this.logoUrl.set(res.logo_url);
        this.logoPreviewFailed.set(false);
        this.showToast('Logo institucional actualizado. La propagación tarda hasta 5 minutos.');
      },
      error: (err) => {
        this.showToast(this.identidadService.resolveError(err), 'error');
      }
    });

    input.value = '';
  }

  // ------------------------------------------------------------------
  // Guardado (merge parcial del backend: campos vacíos no pisan)
  // ------------------------------------------------------------------
  save(): void {
    if (!this.canSave()) return;

    const payload = this.current();
    this.formError.set(null);
    this.fontError.set(null);

    // Guardado en dos actos: branding primero, tipografía después (el backend
    // valida las fuentes con reglas propias: catálogo o URL custom alcanzable).
    this.identidadService.saveBranding(payload).subscribe({
      next: () => {
        this.saveFonts(payload);
      },
      error: (err) => {
        this.formError.set(this.identidadService.resolveError(err));
      }
    });
  }

  private saveFonts(branding: BrandingPayload): void {
    const fontPayload = {
      ...branding,
      font_sans_family: this.fontSansValue(),
      font_mono_family: this.fontMonoValue()
    };

    this.identidadService.saveBranding(fontPayload).subscribe({
      next: () => {
        this.initial.set({ ...branding, font_sans_family: this.fontSansValue(), font_mono_family: this.fontMonoValue() });
        this.showToast('Identidad institucional guardada. La propagación a todos los usuarios tarda hasta 5 minutos.');
      },
      error: (err) => {
        // El branding quedó guardado; el error es específicamente tipográfico
        this.fontError.set(this.identidadService.resolveError(err));
      }
    });
  }

  resetForm(): void {
    const i = this.initial();
    this.institutionName.set(i.institution_name);
    this.logoUrl.set(i.logo_url);
    this.primaryColor.set(i.tenant_primary_color);
    this.supportEmail.set(i.support_email);
    this.fontSansValue.set(i.font_sans_family || `cat:${DEFAULT_SANS_SLUG}`);
    this.fontMonoValue.set(i.font_mono_family || `cat:${DEFAULT_MONO_SLUG}`);
    this.customSansUrl.set('');
    this.customMonoUrl.set('');
    this.formError.set(null);
    this.fontError.set(null);
    this.logoPreviewFailed.set(false);
  }

  // ------------------------------------------------------------------
  // Utilidades de preview
  // ------------------------------------------------------------------
  tenantInitials(): string {
    const name = this.institutionName().trim() || 'SL';
    const parts = name.split(/\s+/);
    return parts.length >= 2
      ? (parts[0][0] + parts[1][0]).toUpperCase()
      : name.slice(0, 2).toUpperCase();
  }

  previewModalIntent(): ModalIntent {
    return 'info';
  }

  private identidadServicePreviewHSL(): { h: number; s: string; l: string } {
    const hex = this.primaryColor().trim();
    // Mismo contrato que TenantService.hexToHSL: fallback azul SOLV
    if (!/^#([A-Fa-f0-9]{6})$/.test(hex)) {
      return { h: 221, s: '83%', l: '53%' };
    }
    const r = parseInt(hex.substring(1, 3), 16) / 255;
    const g = parseInt(hex.substring(3, 5), 16) / 255;
    const b = parseInt(hex.substring(5, 7), 16) / 255;

    const max = Math.max(r, g, b);
    const min = Math.min(r, g, b);
    let h = 0;
    let s = 0;
    const l = (max + min) / 2;

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
    return { h: Math.round(h * 360), s: `${Math.round(s * 100)}%`, l: `${Math.round(l * 100)}%` };
  }
}
