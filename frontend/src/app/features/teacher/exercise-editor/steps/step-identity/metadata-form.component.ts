import { Component, inject, signal, computed, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { LucideCalendar } from '@lucide/angular';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { FieldMessageComponent } from '@shared/components/field-message/field-message.component';
import { TagInputComponent } from '@shared/components/tag-input/tag-input.component';
import { ExerciseEditorStore } from '../../exercise-editor.store';

@Component({
  selector: 'metadata-form',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideCalendar,
    ComboboxComponent,
    FieldMessageComponent,
    TagInputComponent
  ],
  template: `
    <div class="metadata-form-grid">
      <!-- Título del ejercicio -->
      <div class="form-group full-width">
        <label for="exercise-title" class="form-label">
          {{ titleLabel() }} <span class="required">*</span>
        </label>
        <input 
          id="exercise-title"
          type="text" 
          class="form-input" 
          [class.has-error]="showTitleError()"
          placeholder="Ej: API de productos, inversión de matrices..."
          [ngModel]="store.metadata().title"
          (ngModelChange)="onTitleChange($event)"
          (blur)="onTitleBlur()"
          maxlength="200"
          required
        />
        @if (showTitleError()) {
          <field-message variant="error" [message]="'El título es obligatorio'"></field-message>
        }
      </div>

      @if (store.metadata().environment_type === 'JUEZ_EFIMERO') {
        <!-- Complejidad Esperada (Opcional) -->
        <div class="form-group" id="exercise-complexity">
          <label class="form-label">Complejidad esperada <span class="optional-label">(opcional)</span></label>
          <combobox
            [options]="complexityOptions"
            [value]="store.metadata().expected_complexity || ''"
            [placeholder]="'Sin especificar...'"
            [showSearchIcon]="false"
            [compact]="true"
            (optionSelected)="onComplexitySelect($event)"
            inputAriaLabel="Complejidad algorítmica esperada"
          />
          <span class="field-hint">Cota esperada para validar la eficiencia de la solución.</span>
        </div>

        <!-- Lenguaje de ejecución -->
        <div class="form-group" id="exercise-language">
          <label class="form-label">
            Lenguaje de ejecución <span class="required">*</span>
          </label>
          <combobox
            [options]="languageOptions()"
            [value]="store.metadata().language"
            [maxSuggestions]="8"
            [placeholder]="'Seleccionar lenguaje...'"
            [showSearchIcon]="true"
            [compact]="true"
            [hasError]="showLanguageError()"
            (optionSelected)="onLanguageSelect($event)"
            inputAriaLabel="Lenguaje de ejecución"
          />
          @if (showLanguageError()) {
            <field-message variant="error" [message]="'Debes seleccionar un lenguaje de ejecución'"></field-message>
          }
        </div>

        <!-- Tiempo Límite por Caso (Avanzado) -->
        <details class="advanced-limit-details form-group full-width">
          <summary class="advanced-summary">Ajuste de tiempo límite por caso (opcional)</summary>
          <div class="form-group" style="margin-top: var(--space-2); max-width: 320px;">
            <label for="time-limit" class="form-label">
              Tiempo límite por caso (ms)
            </label>
            <input 
              id="time-limit"
              type="text" 
              class="form-input font-mono" 
              [class.has-error]="showTimeLimitError()"
              [value]="rawTimeLimit()"
              (input)="onTimeLimitInput($event)"
              (blur)="onTimeLimitBlur()"
              placeholder="Ej: 1000"
              maxlength="7"
            />
            @if (showTimeLimitError()) {
              <field-message variant="error" [message]="timeLimitError()!"></field-message>
            } @else {
              <span class="field-hint">Tiempo máximo para cada caso individual (por defecto 1000 ms)</span>
            }
          </div>
        </details>
      }

      <!-- Fecha y hora de entrega para Entorno de Evaluación -->
      @if (store.metadata().environment_type === 'JUEZ_EFIMERO') {
        <div class="form-group due-date-group full-width">
          <div class="due-date-header-row">
            <label class="form-label">Fecha y hora de entrega <span class="optional-label">(opcional)</span></label>
            @if (hasDueDate()) {
              <div class="preset-chips">
                <button type="button" class="preset-pill" (click)="applyPresetDueDate(7)">+1 semana</button>
                <button type="button" class="preset-pill" (click)="applyPresetDueDate('sunday')">Domingo 23:59</button>
              </div>
            }
          </div>

          <div class="due-date-toggle-row">
            <button 
              type="button" 
              class="mode-toggle-btn" 
              [class.active]="!hasDueDate()"
              (click)="setHasDueDate(false)"
            >
              Sin fecha límite
            </button>
            <button 
              type="button" 
              class="mode-toggle-btn" 
              [class.active]="hasDueDate()"
              (click)="setHasDueDate(true)"
            >
              <svg lucideCalendar class="btn-icon"></svg>
              <span>Con fecha límite</span>
            </button>
          </div>

          @if (hasDueDate()) {
            <div class="due-date-input-container">
              <input 
                id="due-date"
                type="datetime-local" 
                class="form-input due-date-picker font-mono" 
                [ngModel]="store.metadata().due_date"
                (ngModelChange)="onDueDateChange($event)"
              />
            </div>
          }
        </div>
      }

      <!-- Tags / Etiquetas -->
      <div class="form-group full-width tags-section">
        <label for="exercise-tag-input" class="form-label">
          Etiquetas <span class="optional-label">(opcional)</span>
        </label>
        <tag-input
          inputId="exercise-tag-input"
          [tags]="store.metadata().tags"
          (tagsChange)="onTagsChange($event)"
        />
        <span class="field-hint">Para clasificar y reutilizar la práctica en tu biblioteca docente.</span>
      </div>
    </div>
  `,
  styleUrl: './metadata-form.component.scss'
})
export class MetadataFormComponent implements OnInit {
  readonly store = inject(ExerciseEditorStore);

  readonly titleTouched = signal<boolean>(false);
  readonly hasDueDate = signal<boolean>(!!this.store.metadata().due_date);

  readonly titleLabel = computed(() => {
    return this.store.metadata().purpose === 'exam' ? 'Título del examen' : 'Título del laboratorio';
  });

  readonly defaultLanguages: ComboboxOption<string>[] = [
    { id: 'python', label: 'Python 3.11 (Python Runner Base)', value: 'python' },
    { id: 'cpp', label: 'C++ 20 (C/C++ Runner Base)', value: 'cpp' },
    { id: 'c', label: 'C 17 (C/C++ Runner Base)', value: 'c' },
    { id: 'java', label: 'Java 21 (Java Runner Base)', value: 'java' },
    { id: 'javascript', label: 'JavaScript (Node.js 20)', value: 'javascript' },
    { id: 'typescript', label: 'TypeScript 5.x', value: 'typescript' },
    { id: 'go', label: 'Go 1.24 (Go Runner Base)', value: 'go' },
    { id: 'rust', label: 'Rust 1.75 (Rust Runner Base)', value: 'rust' },
    { id: 'csharp', label: 'C# (C# Runner Base)', value: 'csharp' }
  ];

  readonly languageOptions = computed<ComboboxOption<string>[]>(() => {
    const runners = this.store.runnerTemplates();
    if (runners.length === 0) return this.defaultLanguages;

    const options: ComboboxOption<string>[] = runners.map(r => {
      const langCode = this.store.deriveLanguageCode(r);
      const rawName = r.name || langCode;
      const label = rawName.toLowerCase().includes(langCode.toLowerCase())
        ? rawName
        : `${rawName} (${langCode.toUpperCase()})`;
      return {
        id: langCode,
        label,
        value: langCode
      };
    });

    const combined = [...options, ...this.defaultLanguages];
    return combined.filter((opt, index, self) => 
      index === self.findIndex(o => o.value === opt.value)
    );
  });

  readonly languageTouched = signal<boolean>(false);
  readonly showLanguageError = computed(() => {
    return !this.store.metadata().language && (this.languageTouched() || this.store.stepAttempted());
  });

  readonly rawTimeLimit = signal<string>(String(this.store.metadata().time_limit_ms || 1000));
  readonly timeLimitTouched = signal<boolean>(false);

  readonly timeLimitError = computed<string | null>(() => {
    const raw = this.rawTimeLimit().trim();
    if (!raw) return 'El tiempo límite es obligatorio';
    if (!/^-?\d+$/.test(raw)) return 'Ingresa un valor numérico válido (en milisegundos)';
    const num = Number(raw);
    if (isNaN(num)) return 'Ingresa un valor numérico válido';
    if (num < 100) return 'El tiempo límite no puede ser negativo ni inferior a 100 ms';
    if (num > 30000) return 'El tiempo límite no puede superar los 30.000 ms (30 segundos)';
    return null;
  });

  readonly showTimeLimitError = computed<boolean>(() => {
    return !!this.timeLimitError() && (this.timeLimitTouched() || this.store.stepAttempted());
  });

  readonly showTitleError = computed(() => {
    const isBlank = !this.store.metadata().title.trim();
    return isBlank && (this.titleTouched() || this.store.stepAttempted());
  });

  readonly complexityOptions: ComboboxOption<string>[] = [
    { id: '', label: 'Sin especificar', value: '' },
    { id: 'O(1)', label: 'O(1) - Constante', value: 'O(1)' },
    { id: 'O(N)', label: 'O(N) - Lineal', value: 'O(N)' },
    { id: 'O(N log N)', label: 'O(N log N) - Log-lineal', value: 'O(N log N)' },
    { id: 'O(N²)', label: 'O(N²) - Cuadrática', value: 'O(N²)' }
  ];

  ngOnInit(): void {
    this.store.loadRunnerTemplates();
  }

  onTitleBlur(): void {
    this.titleTouched.set(true);
    this.store.markFieldTouched('title');
  }

  onTimeLimitBlur(): void {
    this.timeLimitTouched.set(true);
    this.store.markFieldTouched('time_limit');
  }

  onTitleChange(title: string): void {
    this.store.updateMetadata({ title });
  }

  onComplexitySelect(option: ComboboxOption<string>): void {
    this.store.updateMetadata({ expected_complexity: option.value || undefined });
  }

  onLanguageSelect(option: ComboboxOption<string>): void {
    this.languageTouched.set(true);
    const language = option.value || '';
    this.store.setLanguageAndSyncTemplate(language);
  }

  onTimeLimitInput(event: Event): void {
    const val = (event.target as HTMLInputElement).value;
    this.rawTimeLimit.set(val);
    const trimmed = val.trim();
    if (/^\d+$/.test(trimmed)) {
      const num = Number(trimmed);
      if (num >= 100 && num <= 30000) {
        this.store.updateMetadata({ time_limit_ms: num });
        return;
      }
    }
    this.store.updateMetadata({ time_limit_ms: -1 });
  }

  setHasDueDate(enabled: boolean): void {
    this.hasDueDate.set(enabled);
    if (!enabled) {
      this.store.updateMetadata({ due_date: undefined });
    } else if (!this.store.metadata().due_date) {
      this.applyPresetDueDate(7);
    }
  }

  applyPresetDueDate(days: number | 'sunday'): void {
    const d = new Date();
    if (days === 'sunday') {
      const diff = (7 - d.getDay()) % 7 || 7;
      d.setDate(d.getDate() + diff);
    } else {
      d.setDate(d.getDate() + days);
    }
    d.setHours(23, 59, 0, 0);

    const year = d.getFullYear();
    const month = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');
    const hours = String(d.getHours()).padStart(2, '0');
    const mins = String(d.getMinutes()).padStart(2, '0');
    const localIso = `${year}-${month}-${day}T${hours}:${mins}`;

    this.hasDueDate.set(true);
    this.store.updateMetadata({ due_date: localIso });
  }

  onDueDateChange(due_date: string): void {
    this.store.updateMetadata({ due_date: due_date || undefined });
  }

  onTagsChange(tags: string[]): void {
    this.store.updateMetadata({ tags });
  }
}
