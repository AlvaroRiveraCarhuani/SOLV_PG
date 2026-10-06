import { Component, OnInit, inject, signal, computed, effect } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { 
  LucidePlus, 
  LucideTrash2, 
  LucideCode, 
  LucideCheckCircle2, 
  LucideAlertCircle, 
  LucideSparkles, 
  LucideSliders
} from '@lucide/angular';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { ExerciseEditorStore } from '../../../exercise-editor.store';
import { TeacherCourseService } from '../../../../services/teacher-course.service';
import { ExerciseContract, FormatLine, FORMAT_PRESETS, FormatPreset } from './format-builder.models';
import { PresetButtonComponent } from './presets/preset-button.component';

@Component({
  selector: 'format-builder',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucidePlus,
    LucideTrash2,
    LucideCode,
    LucideCheckCircle2,
    LucideAlertCircle,
    LucideSparkles,
    LucideSliders,
    MachineDataDirective,
    PresetButtonComponent
  ],
  templateUrl: './format-builder.component.html',
  styleUrl: './format-builder.component.scss'
})
export class FormatBuilderComponent implements OnInit {
  readonly store = inject(ExerciseEditorStore);
  private courseService = inject(TeacherCourseService);

  readonly presets = FORMAT_PRESETS;

  // Estado del editor visual
  lines = signal<FormatLine[]>([]);
  activePresetId = signal<string | null>(null);

  // Modo JSON
  showJsonEditor = signal<boolean>(false);
  jsonCode = signal<string>('');
  jsonError = signal<string | null>(null);

  // Validación del backend
  isValidating = signal<boolean>(false);
  backendValidation = signal<{ valid: boolean; error?: string } | null>(null);

  constructor() {
    // Sincronizar desde store al iniciar o cuando store cambia externamente
    effect(() => {
      const contract = this.store.contract();
      if (contract?.input?.lines) {
        // Evitar loops si es idéntico
        const currentJson = JSON.stringify(this.buildContractFromLines());
        const storeJson = JSON.stringify(contract);
        if (currentJson !== storeJson) {
          this.lines.set(JSON.parse(JSON.stringify(contract.input.lines)));
          this.jsonCode.set(JSON.stringify(contract, null, 2));
          this.matchPreset(contract);
        }
      }
    });
  }

  ngOnInit() {
    const existing = this.store.contract();
    if (existing?.input?.lines) {
      this.lines.set(JSON.parse(JSON.stringify(existing.input.lines)));
      this.jsonCode.set(JSON.stringify(existing, null, 2));
      this.matchPreset(existing);
      this.validateWithBackend(existing);
    } else {
      // Default: cargar preset "Un entero N"
      this.applyPreset(this.presets[0]);
    }
  }

  // Selección de Preset
  applyPreset(preset: FormatPreset) {
    this.activePresetId.set(preset.id);
    const cloned = JSON.parse(JSON.stringify(preset.contract)) as ExerciseContract;
    this.lines.set(cloned.input.lines);
    this.jsonCode.set(JSON.stringify(cloned, null, 2));
    this.jsonError.set(null);
    this.syncToStore(cloned);
    this.validateWithBackend(cloned);
  }

  private matchPreset(contract: ExerciseContract) {
    const lines = contract.input?.lines;
    if (!lines) {
      this.activePresetId.set(null);
      return;
    }

    const found = this.presets.find(p => {
      const pLines = p.contract.input.lines;
      if (pLines.length !== lines.length) return false;
      return pLines.every((pl, i) => pl.type === lines[i].type && pl.id === lines[i].id);
    });

    this.activePresetId.set(found ? found.id : null);
  }

  // Mutaciones del editor visual
  addLine() {
    const newId = `var_${this.lines().length + 1}`;
    const newLine: FormatLine = {
      id: newId,
      type: 'int',
      min: 1,
      max: 1000000
    };
    this.lines.update(curr => [...curr, newLine]);
    this.onVisualChange();
  }

  removeLine(index: number) {
    this.lines.update(curr => curr.filter((_, i) => i !== index));
    this.onVisualChange();
  }

  updateLineType(index: number, type: FormatLine['type']) {
    this.lines.update(curr => {
      const copy = [...curr];
      const target = { ...copy[index], type };
      
      // Ajustar campos según nuevo tipo
      if (type === 'int') {
        target.min = 1;
        target.max = 1000000;
        delete target.count;
        delete target.rows;
        delete target.cols;
        delete target.item;
        delete target.max_len;
      } else if (type === 'ints') {
        target.count = '=n';
        target.item = { type: 'int', min: -1000000000, max: 1000000000 };
        delete target.rows;
        delete target.cols;
        delete target.max_len;
      } else if (type === 'matrix') {
        target.rows = '=n';
        target.cols = '=m';
        target.item = { type: 'int', min: 0, max: 1000000 };
        delete target.count;
        delete target.max_len;
      } else if (type === 'edges') {
        target.count = '=m';
        delete target.rows;
        delete target.cols;
        delete target.item;
        delete target.max_len;
      } else if (type === 'string') {
        target.max_len = 1000;
        delete target.min;
        delete target.max;
        delete target.count;
        delete target.rows;
        delete target.cols;
        delete target.item;
      } else if (type === 'raw') {
        delete target.min;
        delete target.max;
        delete target.count;
        delete target.rows;
        delete target.cols;
        delete target.item;
        delete target.max_len;
      }

      copy[index] = target;
      return copy;
    });
    this.onVisualChange();
  }

  updateLineId(index: number, id: string) {
    this.lines.update(curr => {
      const copy = [...curr];
      copy[index] = { ...copy[index], id: id.trim() };
      return copy;
    });
    this.onVisualChange();
  }

  updateLineField(index: number, field: keyof FormatLine, value: any) {
    this.lines.update(curr => {
      const copy = [...curr];
      copy[index] = { ...copy[index], [field]: value };
      return copy;
    });
    this.onVisualChange();
  }

  updateItemBounds(index: number, min?: number, max?: number) {
    this.lines.update(curr => {
      const copy = [...curr];
      const target = { ...copy[index] };
      target.item = {
        type: target.item?.type || 'int',
        min: min !== undefined ? min : target.item?.min,
        max: max !== undefined ? max : target.item?.max
      };
      copy[index] = target;
      return copy;
    });
    this.onVisualChange();
  }

  onVisualChange() {
    const contract = this.buildContractFromLines();
    this.jsonCode.set(JSON.stringify(contract, null, 2));
    this.jsonError.set(null);
    this.matchPreset(contract);
    this.syncToStore(contract);
    this.validateWithBackend(contract);
  }

  private buildContractFromLines(): ExerciseContract {
    return {
      version: 1,
      input: {
        lines: this.lines()
      }
    };
  }

  // Edición Manual en JSON
  onJsonCodeChange(rawJson: string) {
    this.jsonCode.set(rawJson);
    try {
      const parsed = JSON.parse(rawJson) as ExerciseContract;
      if (!parsed || typeof parsed !== 'object') {
        this.jsonError.set('El JSON debe ser un objeto válido.');
        return;
      }
      if (!parsed.version || !parsed.input?.lines || !Array.isArray(parsed.input.lines)) {
        this.jsonError.set('Estructura inválida: se requiere { "version": 1, "input": { "lines": [...] } }');
        return;
      }

      // JSON válido: sincronizar con vista visual
      this.jsonError.set(null);
      this.lines.set(JSON.parse(JSON.stringify(parsed.input.lines)));
      this.matchPreset(parsed);
      this.syncToStore(parsed);
      this.validateWithBackend(parsed);
    } catch (e: any) {
      this.jsonError.set(`Error de sintaxis JSON: ${e.message}`);
    }
  }

  toggleJsonView() {
    this.showJsonEditor.update(v => !v);
  }

  resetToFreeText() {
    this.lines.set([]);
    this.activePresetId.set(null);
    this.jsonCode.set('');
    this.jsonError.set(null);
    this.backendValidation.set(null);
    this.store.setContract(null);
  }

  private syncToStore(contract: ExerciseContract) {
    this.store.setContract(contract);
  }

  private validateWithBackend(contract: ExerciseContract) {
    if (!contract?.input?.lines || contract.input.lines.length === 0) {
      this.backendValidation.set(null);
      return;
    }

    this.isValidating.set(true);
    // Validamos contrato pasando entrada vacía
    this.courseService.validateInputFormat(contract, '').subscribe({
      next: (res) => {
        this.isValidating.set(false);
        // Si el error es solo por entrada vacía pero no de contrato, el contrato en sí es válido
        if (res.valid || (res.error && !res.error.includes('contrato'))) {
          this.backendValidation.set({ valid: true });
        } else {
          this.backendValidation.set({ valid: false, error: res.error });
        }
      },
      error: (err) => {
        this.isValidating.set(false);
        this.backendValidation.set({
          valid: false,
          error: err.error?.error || 'Error al validar estructura del contrato'
        });
      }
    });
  }
}
