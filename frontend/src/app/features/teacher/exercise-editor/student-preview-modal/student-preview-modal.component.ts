import { Component, EventEmitter, Input, Output, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import {
  LucideEye,
  LucideX,
  LucideCode,
  LucideTerminal,
  LucideFileText,
  LucideClock,
  LucideCpu
} from '@lucide/angular';
import { marked } from 'marked';
import katex from 'katex';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { ExerciseEditorStore } from '../exercise-editor.store';
import { TestCaseDTO } from '../../models/teacher.models';

function renderMarkdownWithKatex(raw: string): string {
  if (!raw || !raw.trim()) {
    return '<p class="empty-statement">No hay descripción para este ejercicio.</p>';
  }

  // 1. Block math: $$ ... $$
  let processed = raw.replace(/\$\$([\s\S]+?)\$\$/g, (_, math) => {
    try {
      return `<div class="katex-block-wrapper">${katex.renderToString(math.trim(), { displayMode: true, throwOnError: false })}</div>`;
    } catch {
      return `<pre class="katex-error">${math}</pre>`;
    }
  });

  // 2. Inline math: $ ... $
  processed = processed.replace(/\$([^\$\n]+?)\$/g, (_, math) => {
    try {
      return katex.renderToString(math.trim(), { displayMode: false, throwOnError: false });
    } catch {
      return `<code>${math}</code>`;
    }
  });

  try {
    return marked.parse(processed, { async: false }) as string;
  } catch {
    return `<pre class="fallback-preview">${raw}</pre>`;
  }
}

@Component({
  selector: 'student-preview-modal',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideEye,
    LucideX,
    LucideCode,
    LucideTerminal,
    LucideFileText,
    LucideClock,
    LucideCpu,
    MachineDataDirective
  ],
  templateUrl: './student-preview-modal.component.html',
  styleUrl: './student-preview-modal.component.scss'
})
export class StudentPreviewModalComponent {
  readonly store = inject(ExerciseEditorStore);

  @Input() isOpen = false;
  @Output() close = new EventEmitter<void>();

  readonly selectedLanguage = signal<string>('python');

  readonly allowedLanguages = computed(() => {
    const meta = this.store.metadata();
    if (meta.allowed_languages && meta.allowed_languages.length > 0) {
      return meta.allowed_languages;
    }
    return [meta.language || 'python'];
  });

  readonly currentBoilerplate = computed(() => {
    const lang = this.selectedLanguage();
    const map = this.store.boilerplates();
    if (map && map[lang]) {
      return map[lang];
    }
    if (lang === this.store.metadata().language && this.store.boilerplate()) {
      return this.store.boilerplate();
    }
    return this.getDefaultBoilerplate(lang);
  });

  readonly exampleCases = computed(() => {
    return this.store.cases().filter(
      (c: TestCaseDTO) => c.visibility === 'example' || (!c.visibility && c.is_sample)
    );
  });

  readonly renderedDescription = computed(() => {
    return renderMarkdownWithKatex(this.store.statement().description);
  });

  readonly renderedInputText = computed(() => {
    return this.store.statement().input_text
      ? renderMarkdownWithKatex(this.store.statement().input_text)
      : null;
  });

  readonly renderedOutputText = computed(() => {
    return this.store.statement().output_text
      ? renderMarkdownWithKatex(this.store.statement().output_text)
      : null;
  });

  readonly renderedConstraints = computed(() => {
    return this.store.statement().constraints
      ? renderMarkdownWithKatex(this.store.statement().constraints)
      : null;
  });

  constructor() {
    // Si cambia allowedLanguages o se inicializa, sincronizar el lenguaje seleccionado
    const langs = this.allowedLanguages();
    if (langs.length > 0) {
      this.selectedLanguage.set(langs[0]);
    }
  }

  onLanguageChange(lang: string): void {
    this.selectedLanguage.set(lang);
  }

  onClose(): void {
    this.close.emit();
  }

  onBackdropClick(event: MouseEvent): void {
    if ((event.target as HTMLElement).classList.contains('student-preview-backdrop')) {
      this.onClose();
    }
  }

  onKeyDown(event: KeyboardEvent): void {
    if (event.key === 'Escape') {
      this.onClose();
    }
  }

  private getDefaultBoilerplate(lang: string): string {
    switch (lang) {
      case 'python':
        return '# Escribe tu solución aquí\nimport sys\n\ndef main():\n    pass\n\nif __name__ == "__main__":\n    main()\n';
      case 'cpp':
        return '#include <iostream>\nusing namespace std;\n\nint main() {\n    ios_base::sync_with_stdio(false);\n    cin.tie(NULL);\n    \n    // Escribe tu solución aquí\n    return 0;\n}\n';
      case 'java':
        return 'import java.util.Scanner;\n\npublic class Solution {\n    public static void main(String[] args) {\n        Scanner scanner = new Scanner(System.in);\n        // Escribe tu solución aquí\n    }\n}\n';
      case 'c':
        return '#include <stdio.h>\n\nint main() {\n    // Escribe tu solución aquí\n    return 0;\n}\n';
      case 'javascript':
        return 'const fs = require("fs");\n\nfunction main() {\n    const input = fs.readFileSync(0, "utf-8").trim();\n    // Escribe tu solución aquí\n}\n\nmain();\n';
      default:
        return `// Código inicial para ${lang}\n`;
    }
  }
}
