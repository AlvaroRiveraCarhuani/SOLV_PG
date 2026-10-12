import { Component, computed, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { 
  LucideCode2, 
  LucideRotateCcw, 
  LucideFileCode, 
  LucideSparkles,
  LucideFileQuestion,
  LucideCheck,
  LucideLayers
} from '@lucide/angular';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { ExerciseEditorStore } from '../../../exercise-editor.store';

export const STANDARD_IO_BOILERPLATES: Record<string, string> = {
  python: `# Lectura de entrada estándar (stdin) y salida (stdout)
import sys

def solve():
    # Lee todos los datos de entrada
    lines = sys.stdin.read().splitlines()
    if not lines:
        return
    
    # TODO: Procesa la entrada y resuelve el problema
    # resultado = ...
    # print(resultado)

if __name__ == '__main__':
    solve()
`,
  cpp: `#include <iostream>
#include <vector>
#include <string>

using namespace std;

int main() {
    // Optimización de I/O estándar para juez competitivo
    ios_base::sync_with_stdio(false);
    cin.tie(NULL);

    // TODO: Lee de cin y escribe el resultado en cout
    
    return 0;
}
`,
  java: `import java.util.Scanner;

public class Solution {
    public static void main(String[] args) {
        Scanner scanner = new Scanner(System.in);
        
        // TODO: Procesa la entrada y muestra el resultado en System.out
        
        scanner.close();
    }
}
`,
  csharp: `using System;

class Program {
    static void Main(string[] args) {
        // TODO: Procesa la entrada (Console.ReadLine) y escribe el resultado (Console.WriteLine)
    }
}
`,
  javascript: `const fs = require('fs');

function solve() {
    const input = fs.readFileSync(0, 'utf-8').trim();
    if (!input) return;

    // TODO: Procesa la entrada y usa console.log para el resultado
}

solve();
`,
  typescript: `import * as fs from 'fs';

function solve(): void {
    const input = fs.readFileSync(0, 'utf-8').trim();
    if (!input) return;

    // TODO: Procesa la entrada y muestra el resultado con console.log
}

solve();
`
};

export const FUNCTION_SIGNATURE_BOILERPLATES: Record<string, string> = {
  python: `def solve(*args):
    """
    TODO: Implementa la solución al problema.
    """
    pass
`,
  cpp: `// Completa la función solicitada
int solve() {
    // TODO: Escribe tu lógica aquí
    return 0;
}
`,
  java: `public class Solution {
    public int solve() {
        // TODO: Escribe tu lógica aquí
        return 0;
    }
}
`,
  csharp: `public class Solution {
    public int Solve() {
        // TODO: Escribe tu lógica aquí
        return 0;
    }
}
`,
  javascript: `/**
 * @return {any}
 */
function solve() {
    // TODO: Escribe tu lógica aquí
}
`,
  typescript: `function solve(): any {
    // TODO: Escribe tu lógica aquí
}
`
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
    LucideSparkles,
    LucideFileQuestion,
    LucideCheck,
    LucideLayers,
    MachineDataDirective
  ],
  templateUrl: './boilerplate-tabs.component.html',
  styleUrl: './boilerplate-tabs.component.scss'
})
export class BoilerplateTabsComponent implements OnInit {
  readonly store = inject(ExerciseEditorStore);

  readonly activeLanguage = signal<string>('python');
  readonly actionNotice = signal<string | null>(null);

  readonly availableLanguages = computed(() => {
    const meta = this.store.metadata();
    const allowed = meta.allowed_languages || [];
    if (allowed.length > 0) return allowed;
    return [meta.language || 'python'];
  });

  readonly hasReferenceSolution = computed(() => {
    return !!this.store.referenceSolution().trim();
  });

  readonly currentCode = computed(() => {
    const lang = this.activeLanguage();
    const map = this.store.boilerplates();
    if (map[lang] !== undefined) {
      return map[lang];
    }
    if (lang === this.store.metadata().language && this.store.boilerplate()) {
      return this.store.boilerplate();
    }
    return STANDARD_IO_BOILERPLATES[lang] || `// Código inicial para ${lang}\n`;
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

  applyStandardIO(): void {
    const lang = this.activeLanguage();
    const template = STANDARD_IO_BOILERPLATES[lang] || `// Código inicial para ${lang}\n`;
    this.store.setBoilerplateForLanguage(lang, template);
    this.flashNotice('Plantilla estándar I/O aplicada');
  }

  applyFunctionSignature(): void {
    const lang = this.activeLanguage();
    const template = FUNCTION_SIGNATURE_BOILERPLATES[lang] || `// Función inicial para ${lang}\n`;
    this.store.setBoilerplateForLanguage(lang, template);
    this.flashNotice('Plantilla de función aplicada');
  }

  applyBlank(): void {
    const lang = this.activeLanguage();
    this.store.setBoilerplateForLanguage(lang, '');
    this.flashNotice('Lienzo en blanco configurado');
  }

  generateFromReference(): void {
    const ref = this.store.referenceSolution().trim();
    if (!ref) return;

    const lang = this.activeLanguage();
    const derived = this.deriveSkeletonFromCode(ref, lang);
    this.store.setBoilerplateForLanguage(lang, derived);
    this.flashNotice('Plantilla derivada desde la solución oficial');
  }

  private deriveSkeletonFromCode(code: string, lang: string): string {
    const lines = code.split('\n');
    const headerLines: string[] = [];

    // Conservar importaciones / includes y firmas iniciales
    for (const line of lines) {
      const trimmed = line.trim();
      if (
        trimmed.startsWith('import ') ||
        trimmed.startsWith('from ') ||
        trimmed.startsWith('#include') ||
        trimmed.startsWith('using ') ||
        trimmed.startsWith('package ')
      ) {
        headerLines.push(line);
      } else if (trimmed.startsWith('def ') || trimmed.startsWith('function ') || trimmed.startsWith('public class ') || trimmed.startsWith('int main(')) {
        headerLines.push(line);
        break;
      }
    }

    if (headerLines.length === 0) {
      return STANDARD_IO_BOILERPLATES[lang] || code;
    }

    if (lang === 'python') {
      return `${headerLines.join('\n')}\n    # TODO: Escribe tu solución aquí\n    pass\n`;
    }

    return `${headerLines.join('\n')}\n    // TODO: Escribe tu solución aquí\n}\n`;
  }

  private flashNotice(msg: string): void {
    this.actionNotice.set(msg);
    setTimeout(() => {
      if (this.actionNotice() === msg) {
        this.actionNotice.set(null);
      }
    }, 3000);
  }
}

