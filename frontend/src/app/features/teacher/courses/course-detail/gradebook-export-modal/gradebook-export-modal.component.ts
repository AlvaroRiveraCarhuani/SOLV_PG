import { 
  Component, 
  Input, 
  Output, 
  EventEmitter, 
  OnInit, 
  inject, 
  signal, 
  computed,
  HostListener
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { 
  LucideFileSpreadsheet, 
  LucideDownload, 
  LucideX, 
  LucideSlidersHorizontal, 
  LucideChevronDown, 
  LucideChevronUp, 
  LucidePalette, 
  LucideFileText, 
  LucideEye,
  LucideEyeOff,
  LucideRotateCcw
} from '@lucide/angular';
import { TeacherCourseService } from '../../../services/teacher-course.service';
import { CourseGradesMatrix, StudentGradesRow } from '../../../models/teacher.models';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { DismissibleDirective } from '@shared/directives/dismissible.directive';
import {
  ColumnExportConfig,
  PalettePreset,
  PALETTE_PRESETS,
  COLOR_SWATCHES
} from './gradebook-export.presets';
import {
  getStudentScore,
  initializeDefaultColumns,
  buildExcelSpreadsheetXml,
  buildCsvContent,
  downloadBlob
} from './gradebook-export.utils';

export type { ColumnExportConfig, PalettePreset };
export { PALETTE_PRESETS, COLOR_SWATCHES };

@Component({
  selector: 'gradebook-export-modal',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideFileSpreadsheet,
    LucideDownload,
    LucideX,
    LucideSlidersHorizontal,
    LucideChevronDown,
    LucideChevronUp,
    LucidePalette,
    LucideFileText,
    LucideEye,
    LucideEyeOff,
    LucideRotateCcw,
    MachineDataDirective,
    DismissibleDirective
  ],
  templateUrl: './gradebook-export-modal.component.html',
  styleUrl: './gradebook-export-modal.component.scss'
})
export class GradebookExportModalComponent implements OnInit {
  @Input({ required: true }) subjectId!: string;
  @Input({ required: true }) subjectName!: string;
  @Input({ required: true }) subjectCode!: string;
  @Output() dismiss = new EventEmitter<void>();

  private courseService = inject(TeacherCourseService);

  matrix = signal<CourseGradesMatrix | null>(null);
  isLoading = signal<boolean>(true);
  isExporting = signal<boolean>(false);
  isAdvancedOpen = signal<boolean>(false);

  @HostListener('document:keydown.escape')
  onEscapeKey(): void {
    this.onClose();
  }

  columns = signal<ColumnExportConfig[]>([]);
  activePresetId = signal<string>('academic');
  highlightFailing = signal<boolean>(true);
  highlightPassing = signal<boolean>(false);
  failingThreshold = signal<number>(51);

  presets = PALETTE_PRESETS;
  swatches = COLOR_SWATCHES;

  visibleColumns = computed(() => this.columns().filter(c => c.visible));

  totalStudentsCount = computed(() => this.matrix()?.students?.length ?? 0);

  courseAverage = computed(() => {
    const students = this.matrix()?.students || [];
    if (students.length === 0) return 0;
    const sum = students.reduce((acc, s) => acc + s.average, 0);
    return Math.round((sum / students.length) * 10) / 10;
  });

  passingCount = computed(() => {
    const students = this.matrix()?.students || [];
    const threshold = this.failingThreshold();
    return students.filter(s => s.average >= threshold).length;
  });

  passingPercentage = computed(() => {
    const total = this.totalStudentsCount();
    if (total === 0) return 0;
    return Math.round((this.passingCount() / total) * 100);
  });

  ngOnInit(): void {
    this.loadMatrix();
  }

  loadMatrix(): void {
    this.isLoading.set(true);
    this.courseService.getCourseGradesMatrix(this.subjectId).subscribe({
      next: (data) => {
        this.matrix.set(data);
        this.initializeColumns(data);
        this.isLoading.set(false);
      },
      error: () => {
        this.isLoading.set(false);
      }
    });
  }

  initializeColumns(data: CourseGradesMatrix): void {
    this.columns.set(initializeDefaultColumns(data));
  }

  toggleAdvanced(): void {
    this.isAdvancedOpen.update(v => !v);
  }

  applyPreset(preset: PalettePreset): void {
    this.activePresetId.set(preset.id);
    const updated = this.columns().map(col => {
      if (col.key === 'average') {
        return { ...col, headerColor: preset.avgColor, textColor: '#FFFFFF' };
      } else if (col.isGrade) {
        return { ...col, headerColor: preset.labColor, textColor: '#FFFFFF' };
      } else {
        return { ...col, headerColor: preset.infoColor, textColor: '#FFFFFF' };
      }
    });
    this.columns.set(updated);
  }

  setColumnColor(columnKey: string, hexColor: string, textHex = '#FFFFFF'): void {
    this.activePresetId.set('custom');
    const updated = this.columns().map(col => {
      if (col.key === columnKey) {
        return { ...col, headerColor: hexColor, textColor: textHex };
      }
      return col;
    });
    this.columns.set(updated);
  }

  toggleColumnVisibility(columnKey: string): void {
    const updated = this.columns().map(col => {
      if (col.key === columnKey) {
        return { ...col, visible: !col.visible };
      }
      return col;
    });
    this.columns.set(updated);
  }

  resetToDefaults(): void {
    const matrixData = this.matrix();
    if (matrixData) {
      this.activePresetId.set('academic');
      this.initializeColumns(matrixData);
    }
  }

  getStudentScore(student: StudentGradesRow, col: ColumnExportConfig): number {
    return getStudentScore(student, col);
  }

  exportExcel(): void {
    const matrixData = this.matrix();
    if (!matrixData) return;

    this.isExporting.set(true);
    try {
      const activeCols = this.visibleColumns();
      const filename = `calificaciones_${this.subjectCode.toLowerCase()}_${new Date().toISOString().slice(0, 10)}.xls`;
      const xml = buildExcelSpreadsheetXml(
        matrixData,
        activeCols,
        this.highlightFailing(),
        this.highlightPassing(),
        this.failingThreshold()
      );
      downloadBlob(xml, 'application/vnd.ms-excel;charset=utf-8', filename);
      this.isExporting.set(false);
    } catch {
      this.isExporting.set(false);
    }
  }

  exportCsv(): void {
    const matrixData = this.matrix();
    if (!matrixData) return;

    this.isExporting.set(true);
    try {
      const activeCols = this.visibleColumns();
      const csv = buildCsvContent(matrixData, activeCols);
      const filename = `calificaciones_${this.subjectCode.toLowerCase()}_${new Date().toISOString().slice(0, 10)}.csv`;
      downloadBlob(csv, 'text/csv;charset=utf-8', filename);
      this.isExporting.set(false);
    } catch {
      this.isExporting.set(false);
    }
  }

  printReport(): void {
    window.print();
  }

  onClose(): void {
    this.dismiss.emit();
  }
}
