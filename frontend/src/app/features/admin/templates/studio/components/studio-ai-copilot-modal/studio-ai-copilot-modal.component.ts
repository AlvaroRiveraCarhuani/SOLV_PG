import { ChangeDetectionStrategy, Component, input, output } from '@angular/core';
import { FormsModule } from '@angular/forms';
import {
  LucideSparkles,
  LucideCheckCircle2,
  LucideRotateCcw,
  LucideRefreshCw,
  LucideXCircle,
  LucidePlusCircle,
  LucideInfo,
  LucideAlertCircle
} from '@lucide/angular';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { StackConstellationComponent } from '../stack-constellation/stack-constellation.component';
import { TargetEnvironment } from '@features/admin/models/admin-templates.models';
import { AiProposalView, StackConstellation } from '../../template-studio.utils';
import { AIMissingComponent } from '../../../../configuracion/admin-config-ia.service';

@Component({
  selector: 'studio-ai-copilot-modal',
  standalone: true,
  imports: [
    FormsModule,
    ModalShellComponent,
    MachineDataDirective,
    StackConstellationComponent,
    LucideSparkles,
    LucideCheckCircle2,
    LucideRotateCcw,
    LucideRefreshCw,
    LucideXCircle,
    LucidePlusCircle,
    LucideInfo,
    LucideAlertCircle
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
  readonly errorMessage = input<string | null>(null);
  readonly proposalView = input<AiProposalView | null>(null);
  readonly constellation = input<StackConstellation | null>(null);
  readonly isApplied = input<boolean>(false);
  readonly targetEnvironment = input<TargetEnvironment>('IDE_PERSISTENTE');

  readonly promptChange = output<string>();
  readonly suggest = output<void>();
  readonly cancelSuggest = output<void>();
  readonly apply = output<void>();
  readonly discard = output<void>();
  readonly dismiss = output<void>();
  readonly proposalChange = output<Omit<AiProposalView, 'latencyMs'> & { latencyMs: number }>();
  readonly createMissingComponent = output<{ name: string; missing?: AIMissingComponent }>();

  onCreateMissing(name: string, view: AiProposalView): void {
    const item = view.missingComponents?.find(m => m.name.toLowerCase() === name.toLowerCase());
    this.createMissingComponent.emit({ name, missing: item });
  }

  updateProposal<K extends 'runtime' | 'web' | 'database'>(field: K, value: string, view: AiProposalView): void {
    if (field === 'runtime') {
      const runtime = view.runtimeChoices?.find(item => item.id === value) ?? null;
      this.proposalChange.emit({ ...view, latencyMs: view.latencyMs ?? 0, runtime, runtimeSlug: runtime?.slug ?? value });
      return;
    }
    if (field === 'web') {
      const web = view.webChoices?.find(item => item.id === value) ?? null;
      this.proposalChange.emit({ ...view, latencyMs: view.latencyMs ?? 0, web, webSlug: web?.slug ?? value });
      return;
    }
    const database = view.databaseChoices?.find(item => item.id === value) ?? null;
    this.proposalChange.emit({ ...view, latencyMs: view.latencyMs ?? 0, database, databaseSlug: database?.service_engine ?? value });
  }

  toggleAddon(addonId: string, selected: boolean, view: AiProposalView): void {
    const addon = view.addonChoices?.find(item => item.id === addonId);
    if (!addon) return;
    const addons = selected
      ? [...view.addons.filter(item => item.id !== addon.id), addon]
      : view.addons.filter(item => item.id !== addon.id);
    const addonSlugs = selected
      ? [...new Set([...view.addonSlugs, addon.slug])]
      : view.addonSlugs.filter(slug => slug !== addon.slug && slug !== addon.id);
    this.proposalChange.emit({ ...view, latencyMs: view.latencyMs ?? 0, addons, addonSlugs });
  }
}
