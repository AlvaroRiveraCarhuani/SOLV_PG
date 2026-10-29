import { Component, ChangeDetectionStrategy, Input, Output, EventEmitter, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
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
import { AiOperation, AiOperations } from '@shared/services/ai-operation';
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
export class AiExerciseGeneratorModalComponent implements OnInit {
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

  // Estados de ejecución y telemetría: la llamada vive en una operación con
  // dueño en la raíz, así que cerrar el modal no la aborta. Al reabrirlo, el
  // resultado y el estado siguen disponibles.
  private readonly operation: AiOperation<AIGenerateExerciseResponse> = inject(AiOperations).get(
    'generate-exercise',
    {
      resolveError: (err: unknown) => this.copilotService.resolveError(err),
      onError: (message: string) => this.errorMessage.set(message),
      notify: {
        success: 'Ejercicio generado por el Copiloto IA.'
      }
    }
  );

  readonly isGenerating = this.operation.isRunning;
  readonly elapsedSeconds = this.operation.elapsedSeconds;
  readonly generatedResult = this.operation.result;

  readonly errorMessage = signal<string | null>(null);

  // Panel de credencial personal: la lógica vive en el componente compartido.
  readonly showByokPanel = signal(false);

  ngOnInit(): void {
    if (this.defaultLanguage && this.languageOptions.includes(this.defaultLanguage)) {
      this.language.set(this.defaultLanguage);
    }
    this.loadStatus();
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
    this.operation.run(
      this.copilotService.generateExercise({
        prompt,
        language: this.language(),
        difficulty: this.difficulty(),
        public_cases_count: this.publicCasesCount(),
        hidden_cases_count: this.hiddenCasesCount(),
        verify_with_runner: true
      })
    );
  }

  cancelGeneration(): void {
    this.operation.cancel();
    this.errorMessage.set('Generación cancelada por el docente.');
  }

  discardResult(): void {
    this.operation.reset();
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
