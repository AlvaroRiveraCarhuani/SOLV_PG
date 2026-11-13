import {
  Component,
  ElementRef,
  HostListener,
  OnDestroy,
  effect,
  inject,
  signal,
  viewChild
} from '@angular/core';
import { RouterModule } from '@angular/router';
import {
  LucideListChecks,
  LucideX,
  LucideAlertCircle,
  LucideCheckCircle2,
  LucideChevronDown,
  LucideChevronUp
} from '@lucide/angular';
import { InteractiveTutorialService } from './interactive-tutorial.service';
import { TutorialPanelPlacement, pickTutorialPanelPlacement } from './tutorial-placement';

/**
 * Panel del tutorial interactivo. Angular decide el paso y valida el estado
 * real; Driver.js sólo resalta el elemento al que se refiere el paso.
 */
@Component({
  selector: 'tutorial-panel',
  standalone: true,
  imports: [
    RouterModule,
    LucideListChecks,
    LucideX,
    LucideAlertCircle,
    LucideCheckCircle2,
    LucideChevronDown,
    LucideChevronUp
  ],
  template: `
    @if (tutorial.currentStep(); as step) {
      <aside
        #panel
        id="tutorial-panel-root"
        class="tutorial-panel"
        [attr.data-placement]="placement()"
        role="dialog"
        tabindex="-1"
        aria-labelledby="tutorial-panel-title"
      >
        <header class="panel-header">
          <div class="panel-heading">
            <svg lucideListChecks class="panel-icon" aria-hidden="true"></svg>
            <div class="panel-titles">
              <span id="tutorial-panel-title" class="panel-title">
                {{ tutorial.definition()?.title }}
              </span>
              <span class="panel-progress">
                Paso {{ tutorial.index() + 1 }} de {{ tutorial.stepCount() }}
              </span>
            </div>
          </div>
          <div class="panel-controls">
            <button
              type="button"
              class="panel-close"
              (click)="toggleCollapsed()"
              [attr.aria-expanded]="!collapsed()"
              [attr.aria-label]="collapsed() ? 'Expandir el tutorial' : 'Minimizar el tutorial'"
            >
              @if (collapsed()) {
                <svg lucideChevronDown class="panel-close-icon" aria-hidden="true"></svg>
              } @else {
                <svg lucideChevronUp class="panel-close-icon" aria-hidden="true"></svg>
              }
            </button>
            <button
              type="button"
              class="panel-close"
              (click)="tutorial.exit()"
              aria-label="Salir del tutorial"
            >
              <svg lucideX class="panel-close-icon" aria-hidden="true"></svg>
            </button>
          </div>
        </header>

        @if (!collapsed()) {
        <div class="panel-body">
          <h3 class="step-title">{{ step.title }}</h3>
          <p class="step-body">{{ step.body }}</p>

          @if (step.check) {
            @if (tutorial.currentCheckPassed()) {
              <p class="step-status status-done" role="status">
                <svg lucideCheckCircle2 class="status-icon" aria-hidden="true"></svg>
                <span>Comprobado en la pantalla.</span>
              </p>
            } @else {
              <p class="step-status status-pending" role="status">
                <svg lucideAlertCircle class="status-icon" aria-hidden="true"></svg>
                <span>{{ step.check.pendingHint }}</span>
              </p>
            }
          }
        </div>

        <footer class="panel-footer">
          <button
            type="button"
            class="panel-btn"
            (click)="tutorial.previous()"
            [disabled]="tutorial.isFirstStep()"
          >
            Anterior
          </button>
          <button
            type="button"
            class="panel-btn panel-btn-primary"
            (click)="tutorial.next()"
            [disabled]="!tutorial.currentCheckPassed()"
          >
            {{ tutorial.isLastStep() ? 'Finalizar' : 'Siguiente' }}
          </button>
        </footer>

        @if (tutorial.definition()?.manualFragment; as fragment) {
          <a
            class="panel-manual-link"
            routerLink="/admin/manual"
            [fragment]="fragment"
            target="_blank"
            rel="noopener noreferrer"
          >
            Ver esta sección en el manual
          </a>
        }
        }
      </aside>
    }
  `,
  styleUrls: ['./tutorial-panel.component.scss']
})
export class TutorialPanelComponent implements OnDestroy {
  readonly tutorial = inject(InteractiveTutorialService);

  private readonly panel = viewChild<ElementRef<HTMLElement>>('panel');
  private wasActive = false;
  private placementTimer: ReturnType<typeof setTimeout> | null = null;

  /** Esquina que ocupa el panel para no tapar el objetivo del paso actual. */
  readonly placement = signal<TutorialPanelPlacement>('bottom-right');

  /** Panel reducido a su cabecera para liberar la pantalla al operar. */
  readonly collapsed = signal(false);

  toggleCollapsed(): void {
    this.collapsed.update((value) => !value);
  }

  constructor() {
    // El foco entra al panel sólo al abrir la guía; no se mueve en cada paso
    // para no arrebatárselo al botón «Siguiente» que el usuario acaba de usar.
    effect(() => {
      const active = this.tutorial.isActive();
      if (active && !this.wasActive) {
        // Cada guía arranca desplegada: el plegado es una decisión de la sesión
        // anterior, no del tutorial que se abre.
        this.collapsed.set(false);
        setTimeout(() => this.panel()?.nativeElement.focus(), 0);
      }
      this.wasActive = active;
    });

    // Cada paso puede señalar un control distinto: la esquina se recalcula
    // después del desplazamiento que hace Driver.js para centrar el objetivo.
    effect(() => {
      this.tutorial.index();
      this.tutorial.currentStep();
      this.collapsed();
      this.schedulePlacement();
    });
  }

  ngOnDestroy(): void {
    this.clearPlacementTimer();
  }

  @HostListener('window:resize')
  onResize(): void {
    this.schedulePlacement();
  }

  @HostListener('document:keydown.escape')
  onEscape(): void {
    if (this.tutorial.isActive()) {
      this.tutorial.exit();
    }
  }

  private schedulePlacement(): void {
    this.clearPlacementTimer();
    this.placementTimer = setTimeout(() => this.applyPlacement(), 160);
  }

  private clearPlacementTimer(): void {
    if (this.placementTimer !== null) {
      clearTimeout(this.placementTimer);
      this.placementTimer = null;
    }
  }

  private applyPlacement(): void {
    const element = this.panel()?.nativeElement;
    const selector = this.tutorial.currentStep()?.selector;

    if (!element) {
      return;
    }

    const target = selector ? document.querySelector(selector) : null;
    if (!target) {
      this.placement.set('bottom-right');
      return;
    }

    const rect = target.getBoundingClientRect();
    this.placement.set(
      pickTutorialPanelPlacement(
        { left: rect.left, top: rect.top, right: rect.right, bottom: rect.bottom },
        { width: element.offsetWidth, height: element.offsetHeight },
        { width: window.innerWidth, height: window.innerHeight }
      )
    );
  }
}
