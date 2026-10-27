import { Component, ChangeDetectionStrategy, Input, Output, EventEmitter, OnInit, OnDestroy, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Subscription } from 'rxjs';
import {
  LucideSparkles,
  LucideAlertTriangle,
  LucideShieldCheck,
  LucideKeyRound,
  LucideChevronDown,
  LucideChevronUp,
  LucideLoader2,
  LucideSearch
} from '@lucide/angular';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import {
  TeacherAiCopilotService,
  AIGenerateExerciseResponse,
  AIAvailabilityStatus,
  AIUserOverrideDTO
} from '../../services/teacher-ai-copilot.service';

type AIDifficulty = 'easy' | 'medium' | 'hard';

const LANGUAGE_OPTIONS: string[] = [
  'python',
  'cpp',
  'java',
  'javascript',
  'go',
  'rust',
  'c',
  'csharp'
];

const DIFFICULTY_OPTIONS: { value: AIDifficulty; label: string }[] = [
  { value: 'easy', label: 'Fácil' },
  { value: 'medium', label: 'Medio' },
  { value: 'hard', label: 'Difícil' }
];

export interface BYOKProviderOption {
  value: string;
  label: string;
  placeholderKey: string;
  defaultBaseUrl?: string;
  supportsModelListing: boolean;
}

const BYOK_PROVIDERS: BYOKProviderOption[] = [
  { value: 'gemini', label: 'Google Gemini', placeholderKey: 'AIzaSy...', supportsModelListing: true },
  { value: 'openai', label: 'OpenAI', placeholderKey: 'sk-proj-...', supportsModelListing: true },
  { value: 'openrouter', label: 'OpenRouter (Multi-modelo / Modelos Gratis)', placeholderKey: 'sk-or-v1-...', supportsModelListing: true },
  { value: 'deepseek', label: 'DeepSeek', placeholderKey: 'sk-...', supportsModelListing: true }
];

@Component({
  selector: 'ai-exercise-generator-modal',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideSparkles,
    LucideAlertTriangle,
    LucideShieldCheck,
    LucideKeyRound,
    LucideChevronDown,
    LucideChevronUp,
    LucideLoader2,
    LucideSearch,
    ModalShellComponent,
    FormFieldComponent,
    MachineDataDirective
  ],
  templateUrl: './ai-exercise-generator-modal.component.html',
  styleUrl: './ai-exercise-generator-modal.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class AiExerciseGeneratorModalComponent implements OnInit, OnDestroy {
  private readonly copilotService = inject(TeacherAiCopilotService);

  @Input() defaultLanguage = 'python';

  @Output() closed = new EventEmitter<void>();
  @Output() applied = new EventEmitter<AIGenerateExerciseResponse>();

  readonly languageOptions = LANGUAGE_OPTIONS;
  readonly difficultyOptions = DIFFICULTY_OPTIONS;
  readonly byokProviders = BYOK_PROVIDERS;

  // Estado general de IA
  readonly aiStatus = signal<AIAvailabilityStatus | null>(null);
  readonly userOverride = signal<AIUserOverrideDTO | null>(null);

  // Formulario de generación
  readonly prompt = signal('');
  readonly language = signal('python');
  readonly difficulty = signal<AIDifficulty>('medium');
  readonly publicCasesCount = signal(2);
  readonly hiddenCasesCount = signal(3);

  // Estados de ejecución y telemetría
  readonly isGenerating = signal(false);
  readonly elapsedSeconds = signal('0.0');
  private timerInterval: ReturnType<typeof setInterval> | null = null;
  private generateSub: Subscription | null = null;

  readonly errorMessage = signal<string | null>(null);
  readonly generatedResult = signal<AIGenerateExerciseResponse | null>(null);

  // Panel BYOK Docente
  readonly showByokPanel = signal(false);
  readonly byokEnabled = signal(false);
  readonly byokProvider = signal('openrouter');
  readonly byokApiKey = signal('');
  readonly byokModel = signal('');
  readonly byokBaseUrl = signal('');
  readonly byokSaving = signal(false);
  readonly byokTesting = signal(false);
  readonly byokTestMessage = signal<string | null>(null);
  readonly byokTestOk = signal<boolean | null>(null);

  // Buscador y filtro de modelos BYOK
  readonly byokLoadingModels = signal(false);
  readonly byokAvailableModels = signal<string[]>([]);
  readonly byokModelFilter = signal<'all' | 'free'>('all');
  readonly byokModelSearch = signal('');
  readonly byokManualModelMode = signal(false);

  readonly filteredByokModels = computed(() => {
    const rawList = this.byokAvailableModels();
    const list = Array.isArray(rawList) ? rawList : [];
    const filter = this.byokModelFilter();
    const search = this.byokModelSearch().toLowerCase().trim();

    return list.filter((m) => {
      const lower = m.toLowerCase();
      if (filter === 'free' && !lower.includes(':free') && !lower.includes('free')) {
        return false;
      }
      if (search && !lower.includes(search)) {
        return false;
      }
      return true;
    });
  });

  ngOnInit(): void {
    if (this.defaultLanguage && this.languageOptions.includes(this.defaultLanguage)) {
      this.language.set(this.defaultLanguage);
    }
    this.loadStatusAndSettings();
  }

  ngOnDestroy(): void {
    this.stopTimer();
    if (this.generateSub) {
      this.generateSub.unsubscribe();
    }
  }

  loadStatusAndSettings(): void {
    this.copilotService.getStatus().subscribe({
      next: (status) => this.aiStatus.set(status),
      error: () => this.aiStatus.set({ enabled: false, provider: '', model: '' })
    });

    this.copilotService.getMySettings().subscribe({
      next: (settings) => {
        this.userOverride.set(settings);
        this.byokEnabled.set(settings.enabled);
        if (settings.provider) {
          this.byokProvider.set(settings.provider);
        }
        if (settings.model) {
          this.byokModel.set(settings.model);
        }
        if (settings.base_url) {
          this.byokBaseUrl.set(settings.base_url);
        }
      },
      error: () => {}
    });
  }

  toggleByokPanel(): void {
    this.showByokPanel.update((v) => !v);
  }

  onProviderChange(provider: string): void {
    this.byokProvider.set(provider);
    this.byokTestMessage.set(null);
    this.byokTestOk.set(null);
    this.byokAvailableModels.set([]);
  }

  fetchModels(): void {
    this.byokLoadingModels.set(true);
    this.byokTestMessage.set(null);

    this.copilotService
      .listMyModels({
        provider: this.byokProvider(),
        api_key: this.byokApiKey().trim() || undefined,
        base_url: this.byokBaseUrl().trim() || undefined
      })
      .subscribe({
        next: (models) => {
          const safeModels = Array.isArray(models) ? models : [];
          this.byokAvailableModels.set(safeModels);
          this.byokLoadingModels.set(false);
          if (safeModels.length > 0 && !this.byokModel()) {
            this.byokModel.set(safeModels[0]);
          }
        },
        error: (err: unknown) => {
          this.byokLoadingModels.set(false);
          this.byokTestMessage.set(this.copilotService.resolveError(err));
          this.byokTestOk.set(false);
        }
      });
  }

  testConnection(): void {
    this.byokTesting.set(true);
    this.byokTestMessage.set(null);
    this.byokTestOk.set(null);

    this.copilotService
      .testMyConnection({
        provider: this.byokProvider(),
        model: this.byokModel().trim(),
        api_key: this.byokApiKey().trim() || undefined,
        base_url: this.byokBaseUrl().trim() || undefined
      })
      .subscribe({
        next: (res) => {
          this.byokTesting.set(false);
          this.byokTestOk.set(res.ok);
          this.byokTestMessage.set(res.message);
        },
        error: (err: unknown) => {
          this.byokTesting.set(false);
          this.byokTestOk.set(false);
          this.byokTestMessage.set(this.copilotService.resolveError(err));
        }
      });
  }

  saveByokSettings(): void {
    this.byokSaving.set(true);
    this.byokTestMessage.set(null);

    this.copilotService
      .updateMySettings({
        enabled: this.byokEnabled(),
        provider: this.byokProvider(),
        model: this.byokModel().trim(),
        base_url: this.byokBaseUrl().trim() || undefined,
        api_key: this.byokApiKey().trim() || undefined
      })
      .subscribe({
        next: (res) => {
          this.byokSaving.set(false);
          this.userOverride.set(res);
          this.byokApiKey.set('');
          this.byokTestOk.set(true);
          this.byokTestMessage.set('Configuración personal guardada correctamente.');
          this.loadStatusAndSettings();
        },
        error: (err: unknown) => {
          this.byokSaving.set(false);
          this.byokTestOk.set(false);
          this.byokTestMessage.set(this.copilotService.resolveError(err));
        }
      });
  }

  onPromptChange(value: string): void {
    this.prompt.set(value);
  }

  onPublicCountChange(value: number): void {
    this.publicCasesCount.set(this.clamp(value, 1, 5));
  }

  onHiddenCountChange(value: number): void {
    this.hiddenCasesCount.set(this.clamp(value, 1, 10));
  }

  generate(): void {
    const prompt = this.prompt().trim();
    if (prompt.length < 5 || prompt.length > 1500) {
      this.errorMessage.set('La consigna debe tener entre 5 y 1500 caracteres.');
      return;
    }

    this.errorMessage.set(null);
    this.isGenerating.set(true);
    this.startTimer();

    this.generateSub = this.copilotService
      .generateExercise({
        prompt,
        language: this.language(),
        difficulty: this.difficulty(),
        public_cases_count: this.publicCasesCount(),
        hidden_cases_count: this.hiddenCasesCount(),
        verify_with_runner: true
      })
      .subscribe({
        next: (res) => {
          this.stopTimer();
          this.generatedResult.set(res);
          this.isGenerating.set(false);
        },
        error: (err: unknown) => {
          this.stopTimer();
          this.errorMessage.set(this.copilotService.resolveError(err));
          this.isGenerating.set(false);
        }
      });
  }

  cancelGeneration(): void {
    if (this.generateSub) {
      this.generateSub.unsubscribe();
      this.generateSub = null;
    }
    this.stopTimer();
    this.isGenerating.set(false);
    this.errorMessage.set('Generación cancelada por el docente.');
  }

  private startTimer(): void {
    this.stopTimer();
    const startTime = Date.now();
    this.elapsedSeconds.set('0.0');
    this.timerInterval = setInterval(() => {
      const elapsed = ((Date.now() - startTime) / 1000).toFixed(1);
      this.elapsedSeconds.set(elapsed);
    }, 100);
  }

  private stopTimer(): void {
    if (this.timerInterval) {
      clearInterval(this.timerInterval);
      this.timerInterval = null;
    }
  }

  discardResult(): void {
    this.generatedResult.set(null);
    this.errorMessage.set(null);
  }

  applyDraft(): void {
    const res = this.generatedResult();
    if (!res) {
      return;
    }
    this.applied.emit(res);
    this.closed.emit();
  }

  private clamp(value: number, min: number, max: number): number {
    if (!Number.isFinite(value)) {
      return min;
    }
    return Math.min(Math.max(Math.round(value), min), max);
  }
}
