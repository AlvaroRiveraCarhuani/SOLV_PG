/**
 * Catálogo de proveedores habilitados para la credencial personal del docente
 * (BYOK). Es la única fuente de verdad de esta lista: el panel del modal y la
 * vista de configuración la consumen desde acá para no divergir.
 *
 * Todos los proveedores son servicios en la nube con endpoints oficiales fijos
 * forzados por el backend (escudo anti-SSRF). Los motores on-premise (Ollama/vLLM)
 * pertenecen exclusivamente al ámbito de infraestructura y se gestionan desde
 * el motor institucional.
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
