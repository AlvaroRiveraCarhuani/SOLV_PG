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
  LucideArrowRight
} from '@lucide/angular';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { ExerciseEditorStore } from '../../../exercise-editor.store';

export interface PreFlightItem {
  id: string;
  message: string;
  stepTarget: 1 | 2;
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
    MachineDataDirective
  ],
  templateUrl: './publication-checklist.component.html',
  styleUrl: './publication-checklist.component.scss'
})
export class PublicationChecklistComponent {
  readonly store = inject(ExerciseEditorStore);

  readonly blockersOpen = signal<boolean>(true);
  readonly warningsOpen = signal<boolean>(true);
  readonly infoOpen = signal<boolean>(true);

  // Mapeo estructurado para salto directo con diagnóstico preciso
  readonly structuredBlockers = computed<PreFlightItem[]>(() => {
    const raw = this.store.checklistReport()?.blockers || [];
    const meta = this.store.metadata();
    const isIDE = meta.environment_type === 'IDE_PERSISTENTE';

    return raw.map((msg, idx) => {
      let stepTarget: 1 | 2 = 1;
      let actionLabel = 'Corregir en Paso 1';

      if (isIDE) {
        if (msg.includes('plantilla') || msg.includes('rúbrica') || msg.includes('rubric') || msg.includes('peso')) {
          stepTarget = 2;
          actionLabel = msg.includes('plantilla') ? 'Seleccionar Plantilla (Paso 2)' : 'Ajustar Rúbrica (Paso 2)';
        } else {
          stepTarget = 1;
          actionLabel = 'Editar Identidad (Paso 1)';
        }
      } else {
        if (msg.includes('caso') || msg.includes('solución') || msg.includes('referencia')) {
          stepTarget = 2;
          actionLabel = msg.includes('caso') ? 'Gestionar Casos (Paso 2)' : 'Definir Solución (Paso 2)';
        } else {
          stepTarget = 1;
          actionLabel = 'Editar Identidad (Paso 1)';
        }
      }

      return {
        id: `blocker-${idx}`,
        message: msg,
        stepTarget,
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
      let actionLabel = 'Revisar en Paso 2';

      if (isIDE) {
        if (msg.includes('fecha') || msg.includes('entrega')) {
          stepTarget = 2;
          actionLabel = 'Fijar Fecha (Paso 2)';
        } else if (msg.includes('rúbrica')) {
          stepTarget = 2;
          actionLabel = 'Definir Rúbrica (Paso 2)';
        } else {
          stepTarget = 1;
          actionLabel = 'Revisar Paso 1';
        }
      } else {
        if (msg.includes('caso') || msg.includes('oculto') || msg.includes('ejemplo') || msg.includes('referencia')) {
          stepTarget = 2;
          actionLabel = msg.includes('referencia') ? 'Ver Solución (Paso 2)' : 'Balancear Casos (Paso 2)';
        } else {
          stepTarget = 1;
          actionLabel = 'Revisar Paso 1';
        }
      }

      return {
        id: `warning-${idx}`,
        message: msg,
        stepTarget,
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

  goToStep(step: 1 | 2): void {
    this.store.setStep(step);
  }

  onAcceptWarnings(checked: boolean): void {
    this.store.setAcceptedWarnings(checked);
  }
}

