import { Component, DestroyRef, OnInit, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import {
  LucideArrowLeft,
  LucidePlay,
  LucideRotateCcw,
  LucideFileCode,
  LucideTerminal,
  LucideFileText,
  LucideClock,
  LucideCpu,
  LucideAlertTriangle,
  LucideCheckCircle2,
  LucideCopy,
  LucideCheck,
  LucideZoomIn,
  LucideZoomOut,
  LucideLoader2,
  LucideLayers,
  LucideServer
} from '@lucide/angular';
import { marked } from 'marked';
import katex from 'katex';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { CodeEditorComponent } from '@shared/components/code-editor/code-editor.component';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { VerdictBadgeComponent } from '@shared/components/verdict-badge/verdict-badge.component';
import { ErrorDiagnostic, parseErrorDiagnostic, detectWhitespaceDiscrepancy } from '@core/utils/error-diagnostic.util';
import { ExerciseEditorStore } from '../exercise-editor/exercise-editor.store';
import { TeacherCourseService } from '../services/teacher-course.service';
import { TestCaseDTO } from '../models/teacher.models';

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

  // Block math: $$ ... $$
  let processed = raw.replace(/\$\$([\s\S]+?)\$\$/g, (_, math) => {
    try {
      return katex.renderToString(math.trim(), { displayMode: true, throwOnError: false, output: 'mathml' });
    } catch {
      return `<code>${math}</code>`;
    }
  });

  // Inline math: $ ... $
  processed = processed.replace(/\$([^$\n]+?)\$/g, (_, math) => {
    try {
      return katex.renderToString(math.trim(), { displayMode: false, throwOnError: false, output: 'mathml' });
    } catch {
      return `<code>${math}</code>`;
    }
  });

  return marked.parse(processed) as string;
}

@Component({
  selector: 'exercise-preview',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    RouterLink,
    LucideArrowLeft,
    LucidePlay,
    LucideRotateCcw,
    LucideFileCode,
    LucideTerminal,
    LucideFileText,
    LucideClock,
    LucideCpu,
    LucideAlertTriangle,
    LucideCheckCircle2,
    LucideCopy,
    LucideCheck,
    LucideZoomIn,
    LucideZoomOut,
    LucideLoader2,
    LucideLayers,
    LucideServer,
    MachineDataDirective,
    CodeEditorComponent,
    ComboboxComponent,
    VerdictBadgeComponent
  ],
  providers: [ExerciseEditorStore],
  templateUrl: './exercise-preview.component.html',
  styleUrl: './exercise-preview.component.scss'
})
export class ExercisePreviewComponent implements OnInit {
  private route = inject(ActivatedRoute);
  readonly store = inject(ExerciseEditorStore);
  private courseService = inject(TeacherCourseService);
  private destroyRef = inject(DestroyRef);

  readonly courseId = signal<string>('');
  readonly exerciseId = signal<string>('');

  // Modo de visualización: consigna o simulación interactiva
  readonly viewMode = signal<'both' | 'statement' | 'simulation'>('both');

  // Control de tamaño de fuente para proyector (en rem)
  readonly editorFontSize = signal<number>(14);

  // Proporción de ancho de columnas en modo Dividido (porcentaje para consigna)
  readonly splitRatio = signal<number>(45);
  readonly isDraggingSplitter = signal<boolean>(false);

  // Estado del editor y simulación
  readonly selectedLanguage = signal<string>('python');
  readonly simulationCode = signal<string>('');
  readonly isExecuting = signal<boolean>(false);
  readonly executionResults = signal<SimulationRunResult | null>(null);
  readonly executionErrorMessage = signal<string | null>(null);
  readonly activeCaseTab = signal<number>(0);
  readonly copiedCaseIndex = signal<number | null>(null);

  readonly isIDEEnvironment = computed(() => {
    return this.store.metadata().environment_type === 'IDE_PERSISTENTE';
  });

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

  readonly renderedDescription = computed(() => {
    let raw = this.store.statement().description || '';
    // Evitar encabezado duplicado si el markdown ya empieza con 'Descripción del Problema'
    raw = raw.replace(/^##?\s*Descripción del Problema\s*\n+/i, '');
    return renderMarkdownWithKatex(raw);
  });

  readonly renderedConstraints = computed(() => {
    const raw = this.store.statement().constraints;
    return raw ? renderMarkdownWithKatex(raw) : null;
  });

  readonly exampleCases = computed<TestCaseDTO[]>(() => {
    return this.store.cases().filter(c => c.visibility === 'example');
  });

  readonly activeCase = computed<TestCaseSimResult | null>(() => {
    const res = this.executionResults();
    if (!res || res.cases.length === 0) return null;
    const idx = this.activeCaseTab();
    return res.cases[idx] || res.cases[0];
  });

  ngOnInit(): void {
    this.route.paramMap.pipe(takeUntilDestroyed(this.destroyRef)).subscribe(params => {
      const cId = params.get('courseId') || params.get('subjectId') || '';
      const eId = params.get('id') || params.get('exerciseId') || '';
      this.courseId.set(cId);
      this.exerciseId.set(eId);

      if (eId) {
        this.store.loadExercise(eId, cId);
        // Inicializar código con la solución o boilerplate
        setTimeout(() => {
          const lang = this.store.metadata().language || 'python';
          this.selectedLanguage.set(lang);
          const initialCode = this.store.boilerplate() || this.store.referenceSolution() || '';
          this.simulationCode.set(initialCode);
        }, 300);
      }
    });
  }

  onLanguageChange(lang: string): void {
    this.selectedLanguage.set(lang);
    if (!this.simulationCode().trim()) {
      this.simulationCode.set(this.store.boilerplate() || '');
    }
  }

  increaseFontSize(): void {
    if (this.editorFontSize() < 24) {
      this.editorFontSize.update(s => s + 2);
    }
  }

  decreaseFontSize(): void {
    if (this.editorFontSize() > 12) {
      this.editorFontSize.update(s => s - 2);
    }
  }

  setSplitRatio(ratio: number): void {
    this.splitRatio.set(Math.max(25, Math.min(75, Math.round(ratio))));
  }

  onSplitterMouseDown(event: MouseEvent): void {
    event.preventDefault();
    this.isDraggingSplitter.set(true);

    const onMouseMove = (moveEvent: MouseEvent) => {
      const container = document.querySelector('.preview-layout') as HTMLElement;
      if (!container) return;
      const rect = container.getBoundingClientRect();
      const currentX = moveEvent.clientX - rect.left;
      const percent = (currentX / rect.width) * 100;
      this.setSplitRatio(percent);
    };

    const onMouseUp = () => {
      this.isDraggingSplitter.set(false);
      window.removeEventListener('mousemove', onMouseMove);
      window.removeEventListener('mouseup', onMouseUp);
    };

    window.addEventListener('mousemove', onMouseMove);
    window.addEventListener('mouseup', onMouseUp);
  }

  loadBoilerplate(): void {
    this.simulationCode.set(this.store.boilerplate() || '');
  }

  loadReferenceSolution(): void {
    const ref = this.store.referenceSolution();
    if (ref) {
      this.simulationCode.set(ref);
    }
  }

  copyExampleInput(input: string, index: number): void {
    navigator.clipboard.writeText(input).then(() => {
      this.copiedCaseIndex.set(index);
      setTimeout(() => this.copiedCaseIndex.set(null), 2000);
    });
  }

  runSimulation(): void {
    const code = this.simulationCode().trim() || this.store.boilerplate().trim();
    if (!code || this.isExecuting()) return;

    const cases = this.store.cases();
    if (cases.length === 0) {
      this.executionErrorMessage.set('El ejercicio no tiene casos de prueba configurados para ejecutar la simulación.');
      return;
    }

    this.isExecuting.set(true);
    this.executionErrorMessage.set(null);
    this.executionResults.set(null);

    const lang = this.selectedLanguage();
    const inputs = cases.map(c => c.input || '');
    const timeLimit = this.store.metadata().time_limit_ms || 1000;
    const memoryLimit = this.store.metadata().memory_limit_mb || 128;

    this.courseService
      .calculateOutputs({
        language: lang,
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
            let verdict: 'AC' | 'WA' | 'TLE' | 'RE';

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
            const diagnostic = errorDetails ? parseErrorDiagnostic(errorDetails, lang) : undefined;
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

          const firstFailedIndex = caseResults.findIndex(c => c.verdict !== 'AC');
          this.activeCaseTab.set(firstFailedIndex >= 0 ? firstFailedIndex : 0);
          this.isExecuting.set(false);
        },
        error: (err) => {
          this.isExecuting.set(false);
          const msg = err?.error?.message || err?.error?.error || 'Error al conectar con el motor de ejecución en Docker';
          this.executionErrorMessage.set(msg);
        }
      });
  }
}
