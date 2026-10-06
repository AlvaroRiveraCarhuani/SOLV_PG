import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MetadataFormComponent } from './metadata-form.component';
import { ModalityPurposeCardsComponent } from './modality-purpose-cards.component';
import { StatementEditorComponent } from './statement-editor.component';

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
      <!-- Sección: Modalidad y Propósito -->
      <section class="form-section">
        <h3 class="section-title">1. Modalidad y Propósito Pedagógico</h3>
        <modality-purpose-cards />
      </section>

      <!-- Sección: Metadatos -->
      <section class="form-section">
        <h3 class="section-title">2. Identidad y Parámetros Técnicos</h3>
        <metadata-form />
      </section>

      <!-- Sección: Enunciado -->
      <section class="form-section">
        <h3 class="section-title">3. Enunciado y Especificación del Problema</h3>
        <statement-editor />
      </section>
    </div>
  `,
  styleUrl: './step-identity.component.scss'
})
export class StepIdentityComponent {}
