import { Component, computed, inject, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { LucideAlertTriangle, LucideCheck } from '@lucide/angular';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { ExerciseEditorStore } from '../exercise-editor.store';

interface EditorSection {
  id: 1 | 2 | 3;
  label: string;
  status: 'valid' | 'warning' | 'invalid';
}

/** Navegación fija de las tres secciones del editor y resumen de preflight. */
@Component({
  selector: 'stepper-nav',
  standalone: true,
  imports: [CommonModule, LucideCheck, LucideAlertTriangle, MachineDataDirective],
  template: `
    <aside class="editor-nav" aria-label="Secciones del editor">
      <header class="nav-head">
        <span class="nav-label font-mono">SECCIONES</span>
        <span class="nav-count font-mono" machineData>{{ store.currentStep() }} / 3</span>
      </header>

      <ul class="nav-list">
        @for (section of sections(); track section.id) {
          <li>
            <button
              type="button"
              class="nav-item"
              [class.is-active]="store.currentStep() === section.id"
              [attr.aria-current]="store.currentStep() === section.id ? 'step' : null"
              (click)="store.setStep(section.id)"
            >
              <span class="nav-index font-mono" machineData>{{ section.id }}</span>
              <span class="nav-text">{{ section.label }}</span>
              <span class="nav-state" [class]="'is-' + section.status" [attr.aria-label]="stateLabel(section.status)">
                @if (section.status === 'valid') {
                  <svg lucideCheck aria-hidden="true"></svg>
                } @else if (section.status === 'warning') {
                  <svg lucideAlertTriangle aria-hidden="true"></svg>
                } @else {
                  <span class="state-dot" aria-hidden="true"></span>
                }
              </span>
            </button>
          </li>
        }
      </ul>

      <section class="preflight-card" aria-label="Verificación previa a publicar">
        <header class="preflight-head">
          <span class="preflight-title">Verificación</span>
        </header>

        <p class="preflight-summary">
          @if (blockersCount() === 0 && warningsCount() === 0) {
            <span class="summary-dot is-valid" aria-hidden="true"></span>
            Listo para publicar
          } @else if (blockersCount() > 0) {
            <span class="summary-dot is-invalid" aria-hidden="true"></span>
            Faltan <span machineData>{{ blockersCount() }}</span>
            {{ blockersCount() === 1 ? 'requisito' : 'requisitos' }}
            @if (warningsCount() > 0) {
              <span aria-hidden="true">·</span> <span machineData>{{ warningsCount() }}</span>
              {{ warningsCount() === 1 ? 'aviso' : 'avisos' }}
            }
          } @else {
            <span class="summary-dot is-warning" aria-hidden="true"></span>
            Listo con <span machineData>{{ warningsCount() }}</span>
            {{ warningsCount() === 1 ? 'aviso' : 'avisos' }}
          }
        </p>

        <button type="button" class="btn-secondary btn-sm preflight-open" (click)="openPreflight.emit()">
          Ver detalle
        </button>

        <p class="preflight-context font-mono">
          @if (store.isIde()) {
            RAM <span machineData>{{ store.metadata().memory_limit_mb }}</span> MB
            <span aria-hidden="true">·</span>
            {{ store.metadata().template?.name || 'Sin plantilla' }}
          } @else {
            <span machineData>{{ store.totalCasesCount() }}</span> ej.
            <span aria-hidden="true">·</span>
            <span machineData>{{ store.publicCasesCount() + store.exampleCasesCount() }}</span> púb.
            <span aria-hidden="true">·</span>
            <span machineData>{{ store.hiddenCasesCount() }}</span> oc.
          }
        </p>
      </section>
    </aside>
  `,
  styleUrl: './stepper-nav.component.scss'
})
export class StepperNavComponent {
  readonly store = inject(ExerciseEditorStore);
  readonly openPreflight = output<void>();

  readonly sections = computed<EditorSection[]>(() => {
    const isIde = this.store.isIde();
    const labels = isIde
      ? ['Identidad y consigna', 'Entorno y proyecto', 'Entrega y rúbrica']
      : ['Identidad y enunciado', 'Solución y casos', 'Reglas y código inicial'];

    const sections: EditorSection[] = [
      { id: 1, label: labels[0], status: this.store.step1Validation().status as EditorSection['status'] },
      { id: 2, label: labels[1], status: this.store.step2Validation().status as EditorSection['status'] },
      { id: 3, label: labels[2], status: this.store.step3Validation().status as EditorSection['status'] }
    ];
    return sections;
  });

  readonly blockersCount = computed(() => this.store.missingPublishRequirements().length);

  readonly warningsCount = computed(() => {
    const report = this.store.checklistReport();
    const sectionWarnings = [
      ...this.store.step1Validation().warnings,
      ...this.store.step2Validation().warnings
    ].filter(warning => !warning.toLowerCase().includes('rúbrica'));
    return sectionWarnings.length + (report?.warnings.length ?? 0);
  });

  stateLabel(status: EditorSection['status']): string {
    if (status === 'valid') {
      return 'Sección completa';
    }
    if (status === 'warning') {
      return 'Sección con avisos';
    }
    return 'Sección con requisitos pendientes';
  }
}
