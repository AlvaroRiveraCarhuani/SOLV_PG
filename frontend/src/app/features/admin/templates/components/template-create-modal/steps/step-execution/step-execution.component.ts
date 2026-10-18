import { Component, input, output, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { 
  LucideHelpCircle, 
  LucideInfo, 
  LucideX, 
  LucideCode,
  LucideAlertCircle,
  LucideFolder,
  LucideChevronDown
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
  selector: 'step-execution',
  standalone: true,
  imports: [
    CommonModule, 
    FormsModule, 
    LucideHelpCircle, 
    LucideInfo, 
    LucideX, 
    LucideCode,
    LucideAlertCircle,
    LucideFolder,
    LucideChevronDown
  ],
  template: `
    <div class="execution-step-container">
      <div class="step-header-with-help mb-3">
        <h4 class="step-section-title" id="step-title-execution" tabindex="-1" i18n="@@PU-08-HEADING">
          Comandos y servicios de ejecución
        </h4>
        <button 
          type="button" 
          class="btn-step-help" 
          (click)="helpRequested.emit()" 
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
              aria-label="Ver contrato de ejecución del juez virtual"
            >
              <svg lucideInfo class="w-3.5 h-3.5 mr-1"></svg>
              <span>Contrato de ejecución</span>
            </button>
          </div>

          <!-- Popover del Contrato de Ejecución para Juez -->
          @if (showContractPopover()) {
            <div class="contract-popover animate-fade">
              <div class="popover-header">
                <strong class="popover-title" i18n="@@AY-25">Contrato de ejecución del Juez Virtual</strong>
                <button type="button" class="btn-popover-close" (click)="closeContractPopover()" aria-label="Cerrar">
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
                  [class.chip-applied]="appliedJudgeChip() === ex.label"
                  (click)="applyEntrypointExample(ex)"
                  [attr.aria-label]="ex.label + ': ' + ex.desc + ' · Clic para insertar'"
                >
                  <span class="chip-label">{{ ex.label }}</span>
                  @if (appliedJudgeChip() === ex.label) {
                    <span class="chip-badge-inserted">Insertado</span>
                  }
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
              min="1000" 
              max="30000" 
              step="500"
              [ngModel]="timeoutMS()" 
              (ngModelChange)="timeoutMSChange.emit($event)" 
            />
            <small class="form-hint" i18n="@@AY-33">
              Rango de 1 a 30 segundos (default: 5.000 ms). Veredicto TLE al sobrepasarlo.
            </small>
          </div>

          <div class="form-group">
            <label class="form-label font-semibold" i18n="@@PU-10">
              Entrada estándar de prueba (stdin - recomendada):
            </label>
            <input 
              type="text" 
              class="form-control font-mono" 
              placeholder="Ej: 5 10"
              [ngModel]="sampleInput()" 
              (ngModelChange)="sampleInputChange.emit($event)" 
            />
            <small class="form-hint" i18n="@@AY-34">
              Entrada recomendada para alimentar el proceso durante la prueba de arranque.
            </small>
          </div>
        </div>
      } @else {
        <!-- Configuración de Laboratorio Interactivo: Estructura de Directorios y Comandos -->
        <div class="form-group mb-4">
          <label class="form-label font-semibold">
            Estructura inicial de carpetas en el laboratorio:
          </label>
          <div class="directory-chips-selector">
            @for (dir of commonFolders; track dir) {
              <button 
                type="button" 
                class="dir-chip-btn" 
                [class.active]="selectedFolders().includes(dir)"
                (click)="toggleFolder(dir)"
              >
                <svg lucideFolder class="w-3.5 h-3.5 mr-1.5"></svg>
                <span>{{ dir }}</span>
                <span class="chip-status-mark">{{ selectedFolders().includes(dir) ? '✓' : '+' }}</span>
              </button>
            }
          </div>
          <small class="form-hint">
            Carpetas que se crearán de forma automática en <code>/home/workspace</code> cuando el alumno inicie por primera vez.
          </small>
        </div>

        <!-- Modo Avanzado: Scripts adicionales de arranque -->
        <div class="advanced-script-toggle-row mb-2">
          <button 
            type="button" 
            class="btn-toggle-advanced-script"
            (click)="showAdvancedScript.set(!showAdvancedScript())"
            [attr.aria-expanded]="showAdvancedScript()"
          >
            <svg lucideChevronDown class="w-3.5 h-3.5 mr-1" [class.rotated]="showAdvancedScript()"></svg>
            <span>{{ showAdvancedScript() ? 'Ocultar comandos de terminal de arranque' : 'Comandos de arranque del sistema (Opcional - Avanzado)' }}</span>
          </button>
        </div>

        @if (showAdvancedScript()) {
          <div class="advanced-script-panel animate-fade">
            <div class="label-with-popover-row">
              <label class="form-label font-semibold" for="setup-script-textarea">
                Script de inicialización adicional (Bash):
              </label>
              <button 
                type="button" 
                class="btn-contract-info" 
                (click)="toggleContractPopover()"
                [attr.aria-expanded]="showContractPopover()"
                aria-label="Ver detalles de los comandos de inicialización"
              >
                <svg lucideHelpCircle class="w-3.5 h-3.5 mr-1"></svg>
                <span>¿Cómo funciona?</span>
              </button>
            </div>

            <!-- Popover del Contrato de Inicialización para IDE -->
            @if (showContractPopover()) {
              <div class="contract-popover animate-fade">
                <div class="popover-header">
                  <strong class="popover-title" i18n="@@AY-22">Comandos de inicialización del entorno</strong>
                  <button type="button" class="btn-popover-close" (click)="closeContractPopover()" aria-label="Cerrar">
                    <svg lucideX class="w-3.5 h-3.5"></svg>
                  </button>
                </div>
                <ul class="contract-list">
                  <li><strong>Momento de ejecución:</strong> Se ejecutan en segundo plano una sola vez cuando el alumno abre su laboratorio por primera vez.</li>
                  <li><strong>Sin interrupción:</strong> El editor web abre de inmediato sin demoras; los comandos corren en segundo plano.</li>
                  <li><strong>Ubicación:</strong> Se aplican en la carpeta de trabajo del alumno (<code>/home/workspace</code>).</li>
                  <li><strong>Idempotencia:</strong> Escriba comandos que puedan re-ejecutarse con seguridad (ej. <code>mkdir -p</code> o comprobar si el archivo ya existe).</li>
                </ul>
              </div>
            }

            <div class="textarea-wrapper">
              <textarea 
                id="setup-script-textarea"
                class="form-control font-mono text-area-sm" 
                rows="4" 
                spellcheck="false"
                placeholder="# Comandos bash opcionales de sistema..." 
                [ngModel]="setupScript()" 
                (ngModelChange)="setupScriptChange.emit($event)"
              ></textarea>
              @if (toastWarning(); as warn) {
                <div class="inline-toast-warning animate-fade" role="alert">
                  <svg lucideAlertCircle class="w-3.5 h-3.5 mr-1.5 shrink-0"></svg>
                  <span i18n="@@AY-24">{{ warn }}</span>
                </div>
              }
            </div>
          </div>
        }
      }
    </div>
  `,
  styleUrls: ['./step-execution.component.scss']
})
export class StepExecutionComponent {
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
  showAdvancedScript = signal<boolean>(false);
  hoveredIdeExample = signal<ScriptExample | null>(null);
  hoveredJudgeExample = signal<EntrypointExample | null>(null);
  appliedIdeChip = signal<string | null>(null);
  appliedJudgeChip = signal<string | null>(null);
  toastWarning = signal<string | null>(null);
  private warningTimeoutId: ReturnType<typeof setTimeout> | null = null;
  private appliedIdeTimeoutId: ReturnType<typeof setTimeout> | null = null;
  private appliedJudgeTimeoutId: ReturnType<typeof setTimeout> | null = null;

  commonFolders: string[] = ['src', 'backend', 'frontend', 'data', 'docs', 'tests'];
  selectedFolders = signal<string[]>([]);

  toggleFolder(folder: string): void {
    const current = this.selectedFolders();
    const updated = current.includes(folder) 
      ? current.filter(f => f !== folder)
      : [...current, folder];
    this.selectedFolders.set(updated);
    this.syncFoldersToScript(updated);
  }

  private syncFoldersToScript(folders: string[]): void {
    const currentScript = this.setupScript();
    const cleanLines = currentScript
      .split('\n')
      .filter(line => !line.startsWith('mkdir -p /home/workspace/'))
      .join('\n')
      .trim();

    if (folders.length === 0) {
      this.setupScriptChange.emit(cleanLines);
      return;
    }

    const paths = folders.map(f => `/home/workspace/${f}`).join(' ');
    const mkdirCommand = `mkdir -p ${paths}`;
    const newScript = cleanLines ? `${mkdirCommand}\n${cleanLines}` : mkdirCommand;
    this.setupScriptChange.emit(newScript);
  }

  ideExamples: ScriptExample[] = [
    {
      label: 'Instalar dependencias',
      desc: 'Comprueba package.json o requirements.txt e instala dependencias',
      code: '# Instalar dependencias si existen manifiestos\nif [ -f "package.json" ]; then\n  npm install --prefer-offline\nelif [ -f "requirements.txt" ]; then\n  pip install --no-cache-dir -r requirements.txt\nfi'
    },
    {
      label: 'Crear directorios',
      desc: 'Crea estructura de directorios de trabajo de forma idempotente',
      code: '# Crear estructura de directorios de trabajo\nmkdir -p /home/workspace/src /home/workspace/build /home/workspace/data /home/workspace/logs'
    },
    {
      label: 'Cargar datos',
      desc: 'Descarga archivos iniciales de prueba solo si no existen previamente',
      code: '# Cargar datos de prueba iniciales de forma idempotente\nif [ ! -f "data/dataset.csv" ]; then\n  mkdir -p data\n  curl -sSL "https://raw.githubusercontent.com/datasets/sample.csv" -o data/dataset.csv\nfi'
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

  showIdePreview(ex: ScriptExample): void {
    this.hoveredIdeExample.set(ex);
  }

  hideIdePreview(): void {
    this.hoveredIdeExample.set(null);
  }

  showJudgePreview(ex: EntrypointExample): void {
    this.hoveredJudgeExample.set(ex);
  }

  hideJudgePreview(): void {
    this.hoveredJudgeExample.set(null);
  }

  applyScriptExample(ex: ScriptExample): void {
    const current = this.setupScript();
    const marker = `# --- ${ex.label} ---`;
    
    if (current.includes(marker) || current.includes(ex.code)) {
      this.showToastWarning('La plantilla ya está insertada en el editor');
      return;
    }

    const snippetWithMarker = `${marker}\n${ex.code}`;
    const trimmed = current.trim();
    if (!trimmed) {
      this.setupScriptChange.emit(snippetWithMarker);
    } else {
      this.setupScriptChange.emit(`${trimmed}\n\n${snippetWithMarker}`);
    }

    // Feedback visual momentáneo
    this.appliedIdeChip.set(ex.label);
    if (this.appliedIdeTimeoutId) clearTimeout(this.appliedIdeTimeoutId);
    this.appliedIdeTimeoutId = setTimeout(() => {
      this.appliedIdeChip.set(null);
      this.appliedIdeTimeoutId = null;
    }, 2000);
  }

  applyEntrypointExample(ex: EntrypointExample | string): void {
    const cmd = typeof ex === 'string' ? ex : ex.cmd;
    const label = typeof ex === 'string' ? ex : ex.label;
    this.entrypointChange.emit(cmd);
    this.appliedJudgeChip.set(label);
    if (this.appliedJudgeTimeoutId) clearTimeout(this.appliedJudgeTimeoutId);
    this.appliedJudgeTimeoutId = setTimeout(() => {
      this.appliedJudgeChip.set(null);
      this.appliedJudgeTimeoutId = null;
    }, 2000);
  }

  private showToastWarning(msg: string): void {
    this.toastWarning.set(msg);
    if (this.warningTimeoutId) {
      clearTimeout(this.warningTimeoutId);
    }
    this.warningTimeoutId = setTimeout(() => {
      this.toastWarning.set(null);
      this.warningTimeoutId = null;
    }, 3000);
  }
}

export { StepExecutionComponent as SolvStepExecutionComponent };
