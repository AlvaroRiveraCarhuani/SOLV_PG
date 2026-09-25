import { Component, input, output, ElementRef, viewChild } from '@angular/core';
import { CommonModule } from '@angular/common';
import { LucideLayers, LucideTerminal, LucideCheckCircle2, LucideHelpCircle } from '@lucide/angular';
import { TargetEnvironment } from '../../../../../services/admin-templates.service';

@Component({
  selector: 'solv-step-purpose',
  standalone: true,
  imports: [CommonModule, LucideLayers, LucideTerminal, LucideCheckCircle2, LucideHelpCircle],
  template: `
    <div class="purpose-intro-block">
      <div class="step-header-with-help">
        <h4 class="purpose-section-title" id="step-title-purpose" tabindex="-1" i18n="@@PU-01">
          Propósito del entorno
        </h4>
        <button 
          type="button" 
          class="btn-step-help" 
          (click)="helpRequested.emit()" 
          title="Ayuda contextual del paso" 
          aria-label="Ayuda contextual del paso"
          i18n-aria-label="@@AY-02"
        >
          <svg lucideHelpCircle class="w-4 h-4"></svg>
        </button>
      </div>
      <p class="purpose-section-desc">
        Seleccione el modo de ejecución que define el ciclo de vida, recursos y herramientas de la plantilla.
      </p>
    </div>

    <div class="purpose-cards-grid" role="radiogroup" aria-labelledby="step-title-purpose">
      <!-- Tarjeta: Laboratorio Interactivo -->
      <div 
        #ideCard
        id="purpose-card-ide"
        class="purpose-card" 
        role="radio"
        [attr.aria-checked]="targetEnvironment() === 'IDE_PERSISTENTE'"
        [tabindex]="targetEnvironment() === 'IDE_PERSISTENTE' ? 0 : -1"
        [class.selected]="targetEnvironment() === 'IDE_PERSISTENTE'"
        (click)="selectEnvironment('IDE_PERSISTENTE')"
        (keydown)="onKeydown($event, 'IDE_PERSISTENTE')"
      >
        <div class="purpose-card-badge-row">
          <div class="purpose-icon-wrap icon-ide">
            <svg lucideLayers class="w-5 h-5"></svg>
          </div>
          <div class="purpose-status-indicator">
            @if (targetEnvironment() === 'IDE_PERSISTENTE') {
              <svg lucideCheckCircle2 class="w-5 h-5 text-success"></svg>
            }
          </div>
        </div>
        <h5 class="purpose-card-title" i18n="@@PU-02">Laboratorio Interactivo (IDE Persistente)</h5>
        <p class="purpose-card-desc" i18n="@@PU-03">
          Sesiones completas con editor web OpenVSCode, persistencia y soporte para bases de datos adicionales.
        </p>
        <div class="purpose-tags-list">
          <span class="purpose-tag">OpenVSCode</span>
          <span class="purpose-tag">Bases de Datos</span>
          <span class="purpose-tag">Sesión Larga</span>
          <span class="purpose-tag">512 MB - 4 GB</span>
        </div>
      </div>

      <!-- Tarjeta: Juez Virtual -->
      <div 
        #judgeCard
        id="purpose-card-judge"
        class="purpose-card" 
        role="radio"
        [attr.aria-checked]="targetEnvironment() === 'JUEZ_EFIMERO'"
        [tabindex]="targetEnvironment() === 'JUEZ_EFIMERO' ? 0 : -1"
        [class.selected]="targetEnvironment() === 'JUEZ_EFIMERO'"
        (click)="selectEnvironment('JUEZ_EFIMERO')"
        (keydown)="onKeydown($event, 'JUEZ_EFIMERO')"
      >
        <div class="purpose-card-badge-row">
          <div class="purpose-icon-wrap icon-judge">
            <svg lucideTerminal class="w-5 h-5"></svg>
          </div>
          <div class="purpose-status-indicator">
            @if (targetEnvironment() === 'JUEZ_EFIMERO') {
              <svg lucideCheckCircle2 class="w-5 h-5 text-success"></svg>
            }
          </div>
        </div>
        <h5 class="purpose-card-title" i18n="@@PU-04">Juez Virtual (Sandbox Algorítmico)</h5>
        <p class="purpose-card-desc" i18n="@@PU-05">
          Ejecución efímera aislada en terminal para evaluación automática de código y algoritmos. Sin interfaz web ni bases de datos.
        </p>
        <div class="purpose-tags-list">
          <span class="purpose-tag">CLI Sandbox</span>
          <span class="purpose-tag">1-5s Timeout</span>
          <span class="purpose-tag">Ultra-ligero</span>
          <span class="purpose-tag">128 MB - 512 MB</span>
        </div>
      </div>
    </div>
  `,
  styleUrls: ['./step-purpose.component.scss']
})
export class SolvStepPurposeComponent {
  ideCard = viewChild<ElementRef<HTMLElement>>('ideCard');
  judgeCard = viewChild<ElementRef<HTMLElement>>('judgeCard');

  targetEnvironment = input<TargetEnvironment>('IDE_PERSISTENTE');

  targetEnvironmentChange = output<TargetEnvironment>();
  helpRequested = output<void>();
  advance = output<void>();

  selectEnvironment(env: TargetEnvironment): void {
    if (this.targetEnvironment() !== env) {
      this.targetEnvironmentChange.emit(env);
    }
  }

  onKeydown(event: KeyboardEvent, current: TargetEnvironment): void {
    if (event.key === 'ArrowRight' || event.key === 'ArrowDown') {
      event.preventDefault();
      this.selectEnvironment('JUEZ_EFIMERO');
      this.judgeCard()?.nativeElement.focus();
    } else if (event.key === 'ArrowLeft' || event.key === 'ArrowUp') {
      event.preventDefault();
      this.selectEnvironment('IDE_PERSISTENTE');
      this.ideCard()?.nativeElement.focus();
    } else if (event.key === 'Enter') {
      event.preventDefault();
      this.advance.emit();
    }
  }
}
