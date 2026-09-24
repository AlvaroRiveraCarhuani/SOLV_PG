import { Component, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { LucideHelpCircle } from '@lucide/angular';
import { TargetEnvironment } from '../../../../../services/admin-templates.service';

@Component({
  selector: 'solv-step-execution',
  standalone: true,
  imports: [CommonModule, FormsModule, LucideHelpCircle],
  template: `
    <div class="execution-step-container">
      <div class="step-header-with-help mb-3">
        <h4 class="step-section-title" id="step-title-execution" tabindex="-1" i18n="@@PU-08-HEADING">
          Parámetros de ejecución
        </h4>
        <button 
          type="button" 
          class="btn-step-help" 
          (click)="helpRequested.emit()" 
          title="Ayuda contextual del paso" 
          aria-label="Ayuda contextual del paso"
          i18n-aria-label="@@AY-02"
        >
          <svg lucideHelpCircle class="w-4 h-4"></svg>
        </button>
      </div>

      @if (targetEnvironment() === 'JUEZ_EFIMERO') {
        <!-- Configuración de Juez Virtual -->
        <div class="form-group">
          <label class="form-label font-semibold" i18n="@@PU-08">
            Comando de compilación o ejecución: <span class="text-danger">*</span>
          </label>
          <input 
            type="text" 
            class="form-control font-mono" 
            placeholder="Ej: gcc -O2 solution.c -o solution && ./solution"
            [ngModel]="entrypoint()" 
            (ngModelChange)="entrypointChange.emit($event)" 
          />
          <small class="form-hint" i18n="@@PU-15">
            Comando de ejecución obligatorio para plantillas de juez virtual.
          </small>
        </div>

        <div class="grid-2-cols">
          <div class="form-group">
            <label class="form-label font-semibold" i18n="@@PU-09">
              Tiempo límite de ejecución (ms):
            </label>
            <input 
              type="number" 
              class="form-control font-mono" 
              min="500" 
              max="15000" 
              step="500"
              [ngModel]="timeoutMS()" 
              (ngModelChange)="timeoutMSChange.emit($event)" 
            />
            <small class="form-hint">Tiempo máximo antes de veredicto TLE (Time Limit Exceeded).</small>
          </div>

          <div class="form-group">
            <label class="form-label font-semibold" i18n="@@PU-10">
              Entrada estándar de prueba (stdin opcional):
            </label>
            <input 
              type="text" 
              class="form-control font-mono" 
              placeholder="Ej: 5 10"
              [ngModel]="sampleInput()" 
              (ngModelChange)="sampleInputChange.emit($event)" 
            />
            <small class="form-hint">Entrada que se inyectará durante la prueba del smoke test.</small>
          </div>
        </div>
      } @else {
        <!-- Configuración de Laboratorio Interactivo -->
        <div class="form-group">
          <label class="form-label font-semibold">
            Script de Inicialización al Despliegue (Opcional):
          </label>
          <textarea 
            class="form-control font-mono text-area-sm" 
            rows="5" 
            placeholder="#!/bin/bash&#10;echo 'Iniciando entorno'..." 
            [ngModel]="setupScript()" 
            (ngModelChange)="setupScriptChange.emit($event)"
          ></textarea>
          <small class="form-hint">
            Comandos bash que se ejecutarán automáticamente al iniciar cada workspace.
          </small>
        </div>
      }
    </div>
  `,
  styleUrls: ['./step-execution.component.scss']
})
export class SolvStepExecutionComponent {
  targetEnvironment = input<TargetEnvironment>('IDE_PERSISTENTE');
  setupScript = input<string>('');
  entrypoint = input<string>('');
  timeoutMS = input<number>(5000);
  sampleInput = input<string>('');

  setupScriptChange = output<string>();
  entrypointChange = output<string>();
  timeoutMSChange = output<number>();
  sampleInputChange = output<string>();
  helpRequested = output<void>();
  advance = output<void>();
}
