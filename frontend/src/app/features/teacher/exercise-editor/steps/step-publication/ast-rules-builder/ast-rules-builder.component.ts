import { Component, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import {
  LucideShieldAlert,
  LucidePlus,
  LucideTrash2,
  LucideLayers,
  LucideSlidersHorizontal
} from '@lucide/angular';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { ExerciseEditorStore } from '../../../exercise-editor.store';
import { ASTCustomRuleDTO } from '../../../../models/teacher.models';

export interface PedagogicalPolicy {
  key: string;
  title: string;
  description: string;
  isApplied: () => boolean;
  toggle: (checked: boolean) => void;
}

@Component({
  selector: 'ast-rules-builder',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideShieldAlert,
    LucidePlus,
    LucideTrash2,
    LucideLayers,
    LucideSlidersHorizontal,
    MachineDataDirective
  ],
  templateUrl: './ast-rules-builder.component.html',
  styleUrl: './ast-rules-builder.component.scss'
})
export class ASTRulesBuilderComponent {
  readonly store = inject(ExerciseEditorStore);

  readonly activeLanguage = computed(() => {
    return this.store.metadata().language || 'python';
  });

  // Políticas pedagógicas de cátedra estilo CodeGrade (universales, independientes del lenguaje)
  readonly policies = computed<PedagogicalPolicy[]>(() => {
    const lang = this.activeLanguage();
    const rules = this.store.astRules();

    return [
      {
        key: 'sort',
        title: 'Exigir Algoritmo de Ordenamiento Propio',
        description: 'Bloquea el uso de ordenamiento nativo en el lenguaje activo (sort / sorted / std::sort / Arrays.sort).',
        isApplied: () => !!rules.block_native_sort,
        toggle: (checked: boolean) => {
          this.store.updateAstRules({ block_native_sort: checked });
        }
      },
      {
        key: 'recursion',
        title: 'Forzar Solución Recursiva Pura',
        description: 'Bloquea estructuras de repetición iterativas (for, while, do). Obliga al estudiante a resolver con recursión pura.',
        isApplied: () => {
          return (rules.custom_rules || []).some(
            r => r.type === 'syntax' && (r.pattern.includes('for') || r.pattern.includes('while'))
          );
        },
        toggle: (checked: boolean) => {
          const current = rules.custom_rules || [];
          if (checked) {
            const pattern = lang === 'python' ? 'for,while' : 'for,while,do';
            const exists = current.some(r => r.type === 'syntax' && r.pattern === pattern);
            if (!exists) {
              this.store.addCustomAstRule({
                language: lang,
                type: 'syntax',
                pattern,
                message: 'Este ejercicio requiere resolución recursiva. Los bucles for y while están bloqueados.'
              });
            }
          } else {
            const filtered = current.filter(
              r => !(r.type === 'syntax' && (r.pattern.includes('for') || r.pattern.includes('while')))
            );
            this.store.updateAstRules({ custom_rules: filtered });
          }
        }
      },
      {
        key: 'system',
        title: 'Restringir Módulos del Sistema Operativo',
        description: 'Bloquea llamadas a comandos del sistema, procesos o sockets (os, sys, subprocess, etc.).',
        isApplied: () => !!rules.block_system_modules,
        toggle: (checked: boolean) => {
          this.store.updateAstRules({ block_system_modules: checked });
        }
      }
    ];
  });

  // Reglas adicionales específicas agregadas manualmente
  readonly customSpecificRules = computed(() => {
    const all = this.store.astRules().custom_rules || [];
    return all.filter(r => !(r.type === 'syntax' && (r.pattern.includes('for') || r.pattern.includes('while'))));
  });

  readonly newRule = signal<ASTCustomRuleDTO>({
    language: 'python',
    type: 'method',
    pattern: '',
    message: ''
  });

  addSpecificRule(): void {
    const r = this.newRule();
    const raw = r.pattern.trim();
    if (!raw) return;

    // Permitir ingresar múltiples funciones/métodos separados por comas (ej. pop, reverse, math)
    const tokens = raw
      .split(',')
      .map(t => t.trim())
      .filter(Boolean);

    const existing = this.activeSpecificRules();
    const lang = r.language || this.activeLanguage();

    for (const pattern of tokens) {
      // Inferencia simple: si contiene puntos es método/módulo, si no función
      const inferredType = pattern.includes('.') ? 'method' : 'function';

      // Evitar duplicados del mismo patrón en el lenguaje
      const alreadyExists = existing.some(
        existingRule => existingRule.pattern.toLowerCase() === pattern.toLowerCase() && existingRule.language === lang
      );
      if (alreadyExists) continue;

      this.store.addCustomAstRule({
        language: lang,
        type: inferredType,
        pattern,
        message: r.message.trim() || `El uso de "${pattern}" no está permitido para este ejercicio.`
      });
    }

    this.newRule.set({
      language: this.activeLanguage(),
      type: 'function',
      pattern: '',
      message: ''
    });
  }


  removeSpecificRule(targetRule: ASTCustomRuleDTO): void {
    const all = this.store.astRules().custom_rules || [];
    const index = all.findIndex(
      r => r.language === targetRule.language && r.pattern === targetRule.pattern && r.type === targetRule.type
    );
    if (index >= 0) {
      this.store.removeCustomAstRule(index);
    }
  }
}
