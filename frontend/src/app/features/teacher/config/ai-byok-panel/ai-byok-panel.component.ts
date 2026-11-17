import {
  ChangeDetectionStrategy,
  Component,
  OnInit,
  computed,
  inject,
  output,
  signal
} from '@angular/core';
import { FormsModule } from '@angular/forms';
import {
  LucideAlertCircle,
  LucideCheckCircle2,
  LucideCloud,
  LucideKeyRound,
  LucideLoader2,
  LucidePlus,
  LucideRefreshCw,
  LucideSave,
  LucideSearch,
  LucideTrash2,
  LucideWifi
} from '@lucide/angular';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { ConfirmModalComponent } from '@shared/components/confirm-modal/confirm-modal.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import {
  AIByokModel,
  AIUserCustomProviderDTO,
  AIUserOverrideDTO,
  TeacherAiCopilotService
} from '../../services/teacher-ai-copilot.service';
import { BYOK_PROVIDERS, DEFAULT_BYOK_PROVIDER } from '../ai-byok.config';
import type { ByokProviderOption } from '../ai-byok.config';

/**
 * Configuración de la credencial personal de IA del docente (BYOK).
 *
 * Es un componente autocontenido: resuelve sus propias lecturas y escrituras
 * contra `/api/v1/ai/my-settings`, de modo que la vista de configuración y el
 * modal del copiloto lo reutilizan sin duplicar lógica ni formularios.
 *
 * La jerarquía es explícita y no automática: mientras el interruptor esté
 * apagado o no haya clave guardada, el docente opera con el motor
 * institucional; al activarlo, su clave prevalece.
 */
@Component({
  selector: 'ai-byok-panel',
  standalone: true,
  imports: [
    FormsModule,
    FormFieldComponent,
    ModalShellComponent,
    ConfirmModalComponent,
    MachineDataDirective,
    DateTextPipe,
    LucideKeyRound,
    LucideLoader2,
    LucideCloud,
    LucideRefreshCw,
    LucideSearch,
    LucideTrash2,
    LucideCheckCircle2,
    LucideAlertCircle,
      LucideWifi,
  LucideSave,
  LucidePlus
  ],
  templateUrl: './ai-byok-panel.component.html',
  styleUrl: './ai-byok-panel.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class AiByokPanelComponent implements OnInit {
  private readonly copilotService = inject(TeacherAiCopilotService);

  /** Notifica al contenedor que la configuración cambió y debe refrescar estado. */
  readonly changed = output<AIUserOverrideDTO>();

  readonly loading = signal(true);
  readonly enabled = signal(false);
  readonly provider = signal(DEFAULT_BYOK_PROVIDER);
  readonly apiKey = signal('');
  readonly model = signal('');

  readonly saving = signal(false);
  readonly testing = signal(false);
  readonly clearing = signal(false);
  readonly loadingModels = signal(false);

  readonly feedbackMessage = signal<string | null>(null);
  readonly feedbackOk = signal<boolean | null>(null);

  readonly availableModels = signal<AIByokModel[]>([]);
  readonly modelsFetchedAt = signal<string>('');
  readonly modelTierFilter = signal<'all' | 'free' | 'paid'>('all');
  readonly modelSearch = signal('');
  readonly manualModelMode = signal(false);
  readonly iconErrors = signal<Record<string, boolean>>({});

  readonly current = signal<AIUserOverrideDTO | null>(null);

  // Alta de un proveedor propio compatible con OpenAI. La URL queda acotada a
  // https público por el backend: el docente no configura la red del servidor.
  readonly showAddProvider = signal(false);
  readonly newProviderLabel = signal('');
  readonly newProviderBaseURL = signal('');
  readonly newProviderApiKey = signal('');
  readonly newProviderModel = signal('');
  readonly newProviderModels = signal<AIByokModel[]>([]);
  readonly newProviderTesting = signal(false);
  readonly newProviderLoadingModels = signal(false);
  readonly newProviderSaving = signal(false);
  readonly newProviderFeedback = signal<string | null>(null);
  readonly newProviderFeedbackOk = signal<boolean | null>(null);
  readonly providerToDelete = signal<AIUserCustomProviderDTO | null>(null);
  readonly deletingProvider = signal(false);

  readonly newProviderId = computed(() => {
    const slug = this.newProviderLabel()
      .trim()
      .toLowerCase()
      .normalize('NFD')
      .replace(/[\u0300-\u036f]/g, '')
      .replace(/[^a-z0-9]+/g, '_')
      .replace(/^_+|_+$/g, '')
      .slice(0, 50);
    return slug ? `custom_${slug}` : '';
  });

  readonly canQueryNewProvider = computed(
    () => this.newProviderId() !== '' && this.newProviderBaseURL().trim().startsWith('https://')
  );

  /**
   * Catálogo visible para el docente: los cuatro proveedores base más los
   * proveedores personalizados que registró la institución. En estos últimos
   * el endpoint lo define la institución; el docente solo aporta su clave.
   */
  readonly providers = computed<ByokProviderOption[]>(() => {
    const customs = (this.current()?.custom_providers ?? []).map<ByokProviderOption>((cp) => ({
      value: cp.id,
      label: cp.label,
      badge: cp.origin === 'personal' ? 'Personalizado · Propio' : 'Personalizado · Institución',
      iconUrl: cp.icon_url ?? '',
      isCloud: true,
      placeholderKey: 'sk-...'
    }));
    return [...BYOK_PROVIDERS, ...customs];
  });

  /** Motor que la institución tiene activo ahora mismo. */
  readonly institutionalProviderId = computed(() => this.current()?.institutional_provider ?? '');

  /**
   * Un proveedor del catálogo que coincide con el motor institucional activo ya
   * funciona sin clave personal: el docente opera con la cuota compartida.
   */
  isInstitutionalActiveProvider(providerId: string): boolean {
    return this.institutionalAvailable() && this.institutionalProviderId() === providerId;
  }

  readonly selectedPersonalProvider = computed(
    () => this.selectedCustomProvider() !== undefined && this.selectedCustomProvider()?.origin === 'personal'
  );

  readonly hasSavedKey = computed(() => this.current()?.has_api_key === true);
  readonly savedKeyMasked = computed(() => this.current()?.api_key_masked ?? '');
  readonly updatedAt = computed(() => this.current()?.updated_at ?? '');
  readonly institutionalAvailable = computed(
    () => this.current()?.institutional_enabled === true
  );

  /** Proveedor que efectivamente tiene la clave cifrada guardada. */
  readonly savedKeyProvider = computed(() =>
    this.hasSavedKey() ? (this.current()?.provider ?? '') : ''
  );

  readonly sameProviderAsSaved = computed(() => this.provider() === this.savedKeyProvider());

  readonly selectedProvider = computed(() =>
    this.providers().find((item) => item.value === this.provider())
  );

  readonly selectedCustomProvider = computed(() =>
    (this.current()?.custom_providers ?? []).find((cp) => cp.id === this.provider())
  );

  /** Endpoint institucional del proveedor custom; los base usan la URL oficial. */
  readonly selectedBaseURL = computed(() => this.selectedCustomProvider()?.base_url || undefined);

  /**
   * Solo se puede descubrir el catálogo con una credencial propia: la clave
   * guardada de este proveedor o una API key recién pegada. Nunca cae en el
   * motor institucional.
   */
  readonly canFetchModels = computed(() => {
    if (this.hasSavedKey() && this.sameProviderAsSaved()) {
      return true;
    }
    const provider = this.selectedProvider();
    if (!provider) {
      return false;
    }
    return this.apiKey().trim().length > 0;
  });

  /**
   * Solo se puede probar la conexión si hay una credencial disponible y un
   * modelo seleccionado.
   */
  readonly canTestConnection = computed(() => {
    return this.canFetchModels() && this.model().trim().length > 0;
  });

  /**
   * Comprueba si el docente modificó algún valor frente a la configuración
   * persistida actualmente.
   */
  readonly isDirty = computed(() => {
    const initial = this.current();
    if (!initial) {
      return false;
    }
    if (this.enabled() !== initial.enabled) {
      return true;
    }
    const initialProvider = initial.provider || DEFAULT_BYOK_PROVIDER;
    if (this.provider() !== initialProvider) {
      return true;
    }
    const initialCustom = (initial.custom_providers ?? []).find(
      (cp) => cp.id === initialProvider
    );
    const initialModel = (initial.model || initialCustom?.model || '').trim();
    if (this.model().trim() !== initialModel) {
      return true;
    }
    if (this.apiKey().trim().length > 0) {
      return true;
    }
    return false;
  });

  /**
   * Solo se puede guardar si hay cambios pendientes y la configuración es válida.
   * Si está habilitada, exige credencial y modelo; si está inactiva pero se
   * redactó una clave, exige también el modelo.
   */
  readonly canSave = computed(() => {
    if (!this.isDirty()) {
      return false;
    }
    if (this.enabled()) {
      return this.canFetchModels() && this.model().trim().length > 0;
    }
    if (this.apiKey().trim().length > 0) {
      return this.model().trim().length > 0;
    }
    return true;
  });

  readonly modelsLockedHint = computed(() => {
    const provider = this.selectedProvider();
    if (!provider) {
      return 'Seleccioná un proveedor para listar sus modelos.';
    }
    return 'Ingresá primero tu API key personal arriba para descubrir y listar los modelos disponibles de este proveedor.';
  });

  readonly filteredModels = computed(() => {
    const list = this.availableModels();
    const tier = this.modelTierFilter();
    const search = this.modelSearch().toLowerCase().trim();

    return list.filter((item) => {
      const free = this.isFreeModel(item);
      if (tier === 'free' && !free) {
        return false;
      }
      if (tier === 'paid' && free) {
        return false;
      }
      if (!search) {
        return true;
      }
      return (
        item.id.toLowerCase().includes(search) || (item.name ?? '').toLowerCase().includes(search)
      );
    });
  });

  readonly apiKeyHint = computed(() =>
    this.hasSavedKey() && this.sameProviderAsSaved()
      ? 'Dejá vacío para conservar la clave guardada'
      : 'Pega tu clave privada'
  );

  readonly apiKeyPlaceholder = computed(() =>
    this.hasSavedKey() && this.sameProviderAsSaved()
      ? 'Dejar en blanco para conservar la clave actual'
      : `Ej: ${this.selectedProvider()?.placeholderKey ?? 'sk-...'}`
  );

  readonly modelHint = computed(() =>
    this.manualModelMode()
      ? 'Escribí el identificador exacto del modelo'
      : 'Buscá el modelo por nombre o ID, o escríbelo manualmente'
  );

  ngOnInit(): void {
    this.load();
  }

  load(): void {
    this.loading.set(true);
    this.copilotService.getMySettings().subscribe({
      next: (settings) => {
        this.current.set(settings);
        this.enabled.set(settings.enabled);
        if (settings.provider) {
          this.provider.set(settings.provider);
        }
        if (settings.model) {
          this.model.set(settings.model);
        } else {
          const custom = (settings.custom_providers ?? []).find(
            (cp) => cp.id === settings.provider
          );
          this.model.set(custom?.model ?? '');
        }
        this.loading.set(false);
      },
      error: (err: unknown) => {
        this.loading.set(false);
        this.feedbackOk.set(false);
        this.feedbackMessage.set(this.copilotService.resolveError(err));
      }
    });
  }

  onToggleEnabled(active: boolean): void {
    this.enabled.set(active);
    this.feedbackMessage.set(null);
    this.feedbackOk.set(null);
  }

  onProviderChange(value: string): void {
    if (value === this.provider()) {
      return;
    }
    this.provider.set(value);
    this.apiKey.set('');
    this.availableModels.set([]);
    this.modelsFetchedAt.set('');
    this.modelTierFilter.set('all');
    this.manualModelMode.set(false);
    if (value !== this.savedKeyProvider()) {
      const custom = (this.current()?.custom_providers ?? []).find((cp) => cp.id === value);
      this.model.set(custom?.model ?? '');
    } else {
      this.model.set(this.current()?.model ?? '');
    }
    this.feedbackMessage.set(null);
    this.feedbackOk.set(null);
  }

  openAddProvider(): void {
    this.newProviderLabel.set('');
    this.newProviderBaseURL.set('');
    this.newProviderApiKey.set('');
    this.newProviderModel.set('');
    this.newProviderModels.set([]);
    this.newProviderFeedback.set(null);
    this.newProviderFeedbackOk.set(null);
    this.showAddProvider.set(true);
  }

  closeAddProvider(): void {
    this.showAddProvider.set(false);
  }

  /** Descubre el catálogo real del proveedor antes de guardarlo. */
  fetchNewProviderModels(): void {
    if (!this.canQueryNewProvider() || this.newProviderLoadingModels()) {
      this.newProviderFeedbackOk.set(false);
      this.newProviderFeedback.set('Completá el nombre y una URL https:// para listar los modelos.');
      return;
    }
    if (this.newProviderApiKey().trim() === '') {
      this.newProviderFeedbackOk.set(false);
      this.newProviderFeedback.set('Pegá tu API key para consultar el catálogo del proveedor.');
      return;
    }
    this.newProviderLoadingModels.set(true);
    this.newProviderFeedback.set(null);
    this.copilotService
      .listMyModels({
        provider: this.newProviderId(),
        api_key: this.newProviderApiKey().trim(),
        base_url: this.newProviderBaseURL().trim()
      })
      .subscribe({
        next: (models) => {
          this.newProviderLoadingModels.set(false);
          this.newProviderModels.set(models);
          if (models.length > 0 && !this.newProviderModel()) {
            this.newProviderModel.set(models[0].id);
          }
          if (models.length === 0) {
            this.newProviderFeedbackOk.set(false);
            this.newProviderFeedback.set('El proveedor no devolvió modelos. Podés escribir el ID a mano.');
          }
        },
        error: (err: unknown) => {
          this.newProviderLoadingModels.set(false);
          this.newProviderFeedbackOk.set(false);
          this.newProviderFeedback.set(this.copilotService.resolveError(err));
        }
      });
  }

  testNewProvider(): void {
    if (this.newProviderTesting()) {
      return;
    }
    if (!this.canQueryNewProvider() || this.newProviderApiKey().trim() === '' || this.newProviderModel().trim() === '') {
      this.newProviderFeedbackOk.set(false);
      this.newProviderFeedback.set('Completá nombre, URL https://, API key y modelo para probar la conexión.');
      return;
    }
    this.newProviderTesting.set(true);
    this.newProviderFeedback.set(null);
    this.copilotService
      .testMyConnection({
        provider: this.newProviderId(),
        model: this.newProviderModel().trim(),
        api_key: this.newProviderApiKey().trim(),
        base_url: this.newProviderBaseURL().trim()
      })
      .subscribe({
        next: (res) => {
          this.newProviderTesting.set(false);
          this.newProviderFeedbackOk.set(res.ok);
          this.newProviderFeedback.set(res.message);
        },
        error: (err: unknown) => {
          this.newProviderTesting.set(false);
          this.newProviderFeedbackOk.set(false);
          this.newProviderFeedback.set(this.copilotService.resolveError(err));
        }
      });
  }

  saveNewProvider(): void {
    if (this.newProviderSaving()) {
      return;
    }
    if (!this.canQueryNewProvider()) {
      this.newProviderFeedbackOk.set(false);
      this.newProviderFeedback.set('El nombre es obligatorio y la URL base debe usar https://.');
      return;
    }
    this.newProviderSaving.set(true);
    this.newProviderFeedback.set(null);

    this.copilotService
      .updateMySettings({
        enabled: this.current()?.enabled ?? false,
        provider: this.provider(),
        model: this.model(),
        custom_providers: [
          {
            id: this.newProviderId(),
            label: this.newProviderLabel().trim(),
            base_url: this.newProviderBaseURL().trim(),
            model: this.newProviderModel().trim(),
            api_key: this.newProviderApiKey().trim() || undefined
          }
        ]
      })
      .subscribe({
        next: (res) => {
          this.newProviderSaving.set(false);
          this.current.set(res);
          this.provider.set(this.newProviderId());
          this.model.set(this.newProviderModel().trim());
          this.apiKey.set('');
          this.showAddProvider.set(false);
          this.feedbackOk.set(true);
          this.feedbackMessage.set(
            'Proveedor agregado a tu catálogo. Activá tu clave personal para usarlo en lugar del motor institucional.'
          );
          this.changed.emit(res);
        },
        error: (err: unknown) => {
          this.newProviderSaving.set(false);
          this.newProviderFeedbackOk.set(false);
          this.newProviderFeedback.set(this.copilotService.resolveError(err));
        }
      });
  }

  requestDeleteProvider(): void {
    const selected = this.selectedCustomProvider();
    if (selected && selected.origin === 'personal') {
      this.providerToDelete.set(selected);
    }
  }

  cancelDeleteProvider(): void {
    this.providerToDelete.set(null);
  }

  executeDeleteProvider(provider: AIUserCustomProviderDTO): void {
    this.deletingProvider.set(true);
    this.copilotService
      .updateMySettings({
        enabled: false,
        provider: '',
        model: '',
        remove_custom_providers: [provider.id]
      })
      .subscribe({
        next: (res) => {
          this.deletingProvider.set(false);
          this.providerToDelete.set(null);
          this.current.set(res);
          this.provider.set(res.provider || DEFAULT_BYOK_PROVIDER);
          this.model.set(res.model ?? '');
          this.apiKey.set('');
          this.feedbackOk.set(true);
          this.feedbackMessage.set('Proveedor eliminado de tu catálogo personal.');
          this.changed.emit(res);
        },
        error: (err: unknown) => {
          this.deletingProvider.set(false);
          this.providerToDelete.set(null);
          this.feedbackOk.set(false);
          this.feedbackMessage.set(this.copilotService.resolveError(err));
        }
      });
  }

  onIconError(providerId: string): void {
    this.iconErrors.update((current) => ({ ...current, [providerId]: true }));
  }

  isFreeModel(item: AIByokModel): boolean {
    const tier = (item.tier ?? '').toLowerCase();
    if (tier === 'free' || tier === 'local') {
      return true;
    }
    if (tier === 'paid' || tier === 'by_account') {
      return false;
    }
    const id = item.id.toLowerCase();
    return id.includes(':free') || id.includes('/free');
  }

  modelName(item: AIByokModel): string {
    return item.name && item.name !== item.id ? item.name : item.id;
  }

  selectModel(item: AIByokModel): void {
    this.model.set(item.id);
    this.feedbackMessage.set(null);
    this.feedbackOk.set(null);
  }

  fetchModels(): void {
    if (!this.canFetchModels() || this.loadingModels()) {
      return;
    }
    this.loadingModels.set(true);
    this.feedbackMessage.set(null);

    this.copilotService
      .listMyModels({
        provider: this.provider(),
        api_key: this.apiKey().trim() || undefined,
        base_url: this.selectedBaseURL()
      })
      .subscribe({
        next: (models) => {
          const safe = Array.isArray(models) ? models : [];
          this.availableModels.set(safe);
          this.modelsFetchedAt.set(new Date().toISOString());
          this.loadingModels.set(false);
          if (safe.length > 0 && !this.model()) {
            this.model.set(safe[0].id);
          }
        },
        error: (err: unknown) => {
          this.loadingModels.set(false);
          this.feedbackOk.set(false);
          this.feedbackMessage.set(this.copilotService.resolveError(err));
        }
      });
  }

  testConnection(): void {
    if (!this.canTestConnection() || this.testing()) {
      return;
    }
    this.testing.set(true);
    this.feedbackMessage.set(null);
    this.feedbackOk.set(null);

    this.copilotService
      .testMyConnection({
        provider: this.provider(),
        model: this.model().trim(),
        api_key: this.apiKey().trim() || undefined,
        base_url: this.selectedBaseURL()
      })
      .subscribe({
        next: (res) => {
          this.testing.set(false);
          this.feedbackOk.set(res.ok);
          this.feedbackMessage.set(res.message);
        },
        error: (err: unknown) => {
          this.testing.set(false);
          this.feedbackOk.set(false);
          this.feedbackMessage.set(this.copilotService.resolveError(err));
        }
      });
  }

  save(): void {
    if (!this.canSave()) {
      return;
    }

    if (this.enabled() && !this.model().trim()) {
      this.feedbackOk.set(false);
      this.feedbackMessage.set('Elegí un modelo antes de guardar la configuración.');
      return;
    }

    this.saving.set(true);
    this.feedbackMessage.set(null);

    this.copilotService
      .updateMySettings({
        enabled: this.enabled(),
        provider: this.provider(),
        model: this.model().trim(),
        base_url: this.selectedBaseURL(),
        api_key: this.apiKey().trim() || undefined
      })
      .subscribe({
        next: (res) => {
          this.saving.set(false);
          this.apiKey.set('');
          this.current.set(res);
          this.feedbackOk.set(true);
          this.feedbackMessage.set('Configuración personal guardada.');
          this.changed.emit(res);
        },
        error: (err: unknown) => {
          this.saving.set(false);
          this.feedbackOk.set(false);
          this.feedbackMessage.set(this.copilotService.resolveError(err));
        }
      });
  }

  /**
   * Salida manual cuando la clave personal dejó de funcionar. Sin fallback
   * automático, este es el camino explícito para volver al motor institucional.
   */
  clearKey(): void {
    this.clearing.set(true);
    this.feedbackMessage.set(null);

    this.copilotService
      .updateMySettings({
        enabled: false,
        // La clave que se borra es la del proveedor que la tiene guardada, no la
        // tarjeta que el docente esté mirando en ese momento.
        provider: this.savedKeyProvider() || this.provider(),
        model: this.model().trim(),
        clear_api_key: true
      })
      .subscribe({
        next: (res) => {
          this.clearing.set(false);
          this.enabled.set(false);
          this.apiKey.set('');
          this.current.set(res);
          this.feedbackOk.set(true);
          this.feedbackMessage.set(
            'Clave personal eliminada. Volviste a usar el motor institucional.'
          );
          this.changed.emit(res);
        },
        error: (err: unknown) => {
          this.clearing.set(false);
          this.feedbackOk.set(false);
          this.feedbackMessage.set(this.copilotService.resolveError(err));
        }
      });
  }
}
