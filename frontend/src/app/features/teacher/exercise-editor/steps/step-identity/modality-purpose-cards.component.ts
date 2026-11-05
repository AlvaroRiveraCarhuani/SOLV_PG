import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import {
  LucideCheckCheck,
  LucideTerminal,
  LucideBookOpen,
  LucideGraduationCap,
  LucideDices
} from '@lucide/angular';
import { ExerciseEditorStore } from '../../exercise-editor.store';

import { EnvironmentLabelPipe } from '@shared/pipes/environment-label.pipe';

@Component({
  selector: 'modality-purpose-cards',
  standalone: true,
  imports: [
    EnvironmentLabelPipe,
    CommonModule,
    FormsModule,
    LucideCheckCheck,
    LucideTerminal,
    LucideBookOpen,
    LucideGraduationCap,
    LucideDices
  ],
  template: `
    <div class="compact-config-bar">
      <!-- Modalidad de Ejecución -->
      <div class="config-cluster">
        <span class="cluster-label">Modalidad</span>
        <div class="segmented-control" role="radiogroup" aria-label="Modalidad de ejecución">
          <button
            type="button"
            class="segment-btn"
            [class.active]="store.metadata().modality === 'environment'"
            (click)="selectModality('environment')"
            role="radio"
            [attr.aria-checked]="store.metadata().modality === 'environment'"
          >
            <svg lucideTerminal class="segment-icon"></svg>
            <span>{{ 'IDE_PERSISTENTE' | environmentLabel }}</span>
          </button>

          <button
            type="button"
            class="segment-btn"
            [class.active]="store.metadata().modality === 'judge'"
            (click)="selectModality('judge')"
            role="radio"
            [attr.aria-checked]="store.metadata().modality === 'judge'"
          >
            <svg lucideCheckCheck class="segment-icon"></svg>
            <span>{{ 'JUEZ_EFIMERO' | environmentLabel }}</span>
          </button>
        </div>
      </div>

      <!-- Propósito Pedagógico -->
      <div class="config-cluster">
        <span class="cluster-label">Propósito</span>
        <div class="segmented-control" role="radiogroup" aria-label="Propósito pedagógico">
          <button
            type="button"
            class="segment-btn"
            [class.active]="store.metadata().purpose === 'class'"
            (click)="selectPurpose('class')"
            role="radio"
            [attr.aria-checked]="store.metadata().purpose === 'class'"
          >
            <svg lucideBookOpen class="segment-icon"></svg>
            <span>Clase / Formativo</span>
          </button>

          <button
            type="button"
            class="segment-btn"
            [class.active]="store.metadata().purpose === 'exam'"
            (click)="selectPurpose('exam')"
            role="radio"
            [attr.aria-checked]="store.metadata().purpose === 'exam'"
          >
            <svg lucideGraduationCap class="segment-icon"></svg>
            <span>Examen / Calificado</span>
          </button>
        </div>
      </div>

      <!-- Chip Contextual Anti-Copia (Solo revelado en Examen, sin carteles molestos) -->
      @if (store.metadata().purpose === 'exam') {
        <div
          id="student-seed-toggle"
          class="anticopia-chip"
          [class.active]="store.metadata().per_student_seed"
          [class.inactive]="!store.metadata().per_student_seed"
          (click)="onSeedContainerClick()"
          tabindex="0"
          (keydown.enter)="onSeedContainerClick()"
          role="switch"
          [attr.aria-checked]="store.metadata().per_student_seed"
          title="Genera datos y parámetros únicos por estudiante a partir de su ID para evitar plagio"
        >
          <svg lucideDices class="chip-icon"></svg>
          <span class="chip-text">
            Anti-copia:
            <strong>{{ store.metadata().per_student_seed ? 'Activado (Semilla única)' : 'Desactivado' }}</strong>
          </span>
        </div>
      }
    </div>
  `,
  styleUrl: './modality-purpose-cards.component.scss'
})
export class ModalityPurposeCardsComponent {
  readonly store = inject(ExerciseEditorStore);

  selectModality(modality: 'judge' | 'environment'): void {
    if (modality === 'judge') {
      this.store.updateMetadata({ modality, environment_type: 'JUEZ_EFIMERO' });
    } else {
      this.store.updateMetadata({ modality, environment_type: 'IDE_PERSISTENTE' });
    }
  }

  selectPurpose(purpose: 'class' | 'exam'): void {
    if (purpose === 'class') {
      this.store.updateMetadata({ purpose, per_student_seed: false });
    } else {
      this.store.updateMetadata({ purpose });
    }
  }

  onSeedContainerClick(): void {
    if (this.store.metadata().purpose !== 'exam') {
      return;
    }
    this.store.updateMetadata({ per_student_seed: !this.store.metadata().per_student_seed });
  }
}
