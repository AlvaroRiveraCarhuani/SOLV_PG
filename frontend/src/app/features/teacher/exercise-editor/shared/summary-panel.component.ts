import { Component, computed, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import {
  LucideClock,
  LucideCpu,
  LucideAlertTriangle,
  LucideLayers
} from '@lucide/angular';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { ExerciseEditorStore } from '../exercise-editor.store';

@Component({
  selector: 'summary-panel',
  standalone: true,
  imports: [
    CommonModule,
    LucideClock,
    LucideCpu,
    LucideAlertTriangle,
    LucideLayers,
    MachineDataDirective
  ],
  template: `

    <aside class="summary-card" aria-label="Resumen del ejercicio">
      <div class="summary-header">
        <div class="header-badge">
          <svg lucideLayers class="badge-icon"></svg>
          <span class="summary-title">Resumen</span>
        </div>
        <span class="status-pill" [class.published]="store.status() === 'published'">
          {{ store.status() === 'published' ? 'Publicado' : 'Borrador' }}
        </span>
      </div>

      <!-- Sección: Configuración General -->
      <div class="summary-section">
        <h4 class="section-heading">Configuración General</h4>
        <div class="summary-row">
          <span class="row-label">Modalidad:</span>
          <span class="row-value font-mono" machineData>
            {{ store.metadata().environment_type === 'IDE_PERSISTENTE' ? 'WORKSPACE' : 'JUEZ VIRTUAL' }}
          </span>
        </div>
        <div class="summary-row">
          <span class="row-label">Propósito:</span>
          <span class="row-value">{{ store.metadata().purpose === 'exam' ? 'Examen' : 'Clase regular' }}</span>
        </div>
        @if (store.metadata().per_student_seed) {
          <div class="summary-row highlight">
            <span class="row-label">Anti-copia (variantes):</span>
            <span class="row-value seed-badge">Activada</span>
          </div>
        }
        <div class="summary-row">
          <span class="row-label">Dificultad:</span>
          <span class="row-value difficulty-tag" [class]="store.metadata().difficulty || 'none'">
            {{ store.metadata().difficulty ? (store.metadata().difficulty | titlecase) : 'Sin definir' }}
          </span>
        </div>
      </div>

      <!-- Sección: Recursos y Límites -->
      <div class="summary-section">
        <h4 class="section-heading">Límites de Ejecución</h4>
        <div class="metrics-grid">
          <div class="metric-box">
            <div class="metric-header">
              <svg lucideClock class="metric-icon"></svg>
              <span class="metric-label">Tiempo (por caso)</span>
            </div>
            <span class="metric-val font-mono" machineData>{{ store.metadata().time_limit_ms }} ms</span>
          </div>
          <div class="metric-box">
            <div class="metric-header">
              <svg lucideCpu class="metric-icon"></svg>
              <span class="metric-label">RAM máxima</span>
            </div>
            <span class="metric-val font-mono" machineData>{{ store.metadata().memory_limit_mb }} MB</span>
          </div>
        </div>
      </div>

      <!-- Sección: Evaluación Según Modalidad -->
      @if (store.metadata().environment_type === 'IDE_PERSISTENTE') {
        <div class="summary-section">
          <h4 class="section-heading">Evaluación de Laboratorio (Rúbrica)</h4>
          <div class="summary-row">
            <span class="row-label">Criterios:</span>
            <span class="row-value font-mono" machineData>{{ store.rubric()?.criteria?.length || 0 }}</span>
          </div>
          <div class="summary-row">
            <span class="row-label">Ponderación:</span>
            <span class="row-value font-mono" machineData>{{ totalRubricWeight() }}% / 100%</span>
          </div>
          <div class="summary-row">
            <span class="row-label">Plantilla IDE:</span>
            <span class="row-value truncate" machineData>{{ store.metadata().template?.name || store.metadata().template_id || 'Sin asignar' }}</span>
          </div>
          @if (store.metadata().due_date) {
            <div class="summary-row">
              <span class="row-label">Entrega:</span>
              <span class="row-value font-mono" machineData>{{ store.metadata().due_date | slice:0:10 }}</span>
            </div>
          }
        </div>
      } @else {
        <!-- Sección: Casos de Prueba (Juez Virtual) -->
        <div class="summary-section">
          <h4 class="section-heading">Casos de Prueba ({{ store.totalCasesCount() }})</h4>

          <div class="case-breakdown">
            <div class="case-chip example">
              <span class="chip-dot"></span>
              <span class="chip-label">Ejemplo:</span>
              <span class="chip-count font-mono" machineData>{{ store.exampleCasesCount() }}</span>
            </div>
            <div class="case-chip public">
              <span class="chip-dot"></span>
              <span class="chip-label">Público:</span>
              <span class="chip-count font-mono" machineData>{{ store.publicCasesCount() }}</span>
            </div>
            <div class="case-chip hidden">
              <span class="chip-dot"></span>
              <span class="chip-label">Oculto:</span>
              <span class="chip-count font-mono" machineData>{{ store.hiddenCasesCount() }}</span>
            </div>
          </div>

          <div class="weight-distribution">
            <div class="weight-row">
              <span class="weight-label">Peso visible / oculto:</span>
              <span class="weight-val font-mono" machineData>
                {{ store.visibleWeight() | number:'1.0-1' }} pts / {{ store.hiddenWeight() | number:'1.0-1' }} pts
              </span>
            </div>
            <div class="weight-bar-track">
              @if (store.totalWeight() > 0) {
                <div
                  class="weight-bar-visible"
                  [style.width.%]="(store.visibleWeight() / store.totalWeight()) * 100"
                  title="Peso visible"
                ></div>
                <div
                  class="weight-bar-hidden"
                  [style.width.%]="(store.hiddenWeight() / store.totalWeight()) * 100"
                  title="Peso oculto"
                ></div>
              } @else {
                <div class="weight-bar-empty"></div>
              }
            </div>
          </div>
        </div>
      }


      <!-- Sección: Advertencias & Sugerencias -->
      @if (allWarnings().length > 0) {
        <div class="summary-section warnings-box">
          <h4 class="section-heading warning-heading">
            <svg lucideAlertTriangle class="warning-icon"></svg>
            Sugerencias pedagógicas
          </h4>
          <ul class="warning-list">
            @for (warn of allWarnings(); track $index) {
              <li class="warning-item">{{ warn }}</li>
            }
          </ul>
        </div>
      }
    </aside>
  `,
  styleUrl: './summary-panel.component.scss'
})
export class SummaryPanelComponent {
  readonly store = inject(ExerciseEditorStore);

  readonly totalRubricWeight = computed(() => {
    const rub = this.store.rubric();
    if (!rub || !rub.criteria) return 0;
    return rub.criteria.reduce((sum, c) => sum + (c.weight || 0), 0);
  });

  allWarnings() {
    return [
      ...this.store.step1Validation().warnings,
      ...this.store.step2Validation().warnings,
      ...this.store.step3Validation().warnings
    ];
  }
}

