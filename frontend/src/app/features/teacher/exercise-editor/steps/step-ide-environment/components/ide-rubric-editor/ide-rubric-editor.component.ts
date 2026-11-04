import { Component, input, output, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import {
  LucideClipboardList,
  LucidePlus,
  LucideTrash2,
  LucideAlertTriangle,
  LucideCheckCircle2,
  LucideScale,
  LucideChevronDown,
  LucideChevronUp
} from '@lucide/angular';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { RubricCriterionDTO, RubricLevelDTO, RubricDTO } from '../../../../exercise-editor.store';

export interface RubricUpdateEvent {
  index: number;
  update: Partial<RubricCriterionDTO>;
}

export interface LevelUpdateEvent {
  criterionIndex: number;
  levelIndex: number;
  update: Partial<RubricLevelDTO>;
}

export interface LevelRemoveEvent {
  criterionIndex: number;
  levelIndex: number;
}

@Component({
  selector: 'ide-rubric-editor',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideClipboardList,
    LucidePlus,
    LucideTrash2,
    LucideAlertTriangle,
    LucideCheckCircle2,
    LucideScale,
    LucideChevronDown,
    LucideChevronUp,
    MachineDataDirective
  ],
  templateUrl: './ide-rubric-editor.component.html',
  styleUrl: './ide-rubric-editor.component.scss'
})
export class IdeRubricEditorComponent {
  rubric = input<RubricDTO | null>(null);
  totalRubricWeight = input<number>(0);

  addCriterion = output<void>();
  removeCriterion = output<number>();
  updateCriterion = output<RubricUpdateEvent>();
  addLevelToCriterion = output<number>();
  removeLevelFromCriterion = output<LevelRemoveEvent>();
  updateLevel = output<LevelUpdateEvent>();
  balanceEqually = output<void>();

  expandedIndex = signal<number | null>(null);

  toggleLevels(index: number): void {
    if (this.expandedIndex() === index) {
      this.expandedIndex.set(null);
    } else {
      this.expandedIndex.set(index);
    }
  }

  isExpanded(index: number): boolean {
    return this.expandedIndex() === index;
  }
}
