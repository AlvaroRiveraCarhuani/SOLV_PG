import { Component, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import {
  LucideCheckCircle2,
  LucideAlertTriangle,
  LucideAlertCircle,
  LucideInfo,
  LucideRefreshCw,
  LucideChevronDown,
  LucideChevronUp,
  LucideShieldCheck,
  LucideShieldX,
  LucideArrowRight,
  LucideEye
} from '@lucide/angular';

import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { ExerciseEditorStore } from '../../../exercise-editor.store';

export interface PreFlightItem {
  id: string;
  message: string;
  stepTarget: 1 | 2;
  targetElementId?: string;
  actionLabel: string;
  severity: 'blocker' | 'warning' | 'info';
}

@Component({
  selector: 'publication-checklist',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideCheckCircle2,
    LucideAlertTriangle,
    LucideAlertCircle,
    LucideInfo,
    LucideRefreshCw,
    LucideChevronDown,
    LucideChevronUp,
    LucideShieldCheck,
    LucideShieldX,
    LucideArrowRight,
    LucideEye,
    MachineDataDirective
  ],
  templateUrl: './publication-checklist.component.html',
  styleUrl: './publication-checklist.component.scss'
})
export class PublicationChecklistComponent {
  readonly store = inject(ExerciseEditorStore);

  openStudentPreview(): void {
    this.store.setStudentPreviewOpen(true);
  }

  readonly blockersOpen = signal<boolean>(true);
  readonly warningsOpen = signal<boolean>(true);
  readonly infoOpen = signal<boolean>(true);

  // Mapeo estructurado para salto directo con diagnóstico preciso y foco en el campo
  readonly structuredBlockers = computed<PreFlightItem[]>(() => {
    const raw = this.store.checklistReport()?.blockers || [];
    const meta = this.store.metadata();
    const isIDE = meta.environment_type === 'IDE_PERSISTENTE';

    return raw.map((msg, idx) => {
      let stepTarget: 1 | 2 = 1;
      let targetElementId = 'exercise-title';
      let actionLabel = 'Corregir en Paso 1';

      if (isIDE) {
        if (msg.includes('plantilla')) {
          stepTarget = 2;
          targetElementId = 'template-section';
          actionLabel = 'Seleccionar Plantilla (Paso 2)';
        } else if (msg.includes('rúbrica') || msg.includes('rubric') || msg.includes('peso')) {
          stepTarget = 2;
          targetElementId = 'rubric-section';
          actionLabel = 'Ajustar Rúbrica (Paso 2)';
        } else if (msg.includes('título') || msg.includes('titulo')) {
          stepTarget = 1;
          targetElementId = 'exercise-title';
          actionLabel = 'Definir Título (Paso 1)';
        } else if (msg.includes('enunciado') || msg.includes('descripción')) {
          stepTarget = 1;
          targetElementId = 'statement-description';
          actionLabel = 'Redactar Enunciado (Paso 1)';
        } else {
          stepTarget = 1;
          targetElementId = 'exercise-title';
          actionLabel = 'Editar Identidad (Paso 1)';
        }
      } else {
        if (msg.includes('caso')) {
          stepTarget = 2;
          targetElementId = 'cases-table-section';
          actionLabel = 'Gestionar Casos (Paso 2)';
        } else if (msg.includes('solución') || msg.includes('referencia') || msg.includes('formato') || msg.includes('input_format')) {
          stepTarget = 2;
          targetElementId = 'reference-solution-section';
          actionLabel = 'Definir Solución (Paso 2)';
        } else if (msg.includes('título') || msg.includes('titulo')) {
          stepTarget = 1;
          targetElementId = 'exercise-title';
          actionLabel = 'Definir Título (Paso 1)';
        } else if (msg.includes('lenguaje')) {
          stepTarget = 1;
          targetElementId = 'exercise-language';
          actionLabel = 'Elegir Lenguaje (Paso 1)';
        } else if (msg.includes('enunciado') || msg.includes('descripción')) {
          stepTarget = 1;
          targetElementId = 'statement-description';
          actionLabel = 'Redactar Enunciado (Paso 1)';
        } else if (msg.includes('tiempo')) {
          stepTarget = 1;
          targetElementId = 'time-limit';
          actionLabel = 'Ajustar Tiempo Límite (Paso 1)';
        } else {
          stepTarget = 1;
          targetElementId = 'exercise-title';
          actionLabel = 'Editar Identidad (Paso 1)';
        }
      }

      return {
        id: `blocker-${idx}`,
        message: msg,
        stepTarget,
        targetElementId,
        actionLabel,
        severity: 'blocker'
      };
    });
  });

  readonly structuredWarnings = computed<PreFlightItem[]>(() => {
    const raw = this.store.checklistReport()?.warnings || [];
    const meta = this.store.metadata();
    const isIDE = meta.environment_type === 'IDE_PERSISTENTE';

    return raw.map((msg, idx) => {
      let stepTarget: 1 | 2 = 2;
      let targetElementId = 'cases-table-section';
      let actionLabel = 'Revisar en Paso 2';

      if (isIDE) {
        if (msg.includes('fecha') || msg.includes('entrega')) {
          stepTarget = 2;
          targetElementId = 'due-date-section';
          actionLabel = 'Fijar Fecha (Paso 2)';
        } else if (msg.includes('rúbrica')) {
          stepTarget = 2;
          targetElementId = 'rubric-section';
          actionLabel = 'Definir Rúbrica (Paso 2)';
        } else {
          stepTarget = 1;
          targetElementId = 'exercise-title';
          actionLabel = 'Revisar Paso 1';
        }
      } else {
        if (msg.includes('referencia') || msg.includes('solución')) {
          stepTarget = 2;
          targetElementId = 'reference-solution-section';
          actionLabel = 'Ver Solución (Paso 2)';
        } else if (msg.includes('caso') || msg.includes('oculto') || msg.includes('ejemplo')) {
          stepTarget = 2;
          targetElementId = 'cases-table-section';
          actionLabel = 'Balancear Casos (Paso 2)';
        } else if (msg.includes('dificultad')) {
          stepTarget = 1;
          targetElementId = 'exercise-difficulty';
          actionLabel = 'Definir Dificultad (Paso 1)';
        } else if (msg.includes('etiqueta') || msg.includes('tag')) {
          stepTarget = 1;
          targetElementId = 'exercise-tag-input';
          actionLabel = 'Agregar Etiquetas (Paso 1)';
        } else {
          stepTarget = 1;
          targetElementId = 'exercise-title';
          actionLabel = 'Revisar Paso 1';
        }
      }

      return {
        id: `warning-${idx}`,
        message: msg,
        stepTarget,
        targetElementId,
        actionLabel,
        severity: 'warning'
      };
    });
  });

  readonly infoItems = computed(() => this.store.checklistReport()?.info || []);

  readonly hasBlockers = computed(() => this.structuredBlockers().length > 0);
  readonly hasWarnings = computed(() => this.structuredWarnings().length > 0);

  toggleBlockers(): void {
    this.blockersOpen.update(v => !v);
  }

  toggleWarnings(): void {
    this.warningsOpen.update(v => !v);
  }

  toggleInfo(): void {
    this.infoOpen.update(v => !v);
  }

  refreshChecklist(): void {
    this.store.refreshChecklist();
  }

  navigateToItem(item: PreFlightItem): void {
    this.store.setStep(item.stepTarget);
    if (!item.targetElementId) return;

    // Desplazamiento suave y enfoque tras la transición de vista de Angular
    setTimeout(() => {
      const target = document.getElementById(item.targetElementId!);
      if (target) {
        target.scrollIntoView({ behavior: 'smooth', block: 'center' });
        if (target instanceof HTMLInputElement || target instanceof HTMLTextAreaElement) {
          target.focus();
        } else {
          const innerFocusable = target.querySelector('input, textarea, button') as HTMLElement | null;
          if (innerFocusable) {
            innerFocusable.focus();
          }
        }
        target.classList.add('highlight-target-pulse');
        setTimeout(() => target.classList.remove('highlight-target-pulse'), 1800);
      }
    }, 120);
  }

  goToStep(step: 1 | 2): void {
    this.store.setStep(step);
  }

  onAcceptWarnings(checked: boolean): void {
    this.store.setAcceptedWarnings(checked);
  }
}

