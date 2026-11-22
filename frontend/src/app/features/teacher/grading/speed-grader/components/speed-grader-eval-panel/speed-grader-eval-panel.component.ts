import { Component, computed, input, output, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import {
  LucideAlertTriangle,
  LucideCopy,
  LucideChevronDown
} from '@lucide/angular';
import { VerdictBadgeComponent } from '@shared/components/verdict-badge/verdict-badge.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { RubricCriterionDTO, RubricDTO } from '../../../../exercise-editor/exercise-editor.store';
import {
  ComplexityAnalysis,
  ReviewMode,
  TeacherSubmissionReviewDTO,
  EphemeralRunResult,
  TestCaseReview
} from '../../../../models/teacher.models';

export interface LevelSelectedEvent {
  criterionId: string;
  levelName: string;
}

export interface CopyCaseEvent {
  input: string;
  index: number;
}

type PanelMode = 'rubric' | 'direct';

@Component({
  selector: 'speed-grader-eval-panel',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideAlertTriangle,
    LucideCopy,
    LucideChevronDown,
    VerdictBadgeComponent,
    MachineDataDirective,
    DateTextPipe
  ],
  templateUrl: './speed-grader-eval-panel.component.html',
  styleUrl: './speed-grader-eval-panel.component.scss'
})
export class SpeedGraderEvalPanelComponent {
  readonly mode = input.required<ReviewMode>();
  readonly rubric = input<RubricDTO | null>(null);
  readonly rubricSuccessMessage = input<string | null>(null);
  readonly rubricErrorMessage = input<string | null>(null);
  readonly isSubmittingRubric = input<boolean>(false);
  readonly calculatedRubricScore = input<number>(0);
  readonly directScore = input<number>(100);
  readonly review = input.required<TeacherSubmissionReviewDTO>();
  readonly ephemeralResult = input<EphemeralRunResult | null>(null);
  readonly getComplexityClass = input.required<(complexity: string) => string>();
  readonly selections = input<Record<string, string>>({});
  readonly rubricComments = input<string>('');
  readonly copiedIndex = input<number | null>(null);

  readonly levelSelected = output<LevelSelectedEvent>();
  readonly rubricCommentsChange = output<string>();
  readonly saveRubricEvaluation = output<void>();
  readonly directScoreChange = output<number>();
  readonly saveDirectEvaluation = output<void>();
  readonly copyCaseInput = output<CopyCaseEvent>();

  /** Modo de calificación en Desarrollo: rúbrica o nota directa. */
  readonly panelMode = signal<PanelMode>('rubric');

  readonly showRubric = computed(() => this.rubric() !== null && this.panelMode() === 'rubric');

  readonly hasSelections = computed(() =>
    Object.values(this.selections()).some(value => !!value)
  );

  readonly displayedScore = computed<number | null>(() => {
    if (this.showRubric()) {
      return this.hasSelections() ? this.calculatedRubricScore() : null;
    }
    return this.directScore();
  });

  readonly hiddenCases = computed(() =>
    this.orderedTestCases().filter(testCase => testCase.is_hidden).length
  );

  /** Sin salida registrada por caso no se pintan estados individuales. */
  readonly hasRecordedOutputs = computed(() =>
    this.review().test_cases.some(testCase => !!testCase.actual_output)
  );

  readonly astViolation = computed<{ ruleId: string; detail: string } | null>(() => {
    const rev = this.review();
    if (!rev) return null;
    if (rev.verdict !== 'AST_BLOCKED' && rev.verdict !== 'AST_VIOLATION') {
      return null;
    }
    const ast = rev.ast_result as Record<string, unknown> | null;
    const ruleId = (ast?.['rule_id'] as string) || 'solv.security.forbidden-system-call';
    const detail = (ast?.['message'] as string) || (ast?.['detail'] as string) || 'Se detectó una instrucción que el enunciado no permite.';
    return { ruleId, detail };
  });

  readonly orderedTestCases = computed(() => {
    const cases = this.review()?.test_cases || [];
    // Los casos con diferencias van primero: el docente ve antes lo que falló.
    return cases.map((testCase, originalIndex) => ({ ...testCase, originalIndex })).sort((a, b) => {
      if (a.passed === b.passed) return a.originalIndex - b.originalIndex;
      return a.passed ? 1 : -1;
    });
  });

  setPanelMode(mode: PanelMode): void {
    this.panelMode.set(mode);
  }

  /** Descripción íntegra del nivel elegido, sin truncar. */
  levelDescription(criterion: RubricCriterionDTO): string {
    const selected = this.selections()[criterion.id ?? ''];
    if (!selected) {
      return 'Seleccione un nivel para ver el criterio completo.';
    }
    const level = criterion.levels.find(item => item.name === selected);
    return level?.description?.trim() || 'El nivel elegido no tiene descripción registrada.';
  }

  isLevelSelected(criterion: RubricCriterionDTO, levelName: string): boolean {
    return this.selections()[criterion.id ?? ''] === levelName;
  }

  /** Estado humanizado por caso; nunca siglas del juez. */
  caseStateLabel(testCase: TestCaseReview): string {
    if (!this.hasRecordedOutputs()) {
      return 'Sin salida registrada';
    }
    return testCase.passed ? 'Cumple' : 'Con diferencias';
  }

  caseStateTone(testCase: TestCaseReview): 'ok' | 'diff' | 'neutral' {
    if (!this.hasRecordedOutputs()) {
      return 'neutral';
    }
    return testCase.passed ? 'ok' : 'diff';
  }

  complexityNeedsAttention(analysis: ComplexityAnalysis): boolean {
    const expected = this.review().expected_complexity;
    const mismatch =
      !!expected && analysis.time_complexity !== 'unknown' && analysis.time_complexity !== expected;
    const lowConfidence = analysis.confidence < 0.7 && analysis.time_complexity !== 'unknown';
    return mismatch || lowConfidence;
  }
}
