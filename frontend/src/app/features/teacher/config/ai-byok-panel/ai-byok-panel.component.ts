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
  LucideRefreshCw,
  LucideSave,
  LucideSearch,
  LucideTrash2,
  LucideWifi
} from '@lucide/angular';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import {
  AIByokModel,
  AIUserOverrideDTO,
  TeacherAiCopilotService
} from '../../services/teacher-ai-copilot.service';
import { BYOK_PROVIDERS, DEFAULT_BYOK_PROVIDER } from '../ai-byok.config';

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
    LucideSave
  ],
  templateUrl: './ai-byok-panel.component.html',
  styleUrl: './ai-byok-panel.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class AiByokPanelComponent implements OnInit {
  private readonly copilotService = inject(TeacherAiCopilotService);

  /** Notifica al contenedor que la configuración cambió y debe refrescar estado. */
  readonly changed = output<AIUserOverrideDTO>();

  readonly providers = BYOK_PROVIDERS;

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
    this.providers.find((item) => item.value === this.provider())
  );

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
    const initialModel = (initial.model || '').trim();
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
          this.model.set('');
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
      this.model.set('');
    } else {
      this.model.set(this.current()?.model ?? '');
    }
    this.feedbackMessage.set(null);
    this.feedbackOk.set(null);
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
        base_url: undefined
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
        base_url: undefined
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
        base_url: undefined,
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
        provider: this.provider(),
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
