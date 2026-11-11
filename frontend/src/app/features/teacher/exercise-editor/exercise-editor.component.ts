import { Component, OnInit, computed, inject, input, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink, ActivatedRoute, Router } from '@angular/router';
import {
  LucideArrowLeft,
  LucideCheck,
  LucideEye,
  LucideTrash2,
  LucideGlobe,
  LucideLoader2,
  LucideSparkles
} from '@lucide/angular';
import { ExerciseEditorStore } from './exercise-editor.store';
import { StepIdentityComponent } from './steps/step-identity/step-identity.component';
import { StepContractComponent } from './steps/step-contract/step-contract.component';
import { StepIdeEnvironmentComponent } from './steps/step-ide-environment/step-ide-environment.component';
import { StepPublicationComponent } from './steps/step-publication/step-publication.component';
import { StudentPreviewModalComponent } from './student-preview-modal/student-preview-modal.component';
import { AiExerciseGeneratorModalComponent } from './ai-exercise-generator-modal/ai-exercise-generator-modal.component';
import { ConfirmModalComponent } from '@shared/components/confirm-modal/confirm-modal.component';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { TeacherAiCopilotService, AIGenerateExerciseResponse } from '../services/teacher-ai-copilot.service';
import { TeacherDashboardService } from '../services/teacher-dashboard.service';
import { TeacherCourseService } from '../services/teacher-course.service';

@Component({
  selector: 'exercise-editor',
  standalone: true,
  imports: [
    CommonModule,
    RouterLink,
    LucideArrowLeft,
    LucideCheck,
    LucideEye,
    LucideTrash2,
    LucideGlobe,
    LucideLoader2,
    LucideSparkles,
    StepIdentityComponent,
    StepContractComponent,
    StepIdeEnvironmentComponent,
    StepPublicationComponent,
    StudentPreviewModalComponent,
    AiExerciseGeneratorModalComponent,
    ConfirmModalComponent,
    ModalShellComponent
  ],
  template: `
    <div class="exercise-editor-page">
      <!-- Top Navigation & Actions Bar -->
      <header class="editor-topbar">
        <div class="topbar-left">
          <a [routerLink]="['/teacher/cursos', store.courseId()]" class="back-link">
            <svg lucideArrowLeft class="back-icon"></svg>
            <span>Volver al curso{{ courseName() ? ' · ' + courseName() : '' }}</span>
          </a>

          <div class="title-meta">
            <h1 class="editor-heading">
              {{ store.metadata().title || (store.exerciseId() ? 'Editar práctica' : 'Nueva práctica') }}
            </h1>
            <div class="meta-badges">
              <span class="status-badge" [class.published]="store.status() === 'published'">
                {{ store.status() === 'published' ? 'Publicado' : 'Borrador' }}
              </span>

              @if (store.dirty()) {
                <span class="dirty-badge">
                  <span class="dirty-dot"></span>
                  <span>Cambios sin guardar</span>
                </span>
              } @else if (store.lastSavedAt()) {
                <span class="saved-badge">
                  <svg lucideCheck class="saved-icon"></svg>
                  <span>Guardado</span>
                </span>
              }
            </div>

            @if (unitTitle()) {
              <span class="status-badge unit-badge">{{ unitTitle() }}</span>
            }
          </div>
        </div>

        <div class="topbar-actions">
          <button
            type="button"
            class="btn-outline preview-btn"
            (click)="store.setStudentPreviewOpen(true, 'preview')"
            title="Ver exactamente cómo verá el estudiante el enunciado y editor"
          >
            <svg lucideEye class="btn-icon"></svg>
            <span>Vista previa</span>
          </button>

          <button
            type="button"
            class="btn-outline ai-copilot-btn"
            (click)="aiModalOpen.set(true)"
            title="Generar propuesta pedagógica completa con el Copiloto IA"
          >
            <svg lucideSparkles class="btn-icon"></svg>
            <span>Generar con IA</span>
          </button>

          <button
            type="button"
            class="btn-outline delete-draft-btn"
            (click)="confirmDeleteDraft()"
            title="Descartar y borrar los cambios no guardados del borrador"
          >
            <svg lucideTrash2 class="btn-icon"></svg>
            <span>Eliminar borrador</span>
          </button>

          @if (store.status() === 'published') {
            <div class="published-pill">
              <svg lucideCheck class="btn-icon"></svg>
              <span>Publicada</span>
            </div>
          } @else {
            <button
              type="button"
              class="btn-primary btn-publish-step"
              [disabled]="store.isPublishing()"
              (click)="onPublishClick()"
              title="Publicar la práctica para los estudiantes"
            >
              @if (store.isPublishing()) {
                <svg lucideLoader2 class="btn-icon motion-spin"></svg>
                <span>Publicando...</span>
              } @else {
                <svg lucideGlobe class="btn-icon"></svg>
                <span>Publicar</span>
              }
            </button>
          }
        </div>
      </header>

      <!-- Lienzo único: la práctica se completa en una sola pasada -->
      <main class="editor-columns-grid">
        <section class="center-column">
          @if (store.isLoading()) {
            <div class="loading-state">
              <p>Cargando la práctica...</p>
            </div>
          } @else {
            <step-identity />

            @if (store.metadata().environment_type === 'IDE_PERSISTENTE') {
              <step-ide-environment />
            } @else {
              <step-contract />
              <step-publication />
            }
          }
        </section>
      </main>

      <!-- Modal de Vista Previa del Estudiante -->
      <student-preview-modal
        [isOpen]="store.studentPreviewOpen()"
        (closed)="store.setStudentPreviewOpen(false)"
      />

      <!-- Modal de Confirmación para Borrar Borrador -->
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

      <!-- Modal Informativa de Requisitos para Publicar -->
      @if (showValidationModal()) {
        <modal-shell animate.leave="motion-modal-out"
          title="Requisitos para publicar"
          subtitle="Completá los siguientes aspectos antes de publicar la práctica:"
          maxWidth="520px"
          (closed)="showValidationModal.set(false)"
        >
          <div class="missing-reqs-content">
            <ul class="missing-reqs-list" style="margin: 0; padding: 0; list-style: none; display: flex; flex-direction: column; gap: var(--space-2);">
              @for (req of store.missingPublishRequirements(); track $index) {
                <li class="missing-req-item" style="display: flex; align-items: flex-start; gap: var(--space-2); font-size: var(--text-xs); color: var(--text-secondary);">
                  <span class="bullet-dot" style="color: var(--color-warning); font-weight: bold;">•</span>
                  <span>{{ req }}</span>
                </li>
              }
            </ul>

            <div class="modal-shell-actions" style="margin-top: var(--space-4); display: flex; justify-content: flex-end;">
              <button type="button" class="btn-primary" (click)="showValidationModal.set(false)">
                Entendido, voy a completarlo
              </button>
            </div>
          </div>
        </modal-shell>
      }

      <!-- Modal de Confirmación para Publicar la Práctica -->
      @if (store.showPublishConfirmModal()) {
        <confirm-modal
          title="Publicar práctica"
          [message]="'¿Confirmas publicar «' + (store.metadata().title || 'este ejercicio') + '»? Quedará visible inmediatamente para todos los estudiantes inscritos en el curso.'"
          confirmText="Publicar ahora"
          cancelText="Cancelar"
          type="primary"
          [isLoading]="store.isPublishing()"
          (confirm)="onPublishConfirmed()"
          (cancel)="store.showPublishConfirmModal.set(false)"
        />
      }

      <!-- Toast de Confirmación -->
      @if (toastMessage(); as msg) {
        <div class="toast-notification animate-fade" role="status">
          <svg lucideCheck class="toast-icon"></svg>
          <span>{{ msg }}</span>
        </div>
      }

      <!-- Modal del Copiloto IA -->
      @if (aiModalOpen()) {
        <ai-exercise-generator-modal
          [defaultLanguage]="store.metadata().language || 'python'"
          [environmentType]="store.isIde() ? 'IDE_PERSISTENTE' : 'JUEZ_EFIMERO'"
          [templateId]="store.metadata().template_id || ''"
          (closed)="aiModalOpen.set(false)"
          (applied)="onAiDraftApplied($event)"
        />
      }
    </div>
  `,
  styleUrl: './exercise-editor.component.scss'
})
export class ExerciseEditorComponent implements OnInit {
  readonly store = inject(ExerciseEditorStore);
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

  /** Nombre del curso resuelto desde el catálogo del panel docente. */
  readonly courseName = computed(() => {
    const cId = this.store.courseId();
    if (!cId) return '';
    return this.dashboardService.courses().find(c => c.id === cId)?.name || '';
  });

  showDeleteModal = signal<boolean>(false);
  showValidationModal = signal<boolean>(false);
  toastMessage = signal<string | null>(null);
  private toastTimer: ReturnType<typeof setTimeout> | null = null;

  onPublishClick(): void {
    if (this.store.missingPublishRequirements().length > 0) {
      this.showValidationModal.set(true);
    } else {
      this.store.showPublishConfirmModal.set(true);
    }
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
        this.showToast('¡Ejercicio publicado con éxito!');
      },
      () => {
        this.showToast(this.store.publishError() || 'Error al publicar el ejercicio.');
      }
    );
  }

  onAiDraftApplied(draft: AIGenerateExerciseResponse): void {
    this.store.applyAIGeneratedDraft(draft);
    this.aiModalOpen.set(false);
    this.showToast('Borrador generado por IA aplicado al editor.');
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

    // Tracking de telemetría de uso
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
    } else {
      // Al ingresar a crear nuevo laboratorio, iniciar con formulario completamente limpio
      this.store.resetToNew(cId);
      // Preselección desde el Catálogo de Entornos ("Usar en Laboratorio")
      this.store.preselectTemplate(tplId, envType);
    }

    this.aiCopilotService.getStatus().subscribe({
      next: (s) => this.aiEnabled.set(!!s?.enabled),
      error: () => this.aiEnabled.set(false)
    });

    // El lienzo muestra la auditoría, no una validación por paso.
    this.store.setStep(3);
  }
}
