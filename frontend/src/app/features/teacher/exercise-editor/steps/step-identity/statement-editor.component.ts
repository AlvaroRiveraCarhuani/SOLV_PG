import { Component, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { 
  LucideEdit3, 
  LucideEye, 
  LucideCode, 
  LucideBold, 
  LucideItalic, 
  LucideSparkles, 
  LucideColumns, 
  LucideSigma 
} from '@lucide/angular';
import { marked } from 'marked';
import { ExerciseEditorStore } from '../../exercise-editor.store';

@Component({
  selector: 'statement-editor',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideEdit3,
    LucideEye,
    LucideCode,
    LucideBold,
    LucideItalic,
    LucideSparkles,
    LucideColumns,
    LucideSigma
  ],
  template: `
    <div class="statement-editor-container">
      <div class="editor-header">
        <div class="header-left">
          <label class="section-label">Enunciado del Ejercicio (Markdown)</label>
          <span class="section-hint">Describe el problema, especificaciones de entrada/salida y restricciones</span>
        </div>

        <div class="header-actions">
          <button 
            type="button" 
            class="btn-outline example-btn"
            (click)="insertExamplesFromCases()"
            [disabled]="store.exampleCasesCount() === 0"
            title="Inserta automáticamente los casos marcados como Ejemplo en el texto"
          >
            <svg lucideSparkles class="btn-icon"></svg>
            <span>Insertar ejemplos de casos ({{ store.exampleCasesCount() }})</span>
          </button>

          <div class="view-toggle-group">
            <button 
              type="button" 
              class="toggle-btn" 
              [class.active]="viewMode() === 'edit'"
              (click)="viewMode.set('edit')"
              title="Solo editor"
            >
              <svg lucideEdit3 class="toggle-icon"></svg>
            </button>
            <button 
              type="button" 
              class="toggle-btn" 
              [class.active]="viewMode() === 'split'"
              (click)="viewMode.set('split')"
              title="Vista dividida (Editor + Previsualización)"
            >
              <svg lucideColumns class="toggle-icon"></svg>
            </button>
            <button 
              type="button" 
              class="toggle-btn" 
              [class.active]="viewMode() === 'preview'"
              (click)="viewMode.set('preview')"
              title="Solo previsualización"
            >
              <svg lucideEye class="toggle-icon"></svg>
            </button>
          </div>
        </div>
      </div>

      <!-- Barra de Herramientas de Formato Markdown -->
      @if (viewMode() !== 'preview') {
        <div class="formatting-toolbar">
          <button type="button" class="tool-btn" (click)="wrapSelection('**', '**')" title="Negrita">
            <svg lucideBold class="tool-icon"></svg>
          </button>
          <button type="button" class="tool-btn" (click)="wrapSelection('*', '*')" title="Cursiva">
            <svg lucideItalic class="tool-icon"></svg>
          </button>
          <button type="button" class="tool-btn" (click)="wrapSelection('\`', '\`')" title="Código en línea">
            <svg lucideCode class="tool-icon"></svg>
          </button>
          <button type="button" class="tool-btn" (click)="insertCodeBlock()" title="Bloque de código">
            <span class="tool-text font-mono">&lt;/&gt;</span>
          </button>
          <button type="button" class="tool-btn" (click)="wrapSelection('$', '$')" title="Fórmula matemática (KaTeX)">
            <svg lucideSigma class="tool-icon"></svg>
          </button>
          <div class="toolbar-divider"></div>
          <button type="button" class="tool-btn text-tool" (click)="insertTemplateSection('input')" title="Sección Entrada">
            + Entrada
          </button>
          <button type="button" class="tool-btn text-tool" (click)="insertTemplateSection('output')" title="Sección Salida">
            + Salida
          </button>
          <button type="button" class="tool-btn text-tool" (click)="insertTemplateSection('constraints')" title="Sección Restricciones">
            + Restricciones
          </button>
        </div>
      }

      <!-- Contenedor Principal de Edición y Visualización -->
      <div class="editor-workspace" [class]="'mode-' + viewMode()">
        <!-- Panel Editor -->
        @if (viewMode() === 'edit' || viewMode() === 'split') {
          <div class="editor-pane">
            <textarea 
              #editorTextarea
              class="statement-textarea font-mono"
              placeholder="Escribe la descripción del problema en formato Markdown..."
              [ngModel]="store.statement().description"
              (ngModelChange)="onDescriptionChange($event)"
              rows="16"
            ></textarea>
          </div>
        }

        <!-- Panel Previsualización -->
        @if (viewMode() === 'preview' || viewMode() === 'split') {
          <div class="preview-pane">
            <div class="preview-header">
              <span class="preview-title">Previsualización de Enunciado</span>
            </div>
            <div class="rendered-markdown" [innerHTML]="renderedContent()"></div>
          </div>
        }
      </div>

      @if (store.step1Validation().errors.includes('La descripción del enunciado es obligatoria')) {
        <span class="field-error">La descripción del problema es obligatoria</span>
      }
    </div>
  `,
  styleUrl: './statement-editor.component.scss'
})
export class StatementEditorComponent {
  readonly store = inject(ExerciseEditorStore);

  viewMode = signal<'edit' | 'split' | 'preview'>('split');

  renderedContent = computed(() => {
    const raw = this.store.statement().description;
    if (!raw.trim()) {
      return '<p class="empty-preview-text">El enunciado está vacío. Escribe algo en el editor para previsualizarlo.</p>';
    }
    try {
      return marked.parse(raw, { async: false }) as string;
    } catch {
      return `<pre class="fallback-preview">${raw}</pre>`;
    }
  });

  onDescriptionChange(description: string) {
    this.store.updateStatement({ description });
  }

  insertExamplesFromCases() {
    const exampleCases = this.store.cases().filter(
      c => c.visibility === 'example' || (!c.visibility && c.is_sample)
    );

    if (exampleCases.length === 0) return;

    let markdownExamples = '\n\n## Ejemplos de Entrada y Salida\n\n';
    exampleCases.forEach((tc, idx) => {
      markdownExamples += `### Ejemplo ${idx + 1}\n\n`;
      markdownExamples += `**Entrada:**\n\`\`\`text\n${tc.input || '(vacío)'}\n\`\`\`\n\n`;
      markdownExamples += `**Salida esperada:**\n\`\`\`text\n${tc.expected_output}\n\`\`\`\n\n`;
    });

    const current = this.store.statement().description;
    this.store.updateStatement({ description: current + markdownExamples });
  }

  wrapSelection(prefix: string, suffix: string) {
    const current = this.store.statement().description;
    this.store.updateStatement({ description: current + ` ${prefix}texto${suffix} ` });
  }

  insertCodeBlock() {
    const block = '\n```python\n# Escribe aquí el código de ejemplo\n```\n';
    const current = this.store.statement().description;
    this.store.updateStatement({ description: current + block });
  }

  insertTemplateSection(type: 'input' | 'output' | 'constraints') {
    let snippet = '';
    if (type === 'input') {
      snippet = '\n\n### Formato de Entrada\n- La primera línea contiene un número entero $N$.\n- La segunda línea contiene $N$ enteros separados por espacios.\n';
    } else if (type === 'output') {
      snippet = '\n\n### Formato de Salida\n- Imprime la respuesta solicitada en una sola línea.\n';
    } else if (type === 'constraints') {
      snippet = '\n\n### Restricciones\n- $1 \\le N \\le 10^5$\n- $-10^9 \\le A_i \\le 10^9$\n- Tiempo límite: 1.0s\n';
    }

    const current = this.store.statement().description;
    this.store.updateStatement({ description: current + snippet });
  }
}
