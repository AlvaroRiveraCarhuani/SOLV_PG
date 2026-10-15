import { Component, DestroyRef, OnInit, effect, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { Subject, of } from 'rxjs';
import { debounceTime, mergeMap, map, catchError } from 'rxjs/operators';
import {
  LucidePlus,
  LucideTrash2,
  LucideCopy,
  LucideArrowUp,
  LucideArrowDown,
  LucideCheck,
  LucideAlertCircle,
  LucideMaximize2,
  LucideScale,
  LucideLayers,
  LucideLoader2,
  LucidePlay,
  LucideX,
  LucideCode
} from '@lucide/angular';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { ExerciseEditorStore } from '../../../exercise-editor.store';
import { TeacherCourseService } from '../../../../services/teacher-course.service';
import { TestCaseDTO } from '../../../../models/teacher.models';
import { CaseEditModalComponent } from './case-edit-modal/case-edit-modal.component';
import { GenerateCasesModalComponent } from './generate-cases-modal/generate-cases-modal.component';

export interface ValidationItem {
  valid: boolean;
  error?: string;
  loading?: boolean;
}

@Component({
  selector: 'cases-table',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucidePlus,
    LucideTrash2,
    LucideCopy,
    LucideArrowUp,
    LucideArrowDown,
    LucideCheck,
    LucideAlertCircle,
    LucideMaximize2,
    LucideScale,
    LucideLayers,
    LucideLoader2,
    LucidePlay,
    LucideX,
    LucideCode,
    MachineDataDirective,
    ComboboxComponent,
    CaseEditModalComponent,
    GenerateCasesModalComponent
  ],
  templateUrl: './cases-table.component.html',
  styleUrl: './cases-table.component.scss'
})
export class CasesTableComponent implements OnInit {
  readonly store = inject(ExerciseEditorStore);
  private courseService = inject(TeacherCourseService);
  private destroyRef = inject(DestroyRef);

  readonly validationResults = signal<Record<number, ValidationItem>>({});
  readonly isGenerateModalOpen = signal<boolean>(false);
  readonly showAutoCalculatePrompt = signal<boolean>(false);
  readonly isAutoCalculating = signal<boolean>(false);
  readonly isNormalizingWeights = signal<boolean>(false);
  readonly weightsNormalizedSuccess = signal<boolean>(false);
  readonly calculationError = signal<string | null>(null);
  readonly errorCopied = signal<boolean>(false);
  readonly lastGeneratedCount = signal<number>(0);

  // Filtro de Pestañas (Todos / Ejemplos / Ocultos)
  readonly activeFilterTab = signal<'all' | 'example' | 'hidden'>('all');

  // Mapa de índices en cálculo individual
  readonly rowCalculating = signal<Record<number, boolean>>({});
  readonly rowCalculatedSuccess = signal<Record<number, boolean>>({});

  // Casos filtrados con su índice original
  readonly filteredCases = computed(() => {
    const tab = this.activeFilterTab();
    const all = this.store.cases();
    return all
      .map((tc, originalIndex) => ({ tc, originalIndex }))
      .filter(({ tc }) => {
        if (tab === 'all') return true;
        const vis = tc.visibility || (tc.is_hidden ? 'hidden' : (tc.is_sample ? 'example' : 'public'));
        if (tab === 'example') return vis === 'example' || vis === 'public';
        return vis === 'hidden';
      });
  });

  // Opciones de visibilidad para Combobox reutilizable
  readonly visibilityOptions: ComboboxOption<string>[] = [
    { id: 'example', label: 'Ejemplo', value: 'example' },
    { id: 'public', label: 'Público', value: 'public' },
    { id: 'hidden', label: 'Oculto', value: 'hidden' }
  ];

  onVisibilityOptionSelected(index: number, option: ComboboxOption): void {
    const val = option.value as 'example' | 'public' | 'hidden';
    this.onVisibilityChange(index, val);
  }

  readonly exampleCount = computed(() => {
    return this.store.cases().filter(tc => {
      const vis = tc.visibility || (tc.is_hidden ? 'hidden' : (tc.is_sample ? 'example' : 'public'));
      return vis === 'example' || vis === 'public';
    }).length;
  });

  readonly hiddenCount = computed(() => {
    return this.store.cases().filter(tc => {
      const vis = tc.visibility || (tc.is_hidden ? 'hidden' : (tc.is_sample ? 'example' : 'public'));
      return vis === 'hidden';
    }).length;
  });

  private validateSubject = new Subject<{ index: number; input: string }>();

  constructor() {
    // Revalidar cuando el contrato cambie
    effect(() => {
      const contract = this.store.contract();
      const cases = this.store.cases();
      if (!contract) {
        this.validationResults.set({});
        return;
      }

      // Disparar validación para los casos existentes
      cases.forEach((tc, idx) => {
        if (tc.input !== undefined) {
          this.validateCase(idx, tc.input);
        }
      });
    });

    // Auto-limpiar error al modificar la solución de referencia
    effect(() => {
      this.store.referenceSolution();
      this.calculationError.set(null);
    });
  }

  scrollToReferenceEditor(): void {
    const el = document.getElementById('reference-solution-section');
    if (el) {
      el.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }
  }

  copyErrorToClipboard(): void {
    const errorText = this.calculationError();
    if (!errorText) return;
    navigator.clipboard.writeText(errorText).then(() => {
      this.errorCopied.set(true);
      setTimeout(() => this.errorCopied.set(false), 2000);
    }).catch(() => {
      // Ignorar fallo de portapapeles si los permisos están denegados
    });
  }

  ngOnInit(): void {
    this.validateSubject.pipe(
      mergeMap(({ index, input }) => {
        const contract = this.store.contract();
        if (!contract) {
          return of({ index, result: { valid: true, error: undefined as string | undefined } });
        }
        return this.courseService.validateInputFormat(contract, input).pipe(
          map(res => ({
            index,
            result: { valid: res.valid, error: res.error }
          })),
          catchError(() =>
            of({
              index,
              result: { valid: false, error: 'Error al contactar el validador' }
            })
          )
        );
      }),
      takeUntilDestroyed(this.destroyRef)
    ).subscribe(({ index, result }) => {
      this.validationResults.update(prev => ({
        ...prev,
        [index]: { valid: result.valid, error: result.error, loading: false }
      }));
    });
  }

  onInputChange(index: number, value: string): void {
    const normalized = value ? value.replace(/\r\n/g, '\n').replace(/\r/g, '\n') : '';
    this.store.updateCase(index, { input: normalized });
    this.validateCase(index, normalized);
  }

  private validateCase(index: number, input: string): void {
    if (!this.store.contract()) return;

    this.validationResults.update(prev => ({
      ...prev,
      [index]: { ...(prev[index] || { valid: true }), loading: true }
    }));

    this.validateSubject.next({ index, input });
  }

  onOutputChange(index: number, value: string): void {
    const normalized = value ? value.replace(/\r\n/g, '\n').replace(/\r/g, '\n') : '';
    this.store.updateCase(index, { expected_output: normalized });
  }

  onVisibilityChange(index: number, visibility: 'example' | 'public' | 'hidden'): void {
    this.store.updateCase(index, {
      visibility,
      is_hidden: visibility === 'hidden',
      is_sample: visibility === 'example'
    });
  }

  onWeightChange(index: number, weightVal: number): void {
    const w = isNaN(weightVal) || weightVal < 0 ? 0 : Number(weightVal);
    this.store.updateCase(index, { weight: w });
  }

  onDescriptionChange(index: number, description: string): void {
    this.store.updateCase(index, { description });
  }

  hasReferenceSolution = computed(() => !!this.store.referenceSolution()?.trim());
  readonly isModalCalculating = signal<boolean>(false);

  readonly modalState = signal<{
    isOpen: boolean;
    caseIndex: number;
    input: string;
    expected_output: string;
    description: string;
  }>({
    isOpen: false,
    caseIndex: 0,
    input: '',
    expected_output: '',
    description: ''
  });

  getWeightPercentage(weight?: number): string {
    const total = this.store.totalWeight();
    const w = weight ?? 1.0;
    if (total <= 0) return '0.0%';
    const pct = (w / total) * 100;
    return `${pct.toFixed(1)}%`;
  }

  setFilterTab(tab: 'all' | 'example' | 'hidden'): void {
    this.activeFilterTab.set(tab);
  }

  calculateSingleRow(index: number): void {
    const ref = this.store.referenceSolution()?.trim();
    const lang = this.store.metadata().language || 'python';
    const tc = this.store.cases()[index];
    if (!ref || !tc || !tc.input.trim()) return;

    this.rowCalculating.update(m => ({ ...m, [index]: true }));
    this.courseService
      .calculateOutputs({
        language: lang,
        source_code: ref,
        inputs: [tc.input]
      })
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: res => {
          this.rowCalculating.update(m => ({ ...m, [index]: false }));
          if (res.outputs && res.outputs.length > 0) {
            const out = res.outputs[0].expected_output || '';
            this.store.updateCase(index, { expected_output: out });
            this.rowCalculatedSuccess.update(m => ({ ...m, [index]: true }));
            setTimeout(() => {
              this.rowCalculatedSuccess.update(m => ({ ...m, [index]: false }));
            }, 1800);
          }
        },
        error: () => {
          this.rowCalculating.update(m => ({ ...m, [index]: false }));
        }
      });
  }

  openSideBySideModal(index: number): void {
    const cases = this.store.cases();
    if (index < 0 || index >= cases.length) return;
    const tc = cases[index];

    this.modalState.set({
      isOpen: true,
      caseIndex: index,
      input: tc.input || '',
      expected_output: tc.expected_output || '',
      description: tc.description || ''
    });
  }

  onModalCalculateOutput(currentInput: string): void {
    const ref = this.store.referenceSolution()?.trim();
    const lang = this.store.metadata().language || 'python';
    if (!ref || !currentInput.trim()) return;

    this.isModalCalculating.set(true);
    this.courseService
      .calculateOutputs({
        language: lang,
        source_code: ref,
        inputs: [currentInput]
      })
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: res => {
          this.isModalCalculating.set(false);
          if (res.outputs && res.outputs.length > 0) {
            const out = res.outputs[0].expected_output || '';
            this.modalState.update(s => ({ ...s, input: currentInput, expected_output: out }));
          }
        },
        error: () => {
          this.isModalCalculating.set(false);
        }
      });
  }

  onModalSave(payload: { input: string; expected_output: string; description?: string }): void {
    const { caseIndex } = this.modalState();
    this.store.updateCase(caseIndex, {
      input: payload.input,
      expected_output: payload.expected_output,
      description: payload.description
    });
    this.closeModal();
  }

  closeModal(): void {
    this.modalState.update(s => ({ ...s, isOpen: false }));
  }

  addNewCase(): void {
    const currentLength = this.store.cases().length;
    const isFirst = currentLength === 0;
    const newCase: TestCaseDTO = {
      input: '',
      expected_output: '',
      visibility: isFirst ? 'example' : 'public',
      weight: 1.0,
      is_hidden: false,
      is_sample: isFirst,
      order_index: currentLength + 1
    };
    this.store.addCase(newCase);
  }

  duplicateCase(index: number): void {
    this.store.duplicateCase(index);
  }

  removeCase(index: number): void {
    this.store.removeCase(index);
    this.validationResults.update(prev => {
      const next: Record<number, ValidationItem> = {};
      Object.keys(prev).forEach(key => {
        const k = Number(key);
        if (k < index) {
          next[k] = prev[k];
        } else if (k > index) {
          next[k - 1] = prev[k];
        }
      });
      return next;
    });
  }

  moveUp(index: number): void {
    if (index > 0) {
      this.store.reorderCase(index, index - 1);
    }
  }

  moveDown(index: number): void {
    if (index < this.store.cases().length - 1) {
      this.store.reorderCase(index, index + 1);
    }
  }

  normalizeWeights(): void {
    if (this.isNormalizingWeights()) return;
    this.isNormalizingWeights.set(true);

    setTimeout(() => {
      this.store.normalizeWeights();
      this.isNormalizingWeights.set(false);
      this.weightsNormalizedSuccess.set(true);
      setTimeout(() => this.weightsNormalizedSuccess.set(false), 1500);
    }, 400);
  }

  openGenerateModal(): void {
    this.isGenerateModalOpen.set(true);
  }

  closeGenerateModal(): void {
    this.isGenerateModalOpen.set(false);
  }

  onCasesGenerated(newCases: Array<{ input: string; output?: string | null; description?: string }>): void {
    if (!newCases || newCases.length === 0) return;

    const currentLength = this.store.cases().length;
    const dtos: TestCaseDTO[] = newCases.map((c, i) => ({
      input: c.input,
      expected_output: c.output || '',
      description: c.description || `Caso de prueba #${currentLength + i + 1}`,
      visibility: 'hidden',
      weight: 1.0,
      is_hidden: true,
      is_sample: false,
      order_index: currentLength + i + 1
    }));

    this.store.addCases(dtos);
    this.lastGeneratedCount.set(newCases.length);

    // Si hay solución de referencia disponible y algunos casos no tienen salida, ofrecer auto-cálculo
    const hasRef = !!this.store.referenceSolution()?.trim();
    const hasEmptyOutputs = dtos.some(d => !d.expected_output);
    if (hasRef && hasEmptyOutputs) {
      this.showAutoCalculatePrompt.set(true);
    }
  }

  dismissAutoCalculatePrompt(): void {
    this.showAutoCalculatePrompt.set(false);
  }

  autoCalculateOutputs(): void {
    const ref = this.store.referenceSolution()?.trim();
    const lang = this.store.metadata().language || 'python';
    if (!ref) return;

    const allCases = this.store.cases();
    const targetIndices: number[] = [];
    const inputsToCalculate: string[] = [];

    // Recalcular para todos los casos que tengan entrada definida
    allCases.forEach((tc, idx) => {
      if (tc.input && tc.input.trim() !== '') {
        targetIndices.push(idx);
        inputsToCalculate.push(tc.input);
      }
    });

    if (inputsToCalculate.length === 0) {
      this.showAutoCalculatePrompt.set(false);
      return;
    }

    this.isAutoCalculating.set(true);
    this.calculationError.set(null);

    this.courseService
      .calculateOutputs({
        language: lang,
        source_code: ref,
        inputs: inputsToCalculate
      })
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: res => {
          let hasErrors = false;
          let errorMessage = '';

          res.outputs?.forEach((out, localIdx) => {
            const targetIndex = targetIndices[localIdx];
            if (targetIndex !== undefined) {
              if (out.expected_output && out.expected_output.trim() !== '') {
                this.store.updateCase(targetIndex, {
                  expected_output: out.expected_output.trim()
                });
              } else if (out.status === 'error' || out.status === 'timeout') {
                hasErrors = true;
                errorMessage = out.error_details || 'Error al ejecutar la solución de referencia en el contenedor';
              }
            }
          });

          this.isAutoCalculating.set(false);
          this.showAutoCalculatePrompt.set(false);

          if (hasErrors) {
            this.calculationError.set(errorMessage);
          } else {
            this.calculationError.set(null);
          }
        },
        error: (err) => {
          this.isAutoCalculating.set(false);
          const msg = err?.error?.message || err?.error?.error || 'Error al conectar con el motor de cálculo';
          this.calculationError.set(msg);
        }
      });
  }
}
