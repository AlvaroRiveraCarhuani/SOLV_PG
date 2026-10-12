import { Component, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { 
  LucideFileCode2, 
  LucideCode, 
  LucideSliders, 
  LucideChevronDown, 
  LucideChevronUp
} from '@lucide/angular';
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
    LucideCode,
    LucideSliders,
    LucideChevronDown,
    LucideChevronUp,
    ReferenceEditorComponent,
    FormatBuilderComponent,
    CasesTableComponent,
    ScriptGeneratorComponent
  ],
  template: `
    <div class="step-contract-layout">
      <!-- 1. PRIMER PLANO: Solución Oficial de Referencia (Docente) -->
      <section class="form-section highlight-section">
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
      <section class="form-section">
        <cases-table />
      </section>

      <!-- 3. CAPA AVANZADA (Divulgación Progresiva): Generación Asistida y Fuzzing -->
      <section class="form-section advanced-section">
        <div class="advanced-toggle-bar" (click)="toggleAdvanced()" (keydown.enter)="toggleAdvanced()" tabindex="0" role="button" [attr.aria-expanded]="showAdvanced()">
          <div class="toggle-left">
            <div class="title-row">
              <svg lucideSliders class="section-icon text-muted"></svg>
              <h4 class="advanced-title">Herramientas Avanzadas de Generación y Fuzzing</h4>
              <span class="advanced-badge">Opcional</span>
            </div>
            <p class="advanced-desc">
              Crea lotes masivos de pruebas utilizando plantillas de formato o scripts generadores en Python para torneos o exámenes de alto estrés.
            </p>
          </div>

          <div class="toggle-right">
            <button type="button" class="btn-toggle-icon" aria-label="Alternar panel avanzado">
              @if (showAdvanced()) {
                <svg lucideChevronUp class="chevron-icon"></svg>
              } @else {
                <svg lucideChevronDown class="chevron-icon"></svg>
              }
            </button>
          </div>
        </div>

        @if (showAdvanced()) {
          <div class="advanced-content animate-fade">
            <!-- Pestaña / Subsección: Esquema de Formato de Entrada -->
            <div class="advanced-block">
              <div class="block-header">
                <svg lucideFileCode2 class="block-icon"></svg>
                <h5 class="block-title">Esquema Estructural de Entrada (stdin)</h5>
              </div>
              <format-builder />
            </div>

            <!-- Subsección: Generación por Script Python -->
            <div class="advanced-block mt-4">
              <app-script-generator
                [exerciseId]="store.exerciseId() || undefined"
                (casesGenerated)="onCasesGenerated($event)"
              />
            </div>
          </div>
        }
      </section>
    </div>
  `,
  styleUrl: './step-contract.component.scss'
})
export class StepContractComponent {
  readonly store = inject(ExerciseEditorStore);
  readonly showAdvanced = signal<boolean>(false);

  toggleAdvanced(): void {
    this.showAdvanced.update(v => !v);
  }

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
