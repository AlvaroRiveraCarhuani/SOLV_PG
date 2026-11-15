import { Component, input, output } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { LucideAlertCircle } from '@lucide/angular';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { TargetEnvironment } from '@features/admin/models/admin-templates.models';
import { CapacityOption, StudioResourceBreakdown } from '../../template-studio.utils';

@Component({
  selector: 'studio-capacity-selector',
  imports: [FormsModule, MachineDataDirective, LucideAlertCircle],
  templateUrl: './studio-capacity-selector.component.html',
  styleUrls: ['./studio-capacity-selector.component.scss']
})
export class StudioCapacitySelectorComponent {
  readonly targetEnvironment = input.required<TargetEnvironment>();
  readonly hasSelectedStack = input.required<boolean>();
  readonly baseRamMB = input.required<number>();
  readonly capacityOptions = input.required<CapacityOption[]>();
  readonly showManualRamInput = input<boolean>(false);
  readonly minimumFloorMB = input.required<number>();
  // null = el servidor todavía no informó su memoria; no se muestra un número.
  readonly maxAllowedRamMB = input.required<number | null>();
  readonly hostFreeRamMB = input.required<number | null>();
  readonly isRamBelowFloor = input.required<boolean>();
  readonly isRamExceedingHost = input.required<boolean>();
  readonly ramExcessMB = input.required<number>();
  readonly isRamExceedingCurrentFree = input.required<boolean>();
  readonly nominalConcurrentStudents = input.required<number | null>();
  readonly stackSummaryLabel = input.required<string>();
  readonly resourceBreakdown = input.required<StudioResourceBreakdown>();

  readonly selectPreset = output<number>();
  readonly toggleManualInput = output<void>();
  readonly customRamChange = output<number | string>();

  estimateStudentsForPreset(mb: number): number | null {
    const max = this.maxAllowedRamMB();
    return max === null ? null : Math.max(1, Math.floor(max / Math.max(1, mb)));
  }
}
