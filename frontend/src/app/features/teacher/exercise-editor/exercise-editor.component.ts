import { Component, ElementRef, OnDestroy, OnInit, computed, effect, inject, input, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterLink, ActivatedRoute, Router } from '@angular/router';
import {
  LucideArrowLeft,
  LucideCheck,
  LucideEye,
  LucideTrash2,
  LucideGlobe,
  LucideLoader2,
  LucideSparkles,
  LucideTerminal,
  LucideMoreVertical,
  LucidePanelRightOpen
} from '@lucide/angular';
import { ExerciseEditorStore } from './exercise-editor.store';
import { StepIdentityComponent } from './steps/step-identity/step-identity.component';
import { StepContractComponent } from './steps/step-contract/step-contract.component';
import { StepIdeEnvironmentComponent } from './steps/step-ide-environment/step-ide-environment.component';
import { StepPublicationComponent } from './steps/step-publication/step-publication.component';
import { StudentPreviewModalComponent } from './student-preview-modal/student-preview-modal.component';
import { AiExerciseGeneratorModalComponent } from './ai-exercise-generator-modal/ai-exercise-generator-modal.component';
import { StepperNavComponent } from './shared/stepper-nav.component';
import { PreflightModalComponent } from './shared/preflight-modal/preflight-modal.component';
import { ConsultDrawerComponent } from './shared/consult-drawer/consult-drawer.component';
import { ConfirmModalComponent } from '@shared/components/confirm-modal/confirm-modal.component';
import { DismissibleDirective } from '@shared/directives/dismissible.directive';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { EnvironmentLabelPipe } from '@shared/pipes/environment-label.pipe';
import { TeacherAiCopilotService, AIGenerateExerciseResponse } from '../services/teacher-ai-copilot.service';
import { TeacherDashboardService } from '../services/teacher-dashboard.service';
import { TeacherCourseService } from '../services/teacher-course.service';
import { PracticeEnvTestService } from '../services/practice-env-test.service';
import { PracticeEnvTestJob } from '../models/practice-env-test.models';
import { Subscription, catchError, of } from 'rxjs';

@Component({
  selector: 'exercise-editor',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    RouterLink,
    LucideArrowLeft,
    LucideCheck,
    LucideEye,
    LucideTrash2,
    LucideGlobe,
    LucideLoader2,
    LucideSparkles,
    LucideTerminal,
    LucideMoreVertical,
    LucidePanelRightOpen,
    StepIdentityComponent,
    StepContractComponent,
    StepIdeEnvironmentComponent,
    StepPublicationComponent,
    StudentPreviewModalComponent,
    AiExerciseGeneratorModalComponent,
    StepperNavComponent,
    PreflightModalComponent,
    ConsultDrawerComponent,
    ConfirmModalComponent,
    DismissibleDirective,
    MachineDataDirective,
    EnvironmentLabelPipe
  ],
  template: `
    <div class="editor-page">
      <header class="editor-header">
        <div class="header-left">
          <a [routerLink]="['/teacher/cursos', store.courseId()]" class="btn-ghost btn-sm back-link">
            <svg lucideArrowLeft class="btn-icon-sm" aria-hidden="true"></svg>
            {{ courseName() || 'Materia' }}
          </a>
          <span class="header-divider" aria-hidden="true"></span>

          <div class="title-block">
            <h1 class="editor-title">
              {{ store.metadata().title || (store.exerciseId() ? 'Editar práctica' : (store.metadata().purpose === 'exam' ? 'Nuevo examen' : 'Nueva práctica')) }}
            </h1>
            <span class="status-badge" [class.is-published]="store.status() === 'published'">
              {{ store.status() === 'published' ? 'Publicado' : 'Borrador' }}
            </span>
          </div>

          <span class="persist-state" [class.is-dirty]="store.dirty()">
            <span class="persist-dot" aria-hidden="true"></span>
            {{ store.dirty() || !store.exerciseId() ? 'Cambios sin guardar' : 'Guardado' }}
          </span>

          @if (store.dirty() || !store.exerciseId()) {
            <button type="button" class="btn-ghost btn-sm save-btn" [disabled]="store.isSaving()" (click)="saveNow()">
              Guardar
            </button>
          }
        </div>

        <div class="header-actions">
          @if (store.isIde() && isEnvTestRunning() && store.currentStep() !== 2) {
            <button type="button" class="env-pill" (click)="store.setStep(2)" title="Volver a la sección 2">
              <svg lucideLoader2 class="btn-icon-sm motion-spin" aria-hidden="true"></svg>
              Probando entorno… <span machineData>{{ envTestJob()?.progress?.percent ?? 0 }} %</span>
            </button>
          }

          <button type="button" class="btn-secondary btn-sm" (click)="store.setStudentPreviewOpen(true, 'preview')">
            <svg lucideEye class="btn-icon-sm" aria-hidden="true"></svg>
            Vista previa
          </button>

          <button type="button" class="btn-secondary btn-sm" (click)="aiModalOpen.set(true)">
            <svg lucideSparkles class="btn-icon-sm" aria-hidden="true"></svg>
            Generar con IA
            <span class="ai-mode">{{ aiEnabled() ? 'Institucional' : 'BYOK' }}</span>
          </button>

          <div class="more-actions-dropdown" dismissible (dismiss)="showMoreActions.set(false)">
            <button
              type="button"
              class="btn-secondary btn-sm more-toggle"
              aria-label="Más acciones del editor"
              [attr.aria-expanded]="showMoreActions()"
              (click)="showMoreActions.update(v => !v)"
            >
              <svg lucideMoreVertical class="btn-icon-sm" aria-hidden="true"></svg>
            </button>

            @if (showMoreActions()) {
              <div class="dropdown-menu animate-dropdown" animate.leave="motion-dropdown-out">
                <button type="button" class="dropdown-item is-danger" (click)="confirmDeleteDraft(); showMoreActions.set(false)">
                  <svg lucideTrash2 class="btn-icon-sm" aria-hidden="true"></svg>
                  Eliminar borrador
                </button>
              </div>
            }
          </div>

          <button
            type="button"
            class="btn-primary btn-sm publish-btn"
            [disabled]="store.isPublishing()"
            (click)="requestPublish()"
          >
            @if (store.isPublishing()) {
              <svg lucideLoader2 class="btn-icon-sm motion-spin" aria-hidden="true"></svg>
              Publicando…
            } @else if (store.status() === 'published') {
              <svg lucideGlobe class="btn-icon-sm" aria-hidden="true"></svg>
              {{ store.templateChangedSincePublish() ? 'Republicar entorno' : 'Actualizar publicación' }}
            } @else {
              <svg lucideGlobe class="btn-icon-sm" aria-hidden="true"></svg>
              Publicar
            }
          </button>
        </div>
      </header>

      <div class="editor-layout">
        <stepper-nav (openPreflight)="openPreflightModal()" />

        <div class="editor-main">
          <section class="context-strip" aria-label="Contexto de la práctica">
            <div class="context-groups">
              <div class="context-group" role="radiogroup" aria-label="Modalidad">
                <span class="context-label font-mono">MODALIDAD</span>
                <div class="segmented">
                  <button
                    type="button"
                    role="radio"
                    class="segment"
                    [class.is-active]="store.isIde()"
                    [attr.aria-checked]="store.isIde()"
                    (click)="setEnvironment('IDE_PERSISTENTE')"
                  >
                    {{ 'IDE_PERSISTENTE' | environmentLabel }}
                  </button>
                  <button
                    type="button"
                    role="radio"
                    class="segment"
                    [class.is-active]="!store.isIde()"
                    [attr.aria-checked]="!store.isIde()"
                    (click)="setEnvironment('JUEZ_EFIMERO')"
                  >
                    {{ 'JUEZ_EFIMERO' | environmentLabel }}
                  </button>
                </div>
              </div>

              <div class="context-group" role="radiogroup" aria-label="Propósito">
                <span class="context-label font-mono">PROPÓSITO</span>
                <div class="segmented">
                  <button
                    type="button"
                    role="radio"
                    class="segment"
                    [class.is-active]="store.metadata().purpose === 'class'"
                    [attr.aria-checked]="store.metadata().purpose === 'class'"
                    (click)="setPurpose('class')"
                  >
                    Clase / Formativo
                  </button>
                  <button
                    type="button"
                    role="radio"
                    class="segment"
                    [class.is-active]="store.metadata().purpose === 'exam'"
                    [attr.aria-checked]="store.metadata().purpose === 'exam'"
                    (click)="setPurpose('exam')"
                  >
                    Examen / Calificado
                  </button>
                </div>
              </div>
            </div>

            @if (store.metadata().purpose === 'exam') {
              <div class="exam-strip">
                <div class="context-group" role="radiogroup" aria-label="Acceso a internet">
                  <span class="context-label font-mono">ACCESO A INTERNET</span>
                  <div class="segmented">
                    <button
                      type="button"
                      role="radio"
                      class="segment"
                      [class.is-active]="store.metadata().exam_network !== 'egress'"
                      [attr.aria-checked]="store.metadata().exam_network !== 'egress'"
                      (click)="setExamNetwork('none')"
                    >
                      Sin internet
                    </button>
                    <button
                      type="button"
                      role="radio"
                      class="segment"
                      [class.is-active]="store.metadata().exam_network === 'egress'"
                      [attr.aria-checked]="store.metadata().exam_network === 'egress'"
                      (click)="setExamNetwork('egress')"
                    >
                      Excepción con motivo
                    </button>
                  </div>
                </div>

                @if (store.metadata().exam_network === 'egress') {
                  <div class="reason-field">
                    <label class="reason-label" for="exam-reason">Motivo de la excepción *</label>
                    <textarea
                      id="exam-reason"
                      class="form-textarea reason-input"
                      rows="2"
                      maxlength="240"
                      [class.has-warning]="!examReasonValid()"
                      [ngModel]="store.metadata().exam_network_reason"
                      (ngModelChange)="onExamReasonChange($event)"
                      placeholder="Explique por qué la evaluación requiere conexión externa"
                    ></textarea>
                    <span class="reason-meta font-mono">
                      <span machineData>{{ (store.metadata().exam_network_reason || '').length }}</span> / 240
                      @if (!examReasonValid()) {
                        <span class="reason-hint">Mínimo 10 caracteres</span>
                      }
                    </span>
                  </div>
                }

                @if (!store.isIde()) {
                  <label class="seed-chip">
                    <input
                      type="checkbox"
                      [checked]="store.metadata().per_student_seed"
                      (change)="toggleAntiCopy($event)"
                    />
                    Anti-copia (semilla por estudiante)
                  </label>
                }

                <p class="exam-note">
                  El editor y las ejecuciones locales continúan funcionando sin conexión a internet.
                </p>
              </div>
            }
          </section>

          @if (store.status() === 'published') {
            <div class="published-banner" role="status">
              <svg lucideGlobe class="banner-icon" aria-hidden="true"></svg>
              <span>
                Esta práctica está publicada: consigna y rúbrica se aplican al guardar. Si cambia la
                plantilla del entorno, use «Republicar entorno» para que los estudiantes nuevos la reciban.
              </span>
            </div>
          }

          @if (store.isIde() && (envTestJob() || envTestError()) && !isEnvTestRunning()) {
            <div
              class="env-test-strip"
              [class.is-failed]="envTestJob()?.status === 'failed' || !!envTestError()"
              [class.is-success]="envTestJob()?.status === 'success'"
              role="status"
            >
              <span>{{ envTestSummary() }}</span>
            </div>
          }

          @if (store.currentStep() > 1) {
            <div class="consult-bar">
              <button type="button" class="btn-secondary btn-sm" (click)="consultOpen.set(true)">
                <svg lucidePanelRightOpen class="btn-icon-sm" aria-hidden="true"></svg>
                {{ store.isIde() ? 'Ver consigna' : 'Ver enunciado' }}
              </button>
            </div>
          }

          <div class="editor-section-content">
            @if (store.isLoading()) {
              <div class="loading-state">
                <p>Cargando la práctica…</p>
              </div>
            } @else if (store.currentStep() === 1) {
              <step-identity />
            } @else if (store.isIde()) {
              @if (store.currentStep() === 2) {
                <section class="env-test-panel" aria-label="Prueba de entorno">
                  <div class="env-test-copy">
                    <span class="env-test-title">Prueba de entorno</span>
                    <p class="env-test-detail">
                      {{ isEnvTestRunning() ? envTestSummary() : (envTestJob()?.status === 'success' ? 'El entorno quedó listo con la configuración actual.' : 'Verifique el entorno antes de la clase.') }}
                    </p>
                  </div>
                  <button
                    type="button"
                    class="btn-secondary btn-sm"
                    [disabled]="isEnvTestRunning() || !store.exerciseId()"
                    (click)="runEnvTest()"
                  >
                    @if (isEnvTestRunning()) {
                      <svg lucideLoader2 class="btn-icon-sm motion-spin" aria-hidden="true"></svg>
                      Probando…
                    } @else {
                      Probar entorno
                    }
                  </button>
                </section>
              }
              <step-ide-environment />
            } @else if (store.currentStep() === 2) {
              <step-contract />
            } @else {
              <step-publication />
            }
          </div>
        </div>
      </div>

      <student-preview-modal
        [isOpen]="store.studentPreviewOpen()"
        (closed)="store.setStudentPreviewOpen(false)"
      />

      @if (showDeleteModal()) {
        <confirm-modal
          title="¿Eliminar borrador?"
          message="Se descartarán todos los datos y cambios de este ejercicio. Esta acción no se puede deshacer."
          confirmText="Sí, borrar borrador"
          cancelText="Cancelar"
          type="danger"
          (confirm)="onDeleteDraftConfirmed()"
          (cancel)="showDeleteModal.set(false)"
        />
      }

      @if (preflightOpen()) {
        <preflight-modal (closed)="preflightOpen.set(false)" (publish)="onPreflightPublish()" />
      }

      <consult-drawer
        [open]="consultOpen()"
        (closed)="consultOpen.set(false)"
        (editSection)="goToIdentity()"
      />

      @if (store.showPublishConfirmModal()) {
        <confirm-modal
          title="Publicar práctica"
          [message]="publishMessage()"
          confirmText="Publicar ahora"
          cancelText="Cancelar"
          type="primary"
          [isLoading]="store.isPublishing()"
          (confirm)="onPublishConfirmed()"
          (cancel)="store.showPublishConfirmModal.set(false)"
        />
      }

      @if (toastMessage(); as msg) {
        <div class="toast-notification animate-fade" role="status">
          <svg lucideCheck class="toast-icon" aria-hidden="true"></svg>
          <span>{{ msg }}</span>
        </div>
      }

      @if (aiModalOpen()) {
        <ai-exercise-generator-modal
          [defaultLanguage]="store.metadata().language || 'python'"
          [environmentType]="store.isIde() ? 'IDE_PERSISTENTE' : 'JUEZ_EFIMERO'"
          [templateId]="store.metadata().template_id || ''"
          [currentTitle]="store.metadata().title || ''"
          [currentTemplateName]="currentTemplateName()"
          [hasExistingContent]="hasExistingContent()"
          (closed)="aiModalOpen.set(false)"
          (applied)="onAiDraftApplied($event)"
        />
      }
    </div>
  `,
  styleUrl: './exercise-editor.component.scss'
})
export class ExerciseEditorComponent implements OnInit, OnDestroy {
  readonly store = inject(ExerciseEditorStore);
  private readonly host = inject<ElementRef<HTMLElement>>(ElementRef);
  private readonly practiceEnvTest = inject(PracticeEnvTestService);
  private readonly aiCopilotService = inject(TeacherAiCopilotService);
  private route = inject(ActivatedRoute);
  private router = inject(Router);

  courseId = input<string>('');
  id = input<string | null>(null);

  private readonly dashboardService = inject(TeacherDashboardService);
  private readonly courseService = inject(TeacherCourseService);

  readonly aiEnabled = signal(false);
  readonly aiModalOpen = signal(false);
  readonly unitTitle = signal<string>('');

  readonly showMoreActions = signal(false);
  readonly preflightOpen = signal(false);

  constructor() {
    // Cada sección arranca desde su inicio. Si se conserva el scroll de la
    // sección anterior, el primer bloque queda bajo la cabecera fija y la
    // página parece subir o bajar al cambiar de paso.
    effect(() => {
      this.store.currentStep();
      setTimeout(() => this.scrollEditorToTop());
    });
  }

  /** Alinea el inicio del editor con el borde superior de su contenedor de scroll. */
  private scrollEditorToTop(): void {
    const page = this.host.nativeElement.querySelector<HTMLElement>('.editor-page');
    page?.scrollIntoView?.({ block: 'start', inline: 'nearest' });
  }

  /** Nombre del curso resuelto desde el catálogo del panel docente. */
  readonly courseName = computed(() => {
    const cId = this.store.courseId();
    if (!cId) return '';
    return this.dashboardService.courses().find(c => c.id === cId)?.name || '';
  });

  readonly currentTemplateName = computed(() => {
    const meta = this.store.metadata();
    return meta.template?.name || meta.template_id || '';
  });

  readonly hasExistingContent = computed(() => {
    const s = this.store.statement();
    const hasStatement = !!s.description?.trim();
    const hasRubric = (this.store.rubric()?.criteria?.length ?? 0) > 0;
    const hasTitle = !!this.store.metadata().title?.trim();
    return hasStatement || hasRubric || hasTitle;
  });

  readonly examReasonValid = computed(() => {
    const meta = this.store.metadata();
    return meta.exam_network !== 'egress' || (meta.exam_network_reason || '').trim().length >= 10;
  });

  readonly envTestJob = signal<PracticeEnvTestJob | null>(null);
  readonly isEnvTestRunning = signal(false);
  readonly envTestError = signal<string | null>(null);
  private envTestPoll?: Subscription;

  readonly envTestSummary = computed(() => {
    const err = this.envTestError();
    if (err) {
      return err;
    }
    const job = this.envTestJob();
    if (!job) {
      return '';
    }
    switch (job.status) {
      case 'success':
        return 'Entorno probado correctamente: el estudiante puede abrirlo tal como está.';
      case 'failed':
        return `La prueba falló: ${job.error_message || 'revise el script de base de datos y el stack.'}`;
      case 'canceled':
        return 'La última prueba se canceló.';
      default:
        return `Probando el entorno… ${job.progress?.percent ?? 0} %`;
    }
  });

  readonly publishMessage = computed(() => {
    const title = this.store.metadata().title || 'este ejercicio';
    const base = `¿Confirma publicar «${title}»? Quedará visible de inmediato para los estudiantes inscritos en el curso.`;
    if (!this.store.isIde()) {
      return base;
    }
    const job = this.envTestJob();
    if (this.isEnvTestRunning()) {
      return `${base} Hay una prueba de entorno en curso.`;
    }
    if (!job) {
      return `${base} Todavía no se probó el entorno.`;
    }
    if (job.status === 'success') {
      return `${base} La última prueba de entorno salió bien.`;
    }
    if (job.status === 'failed') {
      return `${base} La última prueba de entorno falló: ${job.error_message || 'sin detalle'}. Puede publicar igual.`;
    }
    return base;
  });

  showDeleteModal = signal<boolean>(false);
  consultOpen = signal<boolean>(false);
  toastMessage = signal<string | null>(null);
  private toastTimer: ReturnType<typeof setTimeout> | null = null;

  /** Abre el preflight si hay bloqueos o avisos sin aceptar. */
  requestPublish(): void {
    this.store.refreshChecklist();
    const warnings = this.store.checklistReport()?.warnings ?? [];
    const hasBlockers = this.store.missingPublishRequirements().length > 0 || !this.examReasonValid();
    if (hasBlockers || (warnings.length > 0 && !this.store.acceptedWarnings())) {
      this.preflightOpen.set(true);
      return;
    }
    this.store.showPublishConfirmModal.set(true);
  }

  openPreflightModal(): void {
    this.store.refreshChecklist();
    this.preflightOpen.set(true);
  }

  onPreflightPublish(): void {
    this.store.setAcceptedWarnings(true);
    this.preflightOpen.set(false);
    this.onPublishConfirmed();
  }

  goToIdentity(): void {
    this.consultOpen.set(false);
    this.store.setStep(1);
  }

  saveNow(): void {
    void this.store.saveDraft();
  }

  setEnvironment(environment: 'IDE_PERSISTENTE' | 'JUEZ_EFIMERO'): void {
    this.store.updateMetadata({
      environment_type: environment,
      modality: environment === 'IDE_PERSISTENTE' ? 'environment' : 'judge',
      ...(environment === 'JUEZ_EFIMERO' ? {} : { per_student_seed: false })
    });
  }

  setPurpose(purpose: 'class' | 'exam'): void {
    const keepsSeed = purpose === 'exam' && !this.store.isIde();
    this.store.updateMetadata({
      purpose,
      ...(keepsSeed ? {} : { per_student_seed: false })
    });
  }

  setExamNetwork(network: 'none' | 'egress'): void {
    this.store.updateMetadata({
      exam_network: network,
      ...(network === 'none' ? { exam_network_reason: '' } : {})
    });
  }

  onExamReasonChange(reason: string): void {
    this.store.updateMetadata({ exam_network_reason: reason });
  }

  toggleAntiCopy(event: Event): void {
    this.store.updateMetadata({ per_student_seed: (event.target as HTMLInputElement).checked });
  }

  confirmDeleteDraft() {
    this.showDeleteModal.set(true);
  }

  onDeleteDraftConfirmed() {
    this.showDeleteModal.set(false);
    this.store.resetDraft();
    this.showToast('Borrador restablecido correctamente.');
  }

  onPublishConfirmed() {
    this.store.executePublish(
      () => {
        this.showToast('Ejercicio publicado.');
      },
      () => {
        this.showToast(this.store.publishError() || 'No se pudo publicar el ejercicio.');
      }
    );
  }

  onAiDraftApplied(draft: AIGenerateExerciseResponse): void {
    this.store.applyAIGeneratedDraft(draft);
    this.aiModalOpen.set(false);
    this.showToast('Borrador generado por IA aplicado al editor.');
  }

  async runEnvTest(): Promise<void> {
    const exerciseId = this.store.exerciseId();
    const courseId = this.store.courseId();
    if (!exerciseId || !courseId || this.isEnvTestRunning()) {
      return;
    }

    this.envTestError.set(null);
    this.isEnvTestRunning.set(true);

    if (this.store.dirty()) {
      await this.store.saveDraft();
    }

    this.practiceEnvTest.start(courseId, exerciseId).subscribe({
      next: job => {
        this.envTestJob.set(job);
        this.pollEnvTest(job.id);
      },
      error: err => {
        this.isEnvTestRunning.set(false);
        this.envTestError.set(
          err?.message || err?.error?.message || 'No se pudo iniciar la prueba de entorno.'
        );
      }
    });
  }

  private loadLatestEnvTest(exerciseId: string): void {
    this.practiceEnvTest
      .latest(exerciseId)
      .pipe(catchError(() => of(null)))
      .subscribe(job => {
        if (!job) {
          return;
        }
        this.envTestJob.set(job);
        if (!this.practiceEnvTest.isTerminal(job.status)) {
          this.pollEnvTest(job.id);
        }
      });
  }

  private pollEnvTest(jobId: string): void {
    this.stopEnvTestPoll();
    this.isEnvTestRunning.set(true);
    this.envTestPoll = this.practiceEnvTest.pollJob(jobId).subscribe({
      next: job => {
        this.envTestJob.set(job);
        if (this.practiceEnvTest.isTerminal(job.status)) {
          this.isEnvTestRunning.set(false);
        }
      },
      error: () => {
        this.isEnvTestRunning.set(false);
        this.envTestError.set('Se perdió el estado de la prueba de entorno.');
      }
    });
  }

  private stopEnvTestPoll(): void {
    this.envTestPoll?.unsubscribe();
    this.envTestPoll = undefined;
  }

  ngOnDestroy(): void {
    this.stopEnvTestPoll();
    if (this.toastTimer) {
      clearTimeout(this.toastTimer);
    }
  }

  private showToast(message: string) {
    if (this.toastTimer) {
      clearTimeout(this.toastTimer);
    }
    this.toastMessage.set(message);
    this.toastTimer = setTimeout(() => {
      this.toastMessage.set(null);
    }, 3000);
  }

  ngOnInit() {
    const snapshot = this.route.snapshot;
    const cId = this.courseId() || snapshot.paramMap.get('courseId') || '';
    const exId = this.id() || snapshot.paramMap.get('id') || null;
    const tplId = snapshot.queryParamMap.get('template_id') || '';
    const envType = snapshot.queryParamMap.get('environment_type') || '';
    const moduleId = snapshot.queryParamMap.get('module_id') || '';

    console.info('[AuditLog] exercise_editor_opened', {
      editor_type: 'new_editor',
      exercise_id: exId,
      course_id: cId
    });

    if (cId) {
      this.store.setCourseId(cId);
    }

    if (moduleId) {
      this.store.setModuleId(moduleId);
      this.courseService.getCourseModules(cId).subscribe({
        next: (mods) => {
          const module = mods.find((m) => m.id === moduleId);
          if (module) {
            this.unitTitle.set(`Unidad ${module.order_index} · ${module.title}`);
          }
        },
        error: () => {}
      });
    }

    if (exId) {
      if (this.store.exerciseId() !== exId) {
        this.store.loadExercise(exId, cId);
      }
      this.loadLatestEnvTest(exId);
    } else {
      this.store.resetToNew(cId);
      this.store.preselectTemplate(tplId, envType);
    }

    this.aiCopilotService.getStatus().subscribe({
      next: (s) => this.aiEnabled.set(!!s?.enabled),
      error: () => this.aiEnabled.set(false)
    });

    this.store.setStep(1);
  }
}
