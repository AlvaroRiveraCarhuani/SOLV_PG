import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { LucideCheckCircle2 } from '@lucide/angular';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { CodeEditorComponent } from '@shared/components/code-editor/code-editor.component';
import { ExerciseEditorStore } from '../../exercise-editor.store';

@Component({
  selector: 'reference-editor',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideCheckCircle2,
    MachineDataDirective,
    CodeEditorComponent
  ],
  template: `
    <div class="reference-editor-container">
      <div class="editor-header">
        <div class="header-info">
          <span class="editor-label">Código Canónico</span>
        </div>

        <div class="language-badge">
          <span class="lang-dot"></span>
          <span class="lang-name font-mono" machineData>{{ store.metadata().language | uppercase }}</span>
        </div>
      </div>

      <div class="code-editor-wrapper">
        <code-editor
          [value]="store.referenceSolution()"
          [language]="store.metadata().language"
          (valueChange)="onCodeChange($event)"
          minHeight="280px"
        />
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
