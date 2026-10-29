import { Component, ChangeDetectionStrategy, Input, Output, EventEmitter, OnInit, OnDestroy, inject, signal } from '@angular/core';
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
  LucideLoader2
} from '@lucide/angular';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import {
  TeacherAiCopilotService,
  AIGenerateExerciseResponse,
  AIAvailabilityStatus
} from '../../services/teacher-ai-copilot.service';
import { AiByokPanelComponent } from '../../config/ai-byok-panel/ai-byok-panel.component';

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
    ModalShellComponent,
    FormFieldComponent,
    MachineDataDirective,
    AiByokPanelComponent
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

  // Estado general de IA
  readonly aiStatus = signal<AIAvailabilityStatus | null>(null);

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

  // Panel de credencial personal: la lógica vive en el componente compartido.
  readonly showByokPanel = signal(false);

  ngOnInit(): void {
    if (this.defaultLanguage && this.languageOptions.includes(this.defaultLanguage)) {
      this.language.set(this.defaultLanguage);
    }
    this.loadStatus();
  }

  ngOnDestroy(): void {
    this.stopTimer();
    if (this.generateSub) {
      this.generateSub.unsubscribe();
    }
  }

  loadStatus(): void {
    this.copilotService.getStatus().subscribe({
      next: (status) => this.aiStatus.set(status),
      error: () => this.aiStatus.set({ enabled: false, provider: '', model: '' })
    });
  }


  toggleByokPanel(): void {
    this.showByokPanel.update((v) => !v);
  }

  /** Refresca el estado del motor cuando el panel compartido guarda cambios. */
  onByokChanged(): void {
    this.loadStatus();
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
