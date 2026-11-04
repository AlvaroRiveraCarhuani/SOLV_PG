import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import {
  LucideCpu,
  LucideTerminal,
  LucideBookOpen,
  LucideGraduationCap,
  LucideDices,
  LucideAlertCircle
} from '@lucide/angular';
import { ExerciseEditorStore } from '../../exercise-editor.store';

@Component({
  selector: 'modality-purpose-cards',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideCpu,
    LucideTerminal,
    LucideBookOpen,
    LucideGraduationCap,
    LucideDices,
    LucideAlertCircle
  ],
  template: `
    <div class="cards-section">
      <!-- Selección de Modalidad -->
      <div class="selection-block">
        <label class="block-title">Modalidad de Ejecución</label>
        <div class="cards-grid">
          <!-- Card: Juez Virtual -->
          <div
            class="choice-card"
            [class.selected]="store.metadata().modality === 'judge'"
            (click)="selectModality('judge')"
            tabindex="0"
            (keydown.enter)="selectModality('judge')"
          >
            <div class="card-radio">
              @if (store.metadata().modality === 'judge') {
                <span class="radio-dot"></span>
              }
            </div>
            <div class="card-icon-box">
              <svg lucideCpu class="choice-icon"></svg>
            </div>
            <div class="card-text">
              <span class="card-name">Entorno de Evaluación</span>
              <span class="card-desc">
                Ejecución aislada y corrección automática de entrada/salida estándar por casos de prueba.
              </span>
            </div>
          </div>

          <!-- Card: Environment Docker -->
          <div
            class="choice-card"
            [class.selected]="store.metadata().modality === 'environment'"
            (click)="selectModality('environment')"
            tabindex="0"
            (keydown.enter)="selectModality('environment')"
          >
            <div class="card-radio">
              @if (store.metadata().modality === 'environment') {
                <span class="radio-dot"></span>
              }
            </div>
            <div class="card-icon-box">
              <svg lucideTerminal class="choice-icon"></svg>
            </div>
            <div class="card-text">
              <span class="card-name">Entorno de Desarrollo</span>
              <span class="card-desc">
                Espacio de trabajo interactivo y continuo con editor VS Code.
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- Selección de Propósito -->
      <div class="selection-block">
        <label class="block-title">Propósito Pedagógico</label>
        <div class="cards-grid">
          <!-- Card: Clase / Práctica -->
          <div
            class="choice-card"
            [class.selected]="store.metadata().purpose === 'class'"
            (click)="selectPurpose('class')"
            tabindex="0"
            (keydown.enter)="selectPurpose('class')"
          >
            <div class="card-radio">
              @if (store.metadata().purpose === 'class') {
                <span class="radio-dot"></span>
              }
            </div>
            <div class="card-icon-box">
              <svg lucideBookOpen class="choice-icon"></svg>
            </div>
            <div class="card-text">
              <span class="card-name">Clase / Formativo</span>
              <span class="card-desc">
                Diseñado para el aprendizaje continuo, retroalimentación detallada y práctica libre.
              </span>
            </div>
          </div>

          <!-- Card: Examen / Sumativo -->
          <div
            class="choice-card"
            [class.selected]="store.metadata().purpose === 'exam'"
            (click)="selectPurpose('exam')"
            tabindex="0"
            (keydown.enter)="selectPurpose('exam')"
          >
            <div class="card-radio">
              @if (store.metadata().purpose === 'exam') {
                <span class="radio-dot"></span>
              }
            </div>
            <div class="card-icon-box">
              <svg lucideGraduationCap class="choice-icon"></svg>
            </div>
            <div class="card-text">
              <span class="card-name">Examen / Calificado</span>
              <span class="card-desc">
                Evaluación con ponderación estricta, temporizadores y protección anti-trampas.
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- Toggle Avanzado: Semilla por Estudiante -->
      <div
        id="student-seed-toggle"
        class="seed-toggle-container"
        [class.disabled]="store.metadata().purpose !== 'exam'"
        [class.active]="store.metadata().purpose === 'exam' && store.metadata().per_student_seed"
        [class.inactive]="store.metadata().purpose !== 'exam' || !store.metadata().per_student_seed"
        (click)="onSeedContainerClick()"
        [attr.tabindex]="store.metadata().purpose === 'exam' ? 0 : -1"
        [attr.aria-disabled]="store.metadata().purpose !== 'exam'"
        (keydown.enter)="onSeedContainerClick()"
      >
        <div class="seed-toggle-header">
          <div class="seed-info">
            <div class="seed-title-row">
              <svg lucideDices class="seed-icon"></svg>
              <span class="seed-title">Variantes automáticas por estudiante (Anti-copia)</span>
            </div>
            <span class="seed-description">
              Genera datos personalizados para cada estudiante a partir de su ID para evitar que se copien durante evaluaciones.
            </span>
          </div>

          <div class="switch-toggle" [class.checked]="store.metadata().per_student_seed">
            <span class="slider"></span>
          </div>
        </div>

        @if (store.metadata().purpose !== 'exam') {
          <div class="seed-disabled-hint">
            <svg lucideAlertCircle class="hint-icon"></svg>
            <span>Disponible solo en evaluaciones calificadas. Elegí <strong>Examen / Calificado</strong> para habilitarla.</span>
          </div>
        }
      </div>
    </div>
  `,
  styleUrl: './modality-purpose-cards.component.scss'
})
export class ModalityPurposeCardsComponent {
  readonly store = inject(ExerciseEditorStore);

  selectModality(modality: 'judge' | 'environment') {
    if (modality === 'judge') {
      this.store.updateMetadata({ modality, environment_type: 'JUEZ_EFIMERO' });
    } else {
      this.store.updateMetadata({ modality, environment_type: 'IDE_PERSISTENTE' });
    }
  }

  selectPurpose(purpose: 'class' | 'exam') {
    if (purpose === 'class') {
      this.store.updateMetadata({ purpose, per_student_seed: false });
    } else {
      this.store.updateMetadata({ purpose });
    }
  }

  toggleSeed(event: Event) {
    const checked = (event.target as HTMLInputElement).checked;
    this.store.updateMetadata({ per_student_seed: checked });
  }

  onSeedContainerClick() {
    if (this.store.metadata().purpose !== 'exam') {
      return;
    }
    this.store.updateMetadata({ per_student_seed: !this.store.metadata().per_student_seed });
  }
}
