import { Component, input, output, signal, inject, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { 
  LucideCode, 
  LucideTerminal,
  LucideServer,
  LucideLaptop,
  LucideCpu,
  LucideSparkles,
  LucideUpload, 
  LucideTrash2, 
  LucidePlus,
  LucideAlertCircle, 
  LucideAlertTriangle
} from '@lucide/angular';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import { TeacherCourseService } from '../../services/teacher-course.service';
import { TeacherLabStats } from '../../models/teacher.models';
import { FuzzingModalComponent } from '../../evaluations/fuzzing-modal/fuzzing-modal.component';

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
}

@Component({
  selector: 'exercise-editor-modal',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideCode,
    LucideTerminal,
    LucideServer,
    LucideLaptop,
    LucideCpu,
    LucideSparkles,
    LucideUpload,
    LucideTrash2,
    LucidePlus,
    LucideAlertCircle,
    LucideAlertTriangle,
    ModalShellComponent,
    FormFieldComponent,
    FuzzingModalComponent
  ],
  templateUrl: './exercise-editor-modal.component.html',
  styleUrl: './exercise-editor-modal.component.scss'
})
export class ExerciseEditorModalComponent implements OnInit {
  private courseService = inject(TeacherCourseService);
  private http = inject(HttpClient);

  subjectId = input.required<string>();
  exerciseToEdit = input<TeacherLabStats | null>(null);

  close = output<void>();
  saved = output<void>();

  // Modalidad seleccionada
  labType = signal<'ALGORITMO' | 'IDE_PERSISTENTE'>('ALGORITMO');

  // Campos generales
  title = signal<string>('');
  description = signal<string>('');
  dueDate = signal<string>('');

  // Configuración Juez Virtual (Monaco)
  language = signal<string>('python');
  boilerplate = signal<string>('');
  timeLimitMs = signal<number>(2000);
  memoryLimitMb = signal<number>(256);
  testCases = signal<TestCaseFormItem[]>([
    { input: '', expected_output: '', is_hidden: false }
  ]);

  // Configuración Workspace Completo (OpenVSCode)
  templateId = signal<string>('tpl-python-ds');
  workspaceRamMb = signal<number>(1024);
  workspaceReadme = signal<string>('# Guía del Proyecto\n\nBienvenido a tu entorno de laboratorio en OpenVSCode Server.\nEjecuta los tests locales con tu terminal integrado.');

  // Lista de plantillas de catálogo
  templatesList = signal<WorkspaceTemplateOption[]>([
    {
      id: 'tpl-python-ds',
      name: 'Python Data Science & AI',
      docker_image: 'solv-lab/python-ds:3.11',
      description: 'Entorno con NumPy, Pandas, Scikit-learn, Jupyter y soporte para Python 3.11.',
      base_ram_mb: 1024,
      category: 'Data Science'
    },
    {
      id: 'tpl-node-fullstack',
      name: 'Node.js Fullstack & TypeScript',
      docker_image: 'solv-lab/node-fullstack:20',
      description: 'Node.js 20 LTS con TypeScript, Express, Vite y herramientas de testing Jest/Vitest.',
      base_ram_mb: 1024,
      category: 'Web Development'
    },
    {
      id: 'tpl-cpp-systems',
      name: 'C/C++ Sistemas y Algoritmos',
      docker_image: 'solv-lab/cpp-dev:gcc-13',
      description: 'GCC 13, Clang, GDB, CMake, Valgrind para programación de sistemas y estructuras.',
      base_ram_mb: 512,
      category: 'Systems'
    },
    {
      id: 'tpl-postgres-db',
      name: 'PostgreSQL & Database Lab',
      docker_image: 'solv-lab/postgres-lab:16',
      description: 'Servidor PostgreSQL 16 con DBeaver/psql y scripts DDL precargados.',
      base_ram_mb: 1024,
      category: 'Databases'
    },
    {
      id: 'tpl-go-backend',
      name: 'Go Cloud Native Microservices',
      docker_image: 'solv-lab/golang:1.22',
      description: 'Go 1.22 con linter golangci-lint, gRPC tools y soporte para tests paralelos.',
      base_ram_mb: 512,
      category: 'Backend'
    },
    {
      id: 'tpl-java-spring',
      name: 'Java 21 & Spring Boot',
      docker_image: 'solv-lab/java-spring:21',
      description: 'OpenJDK 21, Maven 3.9, Spring Boot CLI y extensiones Java para VS Code.',
      base_ram_mb: 2048,
      category: 'Enterprise'
    }
  ]);

  // Modales secundarios
  showFuzzingModal = signal<boolean>(false);

  // Estados UI
  csvError = signal<string | null>(null);
  formError = signal<string | null>(null);
  isSubmitting = signal<boolean>(false);

  ngOnInit(): void {
    this.loadApprovedTemplates();

    const edit = this.exerciseToEdit();
    if (edit) {
      this.title.set(edit.title);
      this.dueDate.set(edit.due_date ? edit.due_date.substring(0, 16) : '');
      if (edit.type === 'workspace' || edit.type === 'IDE_PERSISTENTE') {
        this.labType.set('IDE_PERSISTENTE');
        this.templateId.set(edit.template_id || 'tpl-python-ds');
        this.workspaceRamMb.set(edit.memory_limit_mb || 1024);
        this.workspaceReadme.set(edit.boilerplate || '');
      } else {
        this.labType.set('ALGORITMO');
        this.language.set(edit.language || 'python');
        this.boilerplate.set(edit.boilerplate || '');
        this.memoryLimitMb.set(edit.memory_limit_mb || 256);
        this.timeLimitMs.set(edit.time_limit_ms || 2000);
      }
    }
  }

  loadApprovedTemplates(): void {
    this.http.get<{ data: WorkspaceTemplateOption[] }>('/api/v1/templates').subscribe({
      next: (res) => {
        if (res?.data && Array.isArray(res.data) && res.data.length > 0) {
          this.templatesList.set(res.data);
        }
      },
      error: () => {
        // Fallback silently to preloaded canonical catalog
      }
    });
  }

  setLabType(type: 'ALGORITMO' | 'IDE_PERSISTENTE'): void {
    this.labType.set(type);
  }

  setLanguage(lang: string): void {
    this.language.set(lang);
    if (!this.boilerplate().trim()) {
      this.boilerplate.set(this.getDefaultBoilerplate(lang));
    }
  }

  setWorkspaceRam(mb: number): void {
    this.workspaceRamMb.set(mb);
  }

  getDefaultBoilerplate(lang: string): string {
    switch (lang) {
      case 'python':
        return 'def solution():\n    # Escribe tu solución aquí\n    pass\n';
      case 'javascript':
        return 'function solution() {\n  // Escribe tu solución aquí\n}\n\nmodule.exports = { solution };\n';
      case 'cpp':
        return '#include <iostream>\nusing namespace std;\n\nint main() {\n    // Escribe tu solución aquí\n    return 0;\n}\n';
      case 'c':
        return '#include <stdio.h>\n\nint main() {\n    // Escribe tu solución aquí\n    return 0;\n}\n';
      case 'go':
        return 'package main\n\nimport "fmt"\n\nfunc main() {\n    // Escribe tu solución aquí\n    fmt.Println("Hola")\n}\n';
      case 'sql':
        return '-- Escribe tu consulta SQL aquí\nSELECT * FROM data;\n';
      default:
        return '';
    }
  }

  addTestCase(): void {
    this.testCases.update(list => [
      ...list,
      { input: '', expected_output: '', is_hidden: false }
    ]);
  }

  removeTestCase(index: number): void {
    this.testCases.update(list => list.filter((_, i) => i !== index));
  }

  onCsvSelected(event: Event): void {
    const input = event.target as HTMLInputElement;
    if (!input.files || input.files.length === 0) return;

    const file = input.files[0];
    const reader = new FileReader();

    reader.onload = (e) => {
      const text = e.target?.result as string;
      this.parseCsvTestCases(text);
    };

    reader.readAsText(file);
  }

  parseCsvTestCases(csvText: string): void {
    this.csvError.set(null);
    const lines = csvText.split(/\r?\n/).filter(line => line.trim().length > 0);
    if (lines.length === 0) {
      this.csvError.set('El archivo CSV está vacío.');
      return;
    }

    const parsed: TestCaseFormItem[] = [];
    const errors: string[] = [];

    let startIndex = 0;
    if (lines[0].toLowerCase().includes('input') && lines[0].toLowerCase().includes('output')) {
      startIndex = 1;
    }

    for (let i = startIndex; i < lines.length; i++) {
      const rowNum = i + 1;
      const line = lines[i];
      const parts = line.split(',');

      if (parts.length < 2) {
        errors.push(`Línea ${rowNum}: formato inválido (se requiere al menos input y output)`);
        continue;
      }

      const inputVal = parts[0].trim();
      const outputVal = parts[1].trim();
      const isHidden = parts[2] ? parts[2].trim().toLowerCase() === 'true' || parts[2].trim() === '1' : false;

      if (!inputVal || !outputVal) {
        errors.push(`Línea ${rowNum}: campos requeridos vacíos`);
        continue;
      }

      parsed.push({
        input: inputVal,
        expected_output: outputVal,
        is_hidden: isHidden
      });
    }

    if (errors.length > 0) {
      this.csvError.set(`Errores en CSV:\n${errors.slice(0, 3).join('\n')}${errors.length > 3 ? ` (+${errors.length - 3} más)` : ''}`);
      return;
    }

    if (parsed.length > 0) {
      this.testCases.set(parsed);
    }
  }

  openFuzzingModal(): void {
    this.showFuzzingModal.set(true);
  }

  closeFuzzingModal(): void {
    this.showFuzzingModal.set(false);
  }

  onFuzzCasesApplied(addedCount: number): void {
    this.showFuzzingModal.set(false);
    this.testCases.update(current => [
      ...current,
      { input: `[fuzz-test-${addedCount}]`, expected_output: '0', is_hidden: true }
    ]);
  }

  submit(): void {
    this.formError.set(null);
    if (!this.title().trim()) {
      this.formError.set('El título del laboratorio es obligatorio.');
      return;
    }

    this.isSubmitting.set(true);

    const isWorkspace = this.labType() === 'IDE_PERSISTENTE';
    const exerciseType = isWorkspace ? 'workspace' : 'algorithm';
    const selectedMemory = isWorkspace ? this.workspaceRamMb() : this.memoryLimitMb();
    const selectedBoilerplate = isWorkspace ? this.workspaceReadme().trim() : this.boilerplate().trim();

    const edit = this.exerciseToEdit();
    if (edit) {
      this.courseService.updateExercise(edit.id, {
        title: this.title().trim(),
        description: this.description().trim(),
        type: exerciseType,
        language: isWorkspace ? undefined : this.language(),
        boilerplate: selectedBoilerplate,
        template_id: isWorkspace ? this.templateId() : undefined,
        memory_limit_mb: selectedMemory,
        time_limit_ms: isWorkspace ? undefined : this.timeLimitMs(),
        due_date: this.dueDate() ? new Date(this.dueDate()).toISOString() : undefined
      }).subscribe({
        next: () => {
          this.isSubmitting.set(false);
          this.saved.emit();
          this.close.emit();
        },
        error: (err) => {
          this.isSubmitting.set(false);
          this.formError.set(err.error?.message || 'Error al actualizar el ejercicio.');
        }
      });
    } else {
      this.courseService.createExercise({
        subject_id: this.subjectId(),
        title: this.title().trim(),
        description: this.description().trim(),
        type: exerciseType,
        language: isWorkspace ? 'workspace' : this.language(),
        boilerplate: selectedBoilerplate,
        template_id: isWorkspace ? this.templateId() : undefined,
        memory_limit_mb: selectedMemory,
        time_limit_ms: isWorkspace ? 0 : this.timeLimitMs(),
        due_date: this.dueDate() ? new Date(this.dueDate()).toISOString() : undefined
      }).subscribe({
        next: (created) => {
          if (!isWorkspace) {
            const validCases = this.testCases().filter(c => c.input.trim() && c.expected_output.trim());
            if (validCases.length > 0) {
              this.courseService.bulkUploadTestCases(created.id, { test_cases: validCases }).subscribe({
                next: () => {
                  this.isSubmitting.set(false);
                  this.saved.emit();
                  this.close.emit();
                },
                error: () => {
                  this.isSubmitting.set(false);
                  this.saved.emit();
                  this.close.emit();
                }
              });
              return;
            }
          }
          this.isSubmitting.set(false);
          this.saved.emit();
          this.close.emit();
        },
        error: (err) => {
          this.isSubmitting.set(false);
          this.formError.set(err.error?.message || 'Error al crear el ejercicio.');
        }
      });
    }
  }
}
