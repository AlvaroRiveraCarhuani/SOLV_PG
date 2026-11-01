import { Component, computed, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import {
  LucideClipboardList,
  LucideTrendingUp,
  LucideAlertTriangle,
  LucideCheckCircle,
  LucideXCircle,
  LucideTerminal,
  LucideCopy
} from '@lucide/angular';
import { VerdictBadgeComponent } from '@shared/components/verdict-badge/verdict-badge.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { RubricDTO } from '../../../../exercise-editor/exercise-editor.store';
import {
  ReviewMode,
  TeacherSubmissionReviewDTO,
  EphemeralRunResult
} from '../../../../models/teacher.models';

export interface LevelSelectedEvent {
  criterionId: string;
  levelName: string;
}

export interface CopyCaseEvent {
  input: string;
  index: number;
}

@Component({
  selector: 'speed-grader-eval-panel',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    RouterLink,
    LucideClipboardList,
    LucideTrendingUp,
    LucideAlertTriangle,
    LucideCheckCircle,
    LucideXCircle,
    LucideTerminal,
    LucideCopy,
    VerdictBadgeComponent,
    MachineDataDirective
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
  readonly review = input.required<TeacherSubmissionReviewDTO>();
  readonly ephemeralResult = input<EphemeralRunResult | null>(null);
  readonly getComplexityClass = input.required<(complexity: string) => string>();
  readonly selections = input<Record<string, string>>({});
  readonly rubricComments = input<string>('');
  readonly copiedIndex = input<number | null>(null);

  readonly levelSelected = output<LevelSelectedEvent>();
  readonly rubricCommentsChange = output<string>();
  readonly saveRubricEvaluation = output<void>();
  readonly copyCaseInput = output<CopyCaseEvent>();

  readonly astViolation = computed<{ ruleId: string; detail: string } | null>(() => {
    const rev = this.review();
    if (!rev) return null;
    if (rev.verdict !== 'AST_BLOCKED' && rev.verdict !== 'AST_VIOLATION') {
      return null;
    }
    const ast = rev.ast_result as Record<string, unknown> | null;
    const ruleId = (ast?.['rule_id'] as string) || 'solv.security.forbidden-system-call';
    const detail = (ast?.['message'] as string) || (ast?.['detail'] as string) || 'Detección de invocación prohibida o función no permitida en entorno académico.';
    return { ruleId, detail };
  });

  readonly orderedTestCases = computed(() => {
    const cases = this.review()?.test_cases || [];
    // Prioridad pedagógica: casos fallidos al inicio para visibilidad inmediata del docente
    return cases.map((tc, originalIndex) => ({ ...tc, originalIndex })).sort((a, b) => {
      if (a.passed === b.passed) return a.originalIndex - b.originalIndex;
      return a.passed ? 1 : -1;
    });
  });
}
