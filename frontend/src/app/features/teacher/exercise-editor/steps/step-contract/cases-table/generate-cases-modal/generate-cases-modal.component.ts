import { Component, EventEmitter, Input, Output, OnChanges, SimpleChanges, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import {
  LucideLayers,
  LucideX,
  LucideRefreshCw,
  LucideEye,
  LucideSliders,
  LucideTerminal,
  LucidePlay,
  LucideRotateCcw,
  LucideCheck,
  LucideAlertCircle
} from '@lucide/angular';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { CodeEditorComponent } from '@shared/components/code-editor/code-editor.component';
import { TeacherCourseService } from '../../../../../services/teacher-course.service';
import { FORMAT_PRESETS, FormatPreset, ExerciseContract } from '../../format-builder/format-builder.models';
import { PresetButtonComponent } from '../../format-builder/presets/preset-button.component';
import { ScriptGenerationResponse, ScriptCaseValidationItem } from '../../../../../models/teacher.models';

const DEFAULT_PYTHON_SCRIPT = `import json
import random

cases = []
for i in range(10):
    n = random.randint(1, 100)
    arr = [random.randint(-1000, 1000) for _ in range(n)]
    input_str = f"{n}\\n{' '.join(map(str, arr))}"
    expected_output = str(sum(arr))
    
    # Etiqueta pedagógica contextual
    tag = "Caso base: límites mínimos" if i == 0 else ("Caso borde: extremos" if i == 1 else f"Caso aleatorio #{i + 1}")
    cases.append({
        "input": input_str,
        "expected_output": expected_output,
        "description": tag
    })

print(json.dumps(cases))`;

@Component({
  selector: 'generate-cases-modal',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideLayers,
    LucideX,
    LucideRefreshCw,
    LucideEye,
    LucideSliders,
    LucideTerminal,
    LucidePlay,
    LucideRotateCcw,
    LucideCheck,
    LucideAlertCircle,
    MachineDataDirective,
    CodeEditorComponent,
    PresetButtonComponent
  ],
  templateUrl: './generate-cases-modal.component.html',
  styleUrl: './generate-cases-modal.component.scss'
})
export class GenerateCasesModalComponent implements OnChanges {
  private courseService = inject(TeacherCourseService);

  @Input() isOpen = false;
  @Input() contract: any = null;
  @Input() exerciseId: string | null = null;

  @Output() casesGenerated = new EventEmitter<Array<{ input: string; output?: string; expected_output?: string; description?: string }>>();
  @Output() cancel = new EventEmitter<void>();

  // Control de Pestañas
  readonly activeTab = signal<'presets' | 'script'>('presets');

  // Pestaña 1: Presets Estructurados
  readonly presets = FORMAT_PRESETS;
  readonly activePresetId = signal<string>('single_int');

  // Parámetros configurables por preset
  readonly intMin = signal<number>(1);
  readonly intMax = signal<number>(1000000);

  readonly arrNMin = signal<number>(1);
  readonly arrNMax = signal<number>(100000);
  readonly arrElemMin = signal<number>(-1000000000);
  readonly arrElemMax = signal<number>(1000000000);

  readonly matNMax = signal<number>(1000);
  readonly matMMax = signal<number>(1000);
  readonly matElemMin = signal<number>(0);
  readonly matElemMax = signal<number>(1000000);

  readonly graphNMax = signal<number>(100000);
  readonly graphMMax = signal<number>(200000);

  readonly stringMaxLen = signal<number>(1000);

  readonly count = signal<number>(10);
  readonly seed = signal<number | null>(null);

  readonly previewCases = signal<Array<{ input: string }>>([]);
  readonly isLoadingPreview = signal<boolean>(false);
  readonly isGenerating = signal<boolean>(false);
  readonly error = signal<string | null>(null);

  // Pestaña 2: Script Python
  readonly scriptCode = signal<string>(DEFAULT_PYTHON_SCRIPT);
  readonly isExecutingScript = signal<boolean>(false);
  readonly scriptResult = signal<ScriptGenerationResponse | null>(null);
  readonly scriptError = signal<string | null>(null);

  setTab(tab: 'presets' | 'script'): void {
    this.activeTab.set(tab);
    if (tab === 'presets' && this.previewCases().length === 0) {
      this.fetchPreview();
    }
  }

  selectPreset(preset: FormatPreset): void {
    this.activePresetId.set(preset.id);
    this.fetchPreview();
  }

  ngOnChanges(changes: SimpleChanges): void {
    if (changes['isOpen'] && this.isOpen) {
      if (this.activeTab() === 'presets') {
        this.fetchPreview();
      }
    }
  }

  onCountChange(val: number): void {
    const num = isNaN(val) ? 10 : Math.min(100, Math.max(1, Number(val)));
    this.count.set(num);
  }

  onSeedChange(val: string): void {
    if (val === '' || val === null || val === undefined) {
      this.seed.set(null);
    } else {
      const num = Number(val);
      this.seed.set(isNaN(num) ? null : num);
    }
    this.fetchPreview();
  }

  onParamChange(): void {
    this.fetchPreview();
  }

  buildContract(): ExerciseContract {
    const presetId = this.activePresetId();
    switch (presetId) {
      case 'single_int':
        return {
          version: 1,
          input: {
            lines: [{ id: 'n', type: 'int', min: this.intMin(), max: this.intMax() }]
          }
        };
      case 'n_and_ints':
        return {
          version: 1,
          input: {
            lines: [
              { id: 'n', type: 'int', min: this.arrNMin(), max: this.arrNMax() },
              {
                id: 'a',
                type: 'ints',
                count: '=n',
                item: { type: 'int', min: this.arrElemMin(), max: this.arrElemMax() }
              }
            ]
          }
        };
      case 'matrix_nm':
        return {
          version: 1,
          input: {
            lines: [
              { id: 'n', type: 'int', min: 1, max: this.matNMax() },
              { id: 'm', type: 'int', min: 1, max: this.matMMax() },
              {
                id: 'mat',
                type: 'matrix',
                rows: '=n',
                cols: '=m',
                item: { type: 'int', min: this.matElemMin(), max: this.matElemMax() }
              }
            ]
          }
        };
      case 'graph_edges':
        return {
          version: 1,
          input: {
            lines: [
              { id: 'n', type: 'int', min: 1, max: this.graphNMax() },
              { id: 'm', type: 'int', min: 1, max: this.graphMMax() },
              { id: 'edges', type: 'edges', count: '=m' }
            ]
          }
        };
      case 'single_string':
        return {
          version: 1,
          input: {
            lines: [{ id: 's', type: 'string', max_len: this.stringMaxLen() }]
          }
        };
      default:
        return this.presets[0].contract;
    }
  }

  fetchPreview(): void {
    const contract = this.buildContract();
    this.isLoadingPreview.set(true);
    this.error.set(null);

    const seedVal = this.seed() ?? undefined;
    this.courseService.generateCases(contract, 3, seedVal).subscribe({
      next: (res) => {
        this.previewCases.set(res.cases || []);
        this.isLoadingPreview.set(false);
      },
      error: (err) => {
        this.error.set(err?.error?.message || err?.message || 'Error al generar vista previa');
        this.isLoadingPreview.set(false);
      }
    });
  }

  onConfirm(): void {
    const contract = this.buildContract();
    this.isGenerating.set(true);
    this.error.set(null);

    const c = this.count();
    const seedVal = this.seed() ?? undefined;

    this.courseService.generateCases(contract, c, seedVal).subscribe({
      next: (res) => {
        this.isGenerating.set(false);
        const cases = res.cases || [];
        const total = cases.length;
        const mapped = cases.map((item, idx) => {
          let desc = `Caso aleatorio #${idx + 1}`;
          if (idx === 0) {
            desc = 'Caso base: límites mínimos';
          } else if (idx === 1) {
            desc = 'Caso borde: extremos';
          } else if (idx === total - 1 && total >= 4) {
            desc = 'Caso de estrés / escala';
          }
          return {
            input: item.input,
            output: item.output || undefined,
            expected_output: item.output || undefined,
            description: desc
          };
        });
        this.casesGenerated.emit(mapped);
        this.onClose();
      },
      error: (err) => {
        this.error.set(err?.error?.message || err?.message || 'Error al generar casos');
        this.isGenerating.set(false);
      }
    });
  }

  // Métodos Pestaña Script Python
  onScriptCodeChange(code: string): void {
    this.scriptCode.set(code);
  }

  resetScriptTemplate(): void {
    this.scriptCode.set(DEFAULT_PYTHON_SCRIPT);
    this.scriptResult.set(null);
    this.scriptError.set(null);
  }

  executeScript(): void {
    const code = this.scriptCode();
    if (!code.trim() || this.isExecutingScript()) return;

    this.isExecutingScript.set(true);
    this.scriptError.set(null);
    this.scriptResult.set(null);

    const targetId = this.exerciseId || 'draft';

    this.courseService.generateCasesFromScript(targetId, code, true).subscribe({
      next: (res) => {
        this.scriptResult.set(res);
        this.isExecutingScript.set(false);
      },
      error: (err) => {
        this.isExecutingScript.set(false);
        const msg = err?.error?.message || err?.error?.error || 'Error al ejecutar el script en la sandbox';
        this.scriptError.set(msg);
      }
    });
  }

  confirmScriptCases(): void {
    const res = this.scriptResult();
    if (!res || !res.cases) return;

    const total = res.cases.length;
    const validCases = res.cases
      .filter((c: ScriptCaseValidationItem) => c.valid)
      .map((c: any, idx: number) => {
        let fallbackDesc = `Caso aleatorio #${idx + 1}`;
        if (idx === 0) fallbackDesc = 'Caso base: inicial';
        else if (idx === 1) fallbackDesc = 'Caso borde: extremos';
        else if (idx === total - 1 && total >= 4) fallbackDesc = 'Caso de estrés / escala';

        return {
          input: c.input,
          output: c.expected_output,
          expected_output: c.expected_output,
          description: c.description || fallbackDesc
        };
      });

    if (validCases.length === 0) return;

    this.casesGenerated.emit(validCases);
    this.onClose();
  }

  onClose(): void {
    this.cancel.emit();
  }

  onBackdropClick(event: MouseEvent): void {
    if ((event.target as HTMLElement).classList.contains('modal-backdrop')) {
      this.onClose();
    }
  }

  onKeyDown(event: KeyboardEvent): void {
    if (event.key === 'Escape') {
      this.onClose();
    }
  }
}
