import { Injectable, computed, inject, signal } from '@angular/core';
import { Router } from '@angular/router';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { TeacherCourseService } from '../services/teacher-course.service';
import {
  ASTRulesDTO,
  ASTCustomRuleDTO,
  ChecklistReportDTO,
  TestCaseDTO,
  ExerciseTemplateSummary,
  ExerciseDetailDTO,
  ExercisePayloadDTO
} from '../models/teacher.models';
import { ExerciseContract } from './steps/step-contract/format-builder/format-builder.models';

interface ApiResponse<T> {
  data: T;
  message?: string;
}

type RunnerTemplatesApiResponse =
  | ExerciseTemplateSummary[]
  | { data?: ExerciseTemplateSummary[] | { data?: ExerciseTemplateSummary[] } };

export interface RubricLevelDTO {
  name: string;
  score: number;
  description: string;
}

export interface RubricCriterionDTO {
  id?: string;
  name: string;
  description: string;
  weight: number;
  levels: RubricLevelDTO[];
}

export interface RubricDTO {
  id?: string;
  exercise_id?: string;
  criteria: RubricCriterionDTO[];
}

export interface ExerciseMetadata {
  title: string;
  difficulty: 'easy' | 'medium' | 'hard' | null;
  tags: string[];
  modality: 'judge' | 'workspace' | 'database';
  environment_type: 'JUEZ_EFIMERO' | 'IDE_PERSISTENTE';
  template_id: string | null;
  template: ExerciseTemplateSummary | null;
  db_init_script: string;
  allow_resubmit: boolean;
  auto_checkpoint: boolean;
  purpose: 'class' | 'exam';
  per_student_seed: boolean;
  language: string;
  allowed_languages: string[];
  time_limit_ms: number;
  memory_limit_mb: number;
  due_date?: string;
  expected_complexity?: string;
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
  private http = inject(HttpClient);

  // Context
  readonly courseId = signal<string>('');
  readonly exerciseId = signal<string | null>(null);
  readonly status = signal<'draft' | 'published'>('draft');
  readonly isSaving = signal<boolean>(false);
  readonly isPublishing = signal<boolean>(false);
  readonly showPublishConfirmModal = signal<boolean>(false);
  readonly publishError = signal<string | null>(null);
  readonly publishSuccess = signal<boolean>(false);
  readonly isLoading = signal<boolean>(false);
  readonly lastSavedAt = signal<Date | null>(null);
  readonly currentStep = signal<1 | 2 | 3>(1);
  readonly studentPreviewOpen = signal<boolean>(false);
  readonly studentPreviewMode = signal<'preview' | 'simulation'>('preview');

  // Rubrica (IDE Persistente)

  readonly rubric = signal<RubricDTO | null>(null);

  // Paso 1: Identidad y Pedagogía
  readonly metadata = signal<ExerciseMetadata>({
    title: '',
    difficulty: null,
    tags: [],
    modality: 'judge',
    environment_type: 'JUEZ_EFIMERO',
    template_id: null,
    template: null,
    db_init_script: '',
    allow_resubmit: true,
    auto_checkpoint: true,
    purpose: 'class',
    per_student_seed: false,
    language: '',
    allowed_languages: [],
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
  readonly contract = signal<ExerciseContract | null>(null);
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
  readonly stepAttempted = signal<boolean>(false);
  readonly hasInteracted = signal<boolean>(false);
  readonly touchedFields = signal<Set<string>>(new Set<string>());

  // Estado de plantillas runner para Juez Efímero
  readonly runnerTemplates = signal<ExerciseTemplateSummary[]>([]);

  // Snapshot inicial para cálculo de dirty
  private initialSnapshot = signal<string>('');
  private autoSaveTimeout: ReturnType<typeof setTimeout> | null = null;

  constructor() {
    this.initialSnapshot.set(this.serializeState());
  }

  readonly dirty = computed(() => {
    const current = this.serializeState();
    return this.initialSnapshot() !== '' && current !== this.initialSnapshot();
  });

  private getDraftStorageKey(): string {
    const exId = this.exerciseId();
    const cId = this.courseId();
    if (exId) {
      return `solv_draft_${cId}_${exId}`;
    }
    return `solv_draft_${cId}_new`;
  }

  saveDraftToStorage() {
    if (typeof localStorage === 'undefined') return;
    try {
      const state = this.serializeState();
      localStorage.setItem(this.getDraftStorageKey(), state);
    } catch {
      // Ignorar errores de quota en localStorage
    }
  }

  restoreDraftFromStorage(): boolean {
    if (typeof localStorage === 'undefined') return false;
    try {
      const raw = localStorage.getItem(this.getDraftStorageKey());
      if (!raw) return false;
      const parsed = JSON.parse(raw);
      if (parsed.meta) this.metadata.set(parsed.meta);
      if (parsed.stat) this.statement.set(parsed.stat);
      if (parsed.contract) this.contract.set(parsed.contract);
      if (Array.isArray(parsed.cases)) this.cases.set(parsed.cases);
      if (parsed.ref !== undefined) this.referenceSolution.set(parsed.ref);
      if (parsed.ast) this.astRules.set(parsed.ast);
      if (parsed.bp !== undefined) this.boilerplate.set(parsed.bp);
      if (parsed.rub !== undefined) this.rubric.set(parsed.rub);
      this.hasInteracted.set(true);
      return true;
    } catch {
      return false;
    }
  }

  clearLocalDraft() {
    if (typeof localStorage === 'undefined') return;
    try {
      localStorage.removeItem(this.getDraftStorageKey());
    } catch {
      // Storage unavailable: ignore.
    }
  }

  private canAutoSaveRemoteDraft(): boolean {
    const m = this.metadata();
    if (!m.title.trim()) {
      return false;
    }
    if (!this.exerciseId() && m.environment_type === 'IDE_PERSISTENTE' && !m.template_id) {
      return false;
    }
    return true;
  }

  scheduleAutoSave() {
    this.saveDraftToStorage();
    if (this.autoSaveTimeout) {
      clearTimeout(this.autoSaveTimeout);
    }
    this.autoSaveTimeout = setTimeout(() => {
      // Si ya tiene un título y plantilla válida, auto-guardar en backend
      if (this.canAutoSaveRemoteDraft() && this.dirty() && !this.isSaving()) {
        this.saveDraft();
      }
    }, 1500);
  }

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
    if (m.environment_type === 'JUEZ_EFIMERO') {
      if (!m.language) {
        errors.push('Debes seleccionar un lenguaje de ejecución');
      }
      if (typeof m.time_limit_ms !== 'number' || isNaN(m.time_limit_ms) || m.time_limit_ms < 100 || m.time_limit_ms > 30000) {
        errors.push('El tiempo límite debe ser un número entre 100 y 30.000 ms');
      }
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

  readonly touchedStep1Errors = computed(() => {
    const m = this.metadata();
    const s = this.statement();
    const errors: string[] = [];

    if (this.isFieldTouched('title') && !m.title.trim()) {
      errors.push('El título del ejercicio es obligatorio');
    }
    if (this.isFieldTouched('description') && !s.description.trim()) {
      errors.push('La descripción del enunciado es obligatoria');
    }
    if (this.isFieldTouched('seed') && m.per_student_seed && m.purpose !== 'exam') {
      errors.push('La semilla por estudiante solo está permitida en modalidad Examen');
    }
    if (m.environment_type === 'JUEZ_EFIMERO') {
      if (this.isFieldTouched('language') && !m.language) {
        errors.push('Debes seleccionar un lenguaje de ejecución');
      }
      if (this.isFieldTouched('time_limit') && (typeof m.time_limit_ms !== 'number' || isNaN(m.time_limit_ms) || m.time_limit_ms < 100 || m.time_limit_ms > 30000)) {
        errors.push('El tiempo límite debe ser un número entre 100 y 30.000 ms');
      }
    }
    return errors;
  });

  readonly step2Validation = computed(() => {
    const m = this.metadata();
    const isIDE = m.environment_type === 'IDE_PERSISTENTE';
    const errors: string[] = [];
    const warnings: string[] = [];

    if (isIDE) {
      if (!m.template_id) {
        errors.push('Debes seleccionar una plantilla aprobada del catálogo');
      }
      const rub = this.rubric();
      if (!rub || !rub.criteria || rub.criteria.length === 0) {
        warnings.push('No has definido los criterios de la rúbrica de evaluación');
      } else {
        const totalWeight = rub.criteria.reduce((sum, c) => sum + (c.weight || 0), 0);
        if (Math.abs(totalWeight - 100) > 0.01) {
          errors.push(`La suma de pesos de la rúbrica es ${totalWeight}%. Debe ser exactamente 100%`);
        }
      }
    } else {
      const cs = this.cases();
      const ref = this.referenceSolution();
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
        warnings.push('No hay casos ocultos; un alumno podría aprobar mostrando respuestas fijas');
      }
    }

    return {
      isValid: errors.length === 0,
      errors,
      warnings,
      status: errors.length > 0 ? 'invalid' : warnings.length > 0 ? 'warning' : 'valid'
    };
  });

  readonly touchedStep2Errors = computed(() => {
    const m = this.metadata();
    const isIDE = m.environment_type === 'IDE_PERSISTENTE';
    const errors: string[] = [];

    if (isIDE) {
      if (this.isFieldTouched('template') && !m.template_id) {
        errors.push('Debes seleccionar una plantilla aprobada del catálogo');
      }
      const rub = this.rubric();
      if (this.isFieldTouched('rubric') && rub && rub.criteria && rub.criteria.length > 0) {
        const totalWeight = rub.criteria.reduce((sum, c) => sum + (c.weight || 0), 0);
        if (Math.abs(totalWeight - 100) > 0.01) {
          errors.push(`La suma de pesos de la rúbrica es ${totalWeight}%. Debe ser exactamente 100%`);
        }
      }
    } else {
      const cs = this.cases();
      if (this.isFieldTouched('cases') && cs.length === 0) {
        errors.push('Debes incluir al menos un caso de prueba');
      }
    }
    return errors;
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

  setStudentPreviewOpen(open: boolean, mode: 'preview' | 'simulation' = 'preview') {
    this.studentPreviewOpen.set(open);
    this.studentPreviewMode.set(mode);
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

  flushAutoSave(callback?: (err?: unknown, result?: unknown) => void): void {
    if (this.autoSaveTimeout) {
      clearTimeout(this.autoSaveTimeout);
      this.autoSaveTimeout = null;
    }
    this.saveDraftToStorage();
    if (this.canAutoSaveRemoteDraft() && this.dirty() && !this.isSaving()) {
      this.saveDraft(callback);
    } else if (callback) {
      callback();
    }
  }

  refreshChecklist() {
    const id = this.exerciseId();
    const meta = this.metadata();
    const isIDE = meta.environment_type === 'IDE_PERSISTENTE';

    if (isIDE) {
      const s1 = this.step1Validation();
      const errors: string[] = [...s1.errors];
      const warnings: string[] = [...s1.warnings];
      const info: string[] = [];

      if (!meta.template_id) {
        errors.push('No se ha asignado una plantilla homologada del catálogo oficial.');
      } else {
        info.push(`Plantilla vinculada: ${meta.template?.name || meta.template_id}`);
      }

      const rub = this.rubric();
      if (!rub || !rub.criteria || rub.criteria.length === 0) {
        warnings.push('Rúbrica de evaluación no definida para este laboratorio de IDE Persistente.');
      } else {
        const totalWeight = rub.criteria.reduce((sum, c) => sum + (c.weight || 0), 0);
        if (Math.abs(totalWeight - 100) > 0.01) {
          errors.push(`La suma de pesos de la rúbrica es ${totalWeight}%. Debe ser exactamente 100%.`);
        } else {
          info.push(`Rúbrica configurada correctamente (${rub.criteria.length} criterios, 100% peso).`);
        }
      }

      if (!meta.due_date) {
        warnings.push('No hay fecha límite de entrega configurada.');
      }

      if (meta.db_init_script?.trim()) {
        info.push('Script de inicialización DDL/DML configurado.');
      }

      this.checklistReport.set({
        blockers: errors,
        warnings,
        info,
        can_publish: errors.length === 0
      });
      return;
    }

    if (!id) {
      this.evaluateLocalChecklist();
      return;
    }

    if (this.dirty()) {
      this.isChecklistLoading.set(true);
      this.flushAutoSave((err) => {
        const currentId = this.exerciseId();
        if (!err && currentId) {
          this.fetchRemoteChecklist(currentId);
        } else {
          this.evaluateLocalChecklist();
          this.isChecklistLoading.set(false);
        }
      });
      return;
    }

    this.fetchRemoteChecklist(id);
  }

  private evaluateLocalChecklist(): void {
    const s1 = this.step1Validation();
    const errors: string[] = [...s1.errors];
    const warnings: string[] = [...s1.warnings];
    const info: string[] = [];

    if (this.cases().length === 0) {
      errors.push('Debes incluir al menos un caso de prueba.');
    }
    if (this.hiddenCasesCount() === 0) {
      warnings.push('Sin casos ocultos: un alumno podría aprobar mostrando salidas fijas en consola.');
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
  }

  private fetchRemoteChecklist(id: string): void {
    this.isChecklistLoading.set(true);
    const refSol = this.referenceSolution().trim() ? {
      code: this.referenceSolution(),
      language: this.metadata().language
    } : undefined;

    this.courseService.getExerciseChecklist(id, refSol).subscribe({
      next: (report: ChecklistReportDTO) => {
        const s1 = this.step1Validation();
        let remoteBlockers = report.blockers || [];
        if (this.cases().length > 0) {
          remoteBlockers = remoteBlockers.filter(b => !b.toLowerCase().includes('caso de prueba'));
        }
        if (this.referenceSolution().trim()) {
          remoteBlockers = remoteBlockers.filter(b => !b.toLowerCase().includes('solución de referencia') && !b.toLowerCase().includes('solucion de referencia'));
        }

        const mergedBlockers = Array.from(new Set([...s1.errors, ...remoteBlockers]));
        const mergedWarnings = Array.from(new Set([...s1.warnings, ...(report.warnings || [])]));

        this.checklistReport.set({
          ...report,
          blockers: mergedBlockers,
          warnings: mergedWarnings,
          can_publish: mergedBlockers.length === 0
        });
        this.isChecklistLoading.set(false);
      },
      error: () => {
        this.evaluateLocalChecklist();
        this.isChecklistLoading.set(false);
      }
    });
  }

  setRubric(rubric: RubricDTO | null) {
    this.hasInteracted.set(true);
    this.rubric.set(rubric);
    this.scheduleAutoSave();
  }

  setCourseId(id: string) {
    this.courseId.set(id);
  }

  setStep(step: 1 | 2 | 3) {
    if (this.dirty() && this.metadata().title.trim()) {
      this.flushAutoSave();
    }
    this.currentStep.set(step);
    this.stepAttempted.set(false);
  }

  markFieldTouched(fieldName: string) {
    this.touchedFields.update(curr => {
      const next = new Set(curr);
      next.add(fieldName);
      return next;
    });
  }

  isFieldTouched(fieldName: string): boolean {
    return this.touchedFields().has(fieldName) || this.stepAttempted() || this.exerciseId() !== null;
  }

  attemptNextStep(): boolean {
    const current = this.currentStep();
    if (current < 3) {
      if (this.dirty() && this.metadata().title.trim()) {
        this.flushAutoSave();
      }
      this.currentStep.set((current + 1) as 1 | 2 | 3);
      this.stepAttempted.set(false);
      return true;
    }
    return false;
  }

  updateMetadata(update: Partial<ExerciseMetadata>) {
    this.hasInteracted.set(true);
    this.metadata.update(curr => ({ ...curr, ...update }));
    this.scheduleAutoSave();
  }

  loadRunnerTemplates(): void {
    if (this.runnerTemplates().length > 0) return;

    this.http.get<RunnerTemplatesApiResponse>('/api/v1/templates?limit=50').subscribe({
      next: (res) => {
        let items: ExerciseTemplateSummary[] = [];
        if (Array.isArray(res)) {
          items = res;
        } else if (res && typeof res === 'object' && 'data' in res) {
          const data = (res as { data?: unknown }).data;
          if (Array.isArray(data)) {
            items = data as ExerciseTemplateSummary[];
          } else if (data && typeof data === 'object' && 'data' in data && Array.isArray((data as { data: unknown }).data)) {
            items = (data as { data: ExerciseTemplateSummary[] }).data;
          }
        }

        const runners = items.filter(t => {
          const env = t.target_environment || t.environment_type || '';
          const isJudgeCompatible = env === 'JUEZ_EFIMERO' || env === 'JUEZ_VIRTUAL' || env === 'AMBOS';
          const isApproved = t.status === 'approved' || t.status === 'APROBADA' || !t.status;
          return isJudgeCompatible && isApproved;
        });
        this.runnerTemplates.set(runners);

        const currentLang = this.metadata().language;
        if (currentLang) {
          this.syncLanguageWithTemplate(currentLang, runners);
        }
      },
      error: () => {}
    });
  }

  setLanguageAndSyncTemplate(language: string): void {
    this.updateMetadata({
      language,
      allowed_languages: language ? [language] : []
    });

    if (language) {
      this.syncLanguageWithTemplate(language);
    }
  }

  private syncLanguageWithTemplate(lang: string, list?: ExerciseTemplateSummary[]): void {
    const templates = list || this.runnerTemplates();
    const match = templates.find(r => this.deriveLanguageCode(r) === lang);
    if (match) {
      const baseRam = match.base_ram_mb || 128;
      const isDual = (match.target_environment || match.environment_type) === 'AMBOS';
      const resolvedRam = isDual ? (baseRam <= 384 ? 128 : 256) : baseRam;
      this.updateMetadata({
        template_id: match.id,
        template: match,
        memory_limit_mb: resolvedRam
      });
    }
  }

  deriveLanguageCode(template: ExerciseTemplateSummary): string {
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

  updateStatement(update: Partial<ExerciseStatement>) {
    this.hasInteracted.set(true);
    this.statement.update(curr => ({ ...curr, ...update }));
    this.scheduleAutoSave();
  }

  updateAstRules(update: Partial<ASTRulesDTO>) {
    this.astRules.update(curr => ({ ...curr, ...update }));
    this.scheduleAutoSave();
  }

  setBoilerplate(code: string) {
    this.boilerplate.set(code);
    this.scheduleAutoSave();
  }

  setReferenceSolution(code: string) {
    this.referenceSolution.set(code);
    this.scheduleAutoSave();
  }

  setContract(contract: ExerciseContract | null) {
    this.contract.set(contract);
    this.scheduleAutoSave();
  }

  setCases(cases: TestCaseDTO[]) {
    this.cases.set(cases);
    this.scheduleAutoSave();
  }

  addCase(tc: TestCaseDTO) {
    this.cases.update(curr => [...curr, tc]);
    this.scheduleAutoSave();
  }

  addCases(tcs: TestCaseDTO[]) {
    this.cases.update(curr => [...curr, ...tcs]);
    this.scheduleAutoSave();
  }

  updateCase(index: number, update: Partial<TestCaseDTO>) {
    this.cases.update(curr => {
      const copy = [...curr];
      if (index >= 0 && index < copy.length) {
        copy[index] = { ...copy[index], ...update };
      }
      return copy;
    });
    this.scheduleAutoSave();
  }

  removeCase(index: number) {
    this.cases.update(curr => curr.filter((_, i) => i !== index));
    this.scheduleAutoSave();
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
    this.hasInteracted.set(true);
    this.isLoading.set(true);

    this.courseService.getExercise(exerciseId).subscribe({
      next: (ex: ExerciseDetailDTO) => {
        if (ex) {
          this.status.set(ex.status === 'published' ? 'published' : 'draft');
          const isIDE = ex.environment_type === 'IDE_PERSISTENTE' || ex.type === 'workspace' || ex.type === 'project';
          this.metadata.set({
            title: ex.title || '',
            difficulty: (ex.difficulty as 'easy' | 'medium' | 'hard' | null) || null,
            tags: ex.tags || [],
            modality: isIDE ? 'workspace' : ((ex.type as 'judge' | 'workspace' | 'database') || 'judge'),
            environment_type: isIDE ? 'IDE_PERSISTENTE' : 'JUEZ_EFIMERO',
            template_id: ex.template_id || null,
            template: ex.template || null,
            db_init_script: ex.config?.ide?.db_init_script || ex.config?.database?.init_script || '',
            allow_resubmit: ex.config?.ide?.allow_resubmit ?? true,
            auto_checkpoint: ex.config?.ide?.auto_checkpoint ?? true,
            purpose: (ex.purpose as 'class' | 'exam') || 'class',
            per_student_seed: !!ex.per_student_seed,
            language: ex.language || 'python',
            allowed_languages: [ex.language || 'python'],
            time_limit_ms: ex.time_limit_ms || 1000,
            memory_limit_mb: ex.template?.base_ram_mb || ex.memory_limit_mb || 128,
            due_date: ex.due_date ? ex.due_date.substring(0, 16) : '',
            expected_complexity: ex.expected_complexity || undefined
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

          if (isIDE && exerciseId) {
            this.http.get<ApiResponse<RubricDTO> | RubricDTO>(`/api/v1/exercises/${exerciseId}/rubric`).subscribe({
              next: (res) => {
                const data = (res && typeof res === 'object' && 'data' in res ? (res as ApiResponse<RubricDTO>).data : res) as RubricDTO | null;
                if (data && data.criteria) {
                  this.rubric.set(data);
                }
              },
              error: () => {}
            });
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

  private buildPayload(): ExercisePayloadDTO {
    const meta = this.metadata();
    const stat = this.statement();
    const cId = this.courseId();
    const isIDE = meta.environment_type === 'IDE_PERSISTENTE';

    const payload: ExercisePayloadDTO = {
      subject_id: cId,
      title: meta.title || 'Borrador sin título',
      description: stat.description,
      type: isIDE ? 'project' : (meta.modality === 'judge' ? 'algorithm' : meta.modality),
      environment_type: meta.environment_type || (isIDE ? 'IDE_PERSISTENTE' : 'JUEZ_EFIMERO'),
      template_id: meta.template_id || undefined,
      difficulty: meta.difficulty || undefined,
      tags: meta.tags,
      purpose: meta.purpose,
      per_student_seed: meta.per_student_seed,
      language: meta.language,
      time_limit_ms: meta.time_limit_ms,
      due_date: meta.due_date ? new Date(meta.due_date).toISOString() : undefined,
      expected_complexity: meta.expected_complexity || undefined,
      ast_rules: this.astRules(),
      boilerplate: this.boilerplate()
    };

    if (isIDE) {
      payload.db_init_script = meta.db_init_script || undefined;
      payload.allow_resubmit = meta.allow_resubmit;
      payload.auto_checkpoint = meta.auto_checkpoint;
    } else {
      payload.reference_solution = this.referenceSolution();
      payload.input_format = this.contract() || undefined;
      payload.test_cases = this.cases();
    }

    return payload;
  }

  /**
   * Persiste la rúbrica del IDE antes de cerrar el guardado o la publicación.
   * Sin esta espera el PUT competía contra el publish y SpeedGrader podía
   * quedar sin criterios de calificación.
   */
  private async persistIdeRubric(exerciseId: string): Promise<void> {
    const rubric = this.rubric();
    if (this.metadata().environment_type !== 'IDE_PERSISTENTE' || !rubric) {
      return;
    }
    await firstValueFrom(this.http.put(`/api/v1/exercises/${exerciseId}/rubric`, rubric));
  }

  private errorMessage(err: unknown): string {
    if (err && typeof err === 'object') {
      const candidate = err as { error?: { message?: string }; message?: string };
      return candidate.error?.message || candidate.message || '';
    }
    return '';
  }

  // Guardar borrador
  async saveDraft(callback?: (err?: unknown, result?: unknown) => void): Promise<void> {
    const id = this.exerciseId();
    const cId = this.courseId();
    const payload = this.buildPayload();

    this.isSaving.set(true);

    try {
      if (id) {
        await firstValueFrom(this.courseService.updateExercise(id, payload));
        // En borrador la rúbrica no bloquea el guardado: se reintenta al publicar.
        await this.persistIdeRubric(id).catch(() => undefined);
        this.lastSavedAt.set(new Date());
        this.initialSnapshot.set(this.serializeState());
        this.isSaving.set(false);
        if (callback) callback(null, undefined);
        return;
      }

      const created = await firstValueFrom(this.courseService.createExercise(payload));
      const newId = created?.id;
      if (newId) {
        await this.persistIdeRubric(newId).catch(() => undefined);
        this.exerciseId.set(newId);
        this.initialSnapshot.set(this.serializeState());
        this.router.navigate(['/teacher/courses', cId, 'exercises', newId, 'edit'], {
          replaceUrl: true
        });
      }
      this.lastSavedAt.set(new Date());
      this.isSaving.set(false);
      if (callback) callback(null, created);
    } catch (err) {
      this.isSaving.set(false);
      if (callback) callback(err);
    }
  }

  async executePublish(onSuccess?: () => void, onError?: (err: unknown) => void): Promise<void> {
    const id = this.exerciseId();
    const payload = this.buildPayload();

    this.isPublishing.set(true);
    this.publishError.set(null);

    let step: 'save' | 'rubric' | 'publish' = 'save';

    try {
      let exerciseIdToPublish = id;

      if (exerciseIdToPublish) {
        await firstValueFrom(this.courseService.updateExercise(exerciseIdToPublish, payload));
      } else {
        const created = await firstValueFrom(this.courseService.createExercise(payload));
        exerciseIdToPublish = created?.id;
        if (!exerciseIdToPublish) {
          throw new Error('No se pudo obtener el identificador del ejercicio creado.');
        }
        this.exerciseId.set(exerciseIdToPublish);
      }
      this.lastSavedAt.set(new Date());

      step = 'rubric';
      await this.persistIdeRubric(exerciseIdToPublish);

      step = 'publish';
      await firstValueFrom(this.courseService.publishExercise(exerciseIdToPublish));

      this.isPublishing.set(false);
      this.status.set('published');
      this.publishSuccess.set(true);
      this.showPublishConfirmModal.set(false);
      this.initialSnapshot.set(this.serializeState());
      if (onSuccess) onSuccess();
    } catch (err) {
      this.isPublishing.set(false);
      this.showPublishConfirmModal.set(false);
      const msg = this.errorMessage(err) || (step === 'rubric'
        ? 'No se pudo guardar la rúbrica de evaluación del proyecto.'
        : step === 'save'
          ? 'Error al guardar los cambios antes de publicar.'
          : 'No se pudo publicar el ejercicio.');
      this.publishError.set(msg);
      if (onError) onError(err);
    }
  }

  resetDraft() {
    this.metadata.set({
      title: '',
      difficulty: null,
      tags: [],
      modality: 'judge',
      environment_type: 'JUEZ_EFIMERO',
      template_id: null,
      template: null,
      db_init_script: '',
      allow_resubmit: true,
      auto_checkpoint: true,
      purpose: 'class',
      per_student_seed: false,
      language: '',
      allowed_languages: [],
      time_limit_ms: 1000,
      memory_limit_mb: 128,
      due_date: ''
    });
    this.statement.set({
      description: '',
      input_text: '',
      output_text: '',
      constraints: ''
    });
    this.contract.set(null);
    this.cases.set([]);
    this.referenceSolution.set('');
    this.astRules.set({
      block_native_sort: false,
      block_system_modules: false,
      forbidden_imports: [],
      forbidden_functions: [],
      custom_rules: []
    });
    this.boilerplate.set('');
    this.boilerplates.set({});
    this.rubric.set(null);
    this.checklistReport.set(null);
    this.acceptedWarnings.set(false);
    if (this.autoSaveTimeout) {
      clearTimeout(this.autoSaveTimeout);
      this.autoSaveTimeout = null;
    }
    this.clearLocalDraft();
    this.currentStep.set(1);
    this.initialSnapshot.set(this.serializeState());
  }

  private serializeState(): string {
    return JSON.stringify({
      meta: this.metadata(),
      stat: this.statement(),
      contract: this.contract(),
      cases: this.cases(),
      ref: this.referenceSolution(),
      ast: this.astRules(),
      bp: this.boilerplate(),
      rub: this.rubric()
    });
  }
}
