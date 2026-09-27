import { Component, input, output, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { DockerContainerSummary } from '@core/models/admin.model';
import { StatusBadgeComponent } from '@shared/components/status-badge/status-badge.component';
import { LucideSearch, LucidePower, LucideChevronLeft, LucideChevronRight } from '@lucide/angular';

@Component({
  selector: 'container-table',
  standalone: true,
  imports: [
    CommonModule, 
    FormsModule, 
    StatusBadgeComponent, 
    LucideSearch, 
    LucidePower, 
    LucideChevronLeft, 
    LucideChevronRight
  ],
  template: `
    <div class="table-card">
      <div class="table-toolbar">
        <div class="toolbar-left">
          <h3 class="table-title">Contenedores de Laboratorio Activos</h3>
          <span class="count-badge">{{ filteredContainers().length }} instancias</span>
        </div>
        <div class="toolbar-right">
          <div class="search-box">
            <svg lucideSearch class="search-icon"></svg>
            <input 
              type="text" 
              placeholder="Filtrar por estudiante, curso o imagen..." 
              [ngModel]="searchTerm()" 
              (ngModelChange)="onSearchChange($event)"
              class="search-input"
            />
          </div>
        </div>
      </div>

      <div class="table-wrapper">
        <table class="data-table">
          <thead>
            <tr>
              <th>ESTUDIANTE</th>
              <th>CURSO</th>
              <th>IMAGEN DOCKER</th>
              <th>RAM</th>
              <th>TTL RESTANTE</th>
              <th>ESTADO</th>
              <th class="text-right">ACCIONES</th>
            </tr>
          </thead>
          <tbody>
            @for (c of paginatedContainers(); track c.id) {
              <tr>
                <td>
                  <div class="user-cell">
                    <span class="user-name">{{ c.student_name }}</span>
                    <span class="user-email">{{ c.student_email }}</span>
                  </div>
                </td>
                <td>
                  <span class="course-text">{{ c.course_name }}</span>
                </td>
                <td>
                  <span class="mono-tag">{{ c.image_tag }}</span>
                </td>
                <td>
                  <span class="mono-val">{{ c.memory_used_mb }} / {{ c.memory_limit_mb }} MB</span>
                </td>
                <td>
                  <span class="mono-val" [class.ttl-urgent]="c.ttl_remaining_seconds > 0 && c.ttl_remaining_seconds < 600">
                    {{ formatTTL(c.ttl_remaining_seconds) }}
                  </span>
                </td>
                <td>
                  <status-badge [status]="c.status" />
                </td>
                <td class="text-right">
                  <div class="actions-group">
                    <button 
                      class="btn-icon danger" 
                      title="Detener Contenedor" 
                      (click)="stopContainer.emit(c)">
                      <svg lucidePower class="icon"></svg>
                    </button>
                  </div>
                </td>
              </tr>
            } @empty {
              <tr>
                <td colspan="7" class="empty-cell">
                  <span>No se encontraron contenedores que coincidan con la búsqueda.</span>
                </td>
              </tr>
            }
          </tbody>
        </table>
      </div>

      <!-- Paginación -->
      @if (filteredContainers().length > pageSize()) {
        <footer class="pagination-footer">
          <div class="pagination-info">
            Mostrando página <strong>{{ currentPage() }}</strong> de <strong>{{ totalPages() }}</strong> 
            ({{ filteredContainers().length }} instancias)
          </div>
          <div class="pagination-controls">
            <button 
              class="btn-page" 
              [disabled]="currentPage() === 1" 
              (click)="goToPage(currentPage() - 1)">
              <svg lucideChevronLeft class="page-icon"></svg>
              <span>Anterior</span>
            </button>
            <button 
              class="btn-page" 
              [disabled]="currentPage() === totalPages()" 
              (click)="goToPage(currentPage() + 1)">
              <span>Siguiente</span>
              <svg lucideChevronRight class="page-icon"></svg>
            </button>
          </div>
        </footer>
      }
    </div>
  `,
  styleUrl: './container-table.component.scss',
})
export class ContainerTableComponent {
  containers = input.required<DockerContainerSummary[]>();
  stopContainer = output<DockerContainerSummary>();

  searchTerm = signal<string>('');
  pageSize = signal<number>(8);
  currentPage = signal<number>(1);

  filteredContainers = computed(() => {
    const term = this.searchTerm().toLowerCase().trim();
    if (!term) return this.containers();
    return this.containers().filter(c => 
      c.student_name.toLowerCase().includes(term) ||
      c.student_email.toLowerCase().includes(term) ||
      c.course_name.toLowerCase().includes(term) ||
      c.image_tag.toLowerCase().includes(term)
    );
  });

  totalPages = computed(() => Math.max(1, Math.ceil(this.filteredContainers().length / this.pageSize())));

  paginatedContainers = computed(() => {
    const start = (this.currentPage() - 1) * this.pageSize();
    return this.filteredContainers().slice(start, start + this.pageSize());
  });

  onSearchChange(term: string): void {
    this.searchTerm.set(term);
    this.currentPage.set(1);
  }

  goToPage(p: number): void {
    if (p >= 1 && p <= this.totalPages()) {
      this.currentPage.set(p);
    }
  }

  formatTTL(seconds: number): string {
    if (seconds <= 0) return 'Expirado';
    const m = Math.floor(seconds / 60);
    const s = seconds % 60;
    return `${m}m ${s < 10 ? '0' : ''}${s}s`;
  }
}
