import { Component, OnInit, inject, input, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink, ActivatedRoute, Router } from '@angular/router';
import {
  LucideArrowLeft,
  LucideChevronRight,
  LucideChevronLeft,
  LucideCheck,
  LucideEye,
  LucideTrash2,
  LucideGlobe,
  LucideLoader2
} from '@lucide/angular';
import { ExerciseEditorStore } from './exercise-editor.store';
import { StepperNavComponent } from './shared/stepper-nav.component';
import { SummaryPanelComponent } from './shared/summary-panel.component';
import { StepIdentityComponent } from './steps/step-identity/step-identity.component';
import { StepContractComponent } from './steps/step-contract/step-contract.component';
import { StepIdeEnvironmentComponent } from './steps/step-ide-environment/step-ide-environment.component';
import { StepPublicationComponent } from './steps/step-publication/step-publication.component';
import { StudentPreviewModalComponent } from './student-preview-modal/student-preview-modal.component';
import { ConfirmModalComponent } from '@shared/components/confirm-modal/confirm-modal.component';

@Component({
  selector: 'exercise-editor',
  standalone: true,
  imports: [
    CommonModule,
    RouterLink,
    LucideArrowLeft,
    LucideChevronRight,
    LucideChevronLeft,
    LucideCheck,
    LucideEye,
    LucideTrash2,
    LucideGlobe,
    LucideLoader2,
    StepperNavComponent,
    SummaryPanelComponent,
    StepIdentityComponent,
    StepContractComponent,
    StepIdeEnvironmentComponent,
    StepPublicationComponent,
    StudentPreviewModalComponent,
    ConfirmModalComponent
  ],
  template: `
    <div class="exercise-editor-page">
      <!-- Top Navigation & Actions Bar -->
      <header class="editor-topbar">
        <div class="topbar-left">
          <a [routerLink]="['/teacher/cursos', courseId()]" class="back-link">
            <svg lucideArrowLeft class="back-icon"></svg>
            <span>Volver al curso</span>
          </a>

          <div class="title-meta">
            <h1 class="editor-heading">
              {{ store.metadata().title || (store.exerciseId() ? 'Editar Ejercicio' : 'Nuevo Ejercicio') }}
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
            class="btn-outline delete-draft-btn"
            (click)="confirmDeleteDraft()"
            title="Descartar y borrar los cambios no guardados del borrador"
          >
            <svg lucideTrash2 class="btn-icon"></svg>
            <span>Eliminar borrador</span>
          </button>

          <!-- Botones de Navegación de Pasos -->
          @if (store.currentStep() > 1) {
            <button
              type="button"
              class="btn-outline step-btn"
              (click)="prevStep()"
            >
              <svg lucideChevronLeft class="btn-icon"></svg>
              <span>Anterior</span>
            </button>
          }

          @if (store.currentStep() < 3) {
            <button
              type="button"
              class="btn-primary step-btn"
              (click)="nextStep()"
            >
              <span>Siguiente</span>
              <svg lucideChevronRight class="btn-icon"></svg>
            </button>
          } @else {
            @if (store.status() === 'published') {
              <div class="published-pill">
                <svg lucideCheck class="btn-icon"></svg>
                <span>Publicado</span>
              </div>
            } @else {
              <button
                type="button"
                class="btn-primary btn-publish-step"
                [disabled]="!store.canPublish() || store.isPublishing()"
                (click)="store.showPublishConfirmModal.set(true)"
                [title]="store.canPublish() ? 'Publicar ejercicio para los estudiantes' : 'Resuelve los requisitos bloqueantes para publicar'"
              >
                @if (store.isPublishing()) {
                  <svg lucideLoader2 class="btn-icon spin-icon"></svg>
                  <span>Publicando...</span>
                } @else {
                  <svg lucideGlobe class="btn-icon"></svg>
                  <span>Publicar</span>
                }
              </button>
            }
          }
        </div>
      </header>

      <!-- Main 3-Column Layout -->
      <main class="editor-columns-grid">
        <!-- Columna 1: Stepper Fijo (240px) -->
        <aside class="left-column">
          <stepper-nav />
        </aside>

        <!-- Columna 2: Contenido del Paso Activo (Flex) -->
        <section class="center-column">
          @if (store.isLoading()) {
            <div class="loading-state">
              <p>Cargando datos del ejercicio...</p>
            </div>
          } @else {
            @switch (store.currentStep()) {
              @case (1) {
                <step-identity />
              }
              @case (2) {
                @if (store.metadata().environment_type === 'IDE_PERSISTENTE') {
                  <step-ide-environment />
                } @else {
                  <step-contract />
                }
              }
              @case (3) {
                <step-publication />
              }
            }
          }
        </section>

        <!-- Columna 3: Resumen (320px) -->
        <aside class="right-column">
          <summary-panel />
        </aside>
      </main>

      <!-- Modal de Vista Previa del Estudiante -->
      <student-preview-modal
        [isOpen]="store.studentPreviewOpen()"
        (close)="store.setStudentPreviewOpen(false)"
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

      <!-- Modal de Confirmación para Publicar Ejercicio -->
      @if (store.showPublishConfirmModal()) {
        <confirm-modal
          title="Publicar Ejercicio"
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
    </div>
  `,
  styleUrl: './exercise-editor.component.scss'
})
export class ExerciseEditorComponent implements OnInit {
  readonly store = inject(ExerciseEditorStore);
  private route = inject(ActivatedRoute);
  private router = inject(Router);

  courseId = input<string>('');
  id = input<string | null>(null);

  showDeleteModal = signal<boolean>(false);
  toastMessage = signal<string | null>(null);
  private toastTimer: any = null;

  prevStep() {
    const current = this.store.currentStep();
    if (current > 1) {
      this.store.setStep((current - 1) as 1 | 2 | 3);
    }
  }

  nextStep() {
    this.store.attemptNextStep();
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
    const cId = this.courseId() || this.route.snapshot.paramMap.get('courseId') || '';
    const exId = this.id() || this.route.snapshot.paramMap.get('id') || null;

    // Tracking de telemetría de uso
    console.info('[AuditLog] exercise_editor_opened', {
      editor_type: 'new_editor',
      exercise_id: exId,
      course_id: cId
    });

    if (cId) {
      this.store.setCourseId(cId);
    }

    if (exId) {
      this.store.loadExercise(exId, cId);
    } else {
      // Si es nuevo ejercicio, recuperar borrador local persistido automáticamente si existe
      this.store.restoreDraftFromStorage();
    }
  }
}
