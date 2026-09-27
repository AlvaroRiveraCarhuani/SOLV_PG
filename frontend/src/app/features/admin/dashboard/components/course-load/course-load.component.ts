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
  selector: 'course-load',
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
  styleUrl: './course-load.component.scss',
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
