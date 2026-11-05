import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MetadataFormComponent } from './metadata-form.component';
import { ModalityPurposeCardsComponent } from './modality-purpose-cards.component';
import { StatementEditorComponent } from './statement-editor.component';
import { ExerciseEditorStore } from '../../exercise-editor.store';

@Component({
  selector: 'step-identity',
  standalone: true,
  imports: [
    CommonModule,
    MetadataFormComponent,
    ModalityPurposeCardsComponent,
    StatementEditorComponent
  ],
  template: `
    <div class="step-identity-layout">
      <!-- Sección 1: Modalidad y Propósito Compacto -->
      <section class="form-section">
        <modality-purpose-cards />
      </section>

      <!-- Sección 2: Identidad y Datos del Laboratorio -->
      <section class="form-section">
        <metadata-form />
      </section>

      <!-- Sección 3: Consigna Pedagógica (solo cuando es Entorno de Evaluación) -->
      @if (store.metadata().environment_type === 'JUEZ_EFIMERO') {
        <section class="form-section">
          <statement-editor />
        </section>
      }
    </div>
  `,
  styleUrl: './step-identity.component.scss'
})
export class StepIdentityComponent {
  readonly store = inject(ExerciseEditorStore);
}
