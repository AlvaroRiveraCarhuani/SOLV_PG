import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { LucideCode, LucideCheckCircle2 } from '@lucide/angular';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { ExerciseEditorStore } from '../../exercise-editor.store';

@Component({
  selector: 'reference-editor',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideCode,
    LucideCheckCircle2,
    MachineDataDirective
  ],
  template: `
    <div class="reference-editor-container">
      <div class="editor-header">
        <div class="header-info">
          <div class="title-row">
            <svg lucideCode class="header-icon"></svg>
            <span class="editor-title">Solución Oficial de Referencia (Docente)</span>
          </div>
          <span class="editor-subtitle">
            Código canónico del docente utilizado para validar casos de prueba, generar salidas automáticas y fuzzing.
          </span>
        </div>

        <div class="language-badge">
          <span class="lang-dot"></span>
          <span class="lang-name font-mono" machineData>{{ store.metadata().language | uppercase }}</span>
        </div>
      </div>

      <div class="code-editor-wrapper">
        <textarea 
          class="code-textarea font-mono"
          placeholder="# Escribe aquí la solución óptima y completa del docente..."
          [ngModel]="store.referenceSolution()"
          (ngModelChange)="onCodeChange($event)"
          rows="14"
          spellcheck="false"
        ></textarea>
      </div>

      <div class="editor-footer">
        @if (store.referenceSolution().trim()) {
          <div class="status-valid">
            <svg lucideCheckCircle2 class="status-icon"></svg>
            <span>Solución de referencia registrada (<span machineData>{{ store.referenceSolution().length }}</span> caracteres)</span>
          </div>
        } @else {
          <div class="status-warning">
            <span>Se recomienda incluir la solución de referencia para permitir verificación de casos.</span>
          </div>
        }
      </div>
    </div>
  `,
  styleUrl: './reference-editor.component.scss'
})
export class ReferenceEditorComponent {
  readonly store = inject(ExerciseEditorStore);

  onCodeChange(code: string) {
    this.store.setReferenceSolution(code);
  }
}
