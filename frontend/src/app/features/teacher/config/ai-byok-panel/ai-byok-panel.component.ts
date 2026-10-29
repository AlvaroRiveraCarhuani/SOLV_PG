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
  LucideKeyRound,
  LucideLoader2,
  LucideSearch,
  LucideTrash2
} from '@lucide/angular';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import {
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
    LucideSearch,
    LucideTrash2,
    LucideCheckCircle2,
    LucideAlertCircle
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
  readonly baseUrl = signal('');

  readonly saving = signal(false);
  readonly testing = signal(false);
  readonly clearing = signal(false);
  readonly loadingModels = signal(false);

  readonly feedbackMessage = signal<string | null>(null);
  readonly feedbackOk = signal<boolean | null>(null);

  readonly availableModels = signal<string[]>([]);
  readonly modelFilter = signal<'all' | 'free'>('all');
  readonly modelSearch = signal('');
  readonly manualModelMode = signal(false);

  readonly current = signal<AIUserOverrideDTO | null>(null);

  readonly hasSavedKey = computed(() => this.current()?.has_api_key === true);
  readonly savedKeyMasked = computed(() => this.current()?.api_key_masked ?? '');
  readonly updatedAt = computed(() => this.current()?.updated_at ?? '');
  readonly institutionalAvailable = computed(
    () => this.current()?.institutional_enabled === true
  );

  readonly filteredModels = computed(() => {
    const raw = this.availableModels();
    const list = Array.isArray(raw) ? raw : [];
    const filter = this.modelFilter();
    const search = this.modelSearch().toLowerCase().trim();

    return list.filter((item) => {
      const lower = item.toLowerCase();
      if (filter === 'free' && !lower.includes(':free') && !lower.includes('free')) {
        return false;
      }
      return !(search && !lower.includes(search));
    });
  });

  readonly apiKeyHint = computed(() =>
    this.hasSavedKey() ? 'Dejá vacío para conservar la clave guardada' : 'Pega tu clave privada'
  );

  readonly modelHint = computed(() =>
    this.manualModelMode()
      ? 'Escribe el ID del modelo manualmente'
      : 'Elegí del catálogo o buscá por nombre'
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
        }
        if (settings.base_url) {
          this.baseUrl.set(settings.base_url);
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
    this.provider.set(value);
    this.availableModels.set([]);
    this.manualModelMode.set(false);
    this.feedbackMessage.set(null);
    this.feedbackOk.set(null);
  }

  fetchModels(): void {
    this.loadingModels.set(true);
    this.feedbackMessage.set(null);

    this.copilotService
      .listMyModels({
        provider: this.provider(),
        api_key: this.apiKey().trim() || undefined,
        base_url: this.baseUrl().trim() || undefined
      })
      .subscribe({
        next: (models) => {
          const safe = Array.isArray(models) ? models : [];
          this.availableModels.set(safe);
          this.loadingModels.set(false);
          if (safe.length > 0 && !this.model()) {
            this.model.set(safe[0]);
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
    this.testing.set(true);
    this.feedbackMessage.set(null);
    this.feedbackOk.set(null);

    this.copilotService
      .testMyConnection({
        provider: this.provider(),
        model: this.model().trim(),
        api_key: this.apiKey().trim() || undefined,
        base_url: this.baseUrl().trim() || undefined
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
    if (!this.model().trim()) {
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
        base_url: this.baseUrl().trim() || undefined,
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
