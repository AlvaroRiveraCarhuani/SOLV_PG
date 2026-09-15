import { Component, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { CourseLoadSummary } from '@core/models/admin.model';
import { LucideBookOpen, LucideUsers, LucideEye } from '@lucide/angular';

@Component({
  selector: 'solv-course-load',
  standalone: true,
  imports: [CommonModule, LucideBookOpen, LucideUsers, LucideEye],
  template: `
    <div class="card-container">
      <div class="card-header">
        <div class="header-title-group">
          <svg lucideBookOpen class="header-icon"></svg>
          <h3 class="card-title">Distribución de Carga por Materia</h3>
        </div>
        <span class="count-tag">{{ courses().length }} materias activas</span>
      </div>

      <div class="table-wrapper">
        <table class="course-table">
          <thead>
            <tr>
              <th>MATERIA / CURSO</th>
              <th>DOCENTE A CARGO</th>
              <th>ALUMNOS ACTIVOS</th>
              <th>RAM TOTAL</th>
              <th class="text-right">DETALLE</th>
            </tr>
          </thead>
          <tbody>
            @for (course of courses(); track course.id) {
              <tr>
                <td>
                  <span class="course-name">{{ course.course_name }}</span>
                </td>
                <td>
                  <span class="teacher-name">{{ course.teacher_name }}</span>
                </td>
                <td>
                  <div class="students-count">
                    <svg lucideUsers class="count-icon"></svg>
                    <span class="count-mono">{{ course.active_students }}</span>
                    @if (course.hibernated_students > 0) {
                      <span class="hibernated-sub">({{ course.hibernated_students }} hib.)</span>
                    }
                  </div>
                </td>
                <td>
                  <span class="ram-mono">{{ formatRAM(course.ram_used_mb) }}</span>
                </td>
                <td class="text-right">
                  <button 
                    class="btn-detail" 
                    (click)="viewDetails.emit(course)"
                    title="Inspeccionar contenedores de esta materia">
                    <svg lucideEye class="btn-icon"></svg>
                    <span>Ver Detalle</span>
                  </button>
                </td>
              </tr>
            } @empty {
              <tr>
                <td colspan="5" class="empty-state">
                  <span>No hay materias con laboratorios activos en este período.</span>
                </td>
              </tr>
            }
          </tbody>
        </table>
      </div>
    </div>
  `,
  styles: [`
    .card-container {
      background-color: var(--bg-surface, #FFFFFF);
      border: 1px solid var(--border-subtle, #E2E8F0);
      border-radius: var(--radius-lg, 8px);
      display: flex;
      flex-direction: column;
      overflow: hidden;
      height: 100%;
    }

    .card-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: var(--space-4, 16px) var(--space-5, 20px);
      border-bottom: 1px solid var(--border-subtle, #E2E8F0);
    }

    .header-title-group {
      display: flex;
      align-items: center;
      gap: var(--space-2-5, 10px);
    }

    .header-icon {
      width: 16px;
      height: 16px;
      color: var(--tenant-primary, #2563EB);
    }

    .card-title {
      font-size: 14px;
      font-weight: 600;
      color: var(--text-primary, #0F172A);
      margin: 0;
    }

    .count-tag {
      font-size: 11px;
      font-weight: 500;
      color: var(--text-muted, #64748B);
      background-color: var(--bg-canvas, #F1F5F9);
      padding: 2px 8px;
      border-radius: var(--radius-full, 9999px);
    }

    .table-wrapper {
      overflow-x: auto;
    }

    .course-table {
      width: 100%;
      border-collapse: collapse;
      font-size: 13px;
      text-align: left;

      th {
        background-color: #F8FAFC;
        padding: 10px 16px;
        font-size: 11px;
        font-weight: 700;
        color: var(--text-muted, #64748B);
        letter-spacing: 0.05em;
        border-bottom: 1px solid var(--border-subtle, #E2E8F0);
        white-space: nowrap;
      }

      td {
        padding: 12px 16px;
        border-bottom: 1px solid var(--border-subtle, #F1F5F9);
        vertical-align: middle;
      }

      tr:hover td {
        background-color: #F8FAFC;
      }
    }

    .course-name {
      font-weight: 600;
      color: var(--text-primary, #0F172A);
    }

    .teacher-name {
      color: var(--text-secondary, #475569);
    }

    .students-count {
      display: inline-flex;
      align-items: center;
      gap: 6px;
    }

    .count-icon {
      width: 14px;
      height: 14px;
      color: var(--text-muted, #94A3B8);
    }

    .count-mono {
      font-family: 'JetBrains Mono', monospace;
      font-weight: 600;
      color: var(--text-primary, #0F172A);
    }

    .hibernated-sub {
      font-size: 11px;
      color: var(--text-muted, #94A3B8);
    }

    .ram-mono {
      font-family: 'JetBrains Mono', monospace;
      font-size: 12px;
      font-weight: 500;
      color: var(--text-secondary, #334155);
    }

    .text-right {
      text-align: right;
    }

    .btn-detail {
      display: inline-flex;
      align-items: center;
      gap: 5px;
      padding: 5px 10px;
      background-color: var(--bg-surface, #FFFFFF);
      border: 1px solid var(--border-subtle, #CBD5E1);
      border-radius: var(--radius-md, 6px);
      font-size: 12px;
      font-weight: 500;
      color: var(--text-secondary, #334155);
      cursor: pointer;
      transition: all 150ms ease;

      &:hover {
        background-color: var(--tenant-primary-subtle, rgba(37, 99, 235, 0.08));
        color: var(--tenant-primary, #2563EB);
        border-color: var(--tenant-primary, #2563EB);
      }

      .btn-icon {
        width: 13px;
        height: 13px;
      }
    }

    .empty-state {
      text-align: center;
      padding: 32px 16px;
      color: var(--text-muted, #94A3B8);
    }
  `]
})
export class CourseLoadComponent {
  courses = input.required<CourseLoadSummary[]>();
  viewDetails = output<CourseLoadSummary>();

  formatRAM(mb: number): string {
    if (mb >= 1024) {
      return `${(mb / 1024).toFixed(1)} GB`;
    }
    return `${mb} MB`;
  }
}
