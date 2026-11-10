/**
 * Catálogo base de proveedores habilitados para la credencial personal del
 * docente (BYOK). El panel lo combina con los proveedores personalizados que
 * registra la institución, de modo que ambos caminos compartan la misma forma
 * de opción y no dupliquen lógica.
 *
 * Los cuatro proveedores base son servicios en la nube con endpoints oficiales
 * fijos forzados por el backend (escudo anti-SSRF). Los proveedores custom
 * aportan su propio endpoint institucional y el docente solo agrega su clave.
 */
export interface ByokProviderOption {
  value: string;
  label: string;
  badge: string;
  iconUrl: string;
  isCloud: boolean;
  placeholderKey: string;
}

export const BYOK_PROVIDERS: ByokProviderOption[] = [
  {
    value: 'gemini',
    label: 'Google Gemini',
    badge: 'Cloud · Nivel gratuito',
    iconUrl: '/ai-icons/gemini.svg',
    isCloud: true,
    placeholderKey: 'AIzaSy...'
  },
  {
    value: 'openai',
    label: 'OpenAI',
    badge: 'Cloud · Estándar',
    iconUrl: '/ai-icons/openai.svg',
    isCloud: true,
    placeholderKey: 'sk-proj-...'
  },
  {
    value: 'openrouter',
    label: 'OpenRouter',
    badge: 'Cloud · Variedad de modelos y ruteo',
    iconUrl: '/ai-icons/openrouter.svg',
    isCloud: true,
    placeholderKey: 'sk-or-v1-...'
  },
  {
    value: 'deepseek',
    label: 'DeepSeek',
    badge: 'Cloud · Código y razonamiento',
    iconUrl: '/ai-icons/deepseek.svg',
    isCloud: true,
    placeholderKey: 'sk-...'
  }
];

/** Valor por defecto cuando el docente todavía no guardó una preferencia. */
export const DEFAULT_BYOK_PROVIDER = 'openrouter';
