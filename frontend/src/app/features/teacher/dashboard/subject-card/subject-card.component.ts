import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';
import { RouterLink } from '@angular/router';

import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { TeacherOverviewSubject } from '../../models/teacher.models';

/**
 * Tarjeta de materia del canvas de Hoy. Es presentación pura: recibe la fila del
 * read model y no calcula cifras. La tarjeta entera abre la materia; las
 * acciones quedan por encima del enlace estirado y conservan su propio foco.
 */
@Component({
  selector: 'subject-card',
  standalone: true,
  imports: [RouterLink, DateTextPipe],
  templateUrl: './subject-card.component.html',
  styleUrl: './subject-card.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class SubjectCardComponent {
  readonly subject = input.required<TeacherOverviewSubject>();
  /** Materia del aula abierta: cambia la acción de la tarjeta, no su contenido. */
  readonly openSessionSubjectId = input<string | null>(null);

  readonly hasQueue = computed(() => this.subject().pending_review > 0);

  readonly hasAula = computed(() => this.openSessionSubjectId() === this.subject().id);

  /** Sin cola y sin aula la tarjeta no lleva acción: su acción es abrir la materia. */
  readonly showGradeAction = computed(() => this.hasQueue());
  readonly showAulaAction = computed(() => this.hasAula() || this.hasQueue());

  readonly subjectLink = computed(() => ['/teacher/cursos', this.subject().id]);

  /** El enlace se anuncia con la materia y su trabajo, sin obligar a recorrer cifras. */
  readonly linkLabel = computed(() => {
    const subject = this.subject();
    if (this.hasQueue()) {
      return `${subject.code}, ${subject.name}, ${subject.pending_review} por calificar`;
    }
    return `${subject.code}, ${subject.name}`;
  });

  readonly hasMissing = computed(() => {
    const missing = this.subject().without_submission;
    return missing !== null && missing > 0;
  });

  /**
   * "Al día" significa que no queda nada de esta materia: ni corrección
   * pendiente ni alumnos sin entregar la práctica que vence. Con faltantes la
   * tarjeta no afirma que esté al día.
   */
  readonly isClear = computed(() => !this.hasQueue() && !this.hasMissing());
}
