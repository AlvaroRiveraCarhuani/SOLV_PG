import { Component, ElementRef, ViewChild, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import {
  LucideEdit3,
  LucideEye,
  LucideCode,
  LucideBold,
  LucideItalic,
  LucideListPlus,
  LucideColumns,
  LucideSigma,
  LucideSquareCode,
  LucideAlertCircle
} from '@lucide/angular';
import { marked } from 'marked';
import katex from 'katex';
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
    LucideListPlus,
    LucideColumns,
    LucideSigma,
    LucideSquareCode,
    LucideAlertCircle
  ],
  template: `
    <div class="statement-editor-container" id="statement-editor-section">
      <div class="editor-header">
        <div class="header-left">
          <label class="section-label">Enunciado del Ejercicio</label>
          <span class="section-hint">Describe el problema, especificaciones de entrada/salida y restricciones</span>
        </div>

        <div class="header-actions">
          @if (store.cases().length > 0) {
            <button
              type="button"
              class="btn-outline example-btn"
              (mousedown)="$event.preventDefault()"
              (click)="insertExamplesFromCases()"
              [disabled]="store.exampleCasesCount() === 0"
              [title]="exampleCasesTooltip()"
            >
              <svg lucideListPlus class="btn-icon"></svg>
              <span>Insertar ejemplos de casos ({{ store.exampleCasesCount() }})</span>
            </button>
          }

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

      <!-- Barra de Herramientas de Formato Markdown a nivel de Componente -->
      @if (viewMode() !== 'preview') {
        <div class="formatting-toolbar">
          <div class="toolbar-group">
            <button type="button" class="tool-btn" (mousedown)="$event.preventDefault()" (click)="wrapSelection('**', '**')" title="Negrita (Ctrl+B)">
              <svg lucideBold class="tool-icon"></svg>
            </button>
            <button type="button" class="tool-btn" (mousedown)="$event.preventDefault()" (click)="wrapSelection('*', '*')" title="Cursiva (Ctrl+I)">
              <svg lucideItalic class="tool-icon"></svg>
            </button>
            <button type="button" class="tool-btn" (mousedown)="$event.preventDefault()" (click)="wrapSelection('&#96;', '&#96;')" title="Código en línea">
              <svg lucideCode class="tool-icon"></svg>
            </button>
            <button type="button" class="tool-btn" (mousedown)="$event.preventDefault()" (click)="insertCodeBlock()" title="Bloque de código (Ctrl+K)">
              <svg lucideSquareCode class="tool-icon"></svg>
            </button>
            <button type="button" class="tool-btn" (mousedown)="$event.preventDefault()" (click)="wrapSelection('$', '$')" title="Fórmula matemática KaTeX ($fórmula$)">
              <svg lucideSigma class="tool-icon"></svg>
            </button>
          </div>

          <div class="toolbar-divider"></div>

          <div class="toolbar-group sections-group">
            <button type="button" class="tool-chip" (mousedown)="$event.preventDefault()" (click)="insertTemplateSection('input')" title="Insertar sección Entrada">
              + Entrada
            </button>
            <button type="button" class="tool-chip" (mousedown)="$event.preventDefault()" (click)="insertTemplateSection('output')" title="Insertar sección Salida">
              + Salida
            </button>
            <button type="button" class="tool-chip" (mousedown)="$event.preventDefault()" (click)="insertTemplateSection('constraints')" title="Insertar sección Restricciones">
              + Restricciones
            </button>
          </div>
        </div>
      }

      <!-- Contenedor Principal de Edición y Visualización -->
      <div class="editor-workspace" [class]="'mode-' + viewMode()">
        <!-- Panel Editor -->
        @if (viewMode() === 'edit' || viewMode() === 'split') {
          <div class="editor-pane">
            <textarea
              #editorTextarea
              id="statement-description"
              class="statement-textarea font-mono"
              placeholder="Escribe la descripción del problema en formato Markdown y fórmulas LaTeX ($x$)..."
              [ngModel]="store.statement().description"
              (ngModelChange)="onDescriptionChange($event)"
              (blur)="onDescriptionBlur()"
              (keydown)="onTextareaKeyDown($event)"
              (scroll)="onEditorScroll()"
              rows="18"
            ></textarea>

            <div class="editor-footer">
              <span class="editor-stats">
                {{ wordsCount() }} palabras · {{ charsCount() }} caracteres
              </span>
            </div>
          </div>
        }

        <!-- Panel Previsualización -->
        @if (viewMode() === 'preview' || viewMode() === 'split') {
          <div class="preview-pane">
            <div class="preview-header">
              <span class="preview-title">Previsualización en tiempo real</span>
            </div>
            <div
              #previewContainer
              class="rendered-markdown"
              [innerHTML]="renderedContent()"
            ></div>
          </div>
        }
      </div>

      @if (showValidationError()) {
        <div class="field-error-banner" role="alert">
          <svg lucideAlertCircle class="error-icon"></svg>
          <span>La descripción del problema es obligatoria</span>
        </div>
      }
    </div>
  `,
  styleUrl: './statement-editor.component.scss'
})
export class StatementEditorComponent {
  readonly store = inject(ExerciseEditorStore);

  @ViewChild('editorTextarea') editorTextarea?: ElementRef<HTMLTextAreaElement>;
  @ViewChild('previewContainer') previewContainer?: ElementRef<HTMLDivElement>;

  viewMode = signal<'edit' | 'split' | 'preview'>('split');
  isTouched = signal<boolean>(false);

  showValidationError = computed(() => {
    const hasError = this.store.step1Validation().errors.includes('La descripción del enunciado es obligatoria');
    const isEmpty = this.store.statement().description.trim() === '';
    const shouldShow = this.isTouched() || this.store.stepAttempted();
    return hasError && isEmpty && shouldShow;
  });

  wordsCount = computed(() => {
    const text = this.store.statement().description.trim();
    if (!text) return 0;
    return text.split(/\s+/).filter(Boolean).length;
  });

  charsCount = computed(() => {
    return this.store.statement().description.length;
  });

  exampleCasesTooltip = computed(() => {
    const count = this.store.exampleCasesCount();
    if (count === 0) {
      return 'Se habilitará cuando definas casos marcados como "Ejemplo" en el Paso 2 (Contrato de E/S)';
    }
    return `Inserta automáticamente los ${count} caso(s) de ejemplo en el enunciado`;
  });

  renderedContent = computed(() => {
    const raw = this.store.statement().description;
    if (!raw.trim()) {
      return '<div class="empty-preview-placeholder"><span>Vista previa del enunciado</span></div>';
    }

    // 1. Block math: $$ ... $$
    let processed = raw.replace(/\$\$([\s\S]+?)\$\$/g, (_, math) => {
      try {
        return `<div class="katex-block-wrapper">${katex.renderToString(math.trim(), { displayMode: true, throwOnError: false, output: 'mathml' })}</div>`;
      } catch {
        return `<pre class="katex-error">${math}</pre>`;
      }
    });

    // 2. Inline math: $ ... $
    processed = processed.replace(/\$([^\$\n]+?)\$/g, (_, math) => {
      try {
        return katex.renderToString(math.trim(), { displayMode: false, throwOnError: false, output: 'mathml' });
      } catch {
        return `<code>${math}</code>`;
      }
    });

    try {
      return marked.parse(processed, { async: false }) as string;
    } catch {
      return `<pre class="fallback-preview">${raw}</pre>`;
    }
  });

  onDescriptionBlur() {
    this.isTouched.set(true);
    this.store.markFieldTouched('description');
  }

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

    this.insertAtCursor(markdownExamples);
  }

  wrapSelection(prefix: string, suffix: string) {
    const textarea = this.editorTextarea?.nativeElement;
    const current = this.store.statement().description;

    if (!textarea) {
      this.store.updateStatement({ description: current + prefix + suffix });
      return;
    }

    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;
    const selectedText = current.substring(start, end);

    const replacement = prefix + selectedText + suffix;
    const updated = current.substring(0, start) + replacement + current.substring(end);
    this.store.updateStatement({ description: updated });

    setTimeout(() => {
      textarea.focus();
      if (selectedText.length > 0) {
        textarea.setSelectionRange(start, start + replacement.length);
      } else {
        const cursorPosition = start + prefix.length;
        textarea.setSelectionRange(cursorPosition, cursorPosition);
      }
    }, 0);
  }

  insertCodeBlock() {
    const textarea = this.editorTextarea?.nativeElement;
    const current = this.store.statement().description;

    if (!textarea) {
      const block = '\n```python\n# Código\n```\n';
      this.store.updateStatement({ description: current + block });
      return;
    }

    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;
    const selectedText = current.substring(start, end);

    const codeContent = selectedText.length > 0 ? selectedText : '# Código de ejemplo';
    const block = `\n\`\`\`python\n${codeContent}\n\`\`\`\n`;

    const updated = current.substring(0, start) + block + current.substring(end);
    this.store.updateStatement({ description: updated });

    setTimeout(() => {
      textarea.focus();
      const codeStart = start + '\n```python\n'.length;
      textarea.setSelectionRange(codeStart, codeStart + codeContent.length);
    }, 0);
  }

  onTextareaKeyDown(event: KeyboardEvent) {
    if (event.ctrlKey || event.metaKey) {
      if (event.key.toLowerCase() === 'b') {
        event.preventDefault();
        this.wrapSelection('**', '**');
      } else if (event.key.toLowerCase() === 'i') {
        event.preventDefault();
        this.wrapSelection('*', '*');
      } else if (event.key.toLowerCase() === 'k') {
        event.preventDefault();
        this.insertCodeBlock();
      }
    } else if (event.key === 'Tab') {
      event.preventDefault();
      this.insertAtCursor('  ');
    }
  }

  insertTemplateSection(type: 'input' | 'output' | 'constraints') {
    let header = '';
    if (type === 'input') {
      header = '### Formato de Entrada';
    } else if (type === 'output') {
      header = '### Formato de Salida';
    } else if (type === 'constraints') {
      header = '### Restricciones';
    }

    const textarea = this.editorTextarea?.nativeElement;
    const current = this.store.statement().description;

    if (!textarea) {
      this.store.updateStatement({ description: current + `\n\n${header}\n\n` });
      return;
    }

    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;

    // Prefijo inteligente: garantizar línea limpia antes del título
    let prefix = '';
    if (start > 0) {
      if (current.charAt(start - 1) !== '\n') {
        prefix = '\n\n';
      } else if (start > 1 && current.charAt(start - 2) !== '\n') {
        prefix = '\n';
      }
    }

    const snippet = `${prefix}${header}\n`;
    const updated = current.substring(0, start) + snippet + current.substring(end);
    this.store.updateStatement({ description: updated });

    setTimeout(() => {
      textarea.focus();
      const cursorTarget = start + snippet.length;
      textarea.setSelectionRange(cursorTarget, cursorTarget);
    }, 0);
  }

  private insertAtCursor(text: string) {
    const textarea = this.editorTextarea?.nativeElement;
    const current = this.store.statement().description;

    if (!textarea) {
      this.store.updateStatement({ description: current + text });
      return;
    }

    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;

    const updated = current.substring(0, start) + text + current.substring(end);
    this.store.updateStatement({ description: updated });

    setTimeout(() => {
      textarea.focus();
      const nextPos = start + text.length;
      textarea.setSelectionRange(nextPos, nextPos);
    }, 0);
  }

  onEditorScroll() {
    const textarea = this.editorTextarea?.nativeElement;
    const preview = this.previewContainer?.nativeElement;
    if (!textarea || !preview || this.viewMode() !== 'split') return;

    const scrollPercentage = textarea.scrollTop / (textarea.scrollHeight - textarea.clientHeight || 1);
    preview.scrollTop = scrollPercentage * (preview.scrollHeight - preview.clientHeight);
  }
}
