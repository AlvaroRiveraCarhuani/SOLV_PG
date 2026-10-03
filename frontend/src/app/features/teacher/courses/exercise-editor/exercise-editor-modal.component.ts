import { Component, input, output, signal, computed, inject, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { 
  LucideCode, 
  LucideAlertCircle, 
  LucideCheck, 
  LucideChevronRight, 
  LucideChevronLeft 
} from '@lucide/angular';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { TeacherCourseService } from '../../services/teacher-course.service';
import { TeacherLabStats } from '../../models/teacher.models';
import { FuzzingModalComponent } from '../../evaluations/fuzzing-modal/fuzzing-modal.component';
import { TemplateRequestModalComponent } from '../../templates/template-request-modal/template-request-modal.component';
import { StepGeneralComponent } from './steps/step-general/step-general.component';
import { StepTechnicalComponent, type TestCaseFormItem, type WorkspaceTemplateOption } from './steps/step-technical/step-technical.component';
import { StepRulesComponent } from './steps/step-rules/step-rules.component';

export type { TestCaseFormItem, WorkspaceTemplateOption };

@Component({
  selector: 'exercise-editor-modal',
  standalone: true,
  imports: [
    CommonModule,
    LucideCode,
    LucideAlertCircle,
    LucideCheck,
    LucideChevronRight,
    LucideChevronLeft,
    ModalShellComponent,
    FuzzingModalComponent,
    TemplateRequestModalComponent,
    StepGeneralComponent,
    StepTechnicalComponent,
    StepRulesComponent
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
  initialTemplate = input<any | null>(null);

  close = output<void>();
  saved = output<void>();

  // Wizard Step: 1 = Tipo y Enunciado, 2 = Config Técnica, 3 = Reglas y Plantilla
  currentStep = signal<1 | 2 | 3>(1);

  // PASO 1: Metadatos & Modalidad
  title = signal<string>('');
  titleError = signal<string | null>(null);
  description = signal<string>('');
  dueDate = signal<string>('');
  hasDueDate = signal<boolean>(false);
  labType = signal<'ALGORITMO' | 'IDE_PERSISTENTE'>('ALGORITMO');
  pedagogicalPurpose = signal<'PRACTICE' | 'EXAM'>('PRACTICE');
  allowBroadcast = signal<boolean>(true);

  // PASO 2A: Configuración Juez Virtual
  language = signal<string>('python');
  timeLimitMs = signal<number>(1000);
  testCases = signal<TestCaseFormItem[]>([
    { input: '', expected_output: '', is_hidden: false }
  ]);
  csvError = signal<string | null>(null);

  // PASO 2B: Configuración Workspace Docker
  templateId = signal<string>('tpl-python-ds');
  dbInitScript = signal<string>('');

  templatesList = signal<WorkspaceTemplateOption[]>([
    {
      id: 'tpl-python-ds',
      name: 'Python 3.11 Data Science & AI',
      docker_image: 'solv-lab/python-ds:3.11',
      description: 'NumPy, Pandas, Scikit-learn, Jupyter y soporte para Python 3.11.',
      base_ram_mb: 1024,
      category: 'Data Science',
      satellite_service: 'Sin servicios satélite (Entorno autónomo)',
      has_database: false
    },
    {
      id: 'tpl-node-fullstack',
      name: 'Node.js 20 Fullstack & TypeScript',
      docker_image: 'solv-lab/node-fullstack:20',
      description: 'Node.js LTS con TypeScript, Express, Vite y testing Jest/Vitest.',
      base_ram_mb: 1024,
      category: 'Web Development',
      satellite_service: 'Sin servicios satélite',
      has_database: false
    },
    {
      id: 'tpl-cpp-systems',
      name: 'C/C++ Sistemas y Algoritmos',
      docker_image: 'solv-lab/cpp-dev:gcc-13',
      description: 'GCC 13, Clang, GDB, CMake, Valgrind para sistemas y estructuras.',
      base_ram_mb: 512,
      category: 'Systems',
      satellite_service: 'Sin servicios satélite',
      has_database: false
    },
    {
      id: 'tpl-postgres-db',
      name: 'PostgreSQL 16 Database Lab',
      docker_image: 'solv-lab/postgres-lab:16',
      description: 'PostgreSQL 16 con psql y scripts DDL precargados.',
      base_ram_mb: 1024,
      category: 'Databases',
      satellite_service: 'PostgreSQL 16 (Relacional / SQL)',
      has_database: true
    },
    {
      id: 'tpl-go-backend',
      name: 'Go 1.22 Cloud Native',
      docker_image: 'solv-lab/golang:1.22',
      description: 'Go con linter golangci-lint, gRPC y testing integrado.',
      base_ram_mb: 512,
      category: 'Backend',
      satellite_service: 'Sin servicios satélite',
      has_database: false
    },
    {
      id: 'tpl-java-spring',
      name: 'Java 21 & Spring Boot',
      docker_image: 'solv-lab/java-spring:21',
      description: 'OpenJDK 21, Maven 3.9 y extensiones Java para VS Code.',
      base_ram_mb: 2048,
      category: 'Enterprise',
      satellite_service: 'PostgreSQL 16 (Relacional / Spring JPA)',
      has_database: true
    }
  ]);

  selectedTemplate = computed(() => {
    return this.templatesList().find(t => t.id === this.templateId()) || this.templatesList()[0];
  });

  hasDatabaseSatellite = computed(() => {
    const tpl = this.selectedTemplate();
    return !!tpl?.has_database || tpl?.category === 'Databases' || (tpl?.satellite_service ? tpl.satellite_service.toLowerCase().includes('sql') : false);
  });

  // PASO 3: Restricciones AST y Boilerplate
  blockNativeSort = signal<boolean>(false);
  blockSystemModules = signal<boolean>(true);
  forceRecursion = signal<boolean>(false);
  boilerplate = signal<string>('');

  // Modales secundarios
  showFuzzingModal = signal<boolean>(false);
  showTemplateRequestModal = signal<boolean>(false);

  // Estados UI
  formError = signal<string | null>(null);
  isSubmitting = signal<boolean>(false);

  ngOnInit(): void {
    this.loadApprovedTemplates();

    const edit = this.exerciseToEdit();
    if (edit) {
      this.title.set(edit.title);
      if (edit.due_date) {
        this.dueDate.set(edit.due_date.substring(0, 16));
        this.hasDueDate.set(true);
      } else {
        this.dueDate.set('');
        this.hasDueDate.set(false);
      }

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
      const initTpl = this.initialTemplate();
      if (initTpl) {
        const envType = initTpl.environment_type || initTpl.target_environment || '';
        if (envType === 'JUEZ_EFIMERO' || envType === 'ALGORITMO') {
          this.labType.set('ALGORITMO');
        } else {
          this.labType.set('IDE_PERSISTENTE');
          if (initTpl.id) {
            this.templateId.set(initTpl.id);
          }
        }

        if (initTpl.name) {
          this.title.set(`Laboratorio: ${initTpl.name}`);
        }
      }
      this.boilerplate.set(this.getDefaultBoilerplate('python'));
    }
  }

  onTitleChange(val: string): void {
    this.title.set(val);
    if (val.trim()) {
      this.titleError.set(null);
      this.formError.set(null);
    }
  }

  loadApprovedTemplates(): void {
    this.http.get<any>('/api/v1/templates').subscribe({
      next: (res) => {
        let items: any[] = [];
        if (Array.isArray(res)) {
          items = res;
        } else if (Array.isArray(res?.data)) {
          items = res.data;
        } else if (Array.isArray(res?.data?.data)) {
          items = res.data.data;
        }

        if (items.length > 0) {
          const mapped: WorkspaceTemplateOption[] = items.map(t => ({
            id: t.id,
            name: t.name,
            docker_image: t.docker_image,
            description: t.description || '',
            base_ram_mb: t.base_ram_mb || t.default_memory_mb || 512,
            category: t.category || t.category_name || 'General',
            satellite_service: t.satellite_service || 'Sin servicios satélite',
            has_database: !!t.has_database
          }));
          this.templatesList.set(mapped);
        }
      },
      error: () => {
        // Fallback al catálogo precargado
      }
    });
  }

  goToStep(step: 1 | 2 | 3): void {
    this.formError.set(null);
    this.titleError.set(null);
    if (step > 1 && !this.title().trim()) {
      const msg = 'El título del laboratorio es obligatorio para continuar.';
      this.formError.set(msg);
      this.titleError.set(msg);
      return;
    }
    this.currentStep.set(step);
  }

  setLabType(type: 'ALGORITMO' | 'IDE_PERSISTENTE'): void {
    this.labType.set(type);
  }

  setPedagogicalPurpose(purpose: 'PRACTICE' | 'EXAM'): void {
    this.pedagogicalPurpose.set(purpose);
    this.allowBroadcast.set(purpose === 'PRACTICE');
  }

  onLanguageChange(lang: string): void {
    this.language.set(lang);
    if (!this.boilerplate().trim() || this.isDefaultBoilerplate(this.boilerplate())) {
      this.boilerplate.set(this.getDefaultBoilerplate(lang));
    }
  }

  toggleNoDueDate(noDeadline: boolean): void {
    this.hasDueDate.set(!noDeadline);
    if (noDeadline) {
      this.dueDate.set('');
    }
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

  onFuzzCasesApplied(count: number): void {
    this.showFuzzingModal.set(false);
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
        due_date: this.hasDueDate() && this.dueDate() ? new Date(this.dueDate()).toISOString() : undefined
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
        due_date: this.hasDueDate() && this.dueDate() ? new Date(this.dueDate()).toISOString() : undefined
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

  private getDefaultBoilerplate(lang: string): string {
    switch (lang) {
      case 'python':
        return `def solve():\n    # Tu solucion aqui\n    pass\n\nif __name__ == '__main__':\n    solve()`;
      case 'javascript':
        return `function solve() {\n  // Tu solucion aqui\n}\n\nsolve();`;
      case 'cpp':
        return `#include <iostream>\n\nusing namespace std;\n\nint main() {\n    // Tu solucion aqui\n    return 0;\n}`;
      case 'c':
        return `#include <stdio.h>\n\nint main() {\n    // Tu solucion aqui\n    return 0;\n}`;
      case 'go':
        return `package main\n\nimport "fmt"\n\nfunc main() {\n    // Tu solucion aqui\n}`;
      case 'sql':
        return `-- Escribe tu consulta SQL aqui\nSELECT * FROM tabla;`;
      default:
        return '';
    }
  }

  private isDefaultBoilerplate(code: string): boolean {
    const defaults = ['python', 'javascript', 'cpp', 'c', 'go', 'sql'].map(l => this.getDefaultBoilerplate(l));
    return defaults.includes(code.trim());
  }
}
