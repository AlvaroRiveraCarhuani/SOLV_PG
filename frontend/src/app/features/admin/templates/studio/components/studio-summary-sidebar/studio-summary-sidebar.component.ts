import { Component, computed, input, output } from '@angular/core';
import {
  LucideAlertCircle,
  LucideAlertTriangle,
  LucideCheckCircle2,
  LucideCopy,
  LucideDownload,
  LucidePlay,
  LucideRefreshCw,
  LucideSend,
  LucideWrench,
  LucideXCircle
} from '@lucide/angular';
import { CodeEditorComponent } from '@shared/components/code-editor/code-editor.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { TargetEnvironment, TemplateComponent } from '@features/admin/models/admin-templates.models';
import { EnvTestJob } from '../../../../services/env-test-job.service';
import { TechLogoComponent } from '../../../components/tech-logo/tech-logo.component';
import { StudioActionBlocker } from '../../template-studio.utils';

@Component({
  selector: 'studio-summary-sidebar',
  imports: [
    CodeEditorComponent,
    MachineDataDirective,
    TechLogoComponent,
    LucideAlertCircle,
    LucideAlertTriangle,
    LucideCheckCircle2,
    LucideCopy,
    LucideDownload,
    LucidePlay,
    LucideRefreshCw,
    LucideSend,
    LucideWrench,
    LucideXCircle
  ],
  templateUrl: './studio-summary-sidebar.component.html',
  styleUrls: ['./studio-summary-sidebar.component.scss']
})
export class StudioSummarySidebarComponent {
  readonly blueprintTab = input<'human' | 'manifest'>('human');
  readonly name = input<string>('');
  readonly targetEnvironment = input.required<TargetEnvironment>();
  readonly dockerImage = input<string>('');
  readonly needsLocalBuild = input<boolean>(false);
  readonly entrypoint = input<string>('');
  readonly timeoutMS = input<number>(5000);
  readonly selectedStackComponents = input<TemplateComponent[]>([]);
  readonly hasSelectedStack = input<boolean>(false);
  readonly baseRamMB = input<number>(512);
  readonly nominalConcurrentStudents = input<number>(1);
  readonly toolsList = input<string[]>([]);
  readonly activeEnvTestJob = input<EnvTestJob | null>(null);
  readonly isConfigModifiedAfterVerification = input<boolean>(false);
  readonly liveManifestJson = input<string>('');
  readonly publishBlocker = input<StudioActionBlocker | null>(null);
  readonly canRunVerification = input<boolean>(false);
  readonly canPublishTemplate = input<boolean>(false);
  readonly isVerifying = input<boolean>(false);
  readonly isSubmitting = input<boolean>(false);
  readonly isEditMode = input<boolean>(false);
  readonly aiEnabled = input<boolean>(false);
  readonly isRepairingWithAi = input<boolean>(false);
  readonly repairElapsedSeconds = input<string>('0.0');

  readonly displayDockerImage = computed<string>(() => {
    const raw = this.dockerImage().trim();
    if (raw.startsWith('solv-local/stack:')) {
      const primary = this.selectedStackComponents().find(c => c.layer === 'runtime' && !!c.base_image);
      if (primary?.base_image) {
        return primary.base_image;
      }
      const anyBase = this.selectedStackComponents().find(c => !!c.base_image);
      if (anyBase?.base_image) {
        return anyBase.base_image;
      }
    }
    return raw;
  });

  readonly blueprintTabChange = output<'human' | 'manifest'>();
  readonly cancelTest = output<void>();
  readonly copyManifest = output<void>();
  readonly downloadManifest = output<void>();
  readonly runVerification = output<void>();
  readonly publish = output<void>();
  readonly repairWithAi = output<void>();
  readonly cancelRepairWithAi = output<void>();
}
