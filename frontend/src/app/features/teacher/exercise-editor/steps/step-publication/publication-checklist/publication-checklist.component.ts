import { Component, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import {
  LucideCheckCircle2,
  LucideAlertTriangle,
  LucideAlertCircle,
  LucideInfo,
  LucideRefreshCw,
  LucideChevronDown,
  LucideChevronUp,
  LucideShieldCheck,
  LucideShieldX
} from '@lucide/angular';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { ExerciseEditorStore } from '../../../exercise-editor.store';

@Component({
  selector: 'publication-checklist',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideCheckCircle2,
    LucideAlertTriangle,
    LucideAlertCircle,
    LucideInfo,
    LucideRefreshCw,
    LucideChevronDown,
    LucideChevronUp,
    LucideShieldCheck,
    LucideShieldX,
    MachineDataDirective
  ],
  templateUrl: './publication-checklist.component.html',
  styleUrl: './publication-checklist.component.scss'
})
export class PublicationChecklistComponent {
  readonly store = inject(ExerciseEditorStore);

  readonly blockersOpen = signal<boolean>(true);
  readonly warningsOpen = signal<boolean>(true);
  readonly infoOpen = signal<boolean>(true);

  readonly blockers = computed(() => this.store.checklistReport()?.blockers || []);
  readonly warnings = computed(() => this.store.checklistReport()?.warnings || []);
  readonly infoItems = computed(() => this.store.checklistReport()?.info || []);

  readonly hasBlockers = computed(() => this.blockers().length > 0);
  readonly hasWarnings = computed(() => this.warnings().length > 0);

  toggleBlockers(): void {
    this.blockersOpen.update(v => !v);
  }

  toggleWarnings(): void {
    this.warningsOpen.update(v => !v);
  }

  toggleInfo(): void {
    this.infoOpen.update(v => !v);
  }

  refreshChecklist(): void {
    this.store.refreshChecklist();
  }

  onAcceptWarnings(checked: boolean): void {
    this.store.setAcceptedWarnings(checked);
  }
}
