import { Component, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
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
  readonly rubric = input<RubricDTO | null>(null);
  readonly rubricSuccessMessage = input<string | null>(null);
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
}
