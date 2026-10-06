import { Component, OnInit, inject, input, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink, ActivatedRoute } from '@angular/router';
import { 
  LucideArrowLeft, 
  LucideSave, 
  LucideChevronRight, 
  LucideChevronLeft, 
  LucideCheck,
  LucideEye
} from '@lucide/angular';
import { ExerciseEditorStore } from './exercise-editor.store';
import { StepperNavComponent } from './shared/stepper-nav.component';
import { SummaryPanelComponent } from './shared/summary-panel.component';
import { StepIdentityComponent } from './steps/step-identity/step-identity.component';
import { StepContractComponent } from './steps/step-contract/step-contract.component';
import { StepPublicationComponent } from './steps/step-publication/step-publication.component';
import { StudentPreviewModalComponent } from './student-preview-modal/student-preview-modal.component';

@Component({
  selector: 'exercise-editor',
  standalone: true,
  imports: [
    CommonModule,
    RouterLink,
    LucideArrowLeft,
    LucideSave,
    LucideChevronRight,
    LucideChevronLeft,
    LucideCheck,
    LucideEye,
    StepperNavComponent,
    SummaryPanelComponent,
    StepIdentityComponent,
    StepContractComponent,
    StepPublicationComponent,
    StudentPreviewModalComponent
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
            (click)="openStudentPreview()"
            title="Ver exactamente cómo verá el estudiante el enunciado y editor"
          >
            <svg lucideEye class="btn-icon"></svg>
            <span>Vista previa</span>
          </button>

          <button 
            type="button" 
            class="btn-outline draft-btn"
            [disabled]="store.isSaving()"
            (click)="store.saveDraft()"
          >
            <svg lucideSave class="btn-icon"></svg>
            <span>{{ store.isSaving() ? 'Guardando...' : 'Guardar borrador' }}</span>
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
                <step-contract />
              }
              @case (3) {
                <step-publication />
              }
            }
          }
        </section>

        <!-- Columna 3: Resumen en Vivo (320px) -->
        <aside class="right-column">
          <summary-panel />
        </aside>
      </main>

      <!-- Modal de Vista Previa del Estudiante -->
      <student-preview-modal
        [isOpen]="isPreviewModalOpen()"
        (close)="closeStudentPreview()"
      />
    </div>
  `,
  styleUrl: './exercise-editor.component.scss'
})
export class ExerciseEditorComponent implements OnInit {
  readonly store = inject(ExerciseEditorStore);
  private route = inject(ActivatedRoute);

  courseId = input<string>('');
  id = input<string | null>(null);

  readonly isPreviewModalOpen = signal<boolean>(false);

  openStudentPreview(): void {
    this.isPreviewModalOpen.set(true);
  }

  closeStudentPreview(): void {
    this.isPreviewModalOpen.set(false);
  }

  prevStep() {
    const current = this.store.currentStep();
    if (current > 1) {
      this.store.setStep((current - 1) as 1 | 2 | 3);
    }
  }

  nextStep() {
    const current = this.store.currentStep();
    if (current < 3) {
      this.store.setStep((current + 1) as 1 | 2 | 3);
    }
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
    }
  }
}
