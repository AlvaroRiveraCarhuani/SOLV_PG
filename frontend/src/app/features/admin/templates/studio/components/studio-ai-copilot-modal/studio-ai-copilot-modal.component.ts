import { ChangeDetectionStrategy, Component, input, output } from '@angular/core';
import { FormsModule } from '@angular/forms';
import {
  LucideSparkles,
  LucideCheckCircle2,
  LucideAlertTriangle,
  LucideRotateCcw,
  LucideRefreshCw,
  LucideXCircle,
  LucideCode,
  LucideHardDrive,
  LucideCheckCircle
} from '@lucide/angular';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { TechLogoComponent } from '../../../components/tech-logo/tech-logo.component';
import { TargetEnvironment } from '@features/admin/models/admin-templates.models';
import { AiProposalView } from '../../template-studio.utils';

@Component({
  selector: 'studio-ai-copilot-modal',
  standalone: true,
  imports: [
    FormsModule,
    ModalShellComponent,
    MachineDataDirective,
    TechLogoComponent,
    LucideSparkles,
    LucideCheckCircle2,
    LucideAlertTriangle,
    LucideRotateCcw,
    LucideRefreshCw,
    LucideXCircle,
    LucideCode,
    LucideHardDrive,
    LucideCheckCircle
  ],
  templateUrl: './studio-ai-copilot-modal.component.html',
  styleUrl: './studio-ai-copilot-modal.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class StudioAiCopilotModalComponent {
  readonly isOpen = input<boolean>(false);
  readonly prompt = input<string>('');
  readonly isSuggesting = input<boolean>(false);
  readonly elapsedSeconds = input<string>('0.0');
  readonly proposalView = input<AiProposalView | null>(null);
  readonly isApplied = input<boolean>(false);
  readonly targetEnvironment = input<TargetEnvironment>('IDE_PERSISTENTE');

  readonly promptChange = output<string>();
  readonly suggest = output<void>();
  readonly cancelSuggest = output<void>();
  readonly apply = output<void>();
  readonly discard = output<void>();
  readonly close = output<void>();
}
