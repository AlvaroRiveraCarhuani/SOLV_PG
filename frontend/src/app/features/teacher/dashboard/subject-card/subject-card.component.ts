import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';
import { RouterLink } from '@angular/router';
import { LucideArrowRight } from '@lucide/angular';

import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { TeacherOverviewSubject } from '../../models/teacher.models';

/**
 * Tarjeta de materia del inicio del docente. Es presentación pura: recibe la
 * fila del read model y no calcula cifras que el servidor ya resolvió.
 */
@Component({
  selector: 'subject-card',
  standalone: true,
  imports: [RouterLink, DateTextPipe, MachineDataDirective, LucideArrowRight],
  templateUrl: './subject-card.component.html',
  styleUrl: './subject-card.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class SubjectCardComponent {
  readonly subject = input.required<TeacherOverviewSubject>();

  readonly hasQueue = computed(() => this.subject().pending_review > 0);

  readonly hasMissing = computed(() => {
    const missing = this.subject().without_submission;
    return missing !== null && missing > 0;
  });

  /** Sin datos de faltantes no se dibuja una barra que aparente medición. */
  readonly hasDeliveryData = computed(() => this.subject().without_submission !== null);

  readonly deliveryPercent = computed(() => {
    const subject = this.subject();
    if (!subject.enrolled || subject.without_submission === null) {
      return 0;
    }
    const delivered = Math.max(0, subject.enrolled - subject.without_submission);
    return Math.min(100, Math.round((delivered / subject.enrolled) * 100));
  });

  readonly subjectLink = computed(() => ['/teacher/cursos', this.subject().id]);

  /** El enlace se anuncia con la materia y su trabajo pendiente. */
  readonly linkLabel = computed(() => {
    const subject = this.subject();
    if (this.hasQueue()) {
      return `${subject.code}, ${subject.name}, ${subject.pending_review} por calificar`;
    }
    return `${subject.code}, ${subject.name}`;
  });
}
