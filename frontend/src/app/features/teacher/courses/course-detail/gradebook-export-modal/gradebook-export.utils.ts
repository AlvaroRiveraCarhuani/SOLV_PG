import { CourseGradesMatrix, StudentGradesRow } from '../../../models/teacher.models';
import { ColumnExportConfig, PALETTE_PRESETS } from './gradebook-export.presets';
import { formatSolvDate } from '@shared/pipes/date-text.pipe';

export function getStudentScore(student: StudentGradesRow, col: ColumnExportConfig): number {
  if (col.key === 'average') {
    return student.average;
  }
  if (col.exerciseId) {
    return student.grades[col.exerciseId] ?? 0;
  }
  return 0;
}

export function initializeDefaultColumns(data: CourseGradesMatrix): ColumnExportConfig[] {
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

  return cols;
}

export function buildExcelSpreadsheetXml(
  matrixData: CourseGradesMatrix,
  activeCols: ColumnExportConfig[],
  highlightFailing: boolean,
  highlightPassing: boolean,
  failingThreshold: number
): string {
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

  xml += '  <Style ss:ID="CourseTitleStyle">\n';
  xml += '   <Font ss:FontName="Segoe UI" ss:Size="14" ss:Bold="1" ss:Color="#0F172A"/>\n';
  xml += '  </Style>\n';

  xml += '  <Style ss:ID="DataCell">\n';
  xml += '   <Borders>\n';
  xml += '    <Border ss:Position="Bottom" ss:LineStyle="Continuous" ss:Weight="1" ss:Color="#E2E8F0"/>\n';
  xml += '    <Border ss:Position="Left" ss:LineStyle="Continuous" ss:Weight="1" ss:Color="#E2E8F0"/>\n';
  xml += '    <Border ss:Position="Right" ss:LineStyle="Continuous" ss:Weight="1" ss:Color="#E2E8F0"/>\n';
  xml += '    <Border ss:Position="Top" ss:LineStyle="Continuous" ss:Weight="1" ss:Color="#E2E8F0"/>\n';
  xml += '   </Borders>\n';
  xml += '  </Style>\n';

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

  activeCols.forEach(col => {
    const width = col.key === 'student_name' ? 180 : (col.key === 'student_email' ? 200 : 100);
    xml += `   <Column ss:Width="${width}"/>\n`;
  });

  xml += '   <Row ss:Height="28">\n';
  xml += `    <Cell ss:StyleID="CourseTitleStyle"><Data ss:Type="String">${matrixData.subject_name} (${matrixData.subject_code})</Data></Cell>\n`;
  xml += '   </Row>\n';

  xml += '   <Row ss:Height="18">\n';
  xml += `    <Cell><Data ss:Type="String">Fecha de emisión: ${formatSolvDate(new Date(), 'date')} · Total Estudiantes: ${matrixData.students.length}</Data></Cell>\n`;
  xml += '   </Row>\n';

  xml += '   <Row ss:Height="10"/>\n';

  xml += '   <Row ss:Height="24">\n';
  activeCols.forEach((col, idx) => {
    xml += `    <Cell ss:StyleID="HeaderCol_${idx}"><Data ss:Type="String">${col.label}</Data></Cell>\n`;
  });
  xml += '   </Row>\n';

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
        const score = getStudentScore(student, col);
        const isFailing = highlightFailing && score < failingThreshold;
        const isPassing = highlightPassing && score >= failingThreshold;
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

  return xml;
}

export function buildCsvContent(
  matrixData: CourseGradesMatrix,
  activeCols: ColumnExportConfig[]
): string {
  let csvContent = '\uFEFF'; // UTF-8 BOM
  csvContent += activeCols.map(c => `"${c.label.replace(/"/g, '""')}"`).join(',') + '\r\n';

  for (const student of matrixData.students) {
    const rowValues = activeCols.map(col => {
      if (col.key === 'student_id') return `"${student.student_id}"`;
      if (col.key === 'student_name') return `"${student.student_name}"`;
      if (col.key === 'student_email') return `"${student.student_email}"`;
      return `${getStudentScore(student, col)}`;
    });
    csvContent += rowValues.join(',') + '\r\n';
  }

  return csvContent;
}

export function downloadBlob(content: string, mimeType: string, filename: string): void {
  const blob = new Blob([content], { type: mimeType });
  const url = window.URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  window.URL.revokeObjectURL(url);
}
