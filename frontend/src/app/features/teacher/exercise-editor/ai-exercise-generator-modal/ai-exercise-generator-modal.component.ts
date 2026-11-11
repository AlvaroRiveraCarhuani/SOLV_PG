import { Component, ChangeDetectionStrategy, Input, Output, EventEmitter, OnInit, effect, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { AutoGrowDirective } from '@shared/directives/auto-grow.directive';
import {
  LucideSparkles,
  LucideAlertTriangle,
  LucideShieldCheck,
  LucideKeyRound,
  LucideChevronDown,
  LucideChevronUp,
  LucideLayers,
  LucideInfo
} from '@lucide/angular';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { AiProgressIndicatorComponent } from '@shared/components/ai-progress-indicator/ai-progress-indicator.component';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import {
  TeacherAiCopilotService,
  AIGeneratedTestCaseDTO,
  AIGenerateExerciseResponse,
  AIAvailabilityStatus
} from '../../services/teacher-ai-copilot.service';
import { AiOperation, AiOperations } from '@shared/services/ai-operation';
import { AiFlowDrafts } from '@shared/services/ai-flow-drafts';
import { AiByokPanelComponent } from '../../config/ai-byok-panel/ai-byok-panel.component';
import { marked } from 'marked';
import type { RubricCriterionDTO, RubricLevelDTO } from '@core/models/rubric.model';

type AIDifficulty = 'easy' | 'medium' | 'hard';

const DRAFT_KEY = 'generate-exercise';

/**
 * Lo escrito en el modal. La consigna se pierde si el modal se destruye, y el
 * docente suele cerrarlo justamente para mirar el editor mientras el motor
 * trabaja: al volver, el formulario sigue como estaba.
 */
interface ExerciseGeneratorDraft {
  prompt: string;
  language: string;
  difficulty: AIDifficulty;
  publicCasesCount: number;
  hiddenCasesCount: number;
  editedDraft?: AIGenerateExerciseResponse;
}

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
    AutoGrowDirective,
    LucideSparkles,
    LucideAlertTriangle,
    LucideShieldCheck,
    LucideKeyRound,
    LucideChevronDown,
    LucideChevronUp,
    LucideLayers,
    LucideInfo,
    ModalShellComponent,
    AiProgressIndicatorComponent,
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
  private readonly drafts = inject(AiFlowDrafts);
  // Se lee antes de crear el effect que persiste: el borrador guardado manda
  // sobre los valores por defecto del formulario.
  private readonly storedDraft = this.drafts.get<ExerciseGeneratorDraft>(DRAFT_KEY);
  private restoredDraft = false;
  private suppressDraftPersistence = false;

  @Input() defaultLanguage = 'python';
  @Input() templateId = '';
  @Input() currentTitle = '';
  @Input() currentTemplateName = '';
  @Input() hasExistingContent = false;

  private readonly _environmentType = signal<'JUEZ_EFIMERO' | 'IDE_PERSISTENTE'>('JUEZ_EFIMERO');
  @Input()
  set environmentType(val: 'JUEZ_EFIMERO' | 'IDE_PERSISTENTE') {
    this._environmentType.set(val || 'JUEZ_EFIMERO');
  }
  get environmentType(): 'JUEZ_EFIMERO' | 'IDE_PERSISTENTE' {
    return this._environmentType();
  }

  @Output() closed = new EventEmitter<void>();
  @Output() applied = new EventEmitter<AIGenerateExerciseResponse>();

  readonly isIde = computed(() => this._environmentType() === 'IDE_PERSISTENTE');

  readonly promptPlaceholder = computed(() => {
    if (!this.isIde()) {
      return 'Ej.: Suma de los N primeros números primos, con límite de tiempo y casos extremos...';
    }
    const tplName = (this.currentTemplateName || '').toLowerCase();
    if (tplName.includes('python')) {
      return 'Ej.: Análisis exploratorio de datos: procesar dataset CSV, almacenar métricas en PostgreSQL y graficar resultados...';
    }
    if (tplName.includes('postgres') || tplName.includes('sql')) {
      return 'Ej.: Modelado de base de datos relacional: tablas, índices, consultas con agregación y transacciones seguras...';
    }
    if (tplName.includes('node') || tplName.includes('next') || tplName.includes('js') || tplName.includes('full-stack')) {
      return 'Ej.: API REST de gestión académica con endpoints CRUD, validación de esquemas y persistencia en base de datos...';
    }
    return 'Ej.: Práctica de desarrollo guiada con requerimientos estructurados, dependencias instaladas y rúbrica balanceada...';
  });

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
      onSuccess: () => {
        this.editingDraft.set(null);
        this.draftStep.set('review');
      },
      notify: {
        success: 'Ejercicio generado por el Copiloto IA.'
      }
    }
  );

  readonly isGenerating = this.operation.isRunning;
  readonly elapsedSeconds = this.operation.elapsedSeconds;
  readonly generatedResult = this.operation.result;
  readonly editingDraft = signal<AIGenerateExerciseResponse | null>(null);
  readonly draftStep = signal<'prompt' | 'review' | 'edit'>('prompt');
  readonly confirmDiscardDraft = signal(false);
  readonly draftForReview = computed(() => this.editingDraft() || this.generatedResult());
  readonly visibleDraft = computed(() => this.draftStep() === 'prompt' ? null : this.draftForReview());
  readonly canApplyDraft = computed(() => !!this.draftForReview() && !this.isGenerating());
  readonly renderedDescription = computed(() => {
    const description = this.draftForReview()?.description?.trim() || '';
    if (!description) return '<p>La consigna está vacía.</p>';
    try {
      return marked.parse(description, { async: false }) as string;
    } catch {
      return `<p>${description}</p>`;
    }
  });

  readonly errorMessage = signal<string | null>(null);

  // Panel de credencial personal: la lógica vive en el componente compartido.
  readonly showByokPanel = signal(false);

  constructor() {
    const draft = this.storedDraft;
    if (draft) {
      this.restoredDraft = true;
      this.prompt.set(draft.prompt);
      if (this.languageOptions.includes(draft.language)) {
        this.language.set(draft.language);
      }
      this.difficulty.set(draft.difficulty);
      this.publicCasesCount.set(this.clamp(draft.publicCasesCount, 1, 5));
      this.hiddenCasesCount.set(this.clamp(draft.hiddenCasesCount, 1, 10));
      if (draft.editedDraft) {
        this.editingDraft.set(this.copyDraft(draft.editedDraft));
        this.draftStep.set('review');
      }
    }
    if (this.operation.state() === 'ready' && this.generatedResult()) {
      this.draftStep.set('review');
    }

    effect(() => {
      const draft: ExerciseGeneratorDraft = {
        prompt: this.prompt(),
        language: this.language(),
        difficulty: this.difficulty(),
        publicCasesCount: this.publicCasesCount(),
        hiddenCasesCount: this.hiddenCasesCount(),
        editedDraft: this.editingDraft() || undefined
      };
      if (!this.suppressDraftPersistence) {
        this.drafts.save(DRAFT_KEY, draft);
      }
    });

  }

  ngOnInit(): void {
    if (!this.restoredDraft && this.defaultLanguage && this.languageOptions.includes(this.defaultLanguage)) {
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

  onPromptKeydown(event: KeyboardEvent): void {
    if ((event.ctrlKey || event.metaKey) && event.key === 'Enter') {
      event.preventDefault();
      this.generate();
    }
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
    this.draftStep.set('prompt');
    const isIdeMode = this.isIde();
    this.operation.run(
      this.copilotService.generateExercise({
        prompt,
        title: this.currentTitle?.trim() || undefined,
        language: isIdeMode ? undefined : this.language(),
        difficulty: this.difficulty(),
        environment_type: this.environmentType,
        template_id: isIdeMode && this.templateId ? this.templateId : undefined,
        public_cases_count: isIdeMode ? 0 : this.publicCasesCount(),
        hidden_cases_count: isIdeMode ? 0 : this.hiddenCasesCount(),
        verify_with_runner: !isIdeMode
      })
    );
  }

  cancelGeneration(): void {
    this.operation.cancel();
    this.errorMessage.set('Generación cancelada por el docente.');
  }

  startEditingDraft(): void {
    const draft = this.draftForReview();
    if (!draft) return;
    if (!this.editingDraft()) {
      this.editingDraft.set(this.copyDraft(draft));
    }
    this.draftStep.set('edit');
  }

  reviewDraft(): void {
    this.draftStep.set('review');
  }

  editPrompt(): void {
    this.draftStep.set('prompt');
  }

  requestDiscardDraft(): void {
    this.confirmDiscardDraft.set(true);
  }

  cancelDiscardDraft(): void {
    this.confirmDiscardDraft.set(false);
  }

  discardDraft(): void {
    this.operation.reset();
    this.editingDraft.set(null);
    this.draftStep.set('prompt');
    this.confirmDiscardDraft.set(false);
    this.errorMessage.set(null);
  }

  updateDraft(patch: Partial<AIGenerateExerciseResponse>): void {
    this.editingDraft.update(draft => draft ? { ...draft, ...patch } : draft);
  }

  rubricTotal(criteria: RubricCriterionDTO[]): number {
    return criteria.reduce((total, criterion) => total + (Number(criterion.weight) || 0), 0);
  }

  rubricBalanced(criteria: RubricCriterionDTO[]): boolean {
    return Math.abs(this.rubricTotal(criteria) - 100) < 0.01;
  }

  allTestCasesVerified(draft: AIGenerateExerciseResponse): boolean {
    return !!draft.test_cases?.length && draft.test_cases.every(testCase => testCase.verified_by_runner);
  }

  updateTestCase(index: number, patch: Partial<AIGeneratedTestCaseDTO>): void {
    this.editingDraft.update(draft => {
      if (!draft?.test_cases) return draft;
      const test_cases = draft.test_cases.map((testCase, i) =>
        i === index ? { ...testCase, ...patch } : testCase
      );
      const changedVerifiedContent =
        Object.prototype.hasOwnProperty.call(patch, 'input') ||
        Object.prototype.hasOwnProperty.call(patch, 'expected_output');
      if (changedVerifiedContent) {
        test_cases[index] = { ...test_cases[index], verified_by_runner: false };
      }
      return {
        ...draft,
        test_cases,
        ...(changedVerifiedContent
          ? {
              runner_verified_all: false,
              verification_warning: 'Se modificó un caso después de verificarlo. El motor debe volver a comprobar los casos.'
            }
          : {})
      };
    });
  }

  updateRubricCriterion(index: number, patch: Partial<RubricCriterionDTO>): void {
    this.editingDraft.update(draft => {
      if (!draft?.rubric) return draft;
      const criteria = draft.rubric.criteria.map((criterion, i) =>
        i === index ? { ...criterion, ...patch } : criterion
      );
      return { ...draft, rubric: { ...draft.rubric, criteria } };
    });
  }

  updateRubricLevel(criterionIndex: number, levelIndex: number, patch: Partial<RubricLevelDTO>): void {
    this.editingDraft.update(draft => {
      if (!draft?.rubric) return draft;
      const criteria = draft.rubric.criteria.map((criterion, i) => {
        if (i !== criterionIndex) return criterion;
        const levels = criterion.levels.map((level, j) => j === levelIndex ? { ...level, ...patch } : level);
        return { ...criterion, levels };
      });
      return { ...draft, rubric: { ...draft.rubric, criteria } };
    });
  }

  applyDraft(): void {
    const draft = this.editingDraft() || this.generatedResult();
    if (!draft || !this.canApplyDraft()) return;

    // El backend ya resolvió la plantilla contra el catálogo aprobado. Se conserva
    // para que el paso de Entorno la hidrate; sin coincidencia, el docente elige
    // el entorno a mano.
    const appliedDraft = { ...draft };
    this.suppressDraftPersistence = true;
    this.drafts.clear(DRAFT_KEY);
    this.applied.emit(appliedDraft);
    this.operation.reset();
    this.editingDraft.set(null);
    this.confirmDiscardDraft.set(false);
    this.closed.emit();
  }

  private copyDraft(draft: AIGenerateExerciseResponse): AIGenerateExerciseResponse {
    return {
      ...draft,
      rubric: draft.rubric
        ? {
            ...draft.rubric,
            criteria: draft.rubric.criteria.map(criterion => ({
              ...criterion,
              levels: criterion.levels.map(level => ({ ...level }))
            }))
          }
        : undefined,
      test_cases: draft.test_cases?.map(testCase => ({ ...testCase }))
    };
  }

  private clamp(value: number, min: number, max: number): number {
    if (!Number.isFinite(value)) {
      return min;
    }
    return Math.min(Math.max(Math.round(value), min), max);
  }
}
