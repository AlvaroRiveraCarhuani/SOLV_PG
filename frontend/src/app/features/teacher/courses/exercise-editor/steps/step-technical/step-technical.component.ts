import { Component, ChangeDetectionStrategy, model, input, output, computed } from '@angular/core';
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
  LucideServer 
} from '@lucide/angular';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';

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
    FormFieldComponent,
    MachineDataDirective,
    ComboboxComponent
  ],
  templateUrl: './step-technical.component.html',
  styleUrl: './step-technical.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class StepTechnicalComponent {
  labType = input.required<'ALGORITMO' | 'IDE_PERSISTENTE'>();

  readonly languageOptions: ComboboxOption[] = [
    { id: 'python', label: 'Python 3.11', value: 'python' },
    { id: 'javascript', label: 'JavaScript / Node.js 20', value: 'javascript' },
    { id: 'cpp', label: 'C++ (GCC 13 / C++20)', value: 'cpp' },
    { id: 'c', label: 'C (GCC 13 / C17)', value: 'c' },
    { id: 'go', label: 'Go 1.22', value: 'go' },
    { id: 'sql', label: 'PostgreSQL 16 SQL', value: 'sql' }
  ];
  
  // Algoritmo
  language = model.required<string>();
  timeLimitMs = model.required<number>();
  testCases = model.required<TestCaseFormItem[]>();
  csvError = model.required<string | null>();

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
      const parts = line.split(';').map(p => p.trim());
      if (parts.length < 2) {
        errors.push(`Línea ${i + 1}: Formato inválido. Se espera "input;expected_output;is_hidden"`);
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
