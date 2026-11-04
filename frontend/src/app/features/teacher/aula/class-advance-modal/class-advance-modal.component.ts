import { ChangeDetectionStrategy, Component, computed, inject, input, output, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { LucideSend } from '@lucide/angular';

import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { FieldMessageComponent } from '@shared/components/field-message/field-message.component';

import { ClassAdvanceService } from '../../services/class-advance.service';
import {
  AdvanceScope,
  ClassAdvanceEmission,
  ClassSessionStudent,
  EmitAdvanceRequest
} from '../../models/teacher.models';

/**
 * Emisión de un avance del aula (ADR-058). La práctica la fija el aula: lo que
 * se empaqueta se lee y se registra bajo la práctica de la sesión, así que el
 * docente solo elige a quién le llega. Un aula sin práctica no emite.
 *
 * Solo los alumnos que pasaron por el aula son destinatarios posibles: el
 * servidor rechaza a cualquier otro, así que la lista no los ofrece.
 */
@Component({
  selector: 'class-advance-modal',
  standalone: true,
  imports: [CommonModule, LucideSend, ModalShellComponent, FieldMessageComponent],
  templateUrl: './class-advance-modal.component.html',
  styleUrl: './class-advance-modal.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class ClassAdvanceModalComponent {
  readonly sessionId = input.required<string>();
  readonly students = input<ClassSessionStudent[]>([]);
  /** Práctica de la sesión: única fuente de verdad del avance. */
  readonly practiceId = input<string | null>(null);
  readonly practiceTitle = input<string | null>(null);

  readonly closed = output<void>();
  readonly emitted = output<ClassAdvanceEmission>();

  private readonly advances = inject(ClassAdvanceService);

  readonly isEmitting = this.advances.isEmitting;
  readonly serviceError = this.advances.error;

  readonly scope = signal<AdvanceScope>('class');
  readonly selectedIds = signal<string[]>([]);

  /** Sin práctica asociada el aula no tiene entorno que emitir ni recibir. */
  readonly hasPractice = computed(() => Boolean(this.practiceId()?.trim()));

  /** Alumnos que pasaron por el aula, en orden alfabético para elegir sin buscar. */
  readonly participants = computed<ClassSessionStudent[]>(() =>
    this.students()
      .filter(student => student.joined_at !== null)
      .sort((a, b) => a.name.localeCompare(b.name))
  );

  readonly selectedCount = computed(() => this.selectedIds().length);

  readonly canEmit = computed(() => {
    if (this.isEmitting() || !this.hasPractice()) {
      return false;
    }
    if (this.scope() === 'selected') {
      return this.selectedCount() > 0;
    }
    return this.participants().length > 0;
  });

  setScope(scope: AdvanceScope): void {
    this.scope.set(scope);
  }

  isSelected(studentId: string): boolean {
    return this.selectedIds().includes(studentId);
  }

  toggleStudent(studentId: string): void {
    this.selectedIds.update(ids =>
      ids.includes(studentId) ? ids.filter(id => id !== studentId) : [...ids, studentId]
    );
  }

  emit(): void {
    if (!this.canEmit()) {
      return;
    }
    const scope = this.scope();
    const request: EmitAdvanceRequest = { scope };
    if (scope === 'selected') {
      request.student_ids = this.selectedIds();
    }

    this.advances.emit(this.sessionId(), request, this.practiceTitle() ?? 'la práctica').subscribe({
      next: emission => this.emitted.emit(emission),
      error: () => undefined
    });
  }

  onClose(): void {
    if (this.isEmitting()) {
      return;
    }
    this.closed.emit();
  }
}
