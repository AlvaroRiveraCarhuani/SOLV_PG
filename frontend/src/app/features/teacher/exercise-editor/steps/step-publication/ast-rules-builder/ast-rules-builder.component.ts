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
  category: 'loops' | 'builtins' | 'libraries';
  rule: ASTCustomRuleDTO;
}

export const RESTRICTION_PRESETS: RulePreset[] = [
  {
    id: 'py_sort',
    name: 'Prohibir sort() / sorted()',
    description: 'Obliga al estudiante a programar su propio algoritmo de ordenamiento manual.',
    category: 'builtins',
    rule: {
      language: 'python',
      type: 'method',
      pattern: 'sort',
      message: 'No está permitido usar sort() o sorted() nativos. Debes implementar el algoritmo de ordenamiento solicitado.'
    }
  },
  {
    id: 'py_loops',
    name: 'Prohibir bucles (for / while)',
    description: 'Fuerza a resolver el problema usando recursión pura.',
    category: 'loops',
    rule: {
      language: 'python',
      type: 'syntax',
      pattern: 'for,while',
      message: 'Este ejercicio debe resolverse mediante recursión. El uso de bucles for y while está restringido.'
    }
  },
  {
    id: 'py_collections',
    name: 'Prohibir librerías avanzadas (collections / math)',
    description: 'Bloquea estructuras automáticas (Counter, deque) para exigir lógica base.',
    category: 'libraries',
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
    description: 'Bloquea std::sort de <algorithm> para evaluar ordenamiento propio.',
    category: 'builtins',
    rule: {
      language: 'cpp',
      type: 'function',
      pattern: 'std::sort',
      message: 'No se permite invocar std::sort en este ejercicio. Implementa tu propio ordenamiento manual.'
    }
  },
  {
    id: 'java_sort',
    name: 'Java: Prohibir Arrays.sort / Collections.sort',
    description: 'Bloquea métodos utilitarios de ordenamiento en Java.',
    category: 'builtins',
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
  readonly presets = RESTRICTION_PRESETS;

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
