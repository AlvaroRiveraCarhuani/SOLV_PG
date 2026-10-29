/**
 * Catálogo de proveedores habilitados para la credencial personal del docente
 * (BYOK). Es la única fuente de verdad de esta lista: el panel del modal y la
 * vista de configuración la consumen desde acá para no divergir.
 */
export interface ByokProviderOption {
  value: string;
  label: string;
  placeholderKey: string;
}

export const BYOK_PROVIDERS: ByokProviderOption[] = [
  { value: 'gemini', label: 'Google Gemini', placeholderKey: 'AIzaSy...' },
  { value: 'openai', label: 'OpenAI', placeholderKey: 'sk-proj-...' },
  {
    value: 'openrouter',
    label: 'OpenRouter (Multi-modelo / Modelos Gratis)',
    placeholderKey: 'sk-or-v1-...'
  },
  { value: 'deepseek', label: 'DeepSeek', placeholderKey: 'sk-...' }
];

/** Valor por defecto cuando el docente todavía no guardó una preferencia. */
export const DEFAULT_BYOK_PROVIDER = 'openrouter';
