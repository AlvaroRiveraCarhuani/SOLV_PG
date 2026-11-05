import { Component, input, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import {
  LucideCheckCircle2,
  LucideXCircle,
  LucideClock,
  LucideAlertTriangle,
  LucideShieldAlert,
  LucideHelpCircle
} from '@lucide/angular';
import {
  JudgeVerdict,
  getVerdictBadgeClass,
  getVerdictHumanLabel,
  getVerdictOutcomeDetail,
  verdictOriginLabel
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
    LucideHelpCircle,
    MachineDataDirective
  ],
  template: `
    @if (originLabel(); as originText) {
      <span class="verdict-badge verdict-origin" [ngClass]="sizeClass()" [attr.title]="originText">
        <span class="verdict-human-label">{{ originText }}</span>
      </span>
    } @else {
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
        @case ('MLE') {
          <svg lucideClock class="verdict-icon"></svg>
        }
        @case ('RE') {
          <svg lucideAlertTriangle class="verdict-icon"></svg>
        }
        @case ('CE') {
          <svg lucideAlertTriangle class="verdict-icon"></svg>
        }
        @case ('VE') {
          <svg lucideAlertTriangle class="verdict-icon"></svg>
        }
        @case ('AST_BLOCKED') {
          <svg lucideShieldAlert class="verdict-icon"></svg>
        }
        @case ('AST_VIOLATION') {
          <svg lucideShieldAlert class="verdict-icon"></svg>
        }
        @case ('PENDING') {
          <svg lucideClock class="verdict-icon"></svg>
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

      @if (legacyLabel(); as legacyText) {
        <span class="verdict-origin-tag">{{ legacyText }}</span>
      }
    </span>
    }
  `,
  styleUrl: './verdict-badge.component.scss'
})
export class VerdictBadgeComponent {
  readonly verdict = input.required<JudgeVerdict | string>();
  /**
   * Origen del veredicto guardado. Cuando no lo escribió el juez, el badge
   * declara la procedencia en lugar de presentarlo como resultado verificado.
   */
  readonly origin = input<string | null | undefined>(undefined);
  readonly showAcronym = input<boolean>(false);
  readonly showHumanLabel = input<boolean>(true);
  readonly size = input<'sm' | 'md' | 'lg'>('md');

  readonly normalizedVerdict = computed<string>(() => {
    return (this.verdict() || '').trim().toUpperCase();
  });

  private readonly normalizedOrigin = computed<string>(() => {
    return (this.origin() || '').trim().toLowerCase();
  });

  /**
   * La procedencia reemplaza al resultado cuando el veredicto guardado lo
   * escribió otro camino y presentarlo como evaluación del juez sería falso:
   * la rúbrica histórica y el ajuste docente. Una entrega sin veredicto, o con
   * el centinela histórico 'PENDING', no tiene resultado que atribuir.
   */
  readonly originLabel = computed<string | null>(() => {
    const verdict = this.normalizedVerdict();
    if (verdict === '' || verdict === 'PENDING') {
      return null;
    }
    const origin = this.normalizedOrigin();
    if (origin !== 'rubric' && origin !== 'override') {
      return null;
    }
    return verdictOriginLabel(origin);
  });

  /**
   * Un registro heredado no tiene procedencia demostrable, pero su código
   * técnico es un dato real: se conserva y se acompaña con la etiqueta.
   */
  readonly legacyLabel = computed<string | null>(() => {
    const verdict = this.normalizedVerdict();
    if (verdict === '' || verdict === 'PENDING') {
      return null;
    }
    return this.normalizedOrigin() === 'legacy' ? verdictOriginLabel('legacy') : null;
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
    const detail = getVerdictOutcomeDetail(this.normalizedVerdict());
    return this.showAcronym() ? `${detail} (${this.acronym()})` : detail;
  });
}
