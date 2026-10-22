import { Component, OnInit, inject, signal, computed } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { ExerciseEditorStore, RubricCriterionDTO, RubricLevelDTO } from '../../exercise-editor.store';
import { ComboboxOption } from '@shared/components/combobox/combobox.component';
import { TemplateRequestModalComponent } from '../../../templates/template-request-modal/template-request-modal.component';
import { RUBRIC_PRESETS } from '@shared/rubric-presets';
import { IdeTemplateSelectorComponent } from './components/ide-template-selector/ide-template-selector.component';
import { IdeDbInitScriptComponent } from './components/ide-db-init-script/ide-db-init-script.component';
import { IdeDeliveryConfigComponent } from './components/ide-delivery-config/ide-delivery-config.component';
import { IdeRubricEditorComponent } from './components/ide-rubric-editor/ide-rubric-editor.component';

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
    services?: Array<{
      category?: string;
      engine?: string;
      [key: string]: unknown;
    }>;
    [key: string]: unknown;
  };
}

@Component({
  selector: 'step-ide-environment',
  standalone: true,
  imports: [
    IdeTemplateSelectorComponent,
    IdeDbInitScriptComponent,
    IdeDeliveryConfigComponent,
    IdeRubricEditorComponent,
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

      <ide-template-selector
        [approvedTemplates]="approvedTemplates()"
        [selectedTemplate]="selectedTemplate()"
        [templateOptions]="templateOptions()"
        (templateSelected)="onTemplateSelected($event)"
        (requestTemplate)="openRequestModal()"
      />

      <ide-db-init-script
        [dbInitScript]="store.metadata().db_init_script || ''"
        [hasDatabaseService]="hasDatabaseService()"
        (initScriptChange)="onInitScriptChange($event)"
      />

      <ide-delivery-config
        [dueDate]="store.metadata().due_date || ''"
        [allowResubmit]="store.metadata().allow_resubmit || false"
        [autoCheckpoint]="store.metadata().auto_checkpoint || false"
        (dueDateChange)="onDueDateChange($event)"
        (allowResubmitChange)="onAllowResubmitChange($event)"
        (autoCheckpointChange)="onAutoCheckpointChange($event)"
      />

      <ide-rubric-editor
        [rubric]="store.rubric()"
        [rubricPresetOptions]="rubricPresetOptions"
        [totalRubricWeight]="totalRubricWeight()"
        (presetSelected)="applyRubricPreset($event)"
        (addCriterion)="addCriterion()"
        (removeCriterion)="removeCriterion($event)"
        (updateCriterion)="updateCriterion($event.index, $event.update)"
        (addLevelToCriterion)="addLevelToCriterion($event)"
        (removeLevelFromCriterion)="removeLevelFromCriterion($event.criterionIndex, $event.levelIndex)"
        (updateLevel)="updateLevel($event.criterionIndex, $event.levelIndex, $event.update)"
      />

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

  hasDatabaseService = computed(() => this.templateHasDatabase(this.selectedTemplate()));

  readonly templateOptions = computed<ComboboxOption[]>(() => {
    return this.approvedTemplates().map(t => {
      const databaseEngine = this.templateDatabaseEngine(t);
      return {
        id: t.id,
        label: t.name,
        value: t.id,
        badge: `${t.base_ram_mb} MB`,
        description: `${t.docker_image}${databaseEngine ? ' • BD ' + databaseEngine : ''}`
      };
    });
  });

  /**
   * El backend puede serializar services_config con dos formas: el objeto
   * histórico `database.enabled` o la lista `services[]` con category=database.
   */
  private templateHasDatabase(tpl: ApprovedTemplate | null | undefined): boolean {
    return this.templateDatabaseEngine(tpl) !== '';
  }

  private templateDatabaseEngine(tpl: ApprovedTemplate | null | undefined): string {
    const config = tpl?.services_config;
    if (!config) {
      return '';
    }
    if (config.database?.enabled) {
      return config.database.engine || 'SQL';
    }
    const services = Array.isArray(config.services) ? config.services : [];
    const databaseService = services.find(
      service => service.category === 'database' && !!service.engine && service.engine !== 'none'
    );
    return databaseService?.engine || '';
  }

  readonly rubricPresetOptions: ComboboxOption[] = [
    { id: 'api-rest', label: 'API REST Backend', value: 'api-rest', description: 'Endpoints, HTTP y persistencia' },
    { id: 'full-stack', label: 'Aplicación Full-Stack', value: 'full-stack', description: 'Frontend, backend y base de datos' },
    { id: 'proyecto-integrador', label: 'Proyecto Integrador', value: 'proyecto-integrador', description: 'Arquitectura, testing y docs' }
  ];

  ngOnInit(): void {
    this.loadTemplates();
  }

  loadTemplates(): void {
    this.http.get<unknown>('/api/v1/templates?limit=50').subscribe({
      next: (res: unknown) => {
        const envelope = res as { data?: unknown } | null | undefined;
        const inner = (envelope?.data as { data?: unknown } | null | undefined)?.data;
        const list: ApprovedTemplate[] = Array.isArray(res)
          ? (res as ApprovedTemplate[])
          : Array.isArray(envelope?.data)
            ? (envelope.data as ApprovedTemplate[])
            : Array.isArray(inner)
              ? (inner as ApprovedTemplate[])
              : [];
        const filtered = list.filter(t => 
          (t.status === 'approved' || t.status === 'APROBADA' || t.status === 'published' || !t.status) &&
          (t.target_environment === 'IDE_PERSISTENTE' || t.target_environment === 'AMBOS' || !t.target_environment)
        );
        this.approvedTemplates.set(filtered);

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

  onAllowResubmitChange(checked: boolean): void {
    this.store.updateMetadata({ allow_resubmit: checked });
  }

  onAutoCheckpointChange(checked: boolean): void {
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
