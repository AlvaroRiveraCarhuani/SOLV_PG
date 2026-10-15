import { Component, input, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import {
  LucideCheckCircle2,
  LucideXCircle,
  LucideClock,
  LucideAlertTriangle,
  LucideShieldAlert,
  LucideLoader2,
  LucideHelpCircle
} from '@lucide/angular';
import {
  JudgeVerdict,
  getVerdictBadgeClass,
  getVerdictHumanLabel
} from '../../../core/models/verdict.model';
import { MachineDataDirective } from '../../directives/machine-data.directive';

export { getVerdictBadgeClass, getVerdictHumanLabel };

@Component({
  selector: 'verdict-badge',
  standalone: true,
  imports: [
    CommonModule,
    LucideCheckCircle2,
    LucideXCircle,
    LucideClock,
    LucideAlertTriangle,
    LucideShieldAlert,
    LucideLoader2,
    LucideHelpCircle,
    MachineDataDirective
  ],
  template: `
    <span class="verdict-badge" [ngClass]="[badgeClass(), sizeClass()]" [attr.title]="fullTitle()">
      @switch (normalizedVerdict()) {
        @case ('AC') {
          <svg lucideCheckCircle2 class="verdict-icon"></svg>
        }
        @case ('WA') {
          <svg lucideXCircle class="verdict-icon"></svg>
        }
        @case ('TLE') {
          <svg lucideClock class="verdict-icon"></svg>
        }
        @case ('RE') {
          <svg lucideAlertTriangle class="verdict-icon"></svg>
        }
        @case ('AST_BLOCKED') {
          <svg lucideShieldAlert class="verdict-icon"></svg>
        }
        @case ('PENDING') {
          <svg lucideLoader2 class="verdict-icon spin-icon"></svg>
        }
        @default {
          <svg lucideHelpCircle class="verdict-icon"></svg>
        }
      }

      @if (showHumanLabel()) {
        <span class="verdict-human-label">{{ humanLabel() }}</span>
      }

      @if (showAcronym()) {
        <code class="verdict-acronym font-mono" machineData>{{ acronym() }}</code>
      }
    </span>
  `,
  styleUrl: './verdict-badge.component.scss'
})
export class VerdictBadgeComponent {
  readonly verdict = input.required<JudgeVerdict | string>();
  readonly showAcronym = input<boolean>(true);
  readonly showHumanLabel = input<boolean>(true);
  readonly size = input<'sm' | 'md' | 'lg'>('md');

  readonly normalizedVerdict = computed<string>(() => {
    return (this.verdict() || '').trim().toUpperCase();
  });

  readonly badgeClass = computed<string>(() => {
    return getVerdictBadgeClass(this.normalizedVerdict());
  });

  readonly sizeClass = computed<string>(() => {
    return `size-${this.size()}`;
  });

  readonly humanLabel = computed<string>(() => {
    return getVerdictHumanLabel(this.normalizedVerdict());
  });

  readonly acronym = computed<string>(() => {
    const v = this.normalizedVerdict();
    return v || 'UNK';
  });

  readonly fullTitle = computed<string>(() => {
    return `${this.humanLabel()} (${this.acronym()})`;
  });
}
