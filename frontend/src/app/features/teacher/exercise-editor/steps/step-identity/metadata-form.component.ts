import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { LucideTag, LucidePlus, LucideX } from '@lucide/angular';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { ExerciseEditorStore } from '../../exercise-editor.store';

@Component({
  selector: 'metadata-form',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideTag,
    LucidePlus,
    LucideX,
    MachineDataDirective
  ],
  template: `
    <div class="metadata-form-grid">
      <!-- Título del ejercicio -->
      <div class="form-group full-width">
        <label for="exercise-title" class="form-label">
          Título del ejercicio <span class="required">*</span>
        </label>
        <input 
          id="exercise-title"
          type="text" 
          class="form-input" 
          placeholder="Ej: Suma de Dos Números, Inversión de Matrices..."
          [ngModel]="store.metadata().title"
          (ngModelChange)="onTitleChange($event)"
          maxlength="200"
          required
        />
        @if (store.step1Validation().errors.includes('El título del ejercicio es obligatorio')) {
          <span class="field-error">El título es obligatorio</span>
        }
      </div>

      <!-- Dificultad -->
      <div class="form-group">
        <label for="exercise-difficulty" class="form-label">Dificultad sugerida</label>
        <select 
          id="exercise-difficulty"
          class="form-select"
          [ngModel]="store.metadata().difficulty"
          (ngModelChange)="onDifficultyChange($event)"
        >
          <option [ngValue]="null">Sin especificar</option>
          <option value="easy">Fácil (Básico / Introductorio)</option>
          <option value="medium">Medio (Intermedio / Estructuras)</option>
          <option value="hard">Difícil (Avanzado / Optimización)</option>
        </select>
      </div>

      <!-- Lenguaje Principal -->
      <div class="form-group">
        <label for="exercise-language" class="form-label">Lenguaje de ejecución</label>
        <select 
          id="exercise-language"
          class="form-select"
          [ngModel]="store.metadata().language"
          (ngModelChange)="onLanguageChange($event)"
        >
          <option value="python">Python 3.11</option>
          <option value="cpp">C++ 20 (GCC)</option>
          <option value="c">C 17 (GCC)</option>
          <option value="java">Java 21 (OpenJDK)</option>
          <option value="javascript">JavaScript (Node.js 20)</option>
          <option value="typescript">TypeScript 5.x</option>
          <option value="go">Go 1.22</option>
          <option value="rust">Rust 1.75</option>
        </select>
      </div>

      <!-- Tiempo Límite por Caso -->
      <div class="form-group">
        <label for="time-limit" class="form-label">
          Tiempo límite por caso (ms)
        </label>
        <input 
          id="time-limit"
          type="number" 
          class="form-input font-mono" 
          [ngModel]="store.metadata().time_limit_ms"
          (ngModelChange)="onTimeLimitChange($event)"
          min="100"
          max="30000"
          step="100"
        />
        <span class="field-hint">Tiempo máximo de ejecución para cada caso individual</span>
      </div>

      <!-- Límite de Memoria RAM -->
      <div class="form-group">
        <label for="memory-limit" class="form-label">
          Memoria RAM máxima (MB)
        </label>
        <input 
          id="memory-limit"
          type="number" 
          class="form-input font-mono" 
          [ngModel]="store.metadata().memory_limit_mb"
          (ngModelChange)="onMemoryLimitChange($event)"
          min="32"
          max="1024"
          step="32"
        />
        <span class="field-hint">Límite cgroup asignado al contenedor del runner</span>
      </div>

      <!-- Fecha de Entrega / Límite -->
      <div class="form-group">
        <label for="due-date" class="form-label">Fecha y hora de entrega (opcional)</label>
        <input 
          id="due-date"
          type="datetime-local" 
          class="form-input" 
          [ngModel]="store.metadata().due_date"
          (ngModelChange)="onDueDateChange($event)"
        />
        <span class="field-hint">Si se omite, el ejercicio no tiene fecha de cierre</span>
      </div>

      <!-- Tags / Etiquetas temáticas -->
      <div class="form-group full-width">
        <label for="tag-input" class="form-label">Etiquetas temáticas (Tags)</label>
        <div class="tags-container">
          <div class="tag-chips-list">
            @for (tag of store.metadata().tags; track tag) {
              <span class="tag-chip">
                <svg lucideTag class="tag-icon"></svg>
                <span class="tag-text" machineData>{{ tag }}</span>
                <button type="button" class="remove-tag-btn" (click)="removeTag(tag)" title="Eliminar etiqueta">
                  <svg lucideX class="remove-icon"></svg>
                </button>
              </span>
            }
          </div>

          <div class="tag-input-row">
            <input 
              #tagInput
              id="tag-input"
              type="text" 
              class="form-input tag-input" 
              placeholder="Ej: arreglos, dp, recursión, ordenamiento..."
              (keydown.enter)="addTag(tagInput.value); tagInput.value = ''; $event.preventDefault()"
            />
            <button 
              type="button" 
              class="btn-outline add-tag-btn"
              (click)="addTag(tagInput.value); tagInput.value = ''"
            >
              <svg lucidePlus class="btn-icon"></svg>
              <span>Agregar</span>
            </button>
          </div>
        </div>
        <span class="field-hint">Presiona Enter o click en Agregar para registrar cada etiqueta</span>
      </div>
    </div>
  `,
  styleUrl: './metadata-form.component.scss'
})
export class MetadataFormComponent {
  readonly store = inject(ExerciseEditorStore);

  onTitleChange(title: string) {
    this.store.updateMetadata({ title });
  }

  onDifficultyChange(difficulty: 'easy' | 'medium' | 'hard' | null) {
    this.store.updateMetadata({ difficulty });
  }

  onLanguageChange(language: string) {
    this.store.updateMetadata({ language, allowed_languages: [language] });
  }

  onTimeLimitChange(time_limit_ms: number) {
    this.store.updateMetadata({ time_limit_ms: Number(time_limit_ms) || 1000 });
  }

  onMemoryLimitChange(memory_limit_mb: number) {
    this.store.updateMetadata({ memory_limit_mb: Number(memory_limit_mb) || 128 });
  }

  onDueDateChange(due_date: string) {
    this.store.updateMetadata({ due_date });
  }

  addTag(rawTag: string) {
    const tag = rawTag.trim().toLowerCase();
    if (!tag) return;
    const currentTags = this.store.metadata().tags;
    if (!currentTags.includes(tag)) {
      this.store.updateMetadata({ tags: [...currentTags, tag] });
    }
  }

  removeTag(tagToRemove: string) {
    const currentTags = this.store.metadata().tags;
    this.store.updateMetadata({
      tags: currentTags.filter(t => t !== tagToRemove)
    });
  }
}
