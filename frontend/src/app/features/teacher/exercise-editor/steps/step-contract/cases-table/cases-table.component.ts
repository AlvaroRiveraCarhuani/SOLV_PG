import { Component, DestroyRef, OnInit, effect, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { Subject, of } from 'rxjs';
import { debounceTime, switchMap, map, catchError } from 'rxjs/operators';
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
  LucideSparkles,
  LucidePlay,
  LucideX
} from '@lucide/angular';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
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
    LucideSparkles,
    LucidePlay,
    LucideX,
    MachineDataDirective,
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
  readonly lastGeneratedCount = signal<number>(0);

  readonly modalState = signal<{
    isOpen: boolean;
    title: string;
    subtitle: string;
    field: 'input' | 'expected_output';
    caseIndex: number;
    value: string;
  }>({
    isOpen: false,
    title: '',
    subtitle: '',
    field: 'input',
    caseIndex: 0,
    value: ''
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
  }

  ngOnInit(): void {
    this.validateSubject.pipe(
      debounceTime(300),
      switchMap(({ index, input }) => {
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
    this.store.updateCase(index, { input: value });
    this.validateCase(index, value);
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
    this.store.updateCase(index, { expected_output: value });
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

  getWeightPercentage(weight?: number): string {
    const total = this.store.totalWeight();
    const w = weight ?? 1.0;
    if (total <= 0) return '0.0%';
    const pct = (w / total) * 100;
    return `${pct.toFixed(1)}%`;
  }

  openModal(index: number, field: 'input' | 'expected_output'): void {
    const cases = this.store.cases();
    if (index < 0 || index >= cases.length) return;
    const tc = cases[index];
    const val = field === 'input' ? tc.input : tc.expected_output;
    const title =
      field === 'input'
        ? 'Editar Entrada Estándar (stdin)'
        : 'Editar Salida Esperada (stdout)';
    const visLabels: Record<string, string> = {
      example: 'Ejemplo',
      public: 'Público',
      hidden: 'Oculto'
    };
    const vis = tc.visibility || (tc.is_hidden ? 'hidden' : tc.is_sample ? 'example' : 'public');
    const subtitle = `Caso #${index + 1} • Visibilidad: ${visLabels[vis] || vis}`;

    this.modalState.set({
      isOpen: true,
      title,
      subtitle,
      field,
      caseIndex: index,
      value: val || ''
    });
  }

  onModalSave(newValue: string): void {
    const { field, caseIndex } = this.modalState();
    if (field === 'input') {
      this.onInputChange(caseIndex, newValue);
    } else {
      this.onOutputChange(caseIndex, newValue);
    }
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
    this.store.normalizeWeights();
  }

  openGenerateModal(): void {
    this.isGenerateModalOpen.set(true);
  }

  closeGenerateModal(): void {
    this.isGenerateModalOpen.set(false);
  }

  onCasesGenerated(newCases: Array<{ input: string; output?: string | null }>): void {
    if (!newCases || newCases.length === 0) return;

    const currentLength = this.store.cases().length;
    const dtos: TestCaseDTO[] = newCases.map((c, i) => ({
      input: c.input,
      expected_output: c.output || '',
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
    const emptyOutputIndices: number[] = [];
    const inputsToCalculate: string[] = [];

    allCases.forEach((tc, idx) => {
      if (!tc.expected_output || tc.expected_output.trim() === '') {
        emptyOutputIndices.push(idx);
        inputsToCalculate.push(tc.input);
      }
    });

    if (inputsToCalculate.length === 0) {
      this.showAutoCalculatePrompt.set(false);
      return;
    }

    this.isAutoCalculating.set(true);
    this.courseService
      .calculateOutputs({
        language: lang,
        source_code: ref,
        inputs: inputsToCalculate
      })
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: res => {
          res.outputs?.forEach((out, localIdx) => {
            const targetIndex = emptyOutputIndices[localIdx];
            if (targetIndex !== undefined) {
              this.store.updateCase(targetIndex, {
                expected_output: out.expected_output
              });
            }
          });
          this.isAutoCalculating.set(false);
          this.showAutoCalculatePrompt.set(false);
        },
        error: () => {
          this.isAutoCalculating.set(false);
        }
      });
  }
}
