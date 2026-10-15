import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { LucideCode } from '@lucide/angular';
import { ExerciseEditorStore } from '../../exercise-editor.store';
import { ReferenceEditorComponent } from './reference-editor.component';
import { CasesTableComponent } from './cases-table/cases-table.component';

@Component({
  selector: 'step-contract',
  standalone: true,
  imports: [
    CommonModule,
    LucideCode,
    ReferenceEditorComponent,
    CasesTableComponent
  ],
  template: `
    <div class="step-contract-layout">
      <!-- 1. PRIMER PLANO: Solución Oficial de Referencia (Docente) -->
      <section class="form-section highlight-section" id="reference-solution-section">
        <div class="section-header">
          <div class="header-left">
            <div class="title-row">
              <svg lucideCode class="section-icon"></svg>
              <h3 class="section-title">1. Solución Oficial de Referencia</h3>
            </div>
            <p class="section-desc">
              Escribe o pega la solución óptima del ejercicio. El juez la utilizará para calcular automáticamente las salidas de tus casos de prueba y garantizar cero errores humanos.
            </p>
          </div>
        </div>

        <reference-editor />
      </section>

      <!-- 2. NÚCLEO: Casos de Prueba (Públicos y Ocultos) -->
      <section class="form-section" id="cases-table-section">
        <cases-table />
      </section>
    </div>
  `,
  styleUrl: './step-contract.component.scss'
})
export class StepContractComponent {
  readonly store = inject(ExerciseEditorStore);
}
