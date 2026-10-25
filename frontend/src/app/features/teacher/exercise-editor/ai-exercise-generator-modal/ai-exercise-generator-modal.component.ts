import { Component, ChangeDetectionStrategy, Input, Output, EventEmitter, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { LucideSparkles, LucideAlertTriangle, LucideShieldCheck } from '@lucide/angular';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import {
  TeacherAiCopilotService,
  AIGenerateExerciseResponse
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

@Component({
  selector: 'ai-exercise-generator-modal',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideSparkles,
    LucideAlertTriangle,
    LucideShieldCheck,
    ModalShellComponent,
    FormFieldComponent
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

  readonly prompt = signal('');
  readonly language = signal('python');
  readonly difficulty = signal<AIDifficulty>('medium');
  readonly publicCasesCount = signal(2);
  readonly hiddenCasesCount = signal(3);

  readonly isGenerating = signal(false);
  readonly errorMessage = signal<string | null>(null);
  readonly generatedResult = signal<AIGenerateExerciseResponse | null>(null);

  ngOnInit(): void {
    if (this.defaultLanguage && this.languageOptions.includes(this.defaultLanguage)) {
      this.language.set(this.defaultLanguage);
    }
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

    this.copilotService
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
          this.generatedResult.set(res);
          this.isGenerating.set(false);
        },
        error: (err: unknown) => {
          this.errorMessage.set(this.copilotService.resolveError(err));
          this.isGenerating.set(false);
        }
      });
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
