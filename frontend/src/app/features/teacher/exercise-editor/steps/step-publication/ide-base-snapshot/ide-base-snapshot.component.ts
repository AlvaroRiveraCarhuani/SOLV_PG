import { Component, computed, inject } from '@angular/core';
import { CodeEditorComponent } from '@shared/components/code-editor/code-editor.component';
import { ExerciseEditorStore } from '../../../exercise-editor.store';

/**
 * Proyecto base de un ejercicio IDE.
 *
 * Se guarda en `exercises.boilerplate` con el mismo formato del snapshot de
 * entrega (`# === FILE: ruta ===`). SpeedGrader lo parsea y compara cada archivo
 * del alumno contra su homónimo; no hay un segundo formato que mantener.
 */
@Component({
  selector: 'ide-base-snapshot',
  standalone: true,
  imports: [CodeEditorComponent],
  templateUrl: './ide-base-snapshot.component.html',
  styleUrl: './ide-base-snapshot.component.scss'
})
export class IdeBaseSnapshotComponent {
  readonly store = inject(ExerciseEditorStore);
  readonly code = computed(() => this.store.boilerplate());

  readonly fileCount = computed(() => {
    const matches = this.code().match(/^# === FILE: .+ ===$/gm);
    return matches ? matches.length : 0;
  });

  readonly hasFormatWarning = computed(
    () => this.code().trim().length > 0 && this.fileCount() === 0
  );

  onCodeChange(value: string): void {
    this.store.setBoilerplate(value);
  }
}
