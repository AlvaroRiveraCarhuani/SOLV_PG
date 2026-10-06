import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { LucideFileCode2 } from '@lucide/angular';
import { ExerciseEditorStore } from '../../exercise-editor.store';
import { ReferenceEditorComponent } from './reference-editor.component';
import { FormatBuilderComponent } from './format-builder/format-builder.component';
import { CasesTableComponent } from './cases-table/cases-table.component';
import { ScriptGeneratorComponent } from './script-generator/script-generator.component';
import { ScriptCaseValidationItem, TestCaseDTO } from '../../../models/teacher.models';

@Component({
  selector: 'step-contract',
  standalone: true,
  imports: [
    CommonModule,
    LucideFileCode2,
    ReferenceEditorComponent,
    FormatBuilderComponent,
    CasesTableComponent,
    ScriptGeneratorComponent
  ],
  template: `
    <div class="step-contract-layout">
      <!-- Sección 1: Contrato de Formato de Entrada (OA-02 / OA-05) -->
      <section class="form-section">
        <div class="section-header">
          <div class="header-left">
            <div class="title-row">
              <svg lucideFileCode2 class="section-icon"></svg>
              <h3 class="section-title">1. Contrato Declarativo de Formato (input_format)</h3>
            </div>
            <p class="section-desc">
              Define la gramática esperada de la entrada estándar. Permite validación sintáctica en tiempo real y fuzzing estructural.
            </p>
          </div>

          <div class="header-right">
            <span class="contract-status-pill" [class.active]="store.contract() !== null">
              {{ store.contract() ? 'Contrato Activo' : 'Modo Libre (Legacy)' }}
            </span>
          </div>
        </div>

        <format-builder />
      </section>

      <!-- Sección 1.5: Generación por Script Python (OA-18) -->
      <section class="form-section">
        <app-script-generator
          [exerciseId]="store.exerciseId() || undefined"
          (casesGenerated)="onCasesGenerated($event)"
        />
      </section>

      <!-- Sección 2: Casos de Prueba (OA-04) -->
      <section class="form-section">
        <cases-table />
      </section>

      <!-- Sección 3: Solución de Referencia -->
      <section class="form-section">
        <reference-editor />
      </section>
    </div>
  `,
  styleUrl: './step-contract.component.scss'
})
export class StepContractComponent {
  readonly store = inject(ExerciseEditorStore);

  onCasesGenerated(items: ScriptCaseValidationItem[]): void {
    const newCases: TestCaseDTO[] = items.map((item, i) => ({
      input: item.input,
      expected_output: item.expected_output,
      visibility: 'hidden',
      weight: 1.0,
      order_index: this.store.cases().length + i + 1
    }));
    this.store.addCases(newCases);
  }
}
