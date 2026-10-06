import { Injectable, computed, inject, signal } from '@angular/core';
import { Router } from '@angular/router';
import { Observable } from 'rxjs';
import { TeacherCourseService } from '../services/teacher-course.service';
import { ASTRulesDTO, ASTCustomRuleDTO, ChecklistReportDTO, CreateExerciseRequestDTO, TestCaseDTO, UpdateExerciseRequestDTO } from '../models/teacher.models';

export interface ExerciseMetadata {
  title: string;
  difficulty: 'easy' | 'medium' | 'hard' | null;
  tags: string[];
  modality: 'judge' | 'workspace' | 'database';
  purpose: 'class' | 'exam';
  per_student_seed: boolean;
  language: string;
  allowed_languages: string[];
  time_limit_ms: number;
  memory_limit_mb: number;
  due_date?: string;
}

export interface ExerciseStatement {
  description: string;
  input_text: string;
  output_text: string;
  constraints: string;
}

@Injectable({
  providedIn: 'root'
})
export class ExerciseEditorStore {
  private courseService = inject(TeacherCourseService);
  private router = inject(Router);

  // Context
  readonly courseId = signal<string>('');
  readonly exerciseId = signal<string | null>(null);
  readonly status = signal<'draft' | 'published'>('draft');
  readonly isSaving = signal<boolean>(false);
  readonly isLoading = signal<boolean>(false);
  readonly lastSavedAt = signal<Date | null>(null);
  readonly currentStep = signal<1 | 2 | 3>(1);

  // Paso 1: Identidad y Pedagogía
  readonly metadata = signal<ExerciseMetadata>({
    title: '',
    difficulty: null,
    tags: [],
    modality: 'judge',
    purpose: 'class',
    per_student_seed: false,
    language: 'python',
    allowed_languages: ['python'],
    time_limit_ms: 1000,
    memory_limit_mb: 128,
    due_date: ''
  });

  readonly statement = signal<ExerciseStatement>({
    description: '',
    input_text: '',
    output_text: '',
    constraints: ''
  });

  // Paso 2: Contrato, Casos y Referencia
  readonly contract = signal<any | null>(null);
  readonly cases = signal<TestCaseDTO[]>([]);
  readonly referenceSolution = signal<string>('');

  // Paso 3: Reglas AST, Boilerplates y Checklist
  readonly astRules = signal<ASTRulesDTO>({
    block_native_sort: false,
    block_system_modules: false,
    forbidden_imports: [],
    forbidden_functions: [],
    custom_rules: []
  });
  readonly boilerplate = signal<string>('');
  readonly boilerplates = signal<Record<string, string>>({});
  readonly checklistReport = signal<ChecklistReportDTO | null>(null);
  readonly isChecklistLoading = signal<boolean>(false);
  readonly acceptedWarnings = signal<boolean>(false);

  // Snapshot inicial para cálculo de dirty
  private initialSnapshot = signal<string>('');

  readonly dirty = computed(() => {
    const current = this.serializeState();
    return this.initialSnapshot() !== '' && current !== this.initialSnapshot();
  });

  // Métricas de casos
  readonly totalCasesCount = computed(() => this.cases().length);
  readonly exampleCasesCount = computed(() =>
    this.cases().filter(c => c.visibility === 'example' || (!c.visibility && c.is_sample)).length
  );
  readonly publicCasesCount = computed(() =>
    this.cases().filter(c => c.visibility === 'public' || (!c.visibility && !c.is_hidden && !c.is_sample)).length
  );
  readonly hiddenCasesCount = computed(() =>
    this.cases().filter(c => c.visibility === 'hidden' || (!c.visibility && c.is_hidden)).length
  );

  readonly totalWeight = computed(() =>
    this.cases().reduce((sum, c) => sum + (c.weight ?? 1.0), 0)
  );
  readonly visibleWeight = computed(() =>
    this.cases()
      .filter(c => c.visibility !== 'hidden' && (c.visibility || !c.is_hidden))
      .reduce((sum, c) => sum + (c.weight ?? 1.0), 0)
  );
  readonly hiddenWeight = computed(() =>
    this.cases()
      .filter(c => c.visibility === 'hidden' || (!c.visibility && c.is_hidden))
      .reduce((sum, c) => sum + (c.weight ?? 1.0), 0)
  );

  // Validaciones y advertencias por paso
  readonly step1Validation = computed(() => {
    const m = this.metadata();
    const s = this.statement();
    const errors: string[] = [];
    const warnings: string[] = [];

    if (!m.title.trim()) {
      errors.push('El título del ejercicio es obligatorio');
    }
    if (!s.description.trim()) {
      errors.push('La descripción del enunciado es obligatoria');
    }
    if (m.per_student_seed && m.purpose !== 'exam') {
      errors.push('La semilla por estudiante solo está permitida en modalidad Examen');
    }
    if (m.tags.length === 0) {
      warnings.push('Se recomienda agregar al menos una etiqueta temática (tag)');
    }
    if (!m.difficulty) {
      warnings.push('No has definido un nivel de dificultad');
    }

    return {
      isValid: errors.length === 0,
      errors,
      warnings,
      status: errors.length > 0 ? 'invalid' : warnings.length > 0 ? 'warning' : 'valid'
    };
  });

  readonly step2Validation = computed(() => {
    const cs = this.cases();
    const ref = this.referenceSolution();
    const errors: string[] = [];
    const warnings: string[] = [];

    if (cs.length === 0) {
      errors.push('Debes incluir al menos un caso de prueba');
    }
    if (!ref.trim()) {
      warnings.push('No has incluido la solución de referencia del docente');
    }
    if (this.exampleCasesCount() === 0) {
      warnings.push('No hay casos marcados como "Ejemplo" para el enunciado');
    }
    if (this.hiddenCasesCount() === 0) {
      warnings.push('No hay casos ocultos; un print estático podría aprobar');
    }

    return {
      isValid: errors.length === 0,
      errors,
      warnings,
      status: errors.length > 0 ? 'invalid' : warnings.length > 0 ? 'warning' : 'valid'
    };
  });

  readonly step3Validation = computed(() => {
    const cl = this.checklistReport();
    const errors: string[] = cl ? [...cl.blockers] : [];
    const warnings: string[] = cl ? [...cl.warnings] : [];

    const isValid = errors.length === 0;
    const status = errors.length > 0 ? 'invalid' : warnings.length > 0 ? 'warning' : 'valid';

    return {
      isValid,
      errors,
      warnings,
      status: status as 'valid' | 'warning' | 'valid'
    };
  });

  readonly canPublish = computed(() => {
    const s1 = this.step1Validation();
    const s2 = this.step2Validation();
    const s3 = this.step3Validation();
    if (!s1.isValid || !s2.isValid || !s3.isValid) return false;

    const cl = this.checklistReport();
    if (!cl) return true;
    if (cl.blockers.length > 0) return false;
    if (cl.warnings.length > 0 && !this.acceptedWarnings()) return false;
    return true;
  });

  // Mutaciones
  setAcceptedWarnings(val: boolean) {
    this.acceptedWarnings.set(val);
  }

  setBoilerplateForLanguage(lang: string, code: string) {
    this.boilerplates.update(curr => ({ ...curr, [lang]: code }));
    if (lang === this.metadata().language) {
      this.boilerplate.set(code);
    }
  }

  addCustomAstRule(rule: ASTCustomRuleDTO) {
    this.astRules.update(curr => ({
      ...curr,
      custom_rules: [...(curr.custom_rules || []), rule]
    }));
  }

  removeCustomAstRule(index: number) {
    this.astRules.update(curr => ({
      ...curr,
      custom_rules: (curr.custom_rules || []).filter((_, i) => i !== index)
    }));
  }

  refreshChecklist() {
    const id = this.exerciseId();
    if (!id) {
      // Para ejercicios nuevos no persistidos aún, generar un checklist local
      const errors: string[] = [];
      const warnings: string[] = [];
      const info: string[] = [];

      if (this.cases().length === 0) {
        errors.push('Debes incluir al menos un caso de prueba.');
      }
      if (this.hiddenCasesCount() === 0) {
        warnings.push('0 casos ocultos: un print fijo podría aprobar este ejercicio.');
      }
      if (this.exampleCasesCount() === 0) {
        warnings.push('0 casos de ejemplo: el estudiante no verá casos ilustrativos en el enunciado.');
      }
      if (!this.referenceSolution().trim()) {
        warnings.push('No se ha configurado la solución de referencia del docente.');
      }

      info.push(`Total de casos: ${this.cases().length} (${this.exampleCasesCount()} ejemplos, ${this.publicCasesCount()} públicos, ${this.hiddenCasesCount()} ocultos).`);

      this.checklistReport.set({
        blockers: errors,
        warnings,
        info,
        can_publish: errors.length === 0
      });
      return;
    }

    this.isChecklistLoading.set(true);
    const refSol = this.referenceSolution().trim() ? {
      code: this.referenceSolution(),
      language: this.metadata().language
    } : undefined;

    this.courseService.getExerciseChecklist(id, refSol).subscribe({
      next: (report: ChecklistReportDTO) => {
        this.checklistReport.set(report);
        this.isChecklistLoading.set(false);
      },
      error: () => {
        this.isChecklistLoading.set(false);
      }
    });
  }

  // Mutaciones
  setCourseId(id: string) {
    this.courseId.set(id);
  }

  setStep(step: 1 | 2 | 3) {
    this.currentStep.set(step);
  }

  updateMetadata(update: Partial<ExerciseMetadata>) {
    this.metadata.update(curr => ({ ...curr, ...update }));
  }

  updateStatement(update: Partial<ExerciseStatement>) {
    this.statement.update(curr => ({ ...curr, ...update }));
  }

  updateAstRules(update: Partial<ASTRulesDTO>) {
    this.astRules.update(curr => ({ ...curr, ...update }));
  }

  setBoilerplate(code: string) {
    this.boilerplate.set(code);
  }

  setReferenceSolution(code: string) {
    this.referenceSolution.set(code);
  }

  setContract(contract: any) {
    this.contract.set(contract);
  }

  setCases(cases: TestCaseDTO[]) {
    this.cases.set(cases);
  }

  addCase(tc: TestCaseDTO) {
    this.cases.update(curr => [...curr, tc]);
  }

  addCases(tcs: TestCaseDTO[]) {
    this.cases.update(curr => [...curr, ...tcs]);
  }

  updateCase(index: number, update: Partial<TestCaseDTO>) {
    this.cases.update(curr => {
      const copy = [...curr];
      if (index >= 0 && index < copy.length) {
        copy[index] = { ...copy[index], ...update };
      }
      return copy;
    });
  }

  removeCase(index: number) {
    this.cases.update(curr => curr.filter((_, i) => i !== index));
  }

  duplicateCase(index: number) {
    this.cases.update(curr => {
      if (index < 0 || index >= curr.length) return curr;
      const target = curr[index];
      const clone: TestCaseDTO = {
        ...target,
        id: undefined,
        order_index: curr.length + 1
      };
      const next = [...curr];
      next.splice(index + 1, 0, clone);
      return next;
    });
  }

  reorderCase(fromIndex: number, toIndex: number) {
    this.cases.update(curr => {
      if (
        fromIndex < 0 ||
        fromIndex >= curr.length ||
        toIndex < 0 ||
        toIndex >= curr.length ||
        fromIndex === toIndex
      ) {
        return curr;
      }
      const next = [...curr];
      const [moved] = next.splice(fromIndex, 1);
      next.splice(toIndex, 0, moved);
      return next.map((item, idx) => ({ ...item, order_index: idx + 1 }));
    });
  }

  normalizeWeights() {
    this.cases.update(curr => curr.map(item => ({ ...item, weight: 1.0 })));
  }

  // Carga de ejercicio existente
  loadExercise(exerciseId: string, courseId: string) {
    this.exerciseId.set(exerciseId);
    this.courseId.set(courseId);
    this.isLoading.set(true);

    this.courseService.getExercise(exerciseId).subscribe({
      next: (ex: any) => {
        if (ex) {
          this.status.set(ex.status || 'draft');
          this.metadata.set({
            title: ex.title || '',
            difficulty: ex.difficulty || null,
            tags: ex.tags || [],
            modality: ex.type || 'judge',
            purpose: ex.purpose || 'class',
            per_student_seed: !!ex.per_student_seed,
            language: ex.language || 'python',
            allowed_languages: [ex.language || 'python'],
            time_limit_ms: ex.time_limit_ms || 1000,
            memory_limit_mb: ex.memory_limit_mb || 128,
            due_date: ex.due_date ? ex.due_date.substring(0, 16) : ''
          });

          this.statement.set({
            description: ex.description || '',
            input_text: '',
            output_text: '',
            constraints: ''
          });

          this.referenceSolution.set(ex.reference_solution || '');
          this.boilerplate.set(ex.boilerplate || '');

          if (ex.config?.algorithm) {
            this.astRules.set(ex.config.algorithm.ast_rules || {
              block_native_sort: false,
              block_system_modules: false,
              forbidden_imports: [],
              forbidden_functions: []
            });
            this.contract.set(ex.config.algorithm.input_format || ex.config.input_format || null);
            this.cases.set(ex.config.algorithm.test_cases || []);
          }
          this.initialSnapshot.set(this.serializeState());
        }
        this.isLoading.set(false);
      },
      error: () => {
        this.isLoading.set(false);
      }
    });
  }

  // Guardar borrador
  saveDraft(): Observable<any> | void {
    const meta = this.metadata();
    const stat = this.statement();
    const id = this.exerciseId();
    const cId = this.courseId();

    this.isSaving.set(true);

    const payload: CreateExerciseRequestDTO | UpdateExerciseRequestDTO = {
      subject_id: cId,
      title: meta.title || 'Borrador sin título',
      description: stat.description,
      type: meta.modality === 'judge' ? 'algorithm' : meta.modality,
      difficulty: meta.difficulty || undefined,
      tags: meta.tags,
      purpose: meta.purpose,
      per_student_seed: meta.per_student_seed,
      language: meta.language,
      time_limit_ms: meta.time_limit_ms,
      memory_limit_mb: meta.memory_limit_mb,
      due_date: meta.due_date ? new Date(meta.due_date).toISOString() : undefined,
      reference_solution: this.referenceSolution(),
      boilerplate: this.boilerplate(),
      ast_rules: this.astRules(),
      input_format: this.contract(),
      test_cases: this.cases()
    };

    if (id) {
      this.courseService.updateExercise(id, payload).subscribe({
        next: () => {
          this.isSaving.set(false);
          this.lastSavedAt.set(new Date());
          this.initialSnapshot.set(this.serializeState());
        },
        error: () => {
          this.isSaving.set(false);
        }
      });
    } else {
      this.courseService.createExercise(payload as CreateExerciseRequestDTO).subscribe({
        next: (created: any) => {
          this.isSaving.set(false);
          this.lastSavedAt.set(new Date());
          if (created?.id) {
            this.exerciseId.set(created.id);
            this.initialSnapshot.set(this.serializeState());
            this.router.navigate(['/teacher/courses', cId, 'exercises', created.id, 'edit'], {
              replaceUrl: true
            });
          }
        },
        error: () => {
          this.isSaving.set(false);
        }
      });
    }
  }

  private serializeState(): string {
    return JSON.stringify({
      meta: this.metadata(),
      stat: this.statement(),
      contract: this.contract(),
      cases: this.cases(),
      ref: this.referenceSolution(),
      ast: this.astRules(),
      bp: this.boilerplate()
    });
  }
}
