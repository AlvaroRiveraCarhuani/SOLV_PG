import { Component, OnInit, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { 
  LucideServer, 
  LucidePlus, 
  LucideDatabase, 
  LucideCalendar,
  LucideClipboardList,
  LucideTrash2,
  LucideSparkles,
  LucideAlertTriangle,
  LucideCheckCircle2
} from '@lucide/angular';
import { ExerciseEditorStore, RubricCriterionDTO, RubricLevelDTO } from '../../exercise-editor.store';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { TemplateRequestModalComponent } from '../../../templates/template-request-modal/template-request-modal.component';
import { RUBRIC_PRESETS } from '@shared/rubric-presets';

export interface ApprovedTemplate {
  id: string;
  name: string;
  docker_image: string;
  base_ram_mb: number;
  target_environment: string;
  status: string;
  services_config?: {
    database?: {
      enabled: boolean;
      engine?: string;
    };
    [key: string]: any;
  };
}

@Component({
  selector: 'step-ide-environment',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideServer,
    LucidePlus,
    LucideDatabase,
    LucideCalendar,
    LucideClipboardList,
    LucideTrash2,
    LucideSparkles,
    LucideAlertTriangle,
    LucideCheckCircle2,
    MachineDataDirective,
    TemplateRequestModalComponent
  ],
  template: `
    <div class="ide-step-container">
      <header class="step-header">
        <h2 class="step-title">Entorno de Ejecución y Entrega</h2>
        <p class="step-subtitle">
          Configura la plantilla de desarrollo interactivo (OpenVSCode) y las reglas de entrega para los estudiantes.
        </p>
      </header>

      <!-- Ficha de Selección de Plantilla del Catálogo -->
      <section class="card-section">
        <div class="section-header-row">
          <div class="title-with-icon">
            <svg lucideServer class="header-icon"></svg>
            <h3 class="card-title">Plantilla del Catálogo Homologado</h3>
          </div>
          <button type="button" class="btn-outline btn-sm" (click)="openRequestModal()">
            <svg lucidePlus class="btn-icon"></svg>
            <span>Solicitar nueva plantilla</span>
          </button>
        </div>

        <div class="form-group">
          <label for="template-select" class="form-label">
            Seleccionar plantilla aprobada <span class="required">*</span>
          </label>
          <select
            id="template-select"
            class="form-select"
            [ngModel]="store.metadata().template_id"
            (ngModelChange)="onTemplateSelected($event)"
          >
            <option [ngValue]="null">-- Seleccionar plantilla homologada --</option>
            @for (t of approvedTemplates(); track t.id) {
              <option [value]="t.id">
                {{ t.name }} ({{ t.docker_image }} — {{ t.base_ram_mb }} MB RAM)
              </option>
            }
          </select>
        </div>

        <!-- Ficha de la Plantilla Seleccionada -->
        @if (selectedTemplate(); as tpl) {
          <div class="template-detail-card">
            <div class="detail-row">
              <span class="detail-label">Nombre de la plantilla:</span>
              <span class="detail-value font-bold">{{ tpl.name }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">Imagen Docker:</span>
              <code class="detail-code" machineData>{{ tpl.docker_image }}</code>
            </div>
            <div class="detail-row">
              <span class="detail-label">Memoria RAM Asignada:</span>
              <span class="detail-chip" machineData>{{ tpl.base_ram_mb }} MB (Heredada del Admin)</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">Servicios Satélite:</span>
              <div class="satellites-list">
                @if (tpl.services_config?.database?.enabled) {
                  <span class="satellite-badge active">
                    <svg lucideDatabase class="badge-icon"></svg>
                    <span>Base de datos ({{ tpl.services_config?.database?.engine || 'PostgreSQL' }})</span>
                  </span>
                } @else {
                  <span class="satellite-badge neutral">Sin base de datos satélite</span>
                }
              </div>
            </div>
          </div>
        } @else {
          <div class="empty-template-hint">
            <span>Selecciona una plantilla del catálogo oficial para desplegar el entorno OpenVSCode.</span>
          </div>
        }
      </section>

      <!-- Script DDL/DML de Inicialización (Condicional a BD Activa) -->
      @if (hasDatabaseService()) {
        <section class="card-section">
          <div class="title-with-icon">
            <svg lucideDatabase class="header-icon"></svg>
            <h3 class="card-title">Script de Inicialización de Base de Datos (DDL / DML)</h3>
          </div>
          <p class="section-desc">
            Este script SQL se ejecutará automáticamente durante el aprovisionamiento de la base de datos satélite del laboratorio.
          </p>

          <div class="form-group">
            <textarea
              class="form-textarea font-mono"
              rows="8"
              placeholder="CREATE TABLE usuarios (id SERIAL PRIMARY KEY, nombre VARCHAR(100));&#10;INSERT INTO usuarios (nombre) VALUES ('Admin');"
              [ngModel]="store.metadata().db_init_script"
              (ngModelChange)="onInitScriptChange($event)"
            ></textarea>
            <span class="field-hint">Sentencias SQL separadas por punto y coma (;)</span>
          </div>
        </section>
      }

      <!-- Configuración de Entrega -->
      <section class="card-section">
        <div class="title-with-icon">
          <svg lucideCalendar class="header-icon"></svg>
          <h3 class="card-title">Parámetros de Entrega y Checkpoint</h3>
        </div>

        <div class="delivery-grid">
          <div class="form-group">
            <label for="ide-due-date" class="form-label">Fecha y hora límite de entrega</label>
            <input
              id="ide-due-date"
              type="datetime-local"
              class="form-input"
              [ngModel]="store.metadata().due_date"
              (ngModelChange)="onDueDateChange($event)"
            />
          </div>

          <div class="form-group checkbox-group">
            <label class="checkbox-label">
              <input
                type="checkbox"
                [checked]="store.metadata().allow_resubmit"
                (change)="onAllowResubmitChange($event)"
              />
              <span>Permitir re-envíos después de la primera entrega</span>
            </label>
            <span class="field-hint">Permite al estudiante re-entregar solución hasta la fecha límite</span>
          </div>

          <div class="form-group checkbox-group">
            <label class="checkbox-label">
              <input
                type="checkbox"
                [checked]="store.metadata().auto_checkpoint"
                (change)="onAutoCheckpointChange($event)"
              />
              <span>Checkpoint automático cada 5 minutos</span>
            </label>
            <span class="field-hint">Guarda copia del espacio de trabajo periódicamente en segundo plano</span>
          </div>
        </div>
      </section>

      <!-- Rúbrica de Evaluación Cualitativa -->
      <section class="card-section rubric-section">
        <div class="section-header-row">
          <div class="title-with-icon">
            <svg lucideClipboardList class="header-icon"></svg>
            <h3 class="card-title">Rúbrica de Evaluación Cualitativa</h3>
          </div>
          <div class="rubric-header-actions">
            <select class="form-select preset-select" #presetSelect (change)="applyRubricPreset(presetSelect.value); presetSelect.value = ''">
              <option value="">-- Usar plantilla de rúbrica --</option>
              <option value="api-rest">Plantilla: API REST Backend</option>
              <option value="full-stack">Plantilla: Aplicación Full-Stack</option>
              <option value="proyecto-integrador">Plantilla: Proyecto Integrador</option>
            </select>
            <button type="button" class="btn-primary btn-sm" (click)="addCriterion()">
              <svg lucidePlus class="btn-icon"></svg>
              <span>Agregar criterio</span>
            </button>
          </div>
        </div>

        <p class="section-desc">
          Define los criterios de evaluación cualitativa y sus niveles de logro. La suma de pesos de los criterios debe ser exactamente 100%.
        </p>

        <!-- Banner de Ponderación Total -->
        <div class="weight-total-banner" [class.valid]="totalRubricWeight() === 100" [class.invalid]="totalRubricWeight() !== 100">
          @if (totalRubricWeight() === 100) {
            <svg lucideCheckCircle2 class="banner-icon icon-success"></svg>
            <span>Ponderación total balanceada: <strong machineData>100%</strong></span>
          } @else {
            <svg lucideAlertTriangle class="banner-icon icon-danger"></svg>
            <span>Suma de pesos actual: <strong machineData>{{ totalRubricWeight() }}%</strong>. La suma debe ser exactamente 100%.</span>
          }
        </div>

        @if (!store.rubric() || store.rubric()?.criteria?.length === 0) {
          <div class="empty-rubric-box">
            <svg lucideClipboardList class="empty-icon"></svg>
            <p>No se han agregado criterios a la rúbrica.</p>
            <button type="button" class="btn-outline btn-sm" (click)="applyRubricPreset('api-rest')">
              <svg lucideSparkles class="btn-icon"></svg>
              <span>Cargar Plantilla API REST</span>
            </button>
          </div>
        } @else {
          <div class="criteria-list">
            @for (criterion of store.rubric()?.criteria; track $index; let i = $index) {
              <div class="criterion-card">
                <div class="criterion-card-header">
                  <span class="criterion-badge font-mono" machineData>Criterio {{ i + 1 }}</span>
                  <button type="button" class="btn-icon-danger" (click)="removeCriterion(i)" title="Eliminar criterio">
                    <svg lucideTrash2 class="btn-icon-sm"></svg>
                  </button>
                </div>

                <div class="criterion-form-grid">
                  <div class="form-group flex-2">
                    <label class="form-label">Nombre del Criterio</label>
                    <input
                      type="text"
                      class="form-input"
                      placeholder="Ej: Arquitectura de carpetas, Endpoints REST..."
                      [ngModel]="criterion.name"
                      (ngModelChange)="updateCriterion(i, { name: $event })"
                    />
                  </div>

                  <div class="form-group flex-1">
                    <label class="form-label">Peso (%)</label>
                    <input
                      type="number"
                      class="form-input font-mono"
                      min="0"
                      max="100"
                      [ngModel]="criterion.weight"
                      (ngModelChange)="updateCriterion(i, { weight: +$event })"
                    />
                  </div>
                </div>

                <div class="form-group">
                  <label class="form-label">Descripción del Criterio</label>
                  <textarea
                    class="form-textarea"
                    rows="2"
                    placeholder="Descripción detallada de lo que se evaluará en este criterio..."
                    [ngModel]="criterion.description"
                    (ngModelChange)="updateCriterion(i, { description: $event })"
                  ></textarea>
                </div>

                <!-- Niveles de logro -->
                <div class="levels-container">
                  <div class="levels-header-row">
                    <span class="levels-title">Niveles de Logro</span>
                    <button type="button" class="btn-ghost btn-xs" (click)="addLevelToCriterion(i)">
                      <svg lucidePlus class="btn-icon-xs"></svg>
                      <span>Agregar nivel</span>
                    </button>
                  </div>

                  <div class="levels-list">
                    @for (level of criterion.levels; track $index; let j = $index) {
                      <div class="level-card">
                        <div class="level-inputs-row">
                          <input
                            type="text"
                            class="form-input level-name-input"
                            placeholder="Ej: Excelente, Bueno..."
                            [ngModel]="level.name"
                            (ngModelChange)="updateLevel(i, j, { name: $event })"
                          />
                          <input
                            type="number"
                            class="form-input font-mono level-score-input"
                            placeholder="Score (0-100)"
                            min="0"
                            max="100"
                            [ngModel]="level.score"
                            (ngModelChange)="updateLevel(i, j, { score: +$event })"
                          />
                          <button type="button" class="btn-icon-danger" (click)="removeLevelFromCriterion(i, j)" title="Eliminar nivel">
                            <svg lucideTrash2 class="btn-icon-xs"></svg>
                          </button>
                        </div>
                        <textarea
                          class="form-textarea level-desc-input"
                          rows="1"
                          placeholder="Descripción del nivel de logro..."
                          [ngModel]="level.description"
                          (ngModelChange)="updateLevel(i, j, { description: $event })"
                        ></textarea>
                      </div>
                    }
                  </div>
                </div>
              </div>
            }
          </div>
        }
      </section>

      <!-- Modal de solicitud de plantilla -->
      @if (showRequestModal()) {
        <template-request-modal
          [subjectId]="store.courseId()"
          (close)="closeRequestModal()"
          (requested)="onTemplateRequested()"
        />
      }
    </div>
  `,
  styleUrl: './step-ide-environment.component.scss'
})
export class StepIdeEnvironmentComponent implements OnInit {
  readonly store = inject(ExerciseEditorStore);
  private http = inject(HttpClient);

  approvedTemplates = signal<ApprovedTemplate[]>([]);
  showRequestModal = signal<boolean>(false);

  selectedTemplate = computed(() => {
    const tplId = this.store.metadata().template_id;
    if (!tplId) return null;
    return this.approvedTemplates().find(t => t.id === tplId) || this.store.metadata().template || null;
  });

  hasDatabaseService = computed(() => {
    const tpl = this.selectedTemplate();
    return !!(tpl && tpl.services_config && tpl.services_config.database && tpl.services_config.database.enabled);
  });

  ngOnInit(): void {
    this.loadTemplates();
  }

  loadTemplates(): void {
    this.http.get<any>('/api/v1/templates').subscribe({
      next: (res) => {
        const list: ApprovedTemplate[] = Array.isArray(res) ? res : res?.data || [];
        const filtered = list.filter(t => 
          (t.status === 'approved' || t.status === 'published' || !t.status) &&
          (t.target_environment === 'IDE_PERSISTENTE' || !t.target_environment)
        );
        this.approvedTemplates.set(filtered);

        // Si ya había una plantilla vinculada, actualizar resumen en store
        const currentId = this.store.metadata().template_id;
        if (currentId) {
          const match = filtered.find(t => t.id === currentId);
          if (match) {
            this.store.updateMetadata({
              template: match,
              memory_limit_mb: match.base_ram_mb
            });
          }
        }
      },
      error: () => {}
    });
  }

  onTemplateSelected(templateId: string): void {
    const match = this.approvedTemplates().find(t => t.id === templateId) || null;
    this.store.updateMetadata({
      template_id: templateId || null,
      template: match,
      memory_limit_mb: match ? match.base_ram_mb : 128
    });
  }

  onInitScriptChange(db_init_script: string): void {
    this.store.updateMetadata({ db_init_script });
  }

  onDueDateChange(due_date: string): void {
    this.store.updateMetadata({ due_date });
  }

  onAllowResubmitChange(event: Event): void {
    const checked = (event.target as HTMLInputElement).checked;
    this.store.updateMetadata({ allow_resubmit: checked });
  }

  onAutoCheckpointChange(event: Event): void {
    const checked = (event.target as HTMLInputElement).checked;
    this.store.updateMetadata({ auto_checkpoint: checked });
  }

  openRequestModal(): void {
    this.showRequestModal.set(true);
  }

  closeRequestModal(): void {
    this.showRequestModal.set(false);
  }

  onTemplateRequested(): void {
    this.closeRequestModal();
    this.loadTemplates();
  }

  totalRubricWeight = computed(() => {
    const rub = this.store.rubric();
    if (!rub || !rub.criteria) return 0;
    return rub.criteria.reduce((sum, c) => sum + (c.weight || 0), 0);
  });

  applyRubricPreset(presetKey: string): void {
    if (!presetKey || !RUBRIC_PRESETS[presetKey]) return;
    const preset = RUBRIC_PRESETS[presetKey];
    this.store.setRubric({
      criteria: JSON.parse(JSON.stringify(preset.criteria))
    });
  }

  addCriterion(): void {
    const current = this.store.rubric() || { criteria: [] };
    const newCriterion: RubricCriterionDTO = {
      name: 'Nuevo Criterio',
      description: 'Descripción del criterio de evaluación',
      weight: 25,
      levels: [
        { name: 'Excelente', score: 100, description: 'Cumple con excelencia los requisitos.' },
        { name: 'Bueno', score: 75, description: 'Cumple satisfactoriamente.' },
        { name: 'Regular', score: 50, description: 'Cumple parcialmente.' },
        { name: 'Deficiente', score: 25, description: 'No cumple con los requisitos.' }
      ]
    };
    this.store.setRubric({
      ...current,
      criteria: [...(current.criteria || []), newCriterion]
    });
  }

  removeCriterion(index: number): void {
    const current = this.store.rubric();
    if (!current) return;
    const nextCriteria = current.criteria.filter((_, i) => i !== index);
    this.store.setRubric({ ...current, criteria: nextCriteria });
  }

  updateCriterion(index: number, update: Partial<RubricCriterionDTO>): void {
    const current = this.store.rubric();
    if (!current) return;
    const copy = [...current.criteria];
    if (index >= 0 && index < copy.length) {
      copy[index] = { ...copy[index], ...update };
    }
    this.store.setRubric({ ...current, criteria: copy });
  }

  addLevelToCriterion(criterionIndex: number): void {
    const current = this.store.rubric();
    if (!current) return;
    const copy = [...current.criteria];
    if (criterionIndex >= 0 && criterionIndex < copy.length) {
      const levels = [...copy[criterionIndex].levels, { name: 'Nuevo Nivel', score: 50, description: '' }];
      copy[criterionIndex] = { ...copy[criterionIndex], levels };
    }
    this.store.setRubric({ ...current, criteria: copy });
  }

  removeLevelFromCriterion(criterionIndex: number, levelIndex: number): void {
    const current = this.store.rubric();
    if (!current) return;
    const copy = [...current.criteria];
    if (criterionIndex >= 0 && criterionIndex < copy.length) {
      const levels = copy[criterionIndex].levels.filter((_, j) => j !== levelIndex);
      copy[criterionIndex] = { ...copy[criterionIndex], levels };
    }
    this.store.setRubric({ ...current, criteria: copy });
  }

  updateLevel(criterionIndex: number, levelIndex: number, update: Partial<RubricLevelDTO>): void {
    const current = this.store.rubric();
    if (!current) return;
    const copy = [...current.criteria];
    if (criterionIndex >= 0 && criterionIndex < copy.length) {
      const levels = [...copy[criterionIndex].levels];
      if (levelIndex >= 0 && levelIndex < levels.length) {
        levels[levelIndex] = { ...levels[levelIndex], ...update };
      }
      copy[criterionIndex] = { ...copy[criterionIndex], levels };
    }
    this.store.setRubric({ ...current, criteria: copy });
  }
}
