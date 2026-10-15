import { Component, DestroyRef, EventEmitter, Input, Output, computed, effect, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import {
  LucideEye,
  LucideX,
  LucideCode,
  LucideTerminal,
  LucideFileText,
  LucideClock,
  LucideCpu,
  LucideShieldAlert,
  LucideServer,
  LucideExternalLink,
  LucideDatabase,
  LucideCalendar,
  LucidePlay,
  LucideLoader2,
  LucideRotateCcw,
  LucideFileCode,
  LucideAlertTriangle
} from '@lucide/angular';
import { marked } from 'marked';
import katex from 'katex';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { CodeEditorComponent } from '@shared/components/code-editor/code-editor.component';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { VerdictBadgeComponent } from '@shared/components/verdict-badge/verdict-badge.component';
import { ErrorDiagnostic, parseErrorDiagnostic, detectWhitespaceDiscrepancy } from '@core/utils/error-diagnostic.util';
import { ExerciseEditorStore } from '../exercise-editor.store';
import { TestCaseDTO } from '../../models/teacher.models';
import { TeacherCourseService } from '../../services/teacher-course.service';

export interface TestCaseSimResult {
  caseNumber: number;
  description?: string;
  input: string;
  expectedOutput: string;
  actualOutput: string;
  verdict: 'AC' | 'WA' | 'TLE' | 'RE';
  executionTimeMs: number;
  errorDetails?: string;
  diagnostic?: ErrorDiagnostic;
  isFormatDiscrepancy?: boolean;
}

export interface SimulationRunResult {
  passedCount: number;
  totalCount: number;
  allPassed: boolean;
  cases: TestCaseSimResult[];
}

function renderMarkdownWithKatex(raw: string): string {
  if (!raw || !raw.trim()) {
    return '<p class="empty-statement">No hay descripción para este ejercicio.</p>';
  }

  // 1. Block math: $$ ... $$
  let processed = raw.replace(/\$\$([\s\S]+?)\$\$/g, (_, math) => {
    try {
      return `<div class="katex-block-wrapper">${katex.renderToString(math.trim(), { displayMode: true, throwOnError: false, output: 'mathml' })}</div>`;
    } catch {
      return `<pre class="katex-error">${math}</pre>`;
    }
  });

  // 2. Inline math: $ ... $
  processed = processed.replace(/\$([^\$\n]+?)\$/g, (_, math) => {
    try {
      return katex.renderToString(math.trim(), { displayMode: false, throwOnError: false, output: 'mathml' });
    } catch {
      return `<code>${math}</code>`;
    }
  });

  try {
    return marked.parse(processed, { async: false }) as string;
  } catch {
    return `<pre class="fallback-preview">${raw}</pre>`;
  }
}

@Component({
  selector: 'student-preview-modal',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideEye,
    LucideX,
    LucideCode,
    LucideTerminal,
    LucideFileText,
    LucideClock,
    LucideCpu,
    LucideShieldAlert,
    LucideServer,
    LucideExternalLink,
    LucideDatabase,
    LucideCalendar,
    LucidePlay,
    LucideLoader2,
    LucideRotateCcw,
    LucideFileCode,
    LucideAlertTriangle,
    MachineDataDirective,
    CodeEditorComponent,
    ComboboxComponent,
    VerdictBadgeComponent
  ],
  templateUrl: './student-preview-modal.component.html',
  styleUrl: './student-preview-modal.component.scss'
})
export class StudentPreviewModalComponent {
  readonly store = inject(ExerciseEditorStore);
  private courseService = inject(TeacherCourseService);
  private destroyRef = inject(DestroyRef);

  @Input() isOpen = false;
  @Output() close = new EventEmitter<void>();

  readonly selectedLanguage = signal<string>('python');
  readonly isSimulationMode = computed(() => this.store.studentPreviewMode() === 'simulation');

  readonly simulationCode = signal<string>('');
  readonly isExecuting = signal<boolean>(false);
  readonly executionResults = signal<SimulationRunResult | null>(null);
  readonly executionErrorMessage = signal<string | null>(null);
  readonly activeCaseTab = signal<number>(0);

  readonly allowedLanguages = computed(() => {
    const meta = this.store.metadata();
    if (meta.allowed_languages && meta.allowed_languages.length > 0) {
      return meta.allowed_languages;
    }
    return [meta.language || 'python'];
  });

  readonly languageOptions = computed<ComboboxOption[]>(() => {
    return this.allowedLanguages().map(lang => ({
      id: lang,
      label: lang.toUpperCase(),
      value: lang
    }));
  });

  readonly isIDEEnvironment = computed(() => {
    return this.store.metadata().environment_type === 'IDE_PERSISTENTE';
  });

  readonly currentBoilerplate = computed(() => {
    const lang = this.selectedLanguage();
    const map = this.store.boilerplates();
    if (map && map[lang]) {
      return map[lang];
    }
    if (lang === this.store.metadata().language && this.store.boilerplate()) {
      return this.store.boilerplate();
    }
    return this.getDefaultBoilerplate(lang);
  });

  readonly activeRestrictions = computed(() => {
    const rules = this.store.astRules();
    const list: { pattern: string; message?: string }[] = [];

    if (rules.block_native_sort) {
      list.push({ pattern: 'Ordenamiento nativo prohibido', message: 'Debes implementar tu propio algoritmo de ordenamiento.' });
    }
    if (rules.block_system_modules) {
      list.push({ pattern: 'Módulos del sistema restringidos', message: 'No se permite acceso a librerías de sistema operativo.' });
    }
    for (const imp of (rules.forbidden_imports || [])) {
      list.push({ pattern: `Módulo prohibido: ${imp}`, message: 'Uso no permitido en este ejercicio.' });
    }
    for (const fn of (rules.forbidden_functions || [])) {
      list.push({ pattern: `Función prohibida: ${fn}`, message: 'Uso no permitido en este ejercicio.' });
    }
    for (const custom of (rules.custom_rules || [])) {
      list.push({ pattern: custom.pattern, message: custom.message });
    }

    return list;
  });

  readonly exampleCases = computed(() => {
    return this.store.cases().filter(
      (c: TestCaseDTO) => c.visibility === 'example' || (!c.visibility && c.is_sample)
    );
  });

  readonly renderedDescription = computed(() => {
    let raw = this.store.statement().description || '';
    raw = raw.replace(/^##?\s*Descripción del Problema\s*\n+/i, '');
    return renderMarkdownWithKatex(raw);
  });

  readonly renderedInputText = computed(() => {
    return this.store.statement().input_text
      ? renderMarkdownWithKatex(this.store.statement().input_text)
      : null;
  });

  readonly renderedOutputText = computed(() => {
    return this.store.statement().output_text
      ? renderMarkdownWithKatex(this.store.statement().output_text)
      : null;
  });

  readonly renderedConstraints = computed(() => {
    return this.store.statement().constraints
      ? renderMarkdownWithKatex(this.store.statement().constraints)
      : null;
  });

  constructor() {
    // Sincronización reactiva del lenguaje
    effect(() => {
      const defaultLang = this.store.metadata().language || 'python';
      const allowed = this.allowedLanguages();
      if (allowed.includes(defaultLang)) {
        this.selectedLanguage.set(defaultLang);
      } else if (allowed.length > 0) {
        this.selectedLanguage.set(allowed[0]);
      }
    });

    // Inicializar código al abrir o cambiar lenguaje en modo simulación
    effect(() => {
      const bp = this.currentBoilerplate();
      const isSim = this.isSimulationMode();
      if (isSim && !this.simulationCode()) {
        this.simulationCode.set(bp);
      }
    });
  }

  onCodeChange(newCode: string): void {
    this.simulationCode.set(newCode);
  }

  loadBoilerplate(): void {
    this.simulationCode.set(this.currentBoilerplate());
    this.executionResults.set(null);
    this.executionErrorMessage.set(null);
  }

  loadReferenceSolution(): void {
    const ref = this.store.referenceSolution();
    if (ref && ref.trim()) {
      this.simulationCode.set(ref);
    } else {
      this.simulationCode.set(this.currentBoilerplate());
    }
    this.executionResults.set(null);
    this.executionErrorMessage.set(null);
  }

  runSimulation(): void {
    const code = this.simulationCode().trim() || this.currentBoilerplate().trim();
    if (!code) return;

    // Tomar casos de ejemplo y públicos (como los vería el estudiante al probar localmente)
    let cases = this.store.cases().filter(
      (c: TestCaseDTO) => c.visibility === 'example' || c.visibility === 'public' || (!c.visibility && c.is_sample)
    );

    // Fallback: si no hay casos públicos, tomar todos los casos existentes
    if (cases.length === 0) {
      cases = this.store.cases();
    }

    if (cases.length === 0) {
      this.executionErrorMessage.set('No hay casos de prueba configurados para ejecutar.');
      return;
    }

    this.isExecuting.set(true);
    this.executionErrorMessage.set(null);

    const inputs = cases.map(c => c.input || '');
    const timeLimit = this.store.metadata().time_limit_ms || 1000;
    const memoryLimit = this.store.metadata().memory_limit_mb || 128;

    this.courseService
      .calculateOutputs({
        language: this.selectedLanguage(),
        source_code: code,
        inputs,
        time_limit_ms: timeLimit,
        memory_limit_mb: memoryLimit
      })
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (res) => {
          let passed = 0;
          const caseResults: TestCaseSimResult[] = cases.map((tc, idx) => {
            const out = res.outputs?.[idx];
            const rawActual = out?.expected_output || '';
            const rawExpected = tc.expected_output || '';
            const actual = rawActual.trim();
            const expected = rawExpected.trim();
            let verdict: 'AC' | 'WA' | 'TLE' | 'RE' = 'WA';

            if (out?.status === 'timeout') {
              verdict = 'TLE';
            } else if (out?.status === 'error') {
              verdict = 'RE';
            } else if (actual === expected) {
              verdict = 'AC';
              passed++;
            } else {
              verdict = 'WA';
            }

            const errorDetails = out?.error_details;
            const diagnostic = errorDetails ? parseErrorDiagnostic(errorDetails, this.selectedLanguage()) : undefined;
            const isFormatDiscrepancy = (verdict === 'WA') && detectWhitespaceDiscrepancy(rawActual, rawExpected);

            return {
              caseNumber: idx + 1,
              description: tc.description,
              input: tc.input || '',
              expectedOutput: expected,
              actualOutput: actual,
              verdict,
              executionTimeMs: out?.execution_time_ms || 0,
              errorDetails,
              diagnostic,
              isFormatDiscrepancy
            };
          });

          this.executionResults.set({
            passedCount: passed,
            totalCount: cases.length,
            allPassed: passed === cases.length,
            cases: caseResults
          });
          this.activeCaseTab.set(0);
          this.isExecuting.set(false);
        },
        error: (err) => {
          this.isExecuting.set(false);
          const msg = err?.error?.message || err?.error?.error || 'Error al conectar con el motor de ejecución en Docker';
          this.executionErrorMessage.set(msg);
        }
      });
  }

  onLanguageChange(lang: string): void {
    this.selectedLanguage.set(lang);
  }

  onClose(): void {
    this.close.emit();
  }

  onBackdropClick(event: MouseEvent): void {
    if ((event.target as HTMLElement).classList.contains('student-preview-backdrop')) {
      this.onClose();
    }
  }

  onKeyDown(event: KeyboardEvent): void {
    if (event.key === 'Escape') {
      this.onClose();
    }
  }

  private getDefaultBoilerplate(lang: string): string {
    switch (lang) {
      case 'python':
        return '# Escribe tu solución aquí\nimport sys\n\ndef main():\n    pass\n\nif __name__ == "__main__":\n    main()\n';
      case 'cpp':
        return '#include <iostream>\nusing namespace std;\n\nint main() {\n    ios_base::sync_with_stdio(false);\n    cin.tie(NULL);\n    \n    // Escribe tu solución aquí\n    return 0;\n}\n';
      case 'java':
        return 'import java.util.Scanner;\n\npublic class Solution {\n    public static void main(String[] args) {\n        Scanner scanner = new Scanner(System.in);\n        // Escribe tu solución aquí\n    }\n}\n';
      case 'c':
        return '#include <stdio.h>\n\nint main() {\n    // Escribe tu solución aquí\n    return 0;\n}\n';
      case 'javascript':
        return 'const fs = require("fs");\n\nfunction main() {\n    const input = fs.readFileSync(0, "utf-8").trim();\n    // Escribe tu solución aquí\n}\n\nmain();\n';
      default:
        return `// Código inicial para ${lang}\n`;
    }
  }
}

