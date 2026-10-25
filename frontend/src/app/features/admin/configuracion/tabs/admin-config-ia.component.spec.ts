import '@angular/compiler';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { of } from 'rxjs';
import { AdminConfigIaComponent } from './admin-config-ia.component';
import {
  AdminConfigIaService,
  AISettingsDTO,
  AIProviderProfileDTO,
  AICustomProviderDTO,
  UpdateAISettingsDTO,
  TestAIConnectionPayload,
  ListAIModelsPayload,
  ListAIModelsResponse
} from '../admin-config-ia.service';

const GEMINI_URL = 'https://generativelanguage.googleapis.com/v1beta/openai';

const mockProfiles: Record<string, AIProviderProfileDTO> = {
  gemini: { model: 'gemini-2.5-flash', base_url: GEMINI_URL, has_api_key: true, api_key_masked: 'AIzaSy...4x9Q' },
  openai: { model: 'gpt-4o-mini', base_url: 'https://api.openai.com/v1', has_api_key: false, api_key_masked: '' },
  openrouter: { model: 'openrouter/auto', base_url: 'https://openrouter.ai/api/v1', has_api_key: true, api_key_masked: 'sk-or...1a2b' },
  deepseek: { model: '', base_url: 'https://api.deepseek.com', has_api_key: false, api_key_masked: '' },
  ollama_lan: { model: '', base_url: 'http://10.20.1.50:11434/v1', has_api_key: false, api_key_masked: '' }
};

const mockCustomProviders: AICustomProviderDTO[] = [
  {
    id: 'custom_llm_local',
    label: 'LLM Local',
    icon_url: 'https://universidad.edu.bo/logo.svg',
    base_url: 'http://192.168.1.50:11434/v1',
    model: 'qwen2.5-coder:7b',
    has_api_key: true,
    api_key_masked: 'abcd...wxyz'
  },
  {
    id: 'custom_docentes_bot',
    label: 'Docentes Bot',
    base_url: 'https://bot.universidad.edu.bo/v1',
    model: '',
    has_api_key: false,
    api_key_masked: ''
  }
];

const mockSettings: AISettingsDTO = {
  enabled: true,
  provider: 'gemini',
  provider_label: 'Google Gemini',
  model: 'gemini-2.5-flash',
  base_url: GEMINI_URL,
  has_api_key: true,
  api_key_masked: 'AIzaSy...4x9Q',
  hourly_quota_per_user: 30,
  timeout_seconds: 60,
  updated_at: '2026-10-24T12:00:00Z',
  provider_profiles: mockProfiles,
  custom_providers: mockCustomProviders
};

const mockModelsResponse: ListAIModelsResponse = {
  provider: 'openrouter',
  cached: false,
  count: 3,
  fetched_at: '2026-10-24T14:30:00Z',
  models: [
    {
      id: 'meta-llama/llama-3.3-70b-instruct:free',
      name: 'Llama 3.3 70B (free)',
      tier: 'free',
      pricing_hint: 'Gratis'
    },
    {
      id: 'google/gemini-flash-1.5:free',
      name: 'Gemini Flash 1.5 Free',
      tier: 'free',
      pricing_hint: 'Gratis'
    },
    {
      id: 'anthropic/claude-3.5-sonnet',
      name: 'Claude 3.5 Sonnet',
      tier: 'paid',
      pricing_hint: 'De pago'
    }
  ]
};

describe('AdminConfigIaComponent', () => {
  let fixture: ComponentFixture<AdminConfigIaComponent>;
  let component: AdminConfigIaComponent;
  let mockService: {
    settings: ReturnType<typeof signal<AISettingsDTO | null>>;
    getSettings: ReturnType<typeof vi.fn>;
    updateSettings: ReturnType<typeof vi.fn>;
    testConnection: ReturnType<typeof vi.fn>;
    listModels: ReturnType<typeof vi.fn>;
    resolveError: ReturnType<typeof vi.fn>;
  };

  beforeEach(async () => {
    const settingsSignal = signal<AISettingsDTO | null>(mockSettings);
    mockService = {
      settings: settingsSignal,
      getSettings: vi.fn().mockReturnValue(of(mockSettings)),
      updateSettings: vi.fn().mockImplementation((dto: UpdateAISettingsDTO) =>
        of({
          ...mockSettings,
          ...dto,
          provider_profiles: mockProfiles,
          // El backend genera el ID custom_<slug> a partir del label
          custom_providers: (dto.custom_providers ?? []).map((cp) =>
            cp.id === '' ? { ...cp, id: 'custom_paquito_ai' } : cp
          ),
          has_api_key: true,
          api_key_masked: 'AIzaSy...4x9Q',
          updated_at: '2026-10-24T12:05:00Z'
        })
      ),
      testConnection: vi.fn().mockImplementation((payload: TestAIConnectionPayload) =>
        of({
          ok: true,
          provider: payload.provider,
          model: payload.model,
          latency_ms: 180,
          message: 'Conexión exitosa con el motor de inferencia'
        })
      ),
      listModels: vi.fn().mockImplementation((_payload: ListAIModelsPayload) => of(mockModelsResponse)),
      resolveError: vi.fn().mockReturnValue('Error simulado')
    };

    await TestBed.configureTestingModule({
      imports: [AdminConfigIaComponent],
      providers: [{ provide: AdminConfigIaService, useValue: mockService }]
    }).compileComponents();

    fixture = TestBed.createComponent(AdminConfigIaComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('1. Carga inicial: verifica que ngOnInit consume getSettings() y sincroniza los campos', () => {
    expect(mockService.getSettings).toHaveBeenCalled();
    expect(component.enabled()).toBe(true);
    expect(component.provider()).toBe('gemini');
    expect(component.model()).toBe('gemini-2.5-flash');
    expect(component.baseUrl()).toBe(GEMINI_URL);
    expect(component.hourlyQuota()).toBe(30);
    expect(component.timeoutSeconds()).toBe(60);
    expect(component.isCloudProvider()).toBe(true);
    expect(component.customProviders().length).toBe(2);
    expect(component.providerProfiles()['gemini'].model).toBe('gemini-2.5-flash');
  });

  it('2. Selección de proveedor: carga modelo/URL desde el perfil guardado y resetea catálogo', () => {
    component.catalogModels.set(mockModelsResponse.models);
    component.catalogFetchedAt.set('2026-10-24T14:30:00Z');
    component.modelSearchQuery.set('llama');
    component.modelTierFilter.set('free');

    component.selectProvider('deepseek');
    expect(component.provider()).toBe('deepseek');
    expect(component.baseUrl()).toBe('https://api.deepseek.com');
    expect(component.isCloudProvider()).toBe(true);
    expect(component.formMode()).toBe('cloud');
    expect(component.catalogModels()).toEqual([]);
    expect(component.catalogFetchedAt()).toBeNull();
    expect(component.modelSearchQuery()).toBe('');
    expect(component.modelTierFilter()).toBe('all');

    component.selectProvider('ollama_lan');
    expect(component.provider()).toBe('ollama_lan');
    expect(component.baseUrl()).toBe('http://10.20.1.50:11434/v1');
    expect(component.isCloudProvider()).toBe(false);
    expect(component.formMode()).toBe('lan');
  });

  it('3. Renderizado de logos de marca y fallback ante errores', () => {
    const compiled = fixture.nativeElement as HTMLElement;
    const logos = compiled.querySelectorAll<HTMLImageElement>('img.provider-logo');
    // 5 base + custom_llm_local (con icono); custom_docentes_bot queda tras «Ver más»
    expect(logos.length).toBe(6);
    expect(logos[0].getAttribute('src')).toBe('/ai-icons/gemini.svg');

    component.onIconError('gemini');
    fixture.detectChanges();
    expect(component.iconErrors()['gemini']).toBe(true);

    const firstCard = compiled.querySelector('.provider-card');
    const cloudIcon = firstCard?.querySelector('svg.provider-icon');
    expect(cloudIcon).toBeTruthy();
  });

  it('4. Consulta dinámica de modelos, búsqueda, filtrado por tier y selección', () => {
    component.selectProvider('openrouter');
    component.fetchModels();

    expect(mockService.listModels).toHaveBeenCalledWith({
      provider: 'openrouter',
      base_url: 'https://openrouter.ai/api/v1',
      api_key: '',
      refresh: false
    });
    expect(component.catalogModels().length).toBe(3);
    expect(component.filteredCatalogModels().length).toBe(3);

    // Búsqueda por query
    component.modelSearchQuery.set('claude');
    expect(component.filteredCatalogModels().length).toBe(1);
    expect(component.filteredCatalogModels()[0].id).toBe('anthropic/claude-3.5-sonnet');

    // Filtro por tier
    component.modelSearchQuery.set('');
    component.modelTierFilter.set('free');
    expect(component.filteredCatalogModels().length).toBe(2);

    component.modelTierFilter.set('paid');
    expect(component.filteredCatalogModels().length).toBe(1);

    // Selección de modelo
    component.selectCatalogModel(mockModelsResponse.models[0]);
    expect(component.model()).toBe('meta-llama/llama-3.3-70b-instruct:free');
  });

  it('5. Selección de perfiles de gobernanza (cuota y timeout)', () => {
    component.selectQuotaProfile(5);
    expect(component.hourlyQuota()).toBe(5);

    component.selectQuotaProfile(15);
    expect(component.hourlyQuota()).toBe(15);

    component.selectQuotaProfile(50);
    expect(component.hourlyQuota()).toBe(50);

    component.selectTimeoutProfile(20);
    expect(component.timeoutSeconds()).toBe(20);

    component.selectTimeoutProfile(45);
    expect(component.timeoutSeconds()).toBe(45);

    component.selectTimeoutProfile(90);
    expect(component.timeoutSeconds()).toBe(90);
  });

  it('6. Prueba de conexión: invoca testConnection() y almacena la respuesta con latency_ms en testResult()', () => {
    component.selectProvider('openrouter');
    component.testConnection();

    expect(mockService.testConnection).toHaveBeenCalledWith({
      provider: 'openrouter',
      model: 'openrouter/auto',
      base_url: 'https://openrouter.ai/api/v1',
      api_key: ''
    });
    expect(component.testResult()).toEqual({
      ok: true,
      provider: 'openrouter',
      model: 'openrouter/auto',
      latency_ms: 180,
      message: 'Conexión exitosa con el motor de inferencia'
    });
  });

  it('7. Guardado: invoca save(), llama a updateSettings() con el DTO y actualiza el estado', () => {
    component.selectProvider('openai');
    component.hourlyQuota.set(50);
    component.timeoutSeconds.set(45);
    component.save();

    expect(mockService.updateSettings).toHaveBeenCalledWith({
      enabled: true,
      provider: 'openai',
      model: 'gpt-4o-mini',
      base_url: 'https://api.openai.com/v1',
      api_key: '',
      clear_api_key: false,
      hourly_quota_per_user: 50,
      timeout_seconds: 45,
      custom_providers: [
        {
          id: 'custom_llm_local',
          label: 'LLM Local',
          icon_url: 'https://universidad.edu.bo/logo.svg',
          base_url: 'http://192.168.1.50:11434/v1',
          model: 'qwen2.5-coder:7b',
          api_key: '',
          clear_api_key: false
        },
        {
          id: 'custom_docentes_bot',
          label: 'Docentes Bot',
          icon_url: '',
          base_url: 'https://bot.universidad.edu.bo/v1',
          model: '',
          api_key: '',
          clear_api_key: false
        }
      ]
    });
    expect(component.toast()?.type).toBe('success');
    expect(component.toast()?.message).toBe('Configuración de Inteligencia Artificial guardada.');
  });

  it('8. Iniciales del avatar: 1 palabra usa 2 letras; 2+ palabras usan 1 letra por palabra', () => {
    expect(component.providerInitials('Groq')).toBe('GR');
    expect(component.providerInitials('Paquito AI')).toBe('PA');
    expect(component.providerInitials('Estudiante 1 Alumno')).toBe('EA');
    expect(component.providerInitials('  ')).toBe('');
  });

  it('9. Grilla colapsable: con más de 2 filas solo se renderizan 6 celdas y «Ver más» alterna', () => {
    // 5 base + 2 customs + 1 tarjeta de acción = 8 elementos
    expect(component.providerItems().length).toBe(8);
    expect(component.hasHiddenProviders()).toBe(true);
    expect(component.hiddenProviderCount()).toBe(2);
    expect(component.visibleProviderItems().length).toBe(6);

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.querySelectorAll('.provider-card').length).toBe(6);

    const moreBtn = compiled.querySelector<HTMLButtonElement>('.btn-more');
    expect(moreBtn?.textContent).toContain('Ver más proveedores (+2)');

    moreBtn?.click();
    fixture.detectChanges();
    expect(compiled.querySelectorAll('.provider-card').length).toBe(8);
    const lessBtn = compiled.querySelector<HTMLButtonElement>('.btn-more');
    expect(lessBtn?.textContent).toContain('Mostrar menos');

    // El custom sin icon_url muestra sus iniciales tras expandir
    const avatar = Array.from(compiled.querySelectorAll('.provider-avatar')).find((el) =>
      el.textContent?.includes('DB')
    );
    expect(avatar).toBeTruthy();

    lessBtn?.click();
    fixture.detectChanges();
    expect(compiled.querySelectorAll('.provider-card').length).toBe(6);
  });

  it('10. Cambiar de proveedor no pisa el modelo con gemini-2.5-flash cuando no está configurado', () => {
    component.selectProvider('deepseek');
    expect(component.model()).toBe('');

    fixture.detectChanges();
    const compiled = fixture.nativeElement as HTMLElement;
    const deepseekCard = Array.from(compiled.querySelectorAll('.provider-card')).find((card) =>
      card.textContent?.includes('DeepSeek')
    );
    expect(deepseekCard?.querySelector('.model-caption')?.textContent?.trim()).toBe('Sin configurar');

    // El proveedor con modelo guardado sí muestra el modelo configurado
    const geminiCard = Array.from(compiled.querySelectorAll('.provider-card')).find((card) =>
      card.textContent?.includes('Google Gemini')
    );
    expect(geminiCard?.querySelector('.model-caption')?.textContent?.trim()).toContain('Modelo:');
    expect(geminiCard?.querySelector('code')?.textContent?.trim()).toBe('gemini-2.5-flash');
  });

  it('11. Creación de proveedor custom: envía custom_providers con id vacío y selecciona el creado', () => {
    component.startAddCustom();
    expect(component.addingCustom()).toBe(true);
    expect(component.formMode()).toBe('custom-new');

    component.customLabel.set('Paquito AI');
    component.baseUrl.set('https://paquito.ai/v1');
    component.model.set('llama-3.1-8b');
    component.apiKey.set('secret-key');
    component.save();

    expect(mockService.updateSettings).toHaveBeenCalledWith({
      enabled: true,
      provider: 'gemini',
      model: '',
      base_url: '',
      api_key: '',
      clear_api_key: false,
      hourly_quota_per_user: 30,
      timeout_seconds: 60,
      custom_providers: [
        {
          id: 'custom_llm_local',
          label: 'LLM Local',
          icon_url: 'https://universidad.edu.bo/logo.svg',
          base_url: 'http://192.168.1.50:11434/v1',
          model: 'qwen2.5-coder:7b',
          api_key: '',
          clear_api_key: false
        },
        {
          id: 'custom_docentes_bot',
          label: 'Docentes Bot',
          icon_url: '',
          base_url: 'https://bot.universidad.edu.bo/v1',
          model: '',
          api_key: '',
          clear_api_key: false
        },
        {
          id: '',
          label: 'Paquito AI',
          icon_url: '',
          base_url: 'https://paquito.ai/v1',
          model: 'llama-3.1-8b',
          api_key: 'secret-key',
          clear_api_key: false
        }
      ]
    });

    // Tras la respuesta, el backend devuelve el ID generado y el componente lo selecciona
    expect(component.provider()).toBe('custom_paquito_ai');
    expect(component.addingCustom()).toBe(false);
    expect(component.formMode()).toBe('custom');
    expect(component.customLabel()).toBe('Paquito AI');
    expect(component.baseUrl()).toBe('https://paquito.ai/v1');
    expect(component.model()).toBe('llama-3.1-8b');
    expect(component.customProviders().some((c) => c.id === 'custom_paquito_ai')).toBe(true);
    expect(component.toast()?.type).toBe('success');
  });

  it('12. Orden causal del DOM en Tarjeta 2: #ai-api-key aparece antes que Modelo de Inferencia', () => {
    const compiled = fixture.nativeElement as HTMLElement;
    const connectionCard = compiled.querySelectorAll('.ia-card')[1];
    expect(connectionCard).toBeTruthy();

    const title = connectionCard.querySelector('.section-title')?.textContent?.trim();
    expect(title).toBe('Credenciales, Conexión y Modelo');

    const apiKeyInput = connectionCard.querySelector('#ai-api-key');
    expect(apiKeyInput).toBeTruthy();

    const selectedModelBar = connectionCard.querySelector('.selected-model-bar');
    expect(selectedModelBar).toBeTruthy();

    // Comparamos el orden en el árbol DOM: apiKeyInput debe preceder a selectedModelBar
    const position = apiKeyInput!.compareDocumentPosition(selectedModelBar!);
    expect(position & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it('13. Auto-fetch al seleccionar proveedor con API Key guardada', () => {
    mockService.listModels.mockClear();

    // Seleccionamos OpenRouter (que en mockProfiles tiene has_api_key: true)
    component.selectProvider('openrouter');
    fixture.detectChanges();

    expect(mockService.listModels).toHaveBeenCalledWith({
      provider: 'openrouter',
      base_url: 'https://openrouter.ai/api/v1',
      api_key: '',
      refresh: false
    });
    expect(component.catalogModels().length).toBe(3);
  });

  it('14. No auto-fetch en proveedor sin API Key', () => {
    mockService.listModels.mockClear();

    // Seleccionamos DeepSeek (que en mockProfiles tiene has_api_key: false y apiKey() está vacío)
    component.selectProvider('deepseek');
    fixture.detectChanges();

    expect(mockService.listModels).not.toHaveBeenCalled();
    expect(component.canFetchModels()).toBe(false);
    expect(component.catalogModels().length).toBe(0);
  });

  it('15. Auto-fetch reactivo al ingresar API Key nueva con debounce de 500ms', () => {
    vi.useFakeTimers();
    try {
      component.selectProvider('deepseek');
      mockService.listModels.mockClear();

      component.onApiKeyInput('sk-test-1234567890');
      expect(mockService.listModels).not.toHaveBeenCalled();

      // Avanzamos el debounce
      vi.advanceTimersByTime(500);

      expect(mockService.listModels).toHaveBeenCalledWith({
        provider: 'deepseek',
        base_url: 'https://api.deepseek.com',
        api_key: 'sk-test-1234567890',
        refresh: false
      });
      expect(component.catalogModels().length).toBe(3);
    } finally {
      vi.useRealTimers();
    }
  });

  it('16. Distinción en grilla (EN USO vs Configurado vs Sin configurar) y banner de conmutación', () => {
    const compiled = fixture.nativeElement as HTMLElement;
    const cards = compiled.querySelectorAll('.provider-card');

    // 1. Tarjeta gemini (mockSettings.provider === 'gemini', enabled: true)
    const geminiItem = component.providerItems().find((p) => p.id === 'gemini');
    expect(geminiItem?.isActiveInBackend).toBe(true);
    expect(geminiItem?.hasSavedKey).toBe(true);

    const geminiCard = Array.from(cards).find((c) => c.textContent?.includes('Google Gemini'));
    expect(geminiCard).toBeTruthy();
    const activeBadge = geminiCard?.querySelector('.active-engine-badge');
    expect(activeBadge).toBeTruthy();
    expect(activeBadge?.textContent?.trim()).toBe('EN USO');

    // 2. Tarjeta openrouter (has_api_key: true, model: 'openrouter/auto', pero NO es el activo en backend)
    const openrouterItem = component.providerItems().find((p) => p.id === 'openrouter');
    expect(openrouterItem?.isActiveInBackend).toBe(false);
    expect(openrouterItem?.hasSavedKey).toBe(true);

    const openrouterCard = Array.from(cards).find((c) => c.textContent?.includes('OpenRouter'));
    expect(openrouterCard).toBeTruthy();
    expect(openrouterCard?.querySelector('.active-engine-badge')).toBeFalsy();
    expect(openrouterCard?.querySelector('.saved-key-pill')?.textContent?.trim()).toBe('Clave guardada');

    // 3. Tarjeta deepseek (has_api_key: false, model: '')
    const deepseekItem = component.providerItems().find((p) => p.id === 'deepseek');
    expect(deepseekItem?.isActiveInBackend).toBe(false);
    expect(deepseekItem?.hasSavedKey).toBe(false);

    const deepseekCard = Array.from(cards).find((c) => c.textContent?.includes('DeepSeek'));
    expect(deepseekCard?.querySelector('.sin-configurar')?.textContent?.trim()).toBe('Sin configurar');

    // 4. Al seleccionar OpenRouter, se muestra el banner de conmutación sin requerir reingresar clave
    component.selectProvider('openrouter');
    fixture.detectChanges();
    expect(component.showSwitchWithSavedKeyBanner()).toBe(true);

    const banner = compiled.querySelector('.switch-saved-key-banner');
    expect(banner).toBeTruthy();
    expect(banner?.textContent).toContain('Este proveedor ya tiene su API Key cifrada guardada');
    expect(banner?.textContent).toContain(component.savedKey().masked);
  });
});
