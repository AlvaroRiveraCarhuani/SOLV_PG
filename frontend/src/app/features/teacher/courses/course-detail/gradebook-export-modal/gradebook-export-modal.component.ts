import { 
  Component, 
  Input, 
  Output, 
  EventEmitter, 
  OnInit, 
  inject, 
  signal, 
  computed 
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
  LucidePrinter, 
  LucideCheck,
  LucideEye,
  LucideEyeOff,
  LucideRotateCcw
} from '@lucide/angular';
import { TeacherCourseService } from '../../../services/teacher-course.service';
import { CourseGradesMatrix, CourseExerciseHeader, StudentGradesRow } from '../../../models/teacher.models';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { DateTextPipe, formatSolvDate } from '@shared/pipes/date-text.pipe';


export interface ColumnExportConfig {
  key: string;
  label: string;
  visible: boolean;
  headerColor: string;
  textColor: string;
  isGrade?: boolean;
  exerciseId?: string;
}

export interface PalettePreset {
  id: string;
  name: string;
  infoColor: string;
  labColor: string;
  avgColor: string;
}

export const PALETTE_PRESETS: PalettePreset[] = [
  {
    id: 'academic',
    name: 'Académico (Predeterminado)',
    infoColor: '#334155', // Slate
    labColor: '#2563EB',  // Azul Real
    avgColor: '#15803D'   // Verde Esmeralda
  },
  {
    id: 'ocean',
    name: 'Océano Índigo',
    infoColor: '#1E293B',
    labColor: '#0284C7',
    avgColor: '#0D9488'
  },
  {
    id: 'emerald',
    name: 'Esmeralda & Menta',
    infoColor: '#134E4A',
    labColor: '#059669',
    avgColor: '#16A34A'
  },
  {
    id: 'graphite',
    name: 'Grafito Clásico',
    infoColor: '#1F2937',
    labColor: '#4B5563',
    avgColor: '#111827'
  },
  {
    id: 'amber',
    name: 'Cálido Ámbar',
    infoColor: '#78350F',
    labColor: '#D97706',
    avgColor: '#15803D'
  },
  {
    id: 'royal',
    name: 'Púrpura Real',
    infoColor: '#4C1D95',
    labColor: '#7C3AED',
    avgColor: '#0D9488'
  }
];

export const COLOR_SWATCHES: { label: string; hex: string; textHex: string }[] = [
  { label: 'Azul', hex: '#2563EB', textHex: '#FFFFFF' },
  { label: 'Índigo', hex: '#4F46E5', textHex: '#FFFFFF' },
  { label: 'Celeste', hex: '#0284C7', textHex: '#FFFFFF' },
  { label: 'Teal', hex: '#0D9488', textHex: '#FFFFFF' },
  { label: 'Verde', hex: '#15803D', textHex: '#FFFFFF' },
  { label: 'Ámbar', hex: '#D97706', textHex: '#FFFFFF' },
  { label: 'Pizarra', hex: '#334155', textHex: '#FFFFFF' },
  { label: 'Grafito', hex: '#1E293B', textHex: '#FFFFFF' },
  { label: 'Violeta', hex: '#7C3AED', textHex: '#FFFFFF' }
];

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
    LucidePrinter,
    LucideCheck,
    LucideEye,
    LucideEyeOff,
    LucideRotateCcw,
    MachineDataDirective
  ],
  templateUrl: './gradebook-export-modal.component.html',
  styleUrl: './gradebook-export-modal.component.scss'
})
export class GradebookExportModalComponent implements OnInit {
  @Input({ required: true }) subjectId!: string;
  @Input({ required: true }) subjectName!: string;
  @Input({ required: true }) subjectCode!: string;
  @Output() close = new EventEmitter<void>();

  private courseService = inject(TeacherCourseService);

  matrix = signal<CourseGradesMatrix | null>(null);
  isLoading = signal<boolean>(true);
  isExporting = signal<boolean>(false);
  isAdvancedOpen = signal<boolean>(false);

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
    const preset = PALETTE_PRESETS[0];
    const cols: ColumnExportConfig[] = [
      {
        key: 'student_id',
        label: 'ID Estudiante',
        visible: true,
        headerColor: preset.infoColor,
        textColor: '#FFFFFF'
      },
      {
        key: 'student_name',
        label: 'Nombre Completo',
        visible: true,
        headerColor: preset.infoColor,
        textColor: '#FFFFFF'
      },
      {
        key: 'student_email',
        label: 'Email',
        visible: true,
        headerColor: preset.infoColor,
        textColor: '#FFFFFF'
      }
    ];

    for (const ex of data.exercises || []) {
      cols.push({
        key: `ex_${ex.id}`,
        label: ex.title,
        visible: true,
        headerColor: preset.labColor,
        textColor: '#FFFFFF',
        isGrade: true,
        exerciseId: ex.id
      });
    }

    cols.push({
      key: 'average',
      label: 'Promedio Final',
      visible: true,
      headerColor: preset.avgColor,
      textColor: '#FFFFFF',
      isGrade: true
    });

    this.columns.set(cols);
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
    if (col.key === 'average') {
      return student.average;
    }
    if (col.exerciseId) {
      return student.grades[col.exerciseId] ?? 0;
    }
    return 0;
  }

  exportExcel(): void {
    const matrixData = this.matrix();
    if (!matrixData) return;

    this.isExporting.set(true);

    try {
      const activeCols = this.visibleColumns();
      const filename = `calificaciones_${this.subjectCode.toLowerCase()}_${new Date().toISOString().slice(0, 10)}.xls`;

      // Generar XML Spreadsheet 2003 compatible con Microsoft Excel, Google Drive y LibreOffice
      let xml = '<?xml version="1.0" encoding="UTF-8"?>\n';
      xml += '<?mso-application progid="Excel.Sheet"?>\n';
      xml += '<Workbook xmlns="urn:schemas-microsoft-com:office:spreadsheet"\n';
      xml += ' xmlns:o="urn:schemas-microsoft-com:office:office"\n';
      xml += ' xmlns:x="urn:schemas-microsoft-com:office:excel"\n';
      xml += ' xmlns:ss="urn:schemas-microsoft-com:office:spreadsheet"\n';
      xml += ' xmlns:html="http://www.w3.org/TR/REC-html40">\n';

      // Estilos
      xml += ' <Styles>\n';
      xml += '  <Style ss:ID="Default" ss:Name="Normal">\n';
      xml += '   <Alignment ss:Vertical="Center"/>\n';
      xml += '   <Font ss:FontName="Segoe UI" ss:Size="10" ss:Color="#1E293B"/>\n';
      xml += '  </Style>\n';

      // Estilo Título de Curso
      xml += '  <Style ss:ID="CourseTitleStyle">\n';
      xml += '   <Font ss:FontName="Segoe UI" ss:Size="14" ss:Bold="1" ss:Color="#0F172A"/>\n';
      xml += '  </Style>\n';

      // Estilo de datos con bordes
      xml += '  <Style ss:ID="DataCell">\n';
      xml += '   <Borders>\n';
      xml += '    <Border ss:Position="Bottom" ss:LineStyle="Continuous" ss:Weight="1" ss:Color="#E2E8F0"/>\n';
      xml += '    <Border ss:Position="Left" ss:LineStyle="Continuous" ss:Weight="1" ss:Color="#E2E8F0"/>\n';
      xml += '    <Border ss:Position="Right" ss:LineStyle="Continuous" ss:Weight="1" ss:Color="#E2E8F0"/>\n';
      xml += '    <Border ss:Position="Top" ss:LineStyle="Continuous" ss:Weight="1" ss:Color="#E2E8F0"/>\n';
      xml += '   </Borders>\n';
      xml += '  </Style>\n';

      // Estilos dinámicos para encabezados según los colores seleccionados
      activeCols.forEach((col, idx) => {
        xml += `  <Style ss:ID="HeaderCol_${idx}">\n`;
        xml += '   <Alignment ss:Horizontal="Center" ss:Vertical="Center"/>\n';
        xml += `   <Font ss:FontName="Segoe UI" ss:Size="11" ss:Bold="1" ss:Color="${col.textColor}"/>\n`;
        xml += `   <Interior ss:Color="${col.headerColor}" ss:Pattern="Solid"/>\n`;
        xml += '   <Borders>\n';
        xml += '    <Border ss:Position="Bottom" ss:LineStyle="Continuous" ss:Weight="1" ss:Color="#CBD5E1"/>\n';
        xml += '    <Border ss:Position="Left" ss:LineStyle="Continuous" ss:Weight="1" ss:Color="#CBD5E1"/>\n';
        xml += '    <Border ss:Position="Right" ss:LineStyle="Continuous" ss:Weight="1" ss:Color="#CBD5E1"/>\n';
        xml += '    <Border ss:Position="Top" ss:LineStyle="Continuous" ss:Weight="1" ss:Color="#CBD5E1"/>\n';
        xml += '   </Borders>\n';
        xml += '  </Style>\n';
      });

      // Estilos para notas destacadas
      xml += '  <Style ss:ID="AverageScore">\n';
      xml += '   <Alignment ss:Horizontal="Right"/>\n';
      xml += '   <Font ss:FontName="Segoe UI" ss:Size="10" ss:Color="#15803D" ss:Bold="1"/>\n';
      xml += '   <Borders>\n';
      xml += '    <Border ss:Position="Bottom" ss:LineStyle="Continuous" ss:Weight="1" ss:Color="#E2E8F0"/>\n';
      xml += '    <Border ss:Position="Left" ss:LineStyle="Continuous" ss:Weight="1" ss:Color="#E2E8F0"/>\n';
      xml += '    <Border ss:Position="Right" ss:LineStyle="Continuous" ss:Weight="1" ss:Color="#E2E8F0"/>\n';
      xml += '    <Border ss:Position="Top" ss:LineStyle="Continuous" ss:Weight="1" ss:Color="#E2E8F0"/>\n';
      xml += '   </Borders>\n';
      xml += '  </Style>\n';

      xml += '  <Style ss:ID="PassingScore">\n';
      xml += '   <Alignment ss:Horizontal="Right"/>\n';
      xml += '   <Font ss:FontName="Segoe UI" ss:Size="10" ss:Color="#15803D" ss:Bold="1"/>\n';
      xml += '   <Interior ss:Color="#DCFCE7" ss:Pattern="Solid"/>\n';
      xml += '   <Borders>\n';
      xml += '    <Border ss:Position="Bottom" ss:LineStyle="Continuous" ss:Weight="1" ss:Color="#BBF7D0"/>\n';
      xml += '    <Border ss:Position="Left" ss:LineStyle="Continuous" ss:Weight="1" ss:Color="#BBF7D0"/>\n';
      xml += '    <Border ss:Position="Right" ss:LineStyle="Continuous" ss:Weight="1" ss:Color="#BBF7D0"/>\n';
      xml += '    <Border ss:Position="Top" ss:LineStyle="Continuous" ss:Weight="1" ss:Color="#BBF7D0"/>\n';
      xml += '   </Borders>\n';
      xml += '  </Style>\n';

      xml += '  <Style ss:ID="FailingScore">\n';
      xml += '   <Alignment ss:Horizontal="Right"/>\n';
      xml += '   <Font ss:FontName="Segoe UI" ss:Size="10" ss:Color="#DC2626" ss:Bold="1"/>\n';
      xml += '   <Interior ss:Color="#FEE2E2" ss:Pattern="Solid"/>\n';
      xml += '   <Borders>\n';
      xml += '    <Border ss:Position="Bottom" ss:LineStyle="Continuous" ss:Weight="1" ss:Color="#FECACA"/>\n';
      xml += '    <Border ss:Position="Left" ss:LineStyle="Continuous" ss:Weight="1" ss:Color="#FECACA"/>\n';
      xml += '    <Border ss:Position="Right" ss:LineStyle="Continuous" ss:Weight="1" ss:Color="#FECACA"/>\n';
      xml += '    <Border ss:Position="Top" ss:LineStyle="Continuous" ss:Weight="1" ss:Color="#FECACA"/>\n';
      xml += '   </Borders>\n';
      xml += '  </Style>\n';

      xml += ' </Styles>\n';

      // Hoja de trabajo
      xml += ' <Worksheet ss:Name="Calificaciones">\n';
      xml += '  <Table>\n';

      // Column widths
      activeCols.forEach(col => {
        const width = col.key === 'student_name' ? 180 : (col.key === 'student_email' ? 200 : 100);
        xml += `   <Column ss:Width="${width}"/>\n`;
      });

      // Fila 1: Título de la Materia
      xml += '   <Row ss:Height="28">\n';
      xml += `    <Cell ss:StyleID="CourseTitleStyle"><Data ss:Type="String">${matrixData.subject_name} (${matrixData.subject_code})</Data></Cell>\n`;
      xml += '   </Row>\n';

      // Fila 2: Metadata de exportación
      xml += '   <Row ss:Height="18">\n';
      xml += `    <Cell><Data ss:Type="String">Fecha de emisión: ${formatSolvDate(new Date(), 'date')} · Total Estudiantes: ${matrixData.students.length}</Data></Cell>\n`;
      xml += '   </Row>\n';


      // Fila en blanco
      xml += '   <Row ss:Height="10"/>\n';

      // Fila de Encabezados con estilos personalizados
      xml += '   <Row ss:Height="24">\n';
      activeCols.forEach((col, idx) => {
        xml += `    <Cell ss:StyleID="HeaderCol_${idx}"><Data ss:Type="String">${col.label}</Data></Cell>\n`;
      });
      xml += '   </Row>\n';

      // Filas de Estudiantes
      for (const student of matrixData.students) {
        xml += '   <Row ss:Height="20">\n';
        for (const col of activeCols) {
          if (col.key === 'student_id') {
            xml += `    <Cell ss:StyleID="DataCell"><Data ss:Type="String">${student.student_id}</Data></Cell>\n`;
          } else if (col.key === 'student_name') {
            xml += `    <Cell ss:StyleID="DataCell"><Data ss:Type="String">${student.student_name}</Data></Cell>\n`;
          } else if (col.key === 'student_email') {
            xml += `    <Cell ss:StyleID="DataCell"><Data ss:Type="String">${student.student_email}</Data></Cell>\n`;
          } else {
            const score = this.getStudentScore(student, col);
            const isFailing = this.highlightFailing() && score < this.failingThreshold();
            const isPassing = this.highlightPassing() && score >= this.failingThreshold();
            let style = 'DataCell';
            if (isFailing) {
              style = 'FailingScore';
            } else if (isPassing) {
              style = 'PassingScore';
            } else if (col.key === 'average') {
              style = 'AverageScore';
            }
            xml += `    <Cell ss:StyleID="${style}"><Data ss:Type="Number">${score}</Data></Cell>\n`;
          }
        }
        xml += '   </Row>\n';
      }

      xml += '  </Table>\n';
      xml += ' </Worksheet>\n';
      xml += '</Workbook>';

      const blob = new Blob([xml], { type: 'application/vnd.ms-excel;charset=utf-8' });
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      window.URL.revokeObjectURL(url);
      this.isExporting.set(false);
    } catch {
      this.isExporting.set(false);
    }
  }

  exportCsv(): void {
    const matrixData = this.matrix();
    if (!matrixData) return;

    this.isExporting.set(true);
    const activeCols = this.visibleColumns();

    let csvContent = '\uFEFF'; // UTF-8 BOM
    csvContent += activeCols.map(c => `"${c.label.replace(/"/g, '""')}"`).join(',') + '\r\n';

    for (const student of matrixData.students) {
      const rowValues = activeCols.map(col => {
        if (col.key === 'student_id') return `"${student.student_id}"`;
        if (col.key === 'student_name') return `"${student.student_name}"`;
        if (col.key === 'student_email') return `"${student.student_email}"`;
        return `${this.getStudentScore(student, col)}`;
      });
      csvContent += rowValues.join(',') + '\r\n';
    }

    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8' });
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `calificaciones_${this.subjectCode.toLowerCase()}_${new Date().toISOString().slice(0, 10)}.csv`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    window.URL.revokeObjectURL(url);
    this.isExporting.set(false);
  }

  printReport(): void {
    window.print();
  }

  onClose(): void {
    this.close.emit();
  }
}
