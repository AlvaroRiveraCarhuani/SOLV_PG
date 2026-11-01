import { ChangeDetectionStrategy, Component, OnInit, computed, inject, input, output, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { LucideSend } from '@lucide/angular';

import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import { FieldMessageComponent } from '@shared/components/field-message/field-message.component';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';

import { ClassAdvanceService } from '../../services/class-advance.service';
import {
  AdvanceScope,
  ClassAdvanceEmission,
  ClassSessionStudent,
  EmitAdvanceRequest
} from '../../models/teacher.models';

/**
 * Emisión de un avance del aula (ADR-058). El docente elige la práctica y a
 * quién le llega; lo que se emite es el código de su entorno de clase tal como
 * está en este momento, no una copia que siga sus ediciones posteriores.
 *
 * Solo los alumnos que pasaron por el aula son destinatarios posibles: el
 * servidor rechaza a cualquier otro, así que la lista no los ofrece.
 */
@Component({
  selector: 'class-advance-modal',
  standalone: true,
  imports: [
    CommonModule,
    LucideSend,
    ModalShellComponent,
    FormFieldComponent,
    FieldMessageComponent,
    ComboboxComponent
  ],
  templateUrl: './class-advance-modal.component.html',
  styleUrl: './class-advance-modal.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class ClassAdvanceModalComponent implements OnInit {
  readonly sessionId = input.required<string>();
  readonly practices = input<ComboboxOption[]>([]);
  readonly students = input<ClassSessionStudent[]>([]);
  /** Práctica del aula, si la tiene: se ofrece ya elegida. */
  readonly defaultExerciseId = input<string | null>(null);

  readonly closed = output<void>();
  readonly emitted = output<ClassAdvanceEmission>();

  private readonly advances = inject(ClassAdvanceService);

  readonly isEmitting = this.advances.isEmitting;
  readonly serviceError = this.advances.error;

  readonly exerciseId = signal<string>('');
  readonly scope = signal<AdvanceScope>('class');
  readonly selectedIds = signal<string[]>([]);

  /** Alumnos que pasaron por el aula, en orden alfabético para elegir sin buscar. */
  readonly participants = computed<ClassSessionStudent[]>(() =>
    this.students()
      .filter(student => student.joined_at !== null)
      .sort((a, b) => a.name.localeCompare(b.name))
  );

  readonly selectedCount = computed(() => this.selectedIds().length);

  readonly canEmit = computed(() => {
    if (this.isEmitting() || !this.exerciseId()) {
      return false;
    }
    if (this.scope() === 'selected') {
      return this.selectedCount() > 0;
    }
    return this.participants().length > 0;
  });

  ngOnInit(): void {
    const preselected = this.defaultExerciseId();
    if (preselected) {
      this.exerciseId.set(preselected);
    }
  }

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
    const request: EmitAdvanceRequest = {
      exercise_id: this.exerciseId(),
      scope
    };
    if (scope === 'selected') {
      request.student_ids = this.selectedIds();
    }

    this.advances.emit(this.sessionId(), request, this.practiceTitle()).subscribe({
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

  private practiceTitle(): string {
    const id = this.exerciseId();
    return this.practices().find(option => option.id === id)?.label ?? 'la práctica';
  }
}
