import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { CommonModule } from '@angular/common';

export type SemanticBadgeTone = 'success' | 'warning' | 'error' | 'neutral' | 'info';
export type SemanticBadgeVariant = 'soft' | 'plain';

/**
 * Etiqueta semántica genérica. Los estados de entorno siguen usando
 * `status-badge` y los veredictos `verdict-badge`; este componente cubre las
 * etiquetas de dominio que solo necesitan un tono y, opcionalmente, un punto.
 */
@Component({
  selector: 'semantic-badge',
  standalone: true,
  imports: [CommonModule],
  template: `
    <span
      class="semantic-badge"
      [ngClass]="'tone-' + tone()"
      [class.is-plain]="variant() === 'plain'">
      @if (dot()) {
        <span class="badge-dot" aria-hidden="true"></span>
      }
      <span class="badge-label">{{ label() }}</span>
    </span>
  `,
  styleUrl: './semantic-badge.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class SemanticBadgeComponent {
  readonly label = input.required<string>();
  readonly tone = input<SemanticBadgeTone>('neutral');
  readonly variant = input<SemanticBadgeVariant>('soft');
  readonly dot = input<boolean>(true);
}
