export interface TenantConfig {
  tenant_id: string;
  slug: string;
  institution_name: string;
  logo_url: string;
  tenant_primary_color?: string;
  base_domain?: string;
  support_email?: string;
  /** Tipografía white-label: "cat:slug" (catálogo curado) o "url:https://..." (custom validada). */
  font_sans_family?: string;
  font_mono_family?: string;
}

export interface TenantBrandingHSL {
  h: number;
  s: string;
  l: string;
}
