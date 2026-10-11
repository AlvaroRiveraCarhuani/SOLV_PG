import { Component, inject, signal, computed, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { FormsModule } from '@angular/forms';
import { 
  LucideTag, 
  LucidePlus, 
  LucideX, 
  LucideCalendar, 
  LucideClock,
  LucideAlertTriangle
} from '@lucide/angular';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { FieldMessageComponent } from '@shared/components/field-message/field-message.component';
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
    LucideCalendar,
    LucideClock,
    LucideAlertTriangle,
    MachineDataDirective,
    ComboboxComponent,
    FieldMessageComponent
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
          [class.has-warning]="showTitleError()"
          placeholder="Ej: Suma de Dos Números, Inversión de Matrices..."
          [ngModel]="store.metadata().title"
          (ngModelChange)="onTitleChange($event)"
          (blur)="titleTouched.set(true)"
          maxlength="200"
          required
        />
        @if (showTitleError()) {
          <field-message variant="warning" [message]="'El título del ejercicio es obligatorio'"></field-message>
        }
      </div>

      <!-- Dificultad sugerida (Opcional) -->
      <div class="form-group">
        <label class="form-label">Dificultad sugerida <span class="optional-label">(opcional)</span></label>
        <combobox
          [options]="difficultyOptions"
          [value]="store.metadata().difficulty || ''"
          [placeholder]="'Selecciona un nivel o sin especificar...'"
          [showSearchIcon]="false"
          [compact]="true"
          (optionSelected)="onDifficultySelect($event)"
          inputAriaLabel="Dificultad sugerida"
        />
      </div>

      @if (store.metadata().environment_type === 'JUEZ_EFIMERO') {
        <!-- Complejidad Esperada (Opcional) -->
        <div class="form-group">
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
          <span class="field-hint">Si la defines, el sistema la comparará con el rendimiento del código entregado</span>
        </div>

        <!-- Lenguaje de ejecución -->
        <div class="form-group">
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
            (optionSelected)="onLanguageSelect($event)"
            inputAriaLabel="Lenguaje de ejecución"
          />
          @if (showLanguageError()) {
            <field-message variant="warning" [message]="'Debes seleccionar un lenguaje de ejecución'"></field-message>
          } @else {
            <span class="field-hint">Lenguaje en el que los estudiantes escribirán y probarán su código</span>
          }
        </div>

        <!-- Tiempo Límite por Caso -->
        <div class="form-group">
          <label for="time-limit" class="form-label">
            Tiempo límite por caso (ms) <span class="required">*</span>
          </label>
          <input 
            id="time-limit"
            type="text" 
            class="form-input font-mono" 
            [class.has-error]="showTimeLimitError()"
            [value]="rawTimeLimit()"
            (input)="onTimeLimitInput($event)"
            (blur)="timeLimitTouched.set(true)"
            placeholder="Ej: 1000"
            maxlength="7"
          />
          @if (showTimeLimitError()) {
            <field-message variant="error" [message]="timeLimitError()!"></field-message>
          } @else {
            <span class="field-hint">Tiempo máximo de ejecución para cada caso individual (100 - 30.000 ms)</span>
          }
        </div>
      }

      <!-- Límite de Memoria RAM (Solo Lectura) -->
      <div class="form-group">
        <label for="memory-limit-display" class="form-label">
          Memoria RAM asignada
        </label>
        <div class="ram-chip-display" id="memory-limit-display">
          <span class="ram-badge font-mono" machineData>
            {{ store.metadata().template?.base_ram_mb || store.metadata().memory_limit_mb || 128 }} MB
          </span>
          <span class="governance-tag">
            {{ store.metadata().environment_type === 'JUEZ_EFIMERO' ? 'Límite estándar de evaluación' : 'Definida en la plantilla del catálogo' }}
          </span>
        </div>
        <span class="field-hint">
          {{ store.metadata().environment_type === 'JUEZ_EFIMERO' 
            ? 'Límite de memoria asignado al contenedor de ejecución de pruebas' 
            : 'Límite de memoria asignado por la plantilla técnica seleccionada' }}
        </span>
      </div>

      <!-- Fecha y hora de entrega (Opcional con selector previo) -->
      <div class="form-group due-date-group">
        <label class="form-label">Fecha y hora de entrega <span class="optional-label">(opcional)</span></label>
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
              class="form-input due-date-picker" 
              [ngModel]="store.metadata().due_date"
              (ngModelChange)="onDueDateChange($event)"
            />
          </div>
        }
      </div>

      <!-- Tags / Etiquetas temáticas -->
      <div class="form-group full-width tags-section">
        <label for="tag-input" class="form-label">Etiquetas temáticas (Tags)</label>
        <div class="tags-container">
          <div class="tag-input-row">
            <input 
              #tagInput
              id="tag-input"
              type="text" 
              class="form-input tag-input" 
              placeholder="Ej: arreglos, dp, recursión, ordenamiento..."
              (input)="clearTagError()"
              (keydown.enter)="addTag(tagInput); $event.preventDefault()"
            />
            <button 
              type="button" 
              class="btn-outline add-tag-btn"
              (click)="addTag(tagInput)"
            >
              <svg lucidePlus class="btn-icon"></svg>
              <span>Agregar</span>
            </button>
          </div>

          @if (tagError()) {
            <field-message variant="warning" [message]="tagError()!"></field-message>
          }

          @if (store.metadata().tags.length > 0) {
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
          }
        </div>
      </div>
    </div>
  `,
  styleUrl: './metadata-form.component.scss'
})
export class MetadataFormComponent implements OnInit, OnDestroy {
  readonly store = inject(ExerciseEditorStore);
  private readonly http = inject(HttpClient);

  readonly titleTouched = signal<boolean>(false);
  readonly tagError = signal<string | null>(null);
  private tagErrorTimer: ReturnType<typeof setTimeout> | null = null;

  readonly hasDueDate = signal<boolean>(!!this.store.metadata().due_date);

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

  readonly runnerTemplates = signal<any[]>([]);
  readonly languageOptions = signal<ComboboxOption<string>[]>(this.defaultLanguages);

  readonly languageTouched = signal<boolean>(false);
  readonly showLanguageError = computed(() => {
    return !this.store.metadata().language && (this.languageTouched() || this.store.stepAttempted());
  });

  readonly rawTimeLimit = signal<string>(String(this.store.metadata().time_limit_ms || 1000));
  readonly timeLimitTouched = signal<boolean>(false);

  readonly timeLimitError = computed<string | null>(() => {
    const raw = this.rawTimeLimit().trim();
    if (!raw) {
      return 'El tiempo límite es obligatorio';
    }
    if (!/^-?\d+$/.test(raw)) {
      return 'Ingresa un valor numérico válido (en milisegundos)';
    }
    const num = Number(raw);
    if (isNaN(num)) {
      return 'Ingresa un valor numérico válido';
    }
    if (num < 100) {
      return 'El tiempo límite no puede ser negativo ni inferior a 100 ms';
    }
    if (num > 30000) {
      return 'El tiempo límite no puede superar los 30.000 ms (30 segundos)';
    }
    return null;
  });

  readonly showTimeLimitError = computed<boolean>(() => {
    return !!this.timeLimitError() && (this.timeLimitTouched() || this.store.stepAttempted());
  });

  readonly showTitleError = computed(() => {
    const isBlank = !this.store.metadata().title.trim();
    return isBlank && (this.titleTouched() || this.store.stepAttempted());
  });

  readonly difficultyOptions: ComboboxOption<string>[] = [
    { id: '', label: 'Sin especificar', value: '' },
    { id: 'easy', label: 'Fácil (Básico / Introductorio)', value: 'easy' },
    { id: 'medium', label: 'Medio (Intermedio / Estructuras)', value: 'medium' },
    { id: 'hard', label: 'Difícil (Avanzado / Optimización)', value: 'hard' }
  ];

  readonly complexityOptions: ComboboxOption<string>[] = [
    { id: '', label: 'Sin especificar', value: '' },
    { id: 'O(1)', label: 'O(1) - Constante', value: 'O(1)' },
    { id: 'O(N)', label: 'O(N) - Lineal', value: 'O(N)' },
    { id: 'O(N log N)', label: 'O(N log N) - Log-lineal', value: 'O(N log N)' },
    { id: 'O(N²)', label: 'O(N²) - Cuadrática', value: 'O(N²)' }
  ];

  ngOnInit(): void {
    this.loadRunnerTemplates();
  }

  ngOnDestroy(): void {
    if (this.tagErrorTimer) {
      clearTimeout(this.tagErrorTimer);
    }
  }

  private loadRunnerTemplates(): void {
    this.http.get<any>('/api/v1/templates?limit=50').subscribe({
      next: (res) => {
        let items: any[] = [];
        if (Array.isArray(res)) {
          items = res;
        } else if (Array.isArray(res?.data)) {
          items = res.data;
        } else if (Array.isArray(res?.data?.data)) {
          items = res.data.data;
        }

        const runners = items.filter(t => 
          (t.target_environment === 'JUEZ_EFIMERO' || t.environment_type === 'JUEZ_EFIMERO') &&
          (t.status === 'approved' || t.status === 'APROBADA' || !t.status)
        );
        this.runnerTemplates.set(runners);

        if (runners.length > 0) {
          const options: ComboboxOption<string>[] = runners.map(r => {
            const langCode = this.deriveLanguageCode(r);
            return {
              id: langCode,
              label: r.name,
              value: langCode
            };
          });

          // Eliminar duplicados si hubiera
          const uniqueOptions = options.filter((opt, index, self) => 
            index === self.findIndex(o => o.value === opt.value)
          );
          this.languageOptions.set(uniqueOptions);

          // Si el store ya tiene lenguaje, sincronizar con su plantilla
          if (this.store.metadata().language) {
            this.syncLanguageWithTemplate(this.store.metadata().language, runners);
          }
        }
      },
      error: () => {
        // Mantiene defaultLanguages
      }
    });
  }

  private deriveLanguageCode(template: any): string {
    const name = (template.name || '').toLowerCase();
    const image = (template.docker_image || '').toLowerCase();
    if (name.includes('python') || image.includes('python')) return 'python';
    if (name.includes('typescript') || image.includes('typescript')) return 'typescript';
    if (name.includes('javascript') || name.includes('node') || image.includes('node')) return 'javascript';
    if (name.includes('c++') || name.includes('cpp')) return 'cpp';
    if (name.includes('c/') || name === 'c' || image.includes('gcc')) return 'c';
    if (name.includes('java') || image.includes('temurin') || image.includes('java')) return 'java';
    if (name.includes('c#') || name.includes('csharp') || image.includes('mono')) return 'csharp';
    if (name.includes('go') || image.includes('golang')) return 'go';
    if (name.includes('rust') || image.includes('rust')) return 'rust';
    return template.name.toLowerCase().replace(/\s+/g, '-');
  }

  private syncLanguageWithTemplate(lang: string, runners?: any[]) {
    const list = runners || this.runnerTemplates();
    const match = list.find(r => this.deriveLanguageCode(r) === lang);
    if (match) {
      this.store.updateMetadata({
        template_id: match.id,
        template: match,
        memory_limit_mb: match.base_ram_mb || 128
      });
    }
  }

  onTitleChange(title: string) {
    this.store.updateMetadata({ title });
  }

  onDifficultySelect(option: ComboboxOption<string>) {
    const val = (option.value || null) as 'easy' | 'medium' | 'hard' | null;
    this.store.updateMetadata({ difficulty: val });
  }

  onComplexitySelect(option: ComboboxOption<string>) {
    this.store.updateMetadata({ expected_complexity: option.value || undefined });
  }

  onLanguageSelect(option: ComboboxOption<string>) {
    this.languageTouched.set(true);
    const language = option.value || '';
    this.store.updateMetadata({ language, allowed_languages: language ? [language] : [] });
    if (language) {
      this.syncLanguageWithTemplate(language);
    }
  }

  onTimeLimitInput(event: Event) {
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

  onMemoryLimitChange(memory_limit_mb: number) {
    this.store.updateMetadata({ memory_limit_mb: Number(memory_limit_mb) || 128 });
  }

  setHasDueDate(enabled: boolean) {
    this.hasDueDate.set(enabled);
    if (!enabled) {
      this.store.updateMetadata({ due_date: undefined });
    }
  }

  onDueDateChange(due_date: string) {
    this.store.updateMetadata({ due_date: due_date || undefined });
  }

  clearTagError() {
    if (this.tagErrorTimer) {
      clearTimeout(this.tagErrorTimer);
      this.tagErrorTimer = null;
    }
    this.tagError.set(null);
  }

  addTag(inputEl: HTMLInputElement) {
    const rawTag = inputEl.value;
    const tag = rawTag.trim().toLowerCase();
    if (!tag) {
      if (this.tagErrorTimer) clearTimeout(this.tagErrorTimer);
      this.tagError.set('Ingresa un texto para agregar la etiqueta');
      this.tagErrorTimer = setTimeout(() => {
        this.tagError.set(null);
        this.tagErrorTimer = null;
      }, 3500);
      return;
    }
    this.clearTagError();
    const currentTags = this.store.metadata().tags;
    if (!currentTags.includes(tag)) {
      this.store.updateMetadata({ tags: [...currentTags, tag] });
    }
    inputEl.value = '';
  }

  removeTag(tagToRemove: string) {
    const currentTags = this.store.metadata().tags;
    this.store.updateMetadata({
      tags: currentTags.filter(t => t !== tagToRemove)
    });
  }
}
