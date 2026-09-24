import { Component, input, output, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { 
  LucideHelpCircle, 
  LucideInfo, 
  LucideCopy, 
  LucideX, 
  LucideCode 
} from '@lucide/angular';
import { TargetEnvironment } from '../../../../../services/admin-templates.service';

export interface ScriptExample {
  label: string;
  desc: string;
  code: string;
}

export interface EntrypointExample {
  label: string;
  desc: string;
  cmd: string;
}

@Component({
  selector: 'solv-step-execution',
  standalone: true,
  imports: [
    CommonModule, 
    FormsModule, 
    LucideHelpCircle, 
    LucideInfo, 
    LucideCopy, 
    LucideX, 
    LucideCode
  ],
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
          <div class="label-with-popover-row">
            <label class="form-label font-semibold" i18n="@@PU-08">
              Comando de compilación o ejecución: <span class="text-danger">*</span>
            </label>
            <button 
              type="button" 
              class="btn-contract-info" 
              (click)="toggleContractPopover()"
              [attr.aria-expanded]="showContractPopover()"
              title="Ver contrato de ejecución del juez"
            >
              <svg lucideInfo class="w-3.5 h-3.5 mr-1"></svg>
              <span>Contrato de ejecución</span>
            </button>
          </div>

          <!-- Popover del Contrato de Ejecución para Juez -->
          @if (showContractPopover()) {
            <div class="contract-popover animate-fade">
              <div class="popover-header">
                <strong class="popover-title">Contrato de Ejecución en Juez Virtual</strong>
                <button type="button" class="btn-popover-close" (click)="closeContractPopover()">
                  <svg lucideX class="w-3.5 h-3.5"></svg>
                </button>
              </div>
              <ul class="contract-list">
                <li><strong>Aislamiento:</strong> Sandbox efímero descartable sin acceso a red y sistema de archivos temporal.</li>
                <li><strong>Entrada / Salida:</strong> El comando debe leer pruebas desde <code>stdin</code> y emitir resultados a <code>stdout</code>.</li>
                <li><strong>Códigos de retorno:</strong> Código 0 para éxito (AC). Código &ne; 0 produce Runtime Error (RE).</li>
                <li><strong>Límite temporal:</strong> Si sobrepasa el timeout configurado, se aborta con Time Limit Exceeded (TLE).</li>
              </ul>
            </div>
          }

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

          <!-- Ejemplos rápidos para Juez -->
          <div class="quick-examples-box mt-3">
            <span class="examples-header-label">
              <svg lucideCode class="w-3.5 h-3.5 mr-1 text-primary"></svg>
              Ejemplos recomendados para compilar y ejecutar:
            </span>
            <div class="examples-chips-row">
              @for (ex of judgeExamples; track ex.label) {
                <button 
                  type="button" 
                  class="chip-example-btn" 
                  (click)="applyEntrypointExample(ex.cmd)"
                  [title]="ex.desc + ' · Clic para insertar'"
                >
                  <span class="chip-label">{{ ex.label }}</span>
                  <svg lucideCopy class="w-3 h-3 ml-1 chip-icon"></svg>
                </button>
              }
            </div>
          </div>
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
          <div class="label-with-popover-row">
            <label class="form-label font-semibold">
              Script de Inicialización al Despliegue (Opcional):
            </label>
            <button 
              type="button" 
              class="btn-contract-info" 
              (click)="toggleContractPopover()"
              [attr.aria-expanded]="showContractPopover()"
              title="Ver contrato formal del script de inicio"
            >
              <svg lucideInfo class="w-3.5 h-3.5 mr-1"></svg>
              <span>Contrato del script</span>
            </button>
          </div>

          <!-- Popover del Contrato de Inicialización para IDE -->
          @if (showContractPopover()) {
            <div class="contract-popover animate-fade">
              <div class="popover-header">
                <strong class="popover-title">Contrato de Ejecución del Script de Inicialización</strong>
                <button type="button" class="btn-popover-close" (click)="closeContractPopover()">
                  <svg lucideX class="w-3.5 h-3.5"></svg>
                </button>
              </div>
              <ul class="contract-list">
                <li><strong>Intérprete:</strong> Ejecutado con <code>/bin/sh -c</code> o <code>/bin/bash</code> dentro del contenedor activo.</li>
                <li><strong>Ciclo de vida:</strong> Se ejecuta antes de que el editor web comience a recibir conexiones en el puerto 3000.</li>
                <li><strong>Tolerancia a fallos:</strong> Aplica <code>set -e</code>; cualquier fallo con código de salida distinto de 0 aborta el arranque.</li>
                <li><strong>Idempotencia:</strong> Diseñe comandos seguros frente a reejecuciones al reiniciar o reanudar el laboratorio.</li>
              </ul>
            </div>
          }

          <textarea 
            class="form-control font-mono text-area-sm" 
            rows="5" 
            placeholder="#!/bin/bash&#10;echo 'Iniciando entorno'..." 
            [ngModel]="setupScript()" 
            (ngModelChange)="setupScriptChange.emit($event)"
          ></textarea>
          <small class="form-hint">
            Comandos bash que se ejecutarán automáticamente al iniciar cada workspace antes del editor.
          </small>

          <!-- Ejemplos rápidos recomendados para IDE -->
          <div class="quick-examples-box mt-3">
            <span class="examples-header-label">
              <svg lucideCode class="w-3.5 h-3.5 mr-1 text-primary"></svg>
              Plantillas de script recomendadas:
            </span>
            <div class="examples-chips-row">
              @for (ex of ideExamples; track ex.label) {
                <button 
                  type="button" 
                  class="chip-example-btn" 
                  (click)="applyScriptExample(ex.code)"
                  [title]="ex.desc + ' · Clic para insertar en el editor'"
                >
                  <span class="chip-label">{{ ex.label }}</span>
                  <svg lucideCopy class="w-3 h-3 ml-1 chip-icon"></svg>
                </button>
              }
            </div>
          </div>
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

  showContractPopover = signal<boolean>(false);

  ideExamples: ScriptExample[] = [
    {
      label: 'Clonar repositorio',
      desc: 'Clona el repositorio base del curso',
      code: 'if [ ! -d "/workspace/proyecto" ]; then\n  git clone --depth 1 https://github.com/ejemplo/laboratorio-base.git /workspace/proyecto\nfi'
    },
    {
      label: 'Instalar paquetes',
      desc: 'Instala dependencias requeridas',
      code: 'if [ -f "/workspace/requirements.txt" ]; then\n  pip install --no-cache-dir -r /workspace/requirements.txt\nfi'
    },
    {
      label: 'Variables de entorno',
      desc: 'Exporta configuración de entorno',
      code: 'export APP_ENV=development\nexport PORT=3000'
    }
  ];

  judgeExamples: EntrypointExample[] = [
    {
      label: 'C / C++ (GCC)',
      desc: 'Compila y ejecuta binario nativo',
      cmd: 'gcc -O2 solution.c -o solution && ./solution'
    },
    {
      label: 'Python 3',
      desc: 'Ejecución interpretada de script',
      cmd: 'python3 solution.py'
    },
    {
      label: 'Java (OpenJDK)',
      desc: 'Compilación y ejecución JVM',
      cmd: 'javac Solution.java && java -Xmx256m Solution'
    },
    {
      label: 'Go',
      desc: 'Ejecución con compilación al vuelo',
      cmd: 'go run main.go'
    }
  ];

  toggleContractPopover(): void {
    this.showContractPopover.update(v => !v);
  }

  closeContractPopover(): void {
    this.showContractPopover.set(false);
  }

  applyScriptExample(code: string): void {
    const current = this.setupScript().trim();
    if (!current) {
      this.setupScriptChange.emit(code);
    } else {
      this.setupScriptChange.emit(`${current}\n\n${code}`);
    }
  }

  applyEntrypointExample(cmd: string): void {
    this.entrypointChange.emit(cmd);
  }
}
