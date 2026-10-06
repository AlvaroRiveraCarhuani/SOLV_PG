import { Component, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import {
  LucideShieldAlert,
  LucidePlus,
  LucideTrash2,
  LucideSparkles,
  LucideLayers
} from '@lucide/angular';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { ExerciseEditorStore } from '../../../exercise-editor.store';
import { ASTCustomRuleDTO } from '../../../../models/teacher.models';

export interface RulePreset {
  id: string;
  name: string;
  description: string;
  rule: ASTCustomRuleDTO;
}

export const AST_PRESETS: RulePreset[] = [
  {
    id: 'py_sort',
    name: 'Python: Prohibir sort() / sorted()',
    description: 'Bloquea el uso de ordenamiento incorporado de Python.',
    rule: {
      language: 'python',
      type: 'method',
      pattern: 'sort',
      message: 'No está permitido usar sort() o sorted() nativos. Debes implementar el algoritmo de ordenamiento solicitado.'
    }
  },
  {
    id: 'py_collections',
    name: 'Python: Prohibir collections',
    description: 'Bloquea estructuras avanzadas como Counter o deque.',
    rule: {
      language: 'python',
      type: 'module',
      pattern: 'collections',
      message: 'El módulo collections está restringido para este ejercicio. Utiliza tipos y estructuras básicas.'
    }
  },
  {
    id: 'cpp_sort',
    name: 'C++: Prohibir std::sort',
    description: 'Bloquea std::sort de la librería <algorithm>.',
    rule: {
      language: 'cpp',
      type: 'function',
      pattern: 'std::sort',
      message: 'No se permite invocar std::sort en este ejercicio. Implementa tu propio ordenamiento manual.'
    }
  },
  {
    id: 'java_sort',
    name: 'Java: Prohibir Arrays.sort',
    description: 'Bloquea métodos de ordenamiento estáticos de Java.',
    rule: {
      language: 'java',
      type: 'method',
      pattern: 'Arrays.sort',
      message: 'El uso de Arrays.sort o Collections.sort no está permitido para este ejercicio.'
    }
  }
];

@Component({
  selector: 'ast-rules-builder',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideShieldAlert,
    LucidePlus,
    LucideTrash2,
    LucideSparkles,
    LucideLayers,
    MachineDataDirective
  ],
  templateUrl: './ast-rules-builder.component.html',
  styleUrl: './ast-rules-builder.component.scss'
})
export class ASTRulesBuilderComponent {
  readonly store = inject(ExerciseEditorStore);
  readonly presets = AST_PRESETS;

  readonly newRule = signal<ASTCustomRuleDTO>({
    language: 'python',
    type: 'method',
    pattern: '',
    message: ''
  });

  onToggleNativeSort(enabled: boolean): void {
    this.store.updateAstRules({ block_native_sort: enabled });
  }

  onToggleSystemModules(enabled: boolean): void {
    this.store.updateAstRules({ block_system_modules: enabled });
  }

  applyPreset(preset: RulePreset): void {
    const existing = this.store.astRules().custom_rules || [];
    const duplicate = existing.some(
      r => r.language === preset.rule.language && r.pattern === preset.rule.pattern
    );
    if (!duplicate) {
      this.store.addCustomAstRule({ ...preset.rule });
    }
  }

  addRule(): void {
    const r = this.newRule();
    if (!r.pattern.trim()) return;

    this.store.addCustomAstRule({
      language: r.language,
      type: r.type,
      pattern: r.pattern.trim(),
      message: r.message.trim() || `El uso del patrón "${r.pattern.trim()}" está restringido en este ejercicio.`
    });

    // Reset form
    this.newRule.set({
      language: r.language,
      type: 'method',
      pattern: '',
      message: ''
    });
  }

  removeRule(index: number): void {
    this.store.removeCustomAstRule(index);
  }
}
