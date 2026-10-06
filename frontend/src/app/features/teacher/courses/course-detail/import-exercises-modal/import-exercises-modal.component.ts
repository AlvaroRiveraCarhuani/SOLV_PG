import {
  Component,
  Input,
  Output,
  EventEmitter,
  inject,
  signal,
  computed
} from '@angular/core';
import { CommonModule } from '@angular/common';
import {
  LucideUploadCloud,
  LucideFileCode,
  LucideX,
  LucideCheckCircle2,
  LucideAlertTriangle,
  LucideAlertCircle,
  LucideRefreshCw
} from '@lucide/angular';
import { TeacherCourseService } from '../../../services/teacher-course.service';
import { ExerciseImportResponse } from '../../../models/teacher.models';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { DismissibleDirective } from '@shared/directives/dismissible.directive';

@Component({
  selector: 'import-exercises-modal',
  standalone: true,
  imports: [
    CommonModule,
    LucideUploadCloud,
    LucideFileCode,
    LucideX,
    LucideCheckCircle2,
    LucideAlertTriangle,
    LucideAlertCircle,
    LucideRefreshCw,
    MachineDataDirective,
    DismissibleDirective
  ],
  templateUrl: './import-exercises-modal.component.html',
  styleUrl: './import-exercises-modal.component.scss'
})
export class ImportExercisesModalComponent {
  private teacherCourseService = inject(TeacherCourseService);

  @Input({ required: true }) subjectId!: string;
  @Output() closed = new EventEmitter<void>();
  @Output() imported = new EventEmitter<number>();

  selectedFile = signal<File | null>(null);
  isDragOver = signal<boolean>(false);
  isAnalyzing = signal<boolean>(false);
  isImporting = signal<boolean>(false);
  importPreview = signal<ExerciseImportResponse | null>(null);
  errorMessage = signal<string | null>(null);

  hasPreview = computed(() => this.importPreview() !== null);

  onFileSelected(event: Event): void {
    const input = event.target as HTMLInputElement;
    if (input.files && input.files.length > 0) {
      this.setFile(input.files[0]);
    }
  }

  onDragOver(event: DragEvent): void {
    event.preventDefault();
    event.stopPropagation();
    this.isDragOver.set(true);
  }

  onDragLeave(event: DragEvent): void {
    event.preventDefault();
    event.stopPropagation();
    this.isDragOver.set(false);
  }

  onDrop(event: DragEvent): void {
    event.preventDefault();
    event.stopPropagation();
    this.isDragOver.set(false);

    if (event.dataTransfer?.files && event.dataTransfer.files.length > 0) {
      this.setFile(event.dataTransfer.files[0]);
    }
  }

  private setFile(file: File): void {
    const ext = file.name.split('.').pop()?.toLowerCase();
    if (ext !== 'json' && ext !== 'yaml' && ext !== 'yml') {
      this.errorMessage.set('Solo se permiten archivos en formato JSON (.json) o YAML (.yaml, .yml)');
      return;
    }

    if (file.size > 10 * 1024 * 1024) {
      this.errorMessage.set('El archivo excede el tamaño máximo permitido de 10 MB');
      return;
    }

    this.selectedFile.set(file);
    this.errorMessage.set(null);
    this.importPreview.set(null);
    this.analyzeFile();
  }

  clearFile(): void {
    this.selectedFile.set(null);
    this.importPreview.set(null);
    this.errorMessage.set(null);
  }

  analyzeFile(): void {
    const file = this.selectedFile();
    if (!file || !this.subjectId) return;

    this.isAnalyzing.set(true);
    this.errorMessage.set(null);

    this.teacherCourseService.importExercises(this.subjectId, file, true).subscribe({
      next: (res) => {
        this.importPreview.set(res);
        this.isAnalyzing.set(false);
      },
      error: (err) => {
        this.isAnalyzing.set(false);
        const msg = err.error?.message || err.error?.error || 'Error al analizar el archivo de importación';
        this.errorMessage.set(msg);
      }
    });
  }

  confirmImport(): void {
    const file = this.selectedFile();
    const preview = this.importPreview();

    if (!file || !preview?.can_import || this.isImporting()) return;

    this.isImporting.set(true);
    this.errorMessage.set(null);

    this.teacherCourseService.importExercises(this.subjectId, file, false).subscribe({
      next: (res) => {
        this.isImporting.set(false);
        this.imported.emit(res.imported_count || preview.total_valid || 0);
        this.closeModal();
      },
      error: (err) => {
        this.isImporting.set(false);
        const msg = err.error?.message || err.error?.error || 'Error al ejecutar la importación';
        this.errorMessage.set(msg);
      }
    });
  }

  closeModal(): void {
    this.closed.emit();
  }
}
