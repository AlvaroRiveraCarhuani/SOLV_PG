import { Component, ChangeDetectionStrategy, model, input, output, computed, signal, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { 
  LucideCpu, 
  LucideSparkles, 
  LucideUpload, 
  LucideTrash2, 
  LucidePlus, 
  LucideAlertTriangle, 
  LucideCheck, 
  LucideDatabase, 
  LucideServer,
  LucidePlay,
  LucideMaximize2,
  LucideCode,
  LucideSliders,
  LucideX
} from '@lucide/angular';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { TeacherCourseService } from '../../../../services/teacher-course.service';

export interface TestCaseFormItem {
  input: string;
  expected_output: string;
  is_hidden: boolean;
}

export interface WorkspaceTemplateOption {
  id: string;
  name: string;
  docker_image: string;
  description: string;
  base_ram_mb: number;
  category: string;
  satellite_service?: string;
  has_database?: boolean;
}

@Component({
  selector: 'step-technical',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideCpu,
    LucideSparkles,
    LucideUpload,
    LucideTrash2,
    LucidePlus,
    LucideAlertTriangle,
    LucideCheck,
    LucideDatabase,
    LucideServer,
    LucidePlay,
    LucideMaximize2,
    LucideCode,
    LucideSliders,
    LucideX,
    FormFieldComponent,
    MachineDataDirective,
    ComboboxComponent
  ],
  templateUrl: './step-technical.component.html',
  styleUrl: './step-technical.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class StepTechnicalComponent {
  private courseService = inject(TeacherCourseService);

  labType = input.required<'ALGORITMO' | 'IDE_PERSISTENTE'>();

  readonly languageOptions: ComboboxOption[] = [
    { id: 'python', label: 'Python 3.11', value: 'python' },
    { id: 'javascript', label: 'JavaScript / Node.js 20', value: 'javascript' },
    { id: 'cpp', label: 'C++ (GCC 13 / C++20)', value: 'cpp' },
    { id: 'c', label: 'C (GCC 13 / C17)', value: 'c' },
    { id: 'csharp', label: 'C# (.NET / Mono)', value: 'csharp' },
    { id: 'java', label: 'Java (OpenJDK 21)', value: 'java' }
  ];

  readonly comparatorOptions: ComboboxOption[] = [
    { id: 'exact', label: 'Exacto (Estricto estándar)', value: 'exact' },
    { id: 'float', label: 'Tolerancia Flotante (Decimales)', value: 'float' },
    { id: 'unordered', label: 'Salida Desordenada (Ignora orden de líneas)', value: 'unordered' }
  ];
  
  // Algoritmo
  language = model.required<string>();
  timeLimitMs = model.required<number>();
  referenceSolution = model.required<string>();
  comparatorType = model.required<'exact' | 'float' | 'unordered' | 'custom'>();
  comparatorTolerance = model<number>(0.000001);
  testCases = model.required<TestCaseFormItem[]>();
  csvError = model.required<string | null>();

  // Estados de cálculo de salidas con Solución de Referencia
  isCalculatingOutputs = signal<boolean>(false);
  calcOutputsError = signal<string | null>(null);
  calcOutputsSuccess = signal<string | null>(null);

  // Editor Multilínea Modal (para matrices o textos grandes)
  editingMultilineIndex = signal<number | null>(null);
  editingMultilineField = signal<'input' | 'expected_output' | null>(null);
  editingMultilineValue = signal<string>('');

  // Workspace
  templateId = model.required<string>();
  templatesList = input.required<WorkspaceTemplateOption[]>();
  selectedTemplate = input.required<WorkspaceTemplateOption>();
  hasDatabaseSatellite = input.required<boolean>();
  dbInitScript = model.required<string>();

  templateOptions = computed<ComboboxOption[]>(() => {
    return this.templatesList().map(tpl => ({
      id: tpl.id,
      label: `[${tpl.category}] ${tpl.name} (${tpl.docker_image})`,
      value: tpl.id,
      group: tpl.category
    }));
  });

  // Outputs
  requestTemplate = output<void>();
  openFuzzing = output<void>();
  languageChanged = output<string>();

  onLanguageSelect(lang: string): void {
    this.language.set(lang);
    this.languageChanged.emit(lang);
  }

  onComparatorSelect(type: string): void {
    this.comparatorType.set(type as 'exact' | 'float' | 'unordered' | 'custom');
  }

  addTestCase(): void {
    this.testCases.update(cases => [
      ...cases,
      { input: '', expected_output: '', is_hidden: false }
    ]);
  }

  removeTestCase(index: number): void {
    this.testCases.update(cases => {
      if (cases.length <= 1) return cases;
      return cases.filter((_, i) => i !== index);
    });
  }

  openMultilineModal(index: number, field: 'input' | 'expected_output'): void {
    const current = this.testCases()[index];
    if (!current) return;
    this.editingMultilineIndex.set(index);
    this.editingMultilineField.set(field);
    this.editingMultilineValue.set(field === 'input' ? current.input : current.expected_output);
  }

  saveMultilineModal(): void {
    const idx = this.editingMultilineIndex();
    const field = this.editingMultilineField();
    if (idx === null || field === null) return;

    this.testCases.update(cases => {
      return cases.map((c, i) => {
        if (i === idx) {
          return {
            ...c,
            [field]: this.editingMultilineValue()
          };
        }
        return c;
      });
    });

    this.closeMultilineModal();
  }

  closeMultilineModal(): void {
    this.editingMultilineIndex.set(null);
    this.editingMultilineField.set(null);
    this.editingMultilineValue.set('');
  }

  calculateOutputsFromReference(): void {
    this.calcOutputsError.set(null);
    this.calcOutputsSuccess.set(null);

    const refSol = this.referenceSolution().trim();
    if (!refSol) {
      this.calcOutputsError.set('Ingrese la solución de referencia arriba antes de calcular las salidas.');
      return;
    }

    const currentCases = this.testCases();
    const inputsToRun = currentCases.map(c => c.input);
    if (inputsToRun.length === 0 || inputsToRun.every(i => !i.trim())) {
      this.calcOutputsError.set('Debe ingresar al menos un valor de Entrada (Input) para calcular.');
      return;
    }

    this.isCalculatingOutputs.set(true);

    this.courseService.calculateOutputs({
      language: this.language(),
      source_code: refSol,
      inputs: inputsToRun,
      time_limit_ms: this.timeLimitMs()
    }).subscribe({
      next: (res) => {
        this.isCalculatingOutputs.set(false);
        const outputs = res.outputs || [];
        let errorsCount = 0;

        this.testCases.update(cases => {
          return cases.map((c, idx) => {
            const outItem = outputs.find(o => o.index === idx);
            if (outItem) {
              if (outItem.status === 'ok') {
                return { ...c, expected_output: outItem.expected_output };
              } else {
                errorsCount++;
                return { ...c, expected_output: outItem.error_details || 'Error de ejecución' };
              }
            }
            return c;
          });
        });

        if (errorsCount > 0) {
          this.calcOutputsError.set(`La solución generó errores en ${errorsCount} caso(s). Revise el código.`);
        } else {
          this.calcOutputsSuccess.set(`¡Salidas calculadas para ${outputs.length} caso(s) de prueba!`);
        }
      },
      error: (err) => {
        this.isCalculatingOutputs.set(false);
        this.calcOutputsError.set(err.error?.message || 'Error al ejecutar la solución de referencia en el backend.');
      }
    });
  }

  handleCsvUpload(event: Event): void {
    const inputEl = event.target as HTMLInputElement;
    if (!inputEl.files || inputEl.files.length === 0) return;

    const file = inputEl.files[0];
    this.csvError.set(null);

    const reader = new FileReader();
    reader.onload = () => {
      const text = reader.result as string;
      this.parseCsvCases(text);
      inputEl.value = '';
    };
    reader.onerror = () => {
      this.csvError.set('Error al leer el archivo CSV.');
      inputEl.value = '';
    };
    reader.readAsText(file);
  }

  private parseCsvCases(csvText: string): void {
    const lines = csvText.split(/\r?\n/).filter(line => line.trim().length > 0);
    if (lines.length === 0) {
      this.csvError.set('El archivo CSV está vacío.');
      return;
    }

    const parsed: TestCaseFormItem[] = [];
    const errors: string[] = [];

    const startIndex = lines[0].toLowerCase().includes('input') || lines[0].toLowerCase().includes('entrada') ? 1 : 0;

    for (let i = startIndex; i < lines.length; i++) {
      const line = lines[i];
      // Soporta delimitador ';' o ','
      const delimiter = line.includes(';') ? ';' : ',';
      const parts = line.split(delimiter).map(p => p.trim().replace(/^"(.*)"$/, '$1'));
      if (parts.length < 2) {
        errors.push(`Línea ${i + 1}: Formato inválido. Se espera "input${delimiter}expected_output${delimiter}is_hidden"`);
        continue;
      }

      const inputVal = parts[0];
      const outputVal = parts[1];
      const isHidden = parts.length >= 3 ? parts[2].toLowerCase() === 'true' || parts[2] === '1' : false;

      parsed.push({ input: inputVal, expected_output: outputVal, is_hidden: isHidden });
    }

    if (errors.length > 0) {
      this.csvError.set(errors.join('\n'));
      return;
    }

    if (parsed.length > 0) {
      this.testCases.set(parsed);
    }
  }
}
