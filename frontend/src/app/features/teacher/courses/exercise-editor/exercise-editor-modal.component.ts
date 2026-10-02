import { Component, input, output, signal, computed, inject, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { marked } from 'marked';
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
  LucideAlertTriangle,
  LucideCheck,
  LucideChevronRight,
  LucideChevronLeft,
  LucideEye,
  LucideEdit3,
  LucideRadio,
  LucideShield,
  LucideDatabase,
  LucideHelpCircle
} from '@lucide/angular';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { TeacherCourseService } from '../../services/teacher-course.service';
import { TeacherLabStats } from '../../models/teacher.models';
import { FuzzingModalComponent } from '../../evaluations/fuzzing-modal/fuzzing-modal.component';
import { TemplateRequestModalComponent } from '../../templates/template-request-modal/template-request-modal.component';

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
    LucideCheck,
    LucideChevronRight,
    LucideChevronLeft,
    LucideEye,
    LucideEdit3,
    LucideRadio,
    LucideShield,
    LucideDatabase,
    LucideHelpCircle,
    MachineDataDirective,
    ModalShellComponent,
    FormFieldComponent,
    FuzzingModalComponent,
    TemplateRequestModalComponent
  ],
  templateUrl: './exercise-editor-modal.component.html',
  styleUrl: './exercise-editor-modal.component.scss'
})
export class ExerciseEditorModalComponent implements OnInit {
  private courseService = inject(TeacherCourseService);
  private http = inject(HttpClient);

  subjectId = input.required<string>();
  subjectName = input<string>('');
  exerciseToEdit = input<TeacherLabStats | null>(null);

  close = output<void>();
  saved = output<void>();

  // Wizard Step: 1 = Tipo y Enunciado, 2 = Config Técnica, 3 = Reglas y Plantilla
  currentStep = signal<1 | 2 | 3>(1);

  // PASO 1: Metadatos & Modalidad
  title = signal<string>('');
  description = signal<string>('');
  dueDate = signal<string>('');
  labType = signal<'ALGORITMO' | 'IDE_PERSISTENTE'>('ALGORITMO');
  allowBroadcast = signal<boolean>(true); // ADR-007
  activeDescTab = signal<'edit' | 'preview'>('edit');

  renderedDescription = computed(() => {
    const raw = this.description();
    if (!raw.trim()) return '<p class="empty-preview">Sin enunciado especificado.</p>';
    try {
      return marked.parse(raw, { async: false }) as string;
    } catch {
      return raw;
    }
  });

  // PASO 2A: Configuración Juez Virtual
  language = signal<string>('python');
  timeLimitMs = signal<number>(1000);
  testCases = signal<TestCaseFormItem[]>([
    { input: '', expected_output: '', is_hidden: false }
  ]);

  // PASO 2B: Configuración Workspace Docker (ADR-030 & ADR-006)
  templateId = signal<string>('tpl-python-ds');
  databaseService = signal<'none' | 'postgres' | 'mysql' | 'mongodb'>('none');
  dbInitScript = signal<string>('');

  templatesList = signal<WorkspaceTemplateOption[]>([
    {
      id: 'tpl-python-ds',
      name: 'Python 3.11 Data Science & AI',
      docker_image: 'solv-lab/python-ds:3.11',
      description: 'NumPy, Pandas, Scikit-learn, Jupyter y soporte para Python 3.11.',
      base_ram_mb: 1024,
      category: 'Data Science'
    },
    {
      id: 'tpl-node-fullstack',
      name: 'Node.js 20 Fullstack & TypeScript',
      docker_image: 'solv-lab/node-fullstack:20',
      description: 'Node.js LTS con TypeScript, Express, Vite y testing Jest/Vitest.',
      base_ram_mb: 1024,
      category: 'Web Development'
    },
    {
      id: 'tpl-cpp-systems',
      name: 'C/C++ Sistemas y Algoritmos',
      docker_image: 'solv-lab/cpp-dev:gcc-13',
      description: 'GCC 13, Clang, GDB, CMake, Valgrind para sistemas y estructuras.',
      base_ram_mb: 512,
      category: 'Systems'
    },
    {
      id: 'tpl-postgres-db',
      name: 'PostgreSQL 16 Database Lab',
      docker_image: 'solv-lab/postgres-lab:16',
      description: 'PostgreSQL 16 con psql y scripts DDL precargados.',
      base_ram_mb: 1024,
      category: 'Databases'
    },
    {
      id: 'tpl-go-backend',
      name: 'Go 1.22 Cloud Native',
      docker_image: 'solv-lab/golang:1.22',
      description: 'Go con linter golangci-lint, gRPC y testing integrado.',
      base_ram_mb: 512,
      category: 'Backend'
    },
    {
      id: 'tpl-java-spring',
      name: 'Java 21 & Spring Boot',
      docker_image: 'solv-lab/java-spring:21',
      description: 'OpenJDK 21, Maven 3.9 y extensiones Java para VS Code.',
      base_ram_mb: 2048,
      category: 'Enterprise'
    }
  ]);

  selectedTemplate = computed(() => {
    return this.templatesList().find(t => t.id === this.templateId()) || this.templatesList()[0];
  });

  // PASO 3: Restricciones AST (Semgrep ADR-026) y Boilerplate
  blockNativeSort = signal<boolean>(false);
  blockSystemModules = signal<boolean>(true);
  forceRecursion = signal<boolean>(false);
  boilerplate = signal<string>('');

  // Modales secundarios
  showFuzzingModal = signal<boolean>(false);
  showTemplateRequestModal = signal<boolean>(false);

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
        this.boilerplate.set(edit.boilerplate || '');
      } else {
        this.labType.set('ALGORITMO');
        this.language.set(edit.language || 'python');
        this.boilerplate.set(edit.boilerplate || '');
        this.timeLimitMs.set(edit.time_limit_ms || 1000);
      }
    } else {
      this.boilerplate.set(this.getDefaultBoilerplate('python'));
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
        // Fallback al catálogo precargado
      }
    });
  }

  goToStep(step: 1 | 2 | 3): void {
    this.formError.set(null);
    if (step > 1 && !this.title().trim()) {
      this.formError.set('El título del laboratorio es obligatorio para continuar.');
      return;
    }
    this.currentStep.set(step);
  }

  setLabType(type: 'ALGORITMO' | 'IDE_PERSISTENTE'): void {
    this.labType.set(type);
  }

  setLanguage(lang: string): void {
    this.language.set(lang);
    if (!this.boilerplate().trim() || this.isDefaultBoilerplate(this.boilerplate())) {
      this.boilerplate.set(this.getDefaultBoilerplate(lang));
    }
  }

  isDefaultBoilerplate(code: string): boolean {
    const list = ['python', 'javascript', 'cpp', 'c', 'go', 'sql'].map(l => this.getDefaultBoilerplate(l).trim());
    return list.includes(code.trim());
  }

  getDefaultBoilerplate(lang: string): string {
    switch (lang) {
      case 'python':
        return 'def solution(arr):\n    # TODO: Implemente su algoritmo aquí\n    pass\n';
      case 'javascript':
        return 'function solution(arr) {\n  // TODO: Implemente su solución aquí\n}\n\nmodule.exports = { solution };\n';
      case 'cpp':
        return '#include <iostream>\n#include <vector>\nusing namespace std;\n\nint main() {\n    // TODO: Implemente su solución aquí\n    return 0;\n}\n';
      case 'c':
        return '#include <stdio.h>\n\nint main() {\n    // TODO: Implemente su solución aquí\n    return 0;\n}\n';
      case 'go':
        return 'package main\n\nimport "fmt"\n\nfunc main() {\n    // TODO: Implemente su solución aquí\n    fmt.Println("SOLV")\n}\n';
      case 'sql':
        return '-- Escriba su consulta SQL de consulta\nSELECT * FROM data;\n';
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

  openTemplateRequestModal(): void {
    this.showTemplateRequestModal.set(true);
  }

  closeTemplateRequestModal(): void {
    this.showTemplateRequestModal.set(false);
  }

  onTemplateRequested(): void {
    this.loadApprovedTemplates();
  }

  submit(publish: boolean = false): void {
    this.formError.set(null);
    if (!this.title().trim()) {
      this.formError.set('El título del laboratorio es obligatorio.');
      this.currentStep.set(1);
      return;
    }

    this.isSubmitting.set(true);

    const isWorkspace = this.labType() === 'IDE_PERSISTENTE';
    const exerciseType = isWorkspace ? 'workspace' : 'algorithm';
    const tpl = this.selectedTemplate();

    const edit = this.exerciseToEdit();
    if (edit) {
      this.courseService.updateExercise(edit.id, {
        title: this.title().trim(),
        description: this.description().trim(),
        type: exerciseType,
        language: isWorkspace ? undefined : this.language(),
        boilerplate: this.boilerplate().trim(),
        template_id: isWorkspace ? tpl?.id : undefined,
        memory_limit_mb: isWorkspace ? tpl?.base_ram_mb : 256,
        time_limit_ms: isWorkspace ? undefined : this.timeLimitMs(),
        due_date: this.dueDate() ? new Date(this.dueDate()).toISOString() : undefined
      }).subscribe({
        next: () => {
          if (publish) {
            this.courseService.publishExercise(edit.id).subscribe({
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
          } else {
            this.isSubmitting.set(false);
            this.saved.emit();
            this.close.emit();
          }
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
        boilerplate: this.boilerplate().trim(),
        template_id: isWorkspace ? tpl?.id : undefined,
        memory_limit_mb: isWorkspace ? tpl?.base_ram_mb : 256,
        time_limit_ms: isWorkspace ? 0 : this.timeLimitMs(),
        due_date: this.dueDate() ? new Date(this.dueDate()).toISOString() : undefined
      }).subscribe({
        next: (created) => {
          const onFinish = () => {
            if (publish) {
              this.courseService.publishExercise(created.id).subscribe({
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
            } else {
              this.isSubmitting.set(false);
              this.saved.emit();
              this.close.emit();
            }
          };

          if (!isWorkspace) {
            const validCases = this.testCases().filter(c => c.input.trim() && c.expected_output.trim());
            if (validCases.length > 0) {
              this.courseService.bulkUploadTestCases(created.id, { test_cases: validCases }).subscribe({
                next: onFinish,
                error: onFinish
              });
              return;
            }
          }
          onFinish();
        },
        error: (err) => {
          this.isSubmitting.set(false);
          this.formError.set(err.error?.message || 'Error al crear el ejercicio.');
        }
      });
    }
  }
}
