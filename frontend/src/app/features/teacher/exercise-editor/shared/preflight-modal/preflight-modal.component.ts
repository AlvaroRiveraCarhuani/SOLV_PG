import { Component, computed, inject, output, signal } from '@angular/core';
import { LucideAlertTriangle, LucideShieldAlert } from '@lucide/angular';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { ExerciseEditorStore } from '../../exercise-editor.store';

/** Lista de verificación previa a publicar, con salto directo a cada sección. */
@Component({
  selector: 'preflight-modal',
  standalone: true,
  imports: [ModalShellComponent, LucideAlertTriangle, LucideShieldAlert],
  template: `
    <modal-shell
      animate.leave="motion-modal-out"
      title="Lista de verificación previa a publicar"
      subtitle="Revise los bloqueos y acepte los avisos antes de publicar."
      intent="warning"
      maxWidth="480px"
      (close)="closed.emit()"
    >
      <svg modal-icon lucideShieldAlert class="modal-icon-svg"></svg>

      <div class="preflight-body">
        @if (blockers().length > 0) {
          <section class="preflight-block">
            <h3 class="preflight-section-title is-invalid">
              Bloqueos obligatorios
              <span class="count font-mono">{{ blockers().length }}</span>
            </h3>
            <ul class="preflight-list">
              @for (blocker of blockers(); track $index) {
                <li>
                  <button type="button" class="preflight-entry is-blocker" (click)="jumpTo(blocker)">
                    <span class="entry-dot" aria-hidden="true"></span>
                    <span class="entry-text">{{ blocker }}</span>
                    <span class="entry-go">Ir</span>
                  </button>
                </li>
              }
            </ul>
          </section>
        }

        @if (warnings().length > 0) {
          <section class="preflight-block">
            <h3 class="preflight-section-title is-warning">
              Avisos no bloqueantes
              <span class="count font-mono">{{ warnings().length }}</span>
            </h3>
            <ul class="preflight-list">
              @for (warning of visibleWarnings(); track $index) {
                <li class="preflight-entry is-warning">
                  <svg lucideAlertTriangle class="entry-icon" aria-hidden="true"></svg>
                  <span class="entry-text">{{ warning }}</span>
                </li>
              }
            </ul>
            @if (warnings().length > 3) {
              <button type="button" class="btn-ghost btn-sm more-warnings" (click)="showAllWarnings.set(!showAllWarnings())">
                {{ showAllWarnings() ? 'Ver menos' : '+' + (warnings().length - 3) + ' avisos más' }}
              </button>
            }

            <label class="accept-row">
              <input type="checkbox" [checked]="accepted()" (change)="toggleAccepted($event)" />
              <span>Acepto publicar la práctica con los avisos señalados</span>
            </label>
          </section>
        }

        @if (blockers().length === 0 && warnings().length === 0) {
          <p class="preflight-clean">No hay bloqueos ni avisos pendientes.</p>
        }
      </div>

      <ng-container modal-footer>
        <button type="button" class="btn-secondary btn-sm" (click)="closed.emit()">Cancelar</button>
        <button type="button" class="btn-primary btn-sm" [disabled]="!canPublish()" (click)="publish.emit()">
          Publicar ahora
        </button>
      </ng-container>
    </modal-shell>
  `,
  styleUrl: './preflight-modal.component.scss'
})
export class PreflightModalComponent {
  private readonly store = inject(ExerciseEditorStore);
  readonly closed = output<void>();
  readonly publish = output<void>();

  readonly showAllWarnings = signal(false);

  readonly blockers = computed(() => {
    const list = [...this.store.missingPublishRequirements()];
    for (const blocker of this.store.checklistReport()?.blockers ?? []) {
      if (!list.includes(blocker)) {
        list.push(blocker);
      }
    }
    const meta = this.store.metadata();
    if (meta.purpose === 'exam' && meta.exam_network === 'egress' && (meta.exam_network_reason || '').trim().length < 10) {
      list.push('Completar el motivo de la excepción de internet (mínimo 10 caracteres).');
    }
    return list;
  });

  readonly warnings = computed(() => this.store.checklistReport()?.warnings ?? []);

  readonly visibleWarnings = computed(() =>
    this.showAllWarnings() ? this.warnings() : this.warnings().slice(0, 3)
  );

  readonly accepted = computed(() => this.store.acceptedWarnings());

  readonly canPublish = computed(
    () => this.blockers().length === 0 && (this.warnings().length === 0 || this.accepted())
  );

  toggleAccepted(event: Event): void {
    this.store.setAcceptedWarnings((event.target as HTMLInputElement).checked);
  }

  /** Los bloqueos llegan como texto; se deduce la sección por su contenido. */
  jumpTo(blocker: string): void {
    const text = blocker.toLowerCase();
    const step: 1 | 2 | 3 = /título|consigna|enunciado|descripción/.test(text)
      ? 1
      : /plantilla|entorno|caso|script|lenguaje|tiempo/.test(text)
        ? 2
        : 3;
    this.store.setStep(step);
    this.closed.emit();
  }
}
