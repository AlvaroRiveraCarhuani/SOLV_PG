import { ChangeDetectionStrategy, Component, computed, input, output } from '@angular/core';
import { LucideXCircle } from '@lucide/angular';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';

/**
 * Indicador de espera de una operación de IA.
 *
 * Una barra indeterminada, la fase que se está ejecutando, los segundos
 * transcurridos y el botón que corta la espera. No finge un porcentaje: el
 * tiempo de una llamada al modelo no es predecible.
 *
 * `variant="plain"` sirve para los flujos que ya viven dentro de un bloque con
 * borde propio (panel lateral del Studio, formulario de solicitud); el modo
 * `panel` trae su propio contenedor.
 */
@Component({
  selector: 'ai-progress-indicator',
  standalone: true,
  imports: [LucideXCircle, MachineDataDirective],
  templateUrl: './ai-progress-indicator.component.html',
  styleUrl: './ai-progress-indicator.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: {
    role: 'status',
    '[class.is-plain]': "variant() === 'plain'"
  }
})
export class AiProgressIndicatorComponent {
  /** Fase en curso, en las palabras del flujo: «Diseñando arquitectura...». */
  readonly label = input.required<string>();
  /** Nombre accesible de la barra; si falta, se usa `label`. */
  readonly ariaLabel = input<string>('');
  /** Segundos transcurridos, ya formateados por la operación. */
  readonly elapsedSeconds = input<string>('0.0');
  readonly variant = input<'panel' | 'plain'>('panel');
  readonly cancelLabel = input<string>('Cancelar');

  readonly cancelled = output<void>();

  readonly progressAriaLabel = computed<string>(() => this.ariaLabel() || this.label());
  readonly elapsedText = computed<string>(
    () => `Esperando respuesta, ${this.elapsedSeconds()} segundos`
  );
}
