import { Component, ChangeDetectionStrategy, input, output, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import {
  LucideGraduationCap,
  LucideLayers,
  LucideSparkles,
  LucideTerminal,
  LucideCode2
} from '@lucide/angular';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { TechLogoComponent } from '@features/admin/templates/components/tech-logo/tech-logo.component';
import { EnvironmentLabelPipe } from '@shared/pipes/environment-label.pipe';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { PublishedTemplate } from '../teacher-templates.component';

export type TemplateDetailTab = 'pedagogical' | 'technical';

@Component({
  selector: 'template-detail-modal',
  standalone: true,
  imports: [
    CommonModule,
    ModalShellComponent,
    TechLogoComponent,
    EnvironmentLabelPipe,
    MachineDataDirective,
    LucideGraduationCap,
    LucideLayers,
    LucideSparkles,
    LucideTerminal,
    LucideCode2
  ],
  templateUrl: './template-detail-modal.component.html',
  styleUrl: './template-detail-modal.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class TemplateDetailModalComponent {
  readonly template = input.required<PublishedTemplate>();

  readonly closed = output<void>();
  readonly useTemplate = output<PublishedTemplate>();

  readonly activeTab = signal<TemplateDetailTab>('pedagogical');

  setTab(tab: TemplateDetailTab): void {
    this.activeTab.set(tab);
  }

  onUse(): void {
    this.useTemplate.emit(this.template());
  }
}
