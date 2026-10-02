import { Component, input, output, signal, inject, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { 
  LucideCode, 
  LucideAlertCircle, 
  LucideAlertTriangle, 
  LucideUpload, 
  LucideTrash2, 
  LucidePlus 
} from '@lucide/angular';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import { TeacherCourseService } from '../../services/teacher-course.service';
import { TeacherLabStats } from '../../models/teacher.models';

export interface TestCaseFormItem {
  input: string;
  expected_output: string;
  is_hidden: boolean;
}

@Component({
  selector: 'exercise-editor-modal',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideCode,
    LucideAlertCircle,
    LucideAlertTriangle,
    LucideUpload,
    LucideTrash2,
    LucidePlus,
    ModalShellComponent,
    FormFieldComponent
  ],
  templateUrl: './exercise-editor-modal.component.html',
  styleUrl: './exercise-editor-modal.component.scss'
})
export class ExerciseEditorModalComponent implements OnInit {
  private courseService = inject(TeacherCourseService);

  subjectId = input.required<string>();
  exerciseToEdit = input<TeacherLabStats | null>(null);

  close = output<void>();
  saved = output<void>();

  title = signal<string>('');
  description = signal<string>('');
  dueDate = signal<string>('');
  testCases = signal<TestCaseFormItem[]>([
    { input: '', expected_output: '', is_hidden: false }
  ]);

  csvError = signal<string | null>(null);
  formError = signal<string | null>(null);
  isSubmitting = signal<boolean>(false);

  ngOnInit(): void {
    const edit = this.exerciseToEdit();
    if (edit) {
      this.title.set(edit.title);
      this.dueDate.set(edit.due_date ? edit.due_date.substring(0, 16) : '');
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

  submit(): void {
    this.formError.set(null);
    if (!this.title().trim()) {
      this.formError.set('El título del laboratorio es obligatorio.');
      return;
    }

    this.isSubmitting.set(true);

    const edit = this.exerciseToEdit();
    if (edit) {
      this.courseService.updateExercise(edit.id, {
        title: this.title().trim(),
        description: this.description().trim(),
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
        due_date: this.dueDate() ? new Date(this.dueDate()).toISOString() : undefined
      }).subscribe({
        next: (created) => {
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
          } else {
            this.isSubmitting.set(false);
            this.saved.emit();
            this.close.emit();
          }
        },
        error: (err) => {
          this.isSubmitting.set(false);
          this.formError.set(err.error?.message || 'Error al crear el ejercicio.');
        }
      });
    }
  }
}
