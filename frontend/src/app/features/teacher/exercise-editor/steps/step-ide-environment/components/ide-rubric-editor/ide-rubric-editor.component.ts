import { Component, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import {
  LucideClipboardList,
  LucidePlus,
  LucideTrash2,
  LucideAlertTriangle,
  LucideCheckCircle2
} from '@lucide/angular';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
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
    ComboboxComponent,
    MachineDataDirective
  ],
  templateUrl: './ide-rubric-editor.component.html',
  styleUrl: './ide-rubric-editor.component.scss'
})
export class IdeRubricEditorComponent {
  rubric = input<RubricDTO | null>(null);
  rubricPresetOptions = input<ComboboxOption[]>([]);
  totalRubricWeight = input<number>(0);

  presetSelected = output<string>();
  addCriterion = output<void>();
  removeCriterion = output<number>();
  updateCriterion = output<RubricUpdateEvent>();
  addLevelToCriterion = output<number>();
  removeLevelFromCriterion = output<LevelRemoveEvent>();
  updateLevel = output<LevelUpdateEvent>();
}
