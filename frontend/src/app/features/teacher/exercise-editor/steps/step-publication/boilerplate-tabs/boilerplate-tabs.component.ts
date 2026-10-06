import { Component, computed, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { LucideCode2, LucideRotateCcw, LucideFileCode } from '@lucide/angular';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { ExerciseEditorStore } from '../../../exercise-editor.store';

export const DEFAULT_BOILERPLATES: Record<string, string> = {
  python: `# Solución inicial para el estudiante\n\ndef solve():\n    # Escribe tu código aquí\n    pass\n\nif __name__ == '__main__':\n    solve()\n`,
  cpp: `#include <iostream>\nusing namespace std;\n\nint main() {\n    // Escribe tu solución aquí\n    \n    return 0;\n}\n`,
  java: `import java.util.Scanner;\n\npublic class Solution {\n    public static void main(String[] args) {\n        // Escribe tu solución aquí\n    }\n}\n`,
  csharp: `using System;\n\nclass Program {\n    static void Main(string[] args) {\n        // Escribe tu solución aquí\n    }\n}\n`,
  javascript: `// Escribe tu solución aquí\nconst fs = require('fs');\n`
};

@Component({
  selector: 'boilerplate-tabs',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideCode2,
    LucideRotateCcw,
    LucideFileCode,
    MachineDataDirective
  ],
  templateUrl: './boilerplate-tabs.component.html',
  styleUrl: './boilerplate-tabs.component.scss'
})
export class BoilerplateTabsComponent implements OnInit {
  readonly store = inject(ExerciseEditorStore);

  readonly activeLanguage = signal<string>('python');

  readonly availableLanguages = computed(() => {
    const meta = this.store.metadata();
    const allowed = meta.allowed_languages || [];
    if (allowed.length > 0) return allowed;
    return [meta.language || 'python'];
  });

  readonly currentCode = computed(() => {
    const lang = this.activeLanguage();
    const map = this.store.boilerplates();
    if (map[lang] !== undefined) {
      return map[lang];
    }
    // Si coincide con el language principal y hay store.boilerplate()
    if (lang === this.store.metadata().language && this.store.boilerplate()) {
      return this.store.boilerplate();
    }
    return DEFAULT_BOILERPLATES[lang] || `// Código inicial para ${lang}\n`;
  });

  readonly lineCount = computed(() => {
    const code = this.currentCode();
    if (!code) return 0;
    return code.split('\n').length;
  });

  readonly charCount = computed(() => this.currentCode().length);

  ngOnInit(): void {
    const defaultLang = this.store.metadata().language || 'python';
    this.activeLanguage.set(defaultLang);
  }

  selectLanguage(lang: string): void {
    this.activeLanguage.set(lang);
  }

  onCodeChange(val: string): void {
    const lang = this.activeLanguage();
    this.store.setBoilerplateForLanguage(lang, val);
  }

  resetToDefault(): void {
    const lang = this.activeLanguage();
    const template = DEFAULT_BOILERPLATES[lang] || `// Código inicial para ${lang}\n`;
    this.store.setBoilerplateForLanguage(lang, template);
  }
}
