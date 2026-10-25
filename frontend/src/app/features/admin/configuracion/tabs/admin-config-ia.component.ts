import { Component, inject, signal, computed, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import {
  LucideBot,
  LucideServer,
  LucideCloud,
  LucideKey,
  LucideActivity,
  LucideSave,
  LucideCheckCircle2,
  LucideAlertCircle,
  LucideRefreshCw,
  LucideSearch,
  LucidePlus,
  LucideLock,
  LucideWifi
} from '@lucide/angular';
import {
  AdminConfigIaService,
  AISettingsDTO,
  AICustomProviderDTO,
  UpdateAISettingsDTO,
  TestAIConnectionPayload,
  TestAIConnectionResponse,
  AIProviderType,
  AIModelItemDTO,
  ListAIModelsPayload
} from '../admin-config-ia.service';

export interface AIProviderCatalogItem {
  id: AIProviderType;
  label: string;
  badge: string;
  iconUrl: string;
  defaultBaseUrl: string;
  isCloud: boolean;
}

export const PROVIDER_CATALOG: AIProviderCatalogItem[] = [
  {
    id: 'gemini',
    label: 'Google Gemini',
    badge: 'Cloud · Nivel gratuito',
    iconUrl: '/ai-icons/gemini.svg',
    defaultBaseUrl: 'https://generativelanguage.googleapis.com/v1beta/openai',
    isCloud: true
  },
  {
    id: 'openai',
    label: 'OpenAI',
    badge: 'Cloud · Estándar',
    iconUrl: '/ai-icons/openai.svg',
    defaultBaseUrl: 'https://api.openai.com/v1',
    isCloud: true
  },
  {
    id: 'openrouter',
    label: 'OpenRouter',
    badge: 'Cloud · Variedad de modelos y ruteo',
    iconUrl: '/ai-icons/openrouter.svg',
    defaultBaseUrl: 'https://openrouter.ai/api/v1',
    isCloud: true
  },
  {
    id: 'deepseek',
    label: 'DeepSeek',
    badge: 'Cloud · Código y razonamiento',
    iconUrl: '/ai-icons/deepseek.svg',
    defaultBaseUrl: 'https://api.deepseek.com',
    isCloud: true
  },
  {
    id: 'ollama_lan',
    label: 'Servidor Local (LAN / Ollama / vLLM)',
    badge: 'On-Premise · GPU Universitaria',
    iconUrl: '/ai-icons/ollama_lan.svg',
    defaultBaseUrl: 'http://10.20.1.50:11434/v1',
    isCloud: false
  }
];

/** Celdas visibles de la grilla: 3 columnas x 2 filas. */
export const PROVIDER_GRID_COLLAPSED_COUNT = 6;

export type ProviderFormMode = 'cloud' | 'lan' | 'custom' | 'custom-new';

/** Elemento de la grilla de proveedores: base, personalizado o tarjeta de acción. */
export interface ProviderGridItem {
  kind: 'base' | 'custom' | 'add';
  id: string;
  label: string;
  badge: string;
  iconUrl: string;
  model: string;
  isCloud: boolean;
  isActiveInBackend: boolean;
  hasSavedKey: boolean;
}

export interface GovernanceQuotaProfile {
  value: number;
  title: string;
  description: string;
  badge?: string;
}

export const GOVERNANCE_QUOTA_PROFILES: GovernanceQuotaProfile[] = [
  {
    value: 5,
    title: 'Ligero (Cuentas gratuitas)',
    description: 'Consultas puntuales cuidando límites gratuitos.'
  },
  {
    value: 15,
    title: 'Docente estándar',
    description: 'Diseño de 3 a 5 prácticas completas por hora.',
    badge: 'Recomendado'
  },
  {
    value: 50,
    title: 'Intensivo',
    description: 'Carga masiva de ejercicios al inicio del semestre.'
  }
];

export interface GovernanceTimeoutProfile {
  value: number;
  title: string;
  description: string;
  badge?: string;
}

export const GOVERNANCE_TIMEOUT_PROFILES: GovernanceTimeoutProfile[] = [
  {
    value: 20,
    title: 'Rápido',
    description: 'Modelos ágiles (Gemini Flash, GPT-4o mini, DeepSeek Chat).'
  },
  {
    value: 45,
    title: 'Estándar',
    description: 'Generación de ejercicios completos con casos de prueba.',
    badge: 'Recomendado'
  },
  {
    value: 90,
    title: 'Razonamiento / GPU Local',
    description: 'Modelos paso a paso (DeepSeek R1) o servidor Ollama bajo carga.'
  }
];

/** Clave interna de errores de icono para la previsualización del borrador. */
const DRAFT_ICON_ERROR_KEY = 'custom-draft-icon';

const CLOUD_PROVIDER_IDS: ReadonlySet<string> = new Set(['gemini', 'openai', 'openrouter', 'deepseek']);

@Component({
  selector: 'admin-config-ia',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    FormFieldComponent,
    DateTextPipe,
    MachineDataDirective,
    LucideBot,
    LucideServer,
    LucideCloud,
    LucideKey,
    LucideActivity,
    LucideSave,
    LucideCheckCircle2,
    LucideAlertCircle,
    LucideRefreshCw,
    LucideSearch,
    LucidePlus,
    LucideLock,
    LucideWifi
  ],
  templateUrl: './admin-config-ia.component.html',
  styleUrls: ['./admin-config-ia.component.scss']
})
export class AdminConfigIaComponent implements OnInit, OnDestroy {
  private readonly service = inject(AdminConfigIaService);

  readonly providerCatalog = PROVIDER_CATALOG;
  readonly settings = this.service.settings;
  readonly quotaProfiles = GOVERNANCE_QUOTA_PROFILES;
  readonly timeoutProfiles = GOVERNANCE_TIMEOUT_PROFILES;

  readonly loading = signal(false);
  readonly saving = signal(false);
  readonly testing = signal(false);
  readonly toast = signal<{ type: 'success' | 'error'; message: string } | null>(null);
  readonly testResult = signal<TestAIConnectionResponse | null>(null);

  readonly enabled = signal<boolean>(false);
  readonly provider = signal<AIProviderType>('gemini');
  readonly model = signal<string>('');
  readonly baseUrl = signal<string>('https://generativelanguage.googleapis.com/v1beta/openai');
  readonly apiKey = signal<string>('');
  readonly clearApiKey = signal<boolean>(false);
  readonly hourlyQuota = signal<number>(15);
  readonly timeoutSeconds = signal<number>(30);

  // Signals para auto-descubrimiento, catálogo y selector directo
  readonly catalogError = signal<string | null>(null);
  readonly manualModelMode = signal<boolean>(false);
  private autoFetchTimer: ReturnType<typeof setTimeout> | null = null;

  // Estado de los proveedores personalizados y perfiles base guardados
  readonly customProviders = signal<AICustomProviderDTO[]>([]);
  readonly providerProfiles = signal<Record<string, { model: string; base_url: string; has_api_key: boolean; api_key_masked: string }>>({});
  readonly addingCustom = signal<boolean>(false);
  readonly customLabel = signal<string>('');
  readonly customIconUrl = signal<string>('');

  // Validaciones inline
  readonly customLabelTouched = signal<boolean>(false);
  readonly customIconUrlTouched = signal<boolean>(false);
  readonly baseUrlTouched = signal<boolean>(false);
  readonly modelTouched = signal<boolean>(false);
  readonly hourlyQuotaTouched = signal<boolean>(false);
  readonly timeoutSecondsTouched = signal<boolean>(false);

  readonly customLabelError = computed<string>(() => {
    if (!this.isCustomForm()) return '';
    const val = this.customLabel().trim();
    if (!val) {
      return this.customLabelTouched() ? 'El nombre del proveedor es obligatorio.' : '';
    }
    if (val.length > 60) {
      return 'El nombre no puede superar los 60 caracteres.';
    }
    return '';
  });

  readonly customIconUrlError = computed<string>(() => {
    if (!this.isCustomForm()) return '';
    const raw = this.customIconUrl().trim();
    if (!raw) return '';
    if (raw.length > 500) {
      return 'La URL del logo no puede superar los 500 caracteres.';
    }
    if (!/^https?:\/\//i.test(raw)) {
      return 'La URL debe comenzar con http:// o https://';
    }
    return '';
  });

  readonly baseUrlError = computed<string>(() => {
    if (this.formMode() === 'cloud') return '';
    const raw = this.baseUrl().trim();
    if (!raw) {
      return this.baseUrlTouched() ? 'La URL base de inferencia es obligatoria.' : '';
    }
    if (!/^https?:\/\//i.test(raw)) {
      return 'La URL debe comenzar con http:// o https://';
    }
    return '';
  });

  readonly modelError = computed<string>(() => {
    const val = this.model().trim();
    if (!val && this.enabled()) {
      return 'El modelo es obligatorio para habilitar la IA en la institución.';
    }
    return '';
  });

  readonly hourlyQuotaError = computed<string>(() => {
    const val = Number(this.hourlyQuota());
    if (!Number.isInteger(val) || val < 1 || val > 200) {
      return 'La cuota horaria debe ser un número entero entre 1 y 200 solicitudes/hora.';
    }
    return '';
  });

  readonly timeoutSecondsError = computed<string>(() => {
    const val = Number(this.timeoutSeconds());
    if (!Number.isInteger(val) || val < 5 || val > 120) {
      return 'El timeout debe ser un número entero entre 5 y 120 segundos.';
    }
    return '';
  });

  // Grilla colapsable (2 filas de 3 columnas)
  readonly gridExpanded = signal<boolean>(false);

  // Icon error fallback
  readonly iconErrors = signal<Record<string, boolean>>({});

  // Dynamic model catalog
  readonly loadingModels = signal(false);
  readonly catalogModels = signal<AIModelItemDTO[]>([]);
  readonly catalogFetchedAt = signal<string | null>(null);
  readonly modelSearchQuery = signal<string>('');
  readonly modelTierFilter = signal<'all' | 'free' | 'paid'>('all');

  readonly filteredCatalogModels = computed(() => {
    const query = this.modelSearchQuery().trim().toLowerCase();
    const filter = this.modelTierFilter();
    return this.catalogModels().filter((m) => {
      const matchesQuery =
        !query ||
        m.id.toLowerCase().includes(query) ||
        (m.name && m.name.toLowerCase().includes(query));
      const matchesTier = filter === 'all' || m.tier === filter;
      return matchesQuery && matchesTier;
    });
  });

  readonly isCloudProvider = computed(() => CLOUD_PROVIDER_IDS.has(this.provider()));

  readonly currentCatalogItem = computed(
    () => this.providerCatalog.find((p) => p.id === this.provider()) ?? this.providerCatalog[0]
  );

  /** Modo del formulario de conexión según el tipo de proveedor activo. */
  readonly formMode = computed<ProviderFormMode>(() => {
    if (this.addingCustom()) {
      return 'custom-new';
    }
    const active = this.provider();
    if (active === 'ollama_lan') {
      return 'lan';
    }
    if (active.startsWith('custom_')) {
      return 'custom';
    }
    return 'cloud';
  });

  readonly isCustomForm = computed(() => {
    const mode = this.formMode();
    return mode === 'custom' || mode === 'custom-new';
  });

  /** URL oficial de solo lectura del proveedor en la nube activo. */
  readonly officialBaseUrl = computed(
    () => this.providerProfiles()[this.provider()]?.base_url || this.currentCatalogItem().defaultBaseUrl
  );

  /** Clave guardada del proveedor activo (perfil base o custom). */
  readonly savedKey = computed(() => {
    const active = this.provider();
    if (active.startsWith('custom_')) {
      const cp = this.customProviders().find((c) => c.id === active);
      return { has: Boolean(cp?.has_api_key), masked: cp?.api_key_masked ?? '' };
    }
    const profile = this.providerProfiles()[active];
    if (profile) {
      return { has: profile.has_api_key, masked: profile.api_key_masked };
    }
    return {
      has: this.settings()?.has_api_key ?? false,
      masked: this.settings()?.api_key_masked ?? ''
    };
  });

  /** El catálogo de modelos se habilita con API Key cargada (o URL en LAN). */
  readonly canFetchModels = computed(() => {
    switch (this.formMode()) {
      case 'cloud': {
        const profile = this.providerProfiles()[this.provider()];
        return this.apiKey().trim() !== '' || profile?.has_api_key === true;
      }
      case 'lan':
        return this.baseUrl().trim() !== '';
      case 'custom': {
        const cp = this.customProviders().find((c) => c.id === this.provider());
        return this.baseUrl().trim() !== '' && (this.apiKey().trim() !== '' || cp?.has_api_key === true);
      }
      case 'custom-new':
        return this.baseUrl().trim() !== '' && this.apiKey().trim() !== '';
    }
  });

  /** Prueba de conexión disponible con los campos obligatorios cargados. */
  readonly canTestConnection = computed(() => {
    if (this.addingCustom()) {
      return (
        this.customLabel().trim() !== '' &&
        this.baseUrl().trim() !== '' &&
        this.apiKey().trim() !== ''
      );
    }
    if (this.formMode() === 'lan') {
      return this.baseUrl().trim() !== '';
    }
    return true;
  });

  readonly apiKeyPlaceholder = computed(() => {
    const mode = this.formMode();
    if (mode === 'lan') {
      return 'Opcional: dejá vacío si el servidor local no requiere clave';
    }
    return 'Ingresa la API Key (obligatoria para este proveedor)';
  });

  /** Elementos completos de la grilla: 5 base + customs + tarjeta de acción. */
  readonly providerItems = computed<ProviderGridItem[]>(() => {
    const profiles = this.providerProfiles();
    const activeBackendProvider = this.settings()?.provider;
    const isAdding = this.addingCustom();

    const base: ProviderGridItem[] = this.providerCatalog.map((item) => ({
      kind: 'base',
      id: item.id,
      label: item.label,
      badge: item.badge,
      iconUrl: item.iconUrl,
      model: profiles[item.id]?.model ?? '',
      isCloud: item.isCloud,
      isActiveInBackend: !isAdding && activeBackendProvider === item.id,
      hasSavedKey: profiles[item.id]?.has_api_key === true
    }));
    const customs: ProviderGridItem[] = this.customProviders().map((cp) => ({
      kind: 'custom',
      id: cp.id,
      label: cp.label,
      badge: 'Personalizado · Compatible OpenAI',
      iconUrl: cp.icon_url ?? '',
      model: cp.model ?? '',
      isCloud: false,
      isActiveInBackend: !isAdding && activeBackendProvider === cp.id,
      hasSavedKey: cp.has_api_key === true
    }));
    return [
      ...base,
      ...customs,
      {
        kind: 'add',
        id: 'add-provider',
        label: 'Agregar proveedor',
        badge: 'Compatible OpenAI · URL propia',
        iconUrl: '',
        model: '',
        isCloud: false,
        isActiveInBackend: false,
        hasSavedKey: false
      }
    ];
  });

  readonly hasHiddenProviders = computed(
    () => this.providerItems().length > PROVIDER_GRID_COLLAPSED_COUNT
  );

  readonly hiddenProviderCount = computed(() =>
    Math.max(0, this.providerItems().length - PROVIDER_GRID_COLLAPSED_COUNT)
  );

  readonly visibleProviderItems = computed(() => {
    const all = this.providerItems();
    return this.gridExpanded() ? all : all.slice(0, PROVIDER_GRID_COLLAPSED_COUNT);
  });

  /** Indica si se seleccionó un proveedor distinto al activo institucional que ya cuenta con API Key guardada */
  readonly showSwitchWithSavedKeyBanner = computed<boolean>(() => {
    if (this.addingCustom()) return false;
    const currentActive = this.settings()?.provider;
    const selected = this.provider();
    return selected !== currentActive && this.savedKey().has;
  });

  ngOnInit(): void {
    this.loading.set(true);
    this.service.getSettings().subscribe({
      next: (dto) => {
        this.syncFromSettings(dto);
        this.loading.set(false);
        if (this.canFetchModels()) {
          this.fetchModels(false, true);
        }
      },
      error: (err: unknown) => {
        this.loading.set(false);
        this.showToast('error', this.service.resolveError(err));
      }
    });
  }

  ngOnDestroy(): void {
    this.clearAutoFetchTimer();
  }

  private clearAutoFetchTimer(): void {
    if (this.autoFetchTimer !== null) {
      clearTimeout(this.autoFetchTimer);
      this.autoFetchTimer = null;
    }
  }

  onIconError(id: string): void {
    this.iconErrors.update((prev) => ({ ...prev, [id]: true }));
  }

  /** Iniciales estilo EA: 1 palabra → 2 primeras letras; 2+ palabras → 1 letra por palabra (sin numerales). */
  providerInitials(name: string): string {
    const words = name
      .trim()
      .split(/\s+/)
      .filter((w) => w.length > 0 && !/^\d+$/.test(w));
    if (words.length === 0) {
      return '';
    }
    if (words.length === 1) {
      return words[0].substring(0, 2).toUpperCase();
    }
    return (words[0].charAt(0) + words[1].charAt(0)).toUpperCase();
  }

  selectProvider(id: string): void {
    this.clearAutoFetchTimer();
    this.catalogError.set(null);
    this.manualModelMode.set(false);
    this.addingCustom.set(false);
    this.provider.set(id as AIProviderType);

    if (id.startsWith('custom_')) {
      const cp = this.customProviders().find((c) => c.id === id);
      this.customLabel.set(cp?.label ?? '');
      this.customIconUrl.set(cp?.icon_url ?? '');
      this.baseUrl.set(cp?.base_url ?? '');
      this.model.set(cp?.model ?? '');
    } else {
      const item = this.providerCatalog.find((c) => c.id === id) ?? this.providerCatalog[0];
      const profile = this.providerProfiles()[id];
      this.baseUrl.set(profile?.base_url || item.defaultBaseUrl);
      this.model.set(profile?.model ?? '');
      this.customLabel.set('');
      this.customIconUrl.set('');
    }

    this.apiKey.set('');
    this.clearApiKey.set(false);
    this.testResult.set(null);
    this.catalogModels.set([]);
    this.catalogFetchedAt.set(null);
    this.modelSearchQuery.set('');
    this.modelTierFilter.set('all');
    this.customLabelTouched.set(false);
    this.customIconUrlTouched.set(false);
    this.baseUrlTouched.set(false);
    this.modelTouched.set(false);

    if (this.canFetchModels()) {
      this.fetchModels(false, true);
    }
  }

  onApiKeyInput(value: string): void {
    this.apiKey.set(value);
    this.clearAutoFetchTimer();
    this.catalogError.set(null);

    // Si la clave tiene al menos 8 caracteres (o hay clave guardada) y canFetchModels es true, disparar auto-fetch con debounce
    const trimmed = value.trim();
    const hasEnoughKey = trimmed.length >= 8 || this.savedKey().has;
    if (this.canFetchModels() && hasEnoughKey) {
      this.autoFetchTimer = setTimeout(() => {
        this.fetchModels(false, true);
      }, 500);
    }
  }

  onBaseUrlInput(value: string): void {
    this.baseUrl.set(value);
    this.clearAutoFetchTimer();
    this.catalogError.set(null);

    const trimmed = value.trim();
    const isValidUrl = /^https?:\/\//i.test(trimmed);
    if (this.canFetchModels() && (this.formMode() !== 'lan' || isValidUrl)) {
      this.autoFetchTimer = setTimeout(() => {
        this.fetchModels(false, true);
      }, 500);
    }
  }

  /** Entra en modo de alta de proveedor personalizado sin tocar el proveedor activo. */
  startAddCustom(): void {
    this.clearAutoFetchTimer();
    this.catalogError.set(null);
    this.manualModelMode.set(false);
    this.addingCustom.set(true);
    this.customLabel.set('');
    this.customIconUrl.set('');
    this.baseUrl.set('');
    this.model.set('');
    this.apiKey.set('');
    this.clearApiKey.set(false);
    this.customLabelTouched.set(false);
    this.customIconUrlTouched.set(false);
    this.baseUrlTouched.set(false);
    this.modelTouched.set(false);
    this.iconErrors.update((prev) => ({ ...prev, [DRAFT_ICON_ERROR_KEY]: false }));
    this.testResult.set(null);
    this.catalogModels.set([]);
    this.catalogFetchedAt.set(null);
    this.modelSearchQuery.set('');
    this.modelTierFilter.set('all');
  }

  onCustomIconUrlChange(value: string): void {
    this.customIconUrl.set(value);
    this.iconErrors.update((prev) => ({ ...prev, [DRAFT_ICON_ERROR_KEY]: false }));
  }

  /** Elimina el proveedor custom de la lista local; el guardado lo aplica «Guardar Configuración de IA». */
  removeCustomProvider(): void {
    const active = this.provider();
    if (!active.startsWith('custom_')) {
      return;
    }
    const remaining = this.customProviders().filter((c) => c.id !== active);
    this.customProviders.set(remaining);

    // Volver al primer proveedor base con modelo configurado, si lo hay.
    const fallback =
      this.providerCatalog.find((p) => (this.providerProfiles()[p.id]?.model ?? '') !== '')?.id ??
      'gemini';
    this.selectProvider(fallback);
    this.showToast('success', 'Proveedor quitado de la lista. Guardá la configuración para aplicar el cambio.');
  }

  selectCatalogModel(m: AIModelItemDTO): void {
    this.model.set(m.id);
  }

  /** ID provisorio (formato custom_<slug>) para probar el borrador antes de guardarlo. */
  private draftProviderId(): AIProviderType {
    let slug = '';
    for (const ch of this.customLabel().trim().toLowerCase()) {
      if ((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')) {
        slug += ch;
      } else if (ch === ' ' || ch === '-' || ch === '_') {
        slug += '_';
      }
    }
    slug = slug.replace(/^_+|_+$/g, '');
    if (slug === '' || slug === 'custom') {
      slug = 'draft';
    }
    if (slug.length > 50) {
      slug = slug.slice(0, 50);
    }
    return `custom_${slug}` as AIProviderType;
  }

  private effectiveProviderId(): AIProviderType {
    return this.addingCustom() ? this.draftProviderId() : this.provider();
  }

  fetchModels(refresh = false, silent = false): void {
    const payload: ListAIModelsPayload = {
      provider: this.effectiveProviderId(),
      base_url: this.baseUrl(),
      api_key: this.apiKey(),
      refresh
    };
    this.catalogError.set(null);
    this.loadingModels.set(true);
    this.service.listModels(payload).subscribe({
      next: (res) => {
        this.catalogModels.set(res.models || []);
        this.catalogFetchedAt.set(res.fetched_at || new Date().toISOString());
        this.loadingModels.set(false);
      },
      error: (err: unknown) => {
        this.loadingModels.set(false);
        const resolvedError = this.service.resolveError(err);
        this.catalogError.set(resolvedError);
        if (!silent) {
          this.showToast('error', resolvedError);
        }
      }
    });
  }

  selectQuotaProfile(val: number): void {
    this.hourlyQuota.set(val);
  }

  selectTimeoutProfile(val: number): void {
    this.timeoutSeconds.set(val);
  }

  onToggleEnabled(e: Event): void {
    const input = e.target as HTMLInputElement;
    this.enabled.set(input.checked);
  }

  testConnection(): void {
    const payload: TestAIConnectionPayload = {
      provider: this.effectiveProviderId(),
      model: this.model(),
      base_url: this.baseUrl(),
      api_key: this.apiKey()
    };
    this.testing.set(true);
    this.testResult.set(null);
    this.service.testConnection(payload).subscribe({
      next: (res) => {
        this.testResult.set(res);
        this.testing.set(false);
      },
      error: (err: unknown) => {
        this.showToast('error', this.service.resolveError(err));
        this.testing.set(false);
      }
    });
  }

  save(): void {
    this.customLabelTouched.set(true);
    this.customIconUrlTouched.set(true);
    this.baseUrlTouched.set(true);
    this.modelTouched.set(true);
    this.hourlyQuotaTouched.set(true);
    this.timeoutSecondsTouched.set(true);

    if (
      this.customLabelError() ||
      this.customIconUrlError() ||
      this.baseUrlError() ||
      this.modelError() ||
      this.hourlyQuotaError() ||
      this.timeoutSecondsError()
    ) {
      this.showToast('error', 'Por favor, corregí los errores en el formulario antes de guardar.');
      return;
    }

    const dto = this.buildUpdateDto();
    const wasAdding = this.addingCustom();
    const knownCustomIds = new Set(this.customProviders().map((c) => c.id));

    this.saving.set(true);
    this.service.updateSettings(dto).subscribe({
      next: (saved) => {
        this.syncFromSettings(saved);
        if (wasAdding) {
          // El backend genera el ID custom_<slug>; seleccionar el recién creado.
          const created = (saved.custom_providers ?? []).find((c) => !knownCustomIds.has(c.id));
          if (created) {
            this.selectProvider(created.id);
          }
        }
        this.saving.set(false);
        this.showToast('success', 'Configuración de Inteligencia Artificial guardada.');
      },
      error: (err: unknown) => {
        this.showToast('error', this.service.resolveError(err));
        this.saving.set(false);
      }
    });
  }

  private buildUpdateDto(): UpdateAISettingsDTO {
    const adding = this.addingCustom();
    const active = this.provider();

    const customProviders: AICustomProviderDTO[] = this.customProviders().map((cp) => {
      const isActive = !adding && cp.id === active;
      return {
        id: cp.id,
        label: isActive ? this.customLabel().trim() : cp.label,
        icon_url: isActive ? this.customIconUrl().trim() : cp.icon_url ?? '',
        base_url: isActive ? this.baseUrl().trim() : cp.base_url,
        model: isActive ? this.model().trim() : cp.model,
        api_key: isActive ? this.apiKey().trim() : '',
        clear_api_key: isActive ? this.clearApiKey() : false
      };
    });

    if (adding) {
      customProviders.push({
        id: '',
        label: this.customLabel().trim(),
        icon_url: this.customIconUrl().trim(),
        base_url: this.baseUrl().trim(),
        model: this.model().trim(),
        api_key: this.apiKey().trim(),
        clear_api_key: false
      });
    }

    return {
      enabled: this.enabled(),
      provider: active,
      model: adding ? '' : this.model().trim(),
      base_url: adding ? '' : this.baseUrl().trim(),
      api_key: adding ? '' : this.apiKey().trim(),
      clear_api_key: adding ? false : this.clearApiKey(),
      hourly_quota_per_user: this.hourlyQuota(),
      timeout_seconds: this.timeoutSeconds(),
      custom_providers: customProviders
    };
  }

  private syncFromSettings(dto: AISettingsDTO): void {
    const customProviders = dto.custom_providers ?? [];
    this.providerProfiles.set(dto.provider_profiles ?? {});
    this.customProviders.set(customProviders);
    this.enabled.set(dto.enabled);
    this.provider.set(dto.provider);
    this.model.set(dto.model);
    this.baseUrl.set(dto.base_url);
    this.apiKey.set('');
    this.clearApiKey.set(false);
    this.hourlyQuota.set(dto.hourly_quota_per_user);
    this.timeoutSeconds.set(dto.timeout_seconds);

    const activeCustom = customProviders.find((c) => c.id === dto.provider);
    this.customLabel.set(activeCustom?.label ?? '');
    this.customIconUrl.set(activeCustom?.icon_url ?? '');
    this.addingCustom.set(false);
  }

  showToast(type: 'success' | 'error', message: string): void {
    this.toast.set({ type, message });
    setTimeout(() => {
      if (this.toast()?.message === message) {
        this.toast.set(null);
      }
    }, 4000);
  }
}
