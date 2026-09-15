import { Component, input, output, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { CourseLoadSummary } from '@core/models/admin.model';
import { 
  LucideBookOpen, 
  LucideUsers, 
  LucideEye, 
  LucideSearch, 
  LucideChevronLeft, 
  LucideChevronRight,
  LucideX
} from '@lucide/angular';

@Component({
  selector: 'solv-course-load',
  standalone: true,
  imports: [
    CommonModule, 
    FormsModule, 
    LucideBookOpen, 
    LucideUsers, 
    LucideEye, 
    LucideSearch, 
    LucideChevronLeft, 
    LucideChevronRight,
    LucideX
  ],
  template: `
    <div class="card-container">
      <!-- Header -->
      <div class="card-header">
        <div class="header-title-group">
          <svg lucideBookOpen class="header-icon"></svg>
          <h3 class="card-title">Distribución de Carga por Materia</h3>
        </div>
        <span class="count-tag">{{ courses().length }} materias registradas</span>
      </div>

      <!-- Search & Filter Toolbar -->
      <div class="toolbar">
        <div class="search-box">
          <svg lucideSearch class="search-icon"></svg>
          <input 
            type="text" 
            class="search-input" 
            placeholder="Buscar por materia o docente..."
            [ngModel]="searchTerm()"
            (ngModelChange)="onSearchChange($event)"
          />
          @if (searchTerm()) {
            <button class="btn-clear" (click)="clearSearch()" title="Limpiar búsqueda">
              <svg lucideX class="clear-icon"></svg>
            </button>
          }
        </div>

        <div class="filter-meta">
          @if (activeCoursesCount() > 0) {
            <span class="badge-active">{{ activeCoursesCount() }} con laboratorios activos</span>
          } @else {
            <span class="badge-idle">Sin laboratorios activos</span>
          }
        </div>
      </div>

      <!-- Table -->
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
            @for (course of paginatedCourses(); track course.id) {
              <tr [class.row-active]="course.active_students > 0">
                <td>
                  <div class="course-cell">
                    <span class="course-name">{{ course.course_name }}</span>
                    @if (course.active_students > 0) {
                      <span class="active-dot" title="Laboratorio activo ahora"></span>
                    }
                  </div>
                </td>
                <td>
                  <span class="teacher-name">{{ course.teacher_name }}</span>
                </td>
                <td>
                  <div class="students-count">
                    <svg lucideUsers class="count-icon"></svg>
                    <span class="count-mono" [class.highlight]="course.active_students > 0">{{ course.active_students }}</span>
                    @if (course.hibernated_students > 0) {
                      <span class="hibernated-sub">({{ course.hibernated_students }} hib.)</span>
                    }
                  </div>
                </td>
                <td>
                  <span class="ram-mono" [class.highlight]="course.ram_used_mb > 0">{{ formatRAM(course.ram_used_mb) }}</span>
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
                  @if (searchTerm()) {
                    <span>No se encontraron materias que coincidan con "{{ searchTerm() }}".</span>
                  } @else {
                    <span>No hay materias registradas en este período.</span>
                  }
                </td>
              </tr>
            }
          </tbody>
        </table>
      </div>

      <!-- Pagination Footer -->
      @if (totalPages() > 1) {
        <div class="pagination-footer">
          <span class="pagination-info">
            Mostrando {{ paginationStart() }} - {{ paginationEnd() }} de {{ filteredCourses().length }} materias
          </span>
          <div class="pagination-controls">
            <button 
              class="btn-page" 
              [disabled]="currentPage() === 1" 
              (click)="goToPage(currentPage() - 1)"
              title="Página anterior">
              <svg lucideChevronLeft class="page-icon"></svg>
            </button>
            <span class="page-current">Página {{ currentPage() }} de {{ totalPages() }}</span>
            <button 
              class="btn-page" 
              [disabled]="currentPage() === totalPages()" 
              (click)="goToPage(currentPage() + 1)"
              title="Página siguiente">
              <svg lucideChevronRight class="page-icon"></svg>
            </button>
          </div>
        </div>
      }
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

    .toolbar {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: var(--space-3, 12px) var(--space-5, 20px);
      background-color: #FAFAFC;
      border-bottom: 1px solid var(--border-subtle, #E2E8F0);
      gap: var(--space-3, 12px);
      flex-wrap: wrap;
    }

    .search-box {
      position: relative;
      display: flex;
      align-items: center;
      flex: 1;
      max-width: 320px;
    }

    .search-icon {
      position: absolute;
      left: 10px;
      width: 14px;
      height: 14px;
      color: var(--text-muted, #94A3B8);
      pointer-events: none;
    }

    .search-input {
      width: 100%;
      height: 32px;
      padding: 0 30px 0 32px;
      background-color: var(--bg-surface, #FFFFFF);
      border: 1px solid var(--border-subtle, #CBD5E1);
      border-radius: var(--radius-md, 6px);
      font-size: 12px;
      color: var(--text-primary, #0F172A);
      outline: none;
      transition: border-color 150ms ease;

      &:focus {
        border-color: var(--tenant-primary, #2563EB);
        box-shadow: 0 0 0 1px var(--tenant-primary, #2563EB);
      }

      &::placeholder {
        color: var(--text-muted, #94A3B8);
      }
    }

    .btn-clear {
      position: absolute;
      right: 8px;
      background: transparent;
      border: none;
      cursor: pointer;
      color: var(--text-muted, #94A3B8);
      padding: 2px;
      display: flex;
      align-items: center;
      justify-content: center;

      &:hover {
        color: var(--text-primary, #0F172A);
      }

      .clear-icon {
        width: 12px;
        height: 12px;
      }
    }

    .filter-meta {
      font-size: 11px;
    }

    .badge-active {
      background-color: #DCFCE7;
      color: #15803D;
      padding: 2px 8px;
      border-radius: var(--radius-full, 9999px);
      font-weight: 600;
    }

    .badge-idle {
      background-color: var(--bg-canvas, #F1F5F9);
      color: var(--text-muted, #64748B);
      padding: 2px 8px;
      border-radius: var(--radius-full, 9999px);
      font-weight: 500;
    }

    .table-wrapper {
      overflow-x: auto;
      flex: 1;
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
        border-bottom: 1px solid var(--border-subtle, #E2E8F0);
        vertical-align: middle;
      }

      tr:last-child td {
        border-bottom: none;
      }

      tr:hover td {
        background-color: #F8FAFC;
      }

      tr.row-active td {
        background-color: rgba(37, 99, 235, 0.02);
      }

      .text-right {
        text-align: right;
      }
    }

    .course-cell {
      display: flex;
      align-items: center;
      gap: 6px;
    }

    .course-name {
      font-weight: 600;
      color: var(--text-primary, #0F172A);
      display: block;
    }

    .active-dot {
      width: 6px;
      height: 6px;
      border-radius: 50%;
      background-color: #10B981;
      flex-shrink: 0;
    }

    .teacher-name {
      color: var(--text-secondary, #475569);
      font-size: 12px;
    }

    .students-count {
      display: flex;
      align-items: center;
      gap: var(--space-1-5, 6px);
    }

    .count-icon {
      width: 14px;
      height: 14px;
      color: var(--text-muted, #94A3B8);
    }

    .count-mono {
      font-family: var(--font-mono, monospace);
      font-weight: 600;
      color: var(--text-primary, #0F172A);

      &.highlight {
        color: var(--tenant-primary, #2563EB);
      }
    }

    .hibernated-sub {
      font-size: 11px;
      color: var(--text-muted, #94A3B8);
    }

    .ram-mono {
      font-family: var(--font-mono, monospace);
      font-size: 12px;
      color: var(--text-secondary, #475569);

      &.highlight {
        color: #7C3AED;
        font-weight: 600;
      }
    }

    .btn-detail {
      display: inline-flex;
      align-items: center;
      gap: var(--space-1-5, 6px);
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
      font-size: 13px;
    }

    .pagination-footer {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: var(--space-3, 12px) var(--space-5, 20px);
      background-color: #FAFAFC;
      border-top: 1px solid var(--border-subtle, #E2E8F0);
      font-size: 12px;
      color: var(--text-secondary, #64748B);
    }

    .pagination-controls {
      display: flex;
      align-items: center;
      gap: var(--space-2, 8px);
    }

    .btn-page {
      width: 28px;
      height: 28px;
      display: flex;
      align-items: center;
      justify-content: center;
      background-color: var(--bg-surface, #FFFFFF);
      border: 1px solid var(--border-subtle, #CBD5E1);
      border-radius: var(--radius-md, 6px);
      color: var(--text-secondary, #334155);
      cursor: pointer;
      transition: all 150ms ease;

      &:hover:not(:disabled) {
        background-color: var(--bg-canvas, #F1F5F9);
        color: var(--tenant-primary, #2563EB);
      }

      &:disabled {
        opacity: 0.4;
        cursor: not-allowed;
      }

      .page-icon {
        width: 14px;
        height: 14px;
      }
    }

    .page-current {
      font-size: 11px;
      font-weight: 500;
      color: var(--text-primary, #0F172A);
      padding: 0 4px;
    }
  `]
})
export class CourseLoadComponent {
  courses = input.required<CourseLoadSummary[]>();
  viewDetails = output<CourseLoadSummary>();

  searchTerm = signal<string>('');
  currentPage = signal<number>(1);
  pageSize = signal<number>(8);

  activeCoursesCount = computed(() => 
    this.courses().filter(c => c.active_students > 0).length
  );

  filteredCourses = computed(() => {
    const term = this.searchTerm().toLowerCase().trim();
    let list = [...this.courses()];

    // Orden prioritario: Materias con alumnos activos o consumo de RAM primero
    list.sort((a, b) => {
      if (b.active_students !== a.active_students) {
        return b.active_students - a.active_students;
      }
      if (b.ram_used_mb !== a.ram_used_mb) {
        return b.ram_used_mb - a.ram_used_mb;
      }
      return a.course_name.localeCompare(b.course_name);
    });

    if (!term) return list;

    return list.filter(c => 
      c.course_name.toLowerCase().includes(term) ||
      c.teacher_name.toLowerCase().includes(term)
    );
  });

  totalPages = computed(() => 
    Math.max(1, Math.ceil(this.filteredCourses().length / this.pageSize()))
  );

  paginatedCourses = computed(() => {
    const start = (this.currentPage() - 1) * this.pageSize();
    return this.filteredCourses().slice(start, start + this.pageSize());
  });

  paginationStart = computed(() => {
    if (this.filteredCourses().length === 0) return 0;
    return (this.currentPage() - 1) * this.pageSize() + 1;
  });

  paginationEnd = computed(() => {
    return Math.min(this.currentPage() * this.pageSize(), this.filteredCourses().length);
  });

  onSearchChange(term: string): void {
    this.searchTerm.set(term);
    this.currentPage.set(1);
  }

  clearSearch(): void {
    this.searchTerm.set('');
    this.currentPage.set(1);
  }

  goToPage(page: number): void {
    if (page >= 1 && page <= this.totalPages()) {
      this.currentPage.set(page);
    }
  }

  formatRAM(mb: number): string {
    if (mb >= 1024) {
      return `${(mb / 1024).toFixed(1)} GB`;
    }
    return `${mb} MB`;
  }
}
