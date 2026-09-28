/**
 * Catálogo curado de fuentes white-label (proposal tenant-typography / ADR-038).
 *
 * Debe mantenerse sincronizado con `CuratedFontCatalog` en
 * `backend/internal/core/services/typography_service.go` (misma lista, mismo
 * orden). El backend es la fuente de verdad de la validación; este catálogo
 * alimenta los selectores y la carga dinámica de Google Fonts.
 */
export interface CuratedFont {
  slug: string;
  name: string;
  googleQuery: string;
  weights: number[];
  kind: 'sans' | 'mono';
}

export const CURATED_FONTS: CuratedFont[] = [
  // --- UI (sans) ---
  { slug: 'inter', name: 'Inter', googleQuery: 'Inter:wght@400;500;600;700', weights: [400, 500, 600, 700], kind: 'sans' },
  { slug: 'source-sans-3', name: 'Source Sans 3', googleQuery: 'Source+Sans+3:wght@400;500;600;700', weights: [400, 500, 600, 700], kind: 'sans' },
  { slug: 'open-sans', name: 'Open Sans', googleQuery: 'Open+Sans:wght@400;500;600;700', weights: [400, 500, 600, 700], kind: 'sans' },
  { slug: 'public-sans', name: 'Public Sans', googleQuery: 'Public+Sans:wght@400;500;600;700', weights: [400, 500, 600, 700], kind: 'sans' },
  { slug: 'lato', name: 'Lato', googleQuery: 'Lato:wght@400;700', weights: [400, 500, 600, 700], kind: 'sans' },
  { slug: 'roboto', name: 'Roboto', googleQuery: 'Roboto:wght@400;500;600;700', weights: [400, 500, 600, 700], kind: 'sans' },
  // --- Datos de máquina (mono) ---
  { slug: 'jetbrains-mono', name: 'JetBrains Mono', googleQuery: 'JetBrains+Mono:wght@400;500;600', weights: [400, 500, 600], kind: 'mono' },
  { slug: 'fira-code', name: 'Fira Code', googleQuery: 'Fira+Code:wght@400;500;600', weights: [400, 500, 600], kind: 'mono' },
  { slug: 'ibm-plex-mono', name: 'IBM Plex Mono', googleQuery: 'IBM+Plex+Mono:wght@400;500;600', weights: [400, 500, 600], kind: 'mono' }
];

export const DEFAULT_SANS_SLUG = 'inter';
export const DEFAULT_MONO_SLUG = 'jetbrains-mono';

export function findCuratedFont(slug: string): CuratedFont | undefined {
  return CURATED_FONTS.find((f) => f.slug === slug);
}

/** URL de la hoja CSS de una fuente curada (display=swap con pesos exactos). */
export function curatedFontCSSUrl(font: CuratedFont): string {
  return `https://fonts.googleapis.com/css2?family=${font.googleQuery}&display=swap`;
}

/** Extrae la familia principal de un valor de fuente para atribuir font-family local. */
export function fontValueToFamily(value: string | undefined, defaultSlug: string): string {  if (!value) return findCuratedFont(defaultSlug)?.name ?? 'Inter';
  if (value.startsWith('cat:')) return findCuratedFont(value.slice(4))?.name ?? 'Inter';
  if (value.startsWith('url:')) {
    const match = value.match(/family=([^:&]+)/);
    return match ? match[1].replace(/\+/g, ' ') : 'Inter';
  }
  return 'Inter';
}

/** Stack CSS local para el preview (los tokens globales los maneja TenantService). */
export function fontValueToStack(value: string | undefined, defaultSlug: string): string {
  const family = fontValueToFamily(value, defaultSlug);
  return `'${family}', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif`;
}

export function fontValueToMonoStack(value: string | undefined, defaultSlug: string): string {
  const family = fontValueToFamily(value, defaultSlug);
  return `'${family}', Menlo, Monaco, Consolas, monospace`;
}

/** Causa del rechazo inline de una URL custom (espejo barato del backend). */
export type FontUrlIssue = 'https' | 'host' | 'family';

export interface FontUrlCheck {
  ok: boolean;
  issue?: FontUrlIssue;
  message?: string;
}

/** Host permitido por gobernanza para hojas CSS custom (igual que el backend). */
export const ALLOWED_FONT_CSS_HOST = 'fonts.googleapis.com';

/**
 * Valida una URL custom de fuente sin red: exige HTTPS, host permitido y
 * presencia de `family=`. La alcanzabilidad queda solo en el backend
 * (código 422 `font_url_unreachable`).
 */
export function validateCustomFontUrl(raw: string): FontUrlCheck {
  const url = (raw ?? '').trim();
  if (!url || !url.startsWith('https://')) {
    return { ok: false, issue: 'https', message: 'La URL de la fuente debe iniciar con https://.' };
  }
  let host = '';
  try {
    host = new URL(url).host;
  } catch {
    return { ok: false, issue: 'https', message: 'La URL de la fuente no es válida. Usa una URL https:// completa.' };
  }
  if (host !== ALLOWED_FONT_CSS_HOST) {
    return { ok: false, issue: 'host', message: `Solo se permiten hojas CSS de ${ALLOWED_FONT_CSS_HOST}.` };
  }
  if (!/[?&]family=/.test(url)) {
    return { ok: false, issue: 'family', message: 'La URL debe incluir el parámetro family= con la familia a cargar.' };
  }
  return { ok: true };
}
