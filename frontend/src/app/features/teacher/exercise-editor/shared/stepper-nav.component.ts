import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { 
  LucideCheck, 
  LucideAlertTriangle, 
  LucideAlertCircle
} from '@lucide/angular';
import { ExerciseEditorStore } from '../exercise-editor.store';

@Component({
  selector: 'stepper-nav',
  standalone: true,
  imports: [
    CommonModule,
    LucideCheck,
    LucideAlertTriangle,
    LucideAlertCircle
  ],
  template: `
    <nav class="stepper-container" aria-label="Pasos de configuración del ejercicio">
      <div class="stepper-header">
        <span class="stepper-title">Configuración</span>
        <span class="stepper-subtitle">Paso {{ store.currentStep() }} de 3</span>
      </div>

      <ul class="step-list">
        <!-- Paso 1 -->
        <li 
          class="step-item"
          [class.active]="store.currentStep() === 1"
          [class.completed]="store.step1Validation().isValid"
          (click)="store.setStep(1)"
          tabindex="0"
          (keydown.enter)="store.setStep(1)"
        >
          <div class="step-badge">
            @if (store.step1Validation().status === 'valid') {
              <svg lucideCheck class="status-icon valid"></svg>
            } @else if (store.step1Validation().status === 'warning') {
              <svg lucideAlertTriangle class="status-icon warning"></svg>
            } @else {
              <svg lucideAlertCircle class="status-icon invalid"></svg>
            }
          </div>
          <div class="step-content">
            <span class="step-number">Paso 1</span>
            <span class="step-label">Identidad y Pedagogía</span>
            <span class="step-hint">Metadatos, modalidad y enunciado</span>
          </div>
        </li>

        <!-- Paso 2 -->
        <li 
          class="step-item"
          [class.active]="store.currentStep() === 2"
          [class.completed]="store.step2Validation().isValid"
          (click)="store.setStep(2)"
          tabindex="0"
          (keydown.enter)="store.setStep(2)"
        >
          <div class="step-badge">
            @if (store.step2Validation().status === 'valid') {
              <svg lucideCheck class="status-icon valid"></svg>
            } @else if (store.step2Validation().status === 'warning') {
              <svg lucideAlertTriangle class="status-icon warning"></svg>
            } @else {
              <svg lucideAlertCircle class="status-icon invalid"></svg>
            }
          </div>
          <div class="step-content">
            <span class="step-number">Paso 2</span>
            <span class="step-label">Contrato y Casos</span>
            <span class="step-hint">Gramática, casos de prueba y solución</span>
          </div>
        </li>

        <!-- Paso 3 -->
        <li 
          class="step-item"
          [class.active]="store.currentStep() === 3"
          [class.completed]="store.step3Validation().isValid"
          (click)="store.setStep(3)"
          tabindex="0"
          (keydown.enter)="store.setStep(3)"
        >
          <div class="step-badge">
            @if (store.step3Validation().status === 'valid') {
              <svg lucideCheck class="status-icon valid"></svg>
            } @else if (store.step3Validation().status === 'warning') {
              <svg lucideAlertTriangle class="status-icon warning"></svg>
            } @else {
              <svg lucideAlertCircle class="status-icon invalid"></svg>
            }
          </div>
          <div class="step-content">
            <span class="step-number">Paso 3</span>
            <span class="step-label">Reglas y Publicación</span>
            <span class="step-hint">AST rules, boilerplate y checklist</span>
          </div>
        </li>
      </ul>
    </nav>
  `,
  styleUrl: './stepper-nav.component.scss'
})
export class StepperNavComponent {
  readonly store = inject(ExerciseEditorStore);
}
