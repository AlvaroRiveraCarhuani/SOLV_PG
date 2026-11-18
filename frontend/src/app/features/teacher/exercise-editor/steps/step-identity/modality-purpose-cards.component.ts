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

      <!-- Chip Contextual Anti-Copia (Solo en Examen y Entorno de Evaluación) -->
      @if (store.metadata().purpose === 'exam' && store.metadata().environment_type === 'JUEZ_EFIMERO') {
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

      <!-- Red del examen: salida cerrada por defecto, excepción con motivo -->
      @if (store.metadata().purpose === 'exam') {
        <div class="config-cluster network-cluster">
          <span class="cluster-label">Acceso a internet</span>
          <div class="segmented-control" role="radiogroup" aria-label="Acceso a internet del examen">
            <button
              type="button"
              class="segment-btn"
              [class.active]="store.metadata().exam_network === 'none'"
              (click)="selectExamNetwork('none')"
              role="radio"
              [attr.aria-checked]="store.metadata().exam_network === 'none'"
            >
              <span>Sin internet</span>
            </button>
            <button
              type="button"
              class="segment-btn"
              [class.active]="store.metadata().exam_network === 'egress'"
              (click)="selectExamNetwork('egress')"
              role="radio"
              [attr.aria-checked]="store.metadata().exam_network === 'egress'"
            >
              <span>Excepción con motivo</span>
            </button>
          </div>
          @if (store.metadata().exam_network === 'egress') {
            <label class="network-reason-label" for="exam-network-reason">
              Motivo de la excepción (mínimo 10 caracteres)
            </label>
            <textarea
              id="exam-network-reason"
              class="network-reason"
              rows="2"
              placeholder="Ej. El examen consume una API pública de tipos de cambio"
              [value]="store.metadata().exam_network_reason"
              (input)="updateExamNetworkReason($event)"
            ></textarea>
          } @else {
            <p class="network-hint">
              El editor funciona, pero el entorno no tiene salida a internet. La excepción queda auditada.
            </p>
          }
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
      this.store.updateMetadata({ modality, environment_type: 'IDE_PERSISTENTE', per_student_seed: false });
    }
  }

  selectPurpose(purpose: 'class' | 'exam'): void {
    if (purpose === 'class') {
      this.store.updateMetadata({ purpose, per_student_seed: false });
    } else {
      this.store.updateMetadata({ purpose });
    }
  }

  selectExamNetwork(network: 'none' | 'egress'): void {
    this.store.updateMetadata({
      exam_network: network,
      exam_network_reason: network === 'none' ? '' : this.store.metadata().exam_network_reason
    });
  }

  updateExamNetworkReason(event: Event): void {
    const value = (event.target as HTMLTextAreaElement).value;
    this.store.updateMetadata({ exam_network: 'egress', exam_network_reason: value });
  }

  onSeedContainerClick(): void {
    if (this.store.metadata().purpose !== 'exam') {
      return;
    }
    this.store.updateMetadata({ per_student_seed: !this.store.metadata().per_student_seed });
  }
}
