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
        <div class="form-group full-width" id="exercise-complexity">
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

        <!-- Lenguajes de la práctica (Unificado) -->
        <div class="form-group full-width" id="allowed-languages-group">
          <label class="form-label">
            Lenguajes de programación <span class="required">*</span>
          </label>
          <div class="allowed-languages-chips" role="group" aria-label="Lenguajes de programación">
            @for (lang of catalogLanguages(); track lang.id) {
              <button
                type="button"
                class="lang-chip"
                [class.active]="isLanguageAllowed(lang.id)"
                [class.primary]="isPrimaryLanguage(lang.id)"
                (click)="onToggleLanguage(lang.id)"
                [attr.aria-pressed]="isLanguageAllowed(lang.id)"
                [attr.title]="isPrimaryLanguage(lang.id) ? 'Lenguaje de la solución docente' : (isLanguageAllowed(lang.id) ? 'Desactivar lenguaje' : 'Habilitar para la práctica')"
              >
                <span class="lang-chip-label">{{ lang.label }}</span>
                @if (isPrimaryLanguage(lang.id)) {
                  <span class="primary-pill">Solución docente</span>
                }
              </button>
            }
          </div>

          @if (showLanguageError()) {
            <field-message variant="error" [message]="'Debes seleccionar al menos un lenguaje de programación'"></field-message>
          } @else if (hasMultipleLanguages()) {
            <div class="reference-language-picker">
              <span class="picker-label">Escribir solución de referencia en:</span>
              <div class="picker-buttons">
                @for (langId of activeAllowedLanguages(); track langId) {
                  <button
                    type="button"
                    class="ref-lang-btn"
                    [class.selected]="isPrimaryLanguage(langId)"
                    (click)="onSelectPrimaryLanguage(langId)"
                  >
                    {{ getLanguageLabel(langId) }}
                  </button>
                }
              </div>
            </div>
            <span class="field-hint">
              Los alumnos podrán resolver la práctica en cualquiera de los lenguajes seleccionados.
            </span>
          } @else {
            <span class="field-hint">
              Haz clic en los lenguajes que desees permitir para esta práctica.
            </span>
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

  readonly defaultLanguages: { id: string }[] = [
    { id: 'python' },
    { id: 'cpp' },
    { id: 'c' },
    { id: 'java' },
    { id: 'go' },
    { id: 'csharp' },
    { id: 'javascript' },
    { id: 'typescript' },
    { id: 'rust' }
  ];

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

  readonly catalogLanguages = computed<{ id: string; label: string }[]>(() => {
    const runners = this.store.runnerTemplates();
    const map = new Map<string, string>();

    const findImageForCode = (code: string): string => {
      const dedicated = runners.find(r => {
        const rRunners = r.language_runners ?? [];
        if (rRunners.length === 1) return (rRunners[0].language || '').trim().toLowerCase() === code;
        return this.store.deriveLanguageCode(r) === code;
      });
      return dedicated?.docker_image || '';
    };

    for (const tpl of runners) {
      if (tpl.language_runners && tpl.language_runners.length > 0) {
        for (const lr of tpl.language_runners) {
          const code = (lr.language || '').trim().toLowerCase();
          if (code && !map.has(code)) {
            const img = lr.docker_image || findImageForCode(code) || tpl.docker_image;
            map.set(code, this.formatLanguageWithVersion(code, img));
          }
        }
      } else {
        const code = this.store.deriveLanguageCode(tpl);
        if (code && !map.has(code)) {
          const img = findImageForCode(code) || tpl.docker_image;
          map.set(code, this.formatLanguageWithVersion(code, img));
        }
      }
    }

    if (map.size === 0) {
      for (const dl of this.defaultLanguages) {
        map.set(dl.id, this.formatLanguageWithVersion(dl.id));
      }
    }

    return Array.from(map.entries()).map(([id, label]) => ({ id, label }));
  });

  private formatLanguageWithVersion(code: string, dockerImage?: string): string {
    const pretty: Record<string, string> = {
      python: 'Python',
      cpp: 'C++',
      c: 'C',
      java: 'Java',
      go: 'Go',
      csharp: 'C#',
      javascript: 'JavaScript',
      typescript: 'TypeScript',
      rust: 'Rust',
      php: 'PHP',
      ruby: 'Ruby',
      kotlin: 'Kotlin',
      julia: 'Julia',
      lua: 'Lua'
    };
    const baseName = pretty[code.toLowerCase()] || (code ? code.charAt(0).toUpperCase() + code.slice(1) : code);
    if (!dockerImage) return baseName;

    const noDigest = dockerImage.split('@')[0];
    const tag = noDigest.includes(':') ? noDigest.split(':')[1] : '';
    const vMatch = tag.match(/^(\d+(?:\.\d+)?)/);

    if (vMatch) {
      const version = vMatch[1];
      if (noDigest.includes('node')) return `JavaScript (Node ${version})`;
      if (noDigest.includes('temurin')) return `Java ${version}`;
      return `${baseName} ${version}`;
    }

    if (noDigest.includes('mono')) return 'C# (Mono)';
    return baseName;
  }

  readonly activeAllowedLanguages = computed<string[]>(() => {
    const primary = (this.store.metadata().language || '').trim().toLowerCase();
    const raw = (this.store.metadata().allowed_languages ?? [])
      .map(l => (l || '').trim().toLowerCase())
      .filter(Boolean);
    const set = new Set(raw);
    if (primary) set.add(primary);
    return Array.from(set);
  });

  readonly hasMultipleLanguages = computed<boolean>(() => {
    return this.activeAllowedLanguages().length > 1;
  });

  getLanguageLabel(langId: string): string {
    const target = (langId || '').trim().toLowerCase();
    const match = this.catalogLanguages().find(l => l.id === target);
    return match ? match.label : this.formatLanguageWithVersion(target);
  }

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

  isPrimaryLanguage(langId: string): boolean {
    const primary = (this.store.metadata().language || '').trim().toLowerCase();
    return primary === (langId || '').trim().toLowerCase();
  }

  isLanguageAllowed(langId: string): boolean {
    const target = (langId || '').trim().toLowerCase();
    return this.activeAllowedLanguages().includes(target);
  }

  onToggleLanguage(langId: string): void {
    const target = (langId || '').trim().toLowerCase();
    const currentPrimary = (this.store.metadata().language || '').trim().toLowerCase();
    const currentAllowed = this.activeAllowedLanguages();

    this.languageTouched.set(true);

    if (this.isLanguageAllowed(target)) {
      if (currentAllowed.length <= 1) {
        return;
      }
      const remaining = currentAllowed.filter(l => l !== target);
      if (this.isPrimaryLanguage(target)) {
        const newPrimary = remaining[0];
        this.store.setLanguageAndSyncTemplate(newPrimary);
      }
      this.store.setAllowedLanguages(remaining);
    } else {
      if (!currentPrimary) {
        this.store.setLanguageAndSyncTemplate(target);
      } else {
        this.store.toggleAllowedLanguage(target);
      }
    }
  }

  onSelectPrimaryLanguage(langId: string): void {
    const target = (langId || '').trim().toLowerCase();
    if (target && target !== (this.store.metadata().language || '').trim().toLowerCase()) {
      this.store.setLanguageAndSyncTemplate(target);
    }
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
