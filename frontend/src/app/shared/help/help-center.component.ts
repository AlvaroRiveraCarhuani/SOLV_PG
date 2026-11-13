import { Component, inject } from '@angular/core';
import { HelpDrawerComponent } from '@shared/components/help-drawer/help-drawer.component';
import { LucideCompass, LucideListChecks } from '@lucide/angular';
import { ContextHelpService } from './context-help.service';

/**
 * Centro de ayuda del área administrativa, montado una vez en el layout.
 *
 * El botón `?` de la barra superior abre este panel. El contenido sigue a la
 * pantalla registrada: explicación contextual, guías disponibles en esa vista y
 * la sección correspondiente del manual. Nunca inicia nada por sí solo.
 */
@Component({
  selector: 'help-center',
  standalone: true,
  imports: [HelpDrawerComponent, LucideCompass, LucideListChecks],
  template: `
    @if (help.isOpen()) {
      <help-drawer
        [isOpen]="true"
        title="Centro de ayuda"
        manualLinkText="Abrir el manual administrativo"
        [manualFragment]="help.entry()?.manualFragment ?? null"
        (closed)="help.close()"
      >
        @if (help.entry(); as content) {
          <section class="help-section">
            <h4 class="help-section-title">{{ content.title }}</h4>
            <p class="help-intro">{{ content.intro }}</p>
            <ul class="help-items">
              @for (item of content.items; track item.label) {
                <li class="help-item">
                  <strong class="help-item-label">{{ item.label }}</strong>
                  <span class="help-item-text">{{ item.text }}</span>
                </li>
              }
            </ul>
          </section>
        }

        @if (help.guides().tour || help.guides().tutorial) {
          <section class="help-section help-guides">
            <h4 class="help-section-title">Guías de esta pantalla</h4>
            <div class="guide-actions">
              @if (help.guides().tour; as tour) {
                <button type="button" class="guide-card" (click)="startGuide(tour.run)">
                  <svg lucideCompass class="guide-icon" aria-hidden="true"></svg>
                  <span class="guide-text">
                    <strong class="guide-label">{{ tour.label }}</strong>
                    @if (tour.hint) {
                      <span class="guide-hint">{{ tour.hint }}</span>
                    }
                  </span>
                </button>
              }
              @if (help.guides().tutorial; as tutorial) {
                <button type="button" class="guide-card" (click)="startGuide(tutorial.run)">
                  <svg lucideListChecks class="guide-icon" aria-hidden="true"></svg>
                  <span class="guide-text">
                    <strong class="guide-label">{{ tutorial.label }}</strong>
                    @if (tutorial.hint) {
                      <span class="guide-hint">{{ tutorial.hint }}</span>
                    }
                  </span>
                </button>
              }
            </div>
          </section>
        }
      </help-drawer>
    }
  `,
  styleUrls: ['./help-center.component.scss']
})
export class HelpCenterComponent {
  readonly help = inject(ContextHelpService);

  /**
   * Cierra el panel antes de lanzar la guía: el recorrido necesita la pantalla
   * libre y el foco debe volver al botón `?` para restaurarlo al terminar.
   */
  startGuide(run: () => void): void {
    this.help.close();
    setTimeout(() => run(), 0);
  }
}
