import { Component, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import {
  LucideShieldAlert,
  LucidePlus,
  LucideTrash2,
  LucideBookOpen,
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
  // Python
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

  // C++
  {
    id: 'cpp_sort',
    name: 'Prohibir std::sort',
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
    id: 'cpp_loops',
    name: 'Prohibir bucles (for / while / do)',
    description: 'Fuerza a estructurar la solución con llamadas recursivas.',
    category: 'loops',
    rule: {
      language: 'cpp',
      type: 'syntax',
      pattern: 'for,while,do',
      message: 'El uso de estructuras de iteración repetitiva (for, while, do) no está permitido. Usa recursión.'
    }
  },

  // Java
  {
    id: 'java_sort',
    name: 'Prohibir Arrays.sort / Collections.sort',
    description: 'Bloquea métodos utilitarios de ordenamiento en Java.',
    category: 'builtins',
    rule: {
      language: 'java',
      type: 'method',
      pattern: 'Arrays.sort',
      message: 'El uso de Arrays.sort o Collections.sort no está permitido para este ejercicio.'
    }
  },
  {
    id: 'java_loops',
    name: 'Prohibir bucles (for / while / do)',
    description: 'Fuerza a resolver el ejercicio con recursión en Java.',
    category: 'loops',
    rule: {
      language: 'java',
      type: 'syntax',
      pattern: 'for,while,do',
      message: 'Este ejercicio requiere solución recursiva. Los bucles for y while están bloqueados.'
    }
  },

  // JavaScript / TypeScript
  {
    id: 'js_sort',
    name: 'Prohibir Array.prototype.sort',
    description: 'Impide el uso del método sort() nativo de arrays.',
    category: 'builtins',
    rule: {
      language: 'javascript',
      type: 'method',
      pattern: '.sort',
      message: 'El uso del método sort() está restringido. Implementa tu propio algoritmo de ordenamiento.'
    }
  },
  {
    id: 'js_loops',
    name: 'Prohibir bucles (for / while / forEach)',
    description: 'Fuerza a implementar recursión pura en JavaScript.',
    category: 'loops',
    rule: {
      language: 'javascript',
      type: 'syntax',
      pattern: 'for,while,forEach',
      message: 'No está permitido el uso de bucles. Resuelve el problema mediante recursión.'
    }
  },

  // C#
  {
    id: 'csharp_sort',
    name: 'Prohibir Array.Sort / LINQ OrderBy',
    description: 'Bloquea métodos utilitarios de ordenamiento en C#.',
    category: 'builtins',
    rule: {
      language: 'csharp',
      type: 'method',
      pattern: 'Array.Sort',
      message: 'El uso de Array.Sort o LINQ OrderBy no está permitido para este ejercicio.'
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
    LucideBookOpen,
    LucideLayers,
    MachineDataDirective
  ],
  templateUrl: './ast-rules-builder.component.html',
  styleUrl: './ast-rules-builder.component.scss'
})
export class ASTRulesBuilderComponent {
  readonly store = inject(ExerciseEditorStore);

  // Filtro reactivo según los lenguajes configurados para el ejercicio
  readonly relevantPresets = computed(() => {
    const meta = this.store.metadata();
    const activeLangs = new Set<string>();
    if (meta.language) activeLangs.add(meta.language.toLowerCase());
    for (const l of (meta.allowed_languages || [])) {
      activeLangs.add(l.toLowerCase());
    }

    if (activeLangs.size === 0) {
      return RESTRICTION_PRESETS;
    }

    return RESTRICTION_PRESETS.filter(p => activeLangs.has(p.rule.language.toLowerCase()));
  });

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
