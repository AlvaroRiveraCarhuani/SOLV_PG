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
  styles: [`
    .table-card {
      background-color: var(--bg-surface, #FFFFFF);
      border: 1px solid var(--border-subtle, #E2E8F0);
      border-radius: var(--radius-lg, 8px);
      overflow: hidden;
      display: flex;
      flex-direction: column;
    }

    .table-toolbar {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: var(--space-4, 16px) var(--space-5, 20px);
      border-bottom: 1px solid var(--border-subtle, #E2E8F0);
      gap: var(--space-4, 16px);
      flex-wrap: wrap;
    }

    .toolbar-left {
      display: flex;
      align-items: center;
      gap: var(--space-3, 12px);
    }

    .table-title {
      font-size: 15px;
      font-weight: 600;
      color: var(--text-primary, #0F172A);
      margin: 0;
    }

    .count-badge {
      font-size: 12px;
      font-weight: 500;
      color: var(--text-muted, #64748B);
      background-color: var(--bg-canvas, #F1F5F9);
      padding: 2px 8px;
      border-radius: var(--radius-full, 9999px);
    }

    .search-box {
      position: relative;
      width: 280px;
    }

    .search-icon {
      position: absolute;
      left: 10px;
      top: 50%;
      transform: translateY(-50%);
      width: 15px;
      height: 15px;
      color: var(--text-muted, #94A3B8);
    }

    .search-input {
      width: 100%;
      padding: 7px 10px 7px 32px;
      font-size: 13px;
      border: 1px solid var(--border-subtle, #CBD5E1);
      border-radius: var(--radius-md, 6px);
      outline: none;
      transition: border-color 150ms ease;

      &:focus {
        border-color: var(--tenant-primary, #2563EB);
      }
    }

    .table-wrapper {
      overflow-x: auto;
    }

    .data-table {
      width: 100%;
      border-collapse: collapse;
      text-align: left;
      font-size: 13px;

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

    .user-cell {
      display: flex;
      flex-direction: column;
    }

    .user-name {
      font-weight: 500;
      color: var(--text-primary, #0F172A);
    }

    .user-email {
      font-size: 11px;
      color: var(--text-muted, #94A3B8);
    }

    .course-text {
      color: var(--text-secondary, #475569);
    }

    .mono-tag {
      font-family: 'JetBrains Mono', monospace;
      font-size: 11px;
      background-color: #F1F5F9;
      color: #334155;
      padding: 2px 6px;
      border-radius: 4px;
    }

    .mono-val {
      font-family: 'JetBrains Mono', monospace;
      font-size: 12px;
      color: var(--text-secondary, #475569);

      &.ttl-urgent {
        color: #DC2626;
        font-weight: 600;
      }
    }

    .text-right {
      text-align: right;
    }

    .actions-group {
      display: flex;
      justify-content: flex-end;
      gap: 6px;
    }

    .btn-icon {
      background: transparent;
      border: 1px solid transparent;
      border-radius: 4px;
      padding: 6px;
      cursor: pointer;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      transition: all 150ms ease;

      .icon {
        width: 16px;
        height: 16px;
      }

      &.danger {
        color: var(--text-muted, #94A3B8);

        &:hover {
          background-color: #FEF2F2;
          color: #DC2626;
          border-color: #FCA5A5;
        }
      }
    }

    .empty-cell {
      text-align: center;
      padding: 32px 16px;
      color: var(--text-muted, #94A3B8);
    }

    .pagination-footer {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 0.75rem 1.125rem;
      background-color: var(--bg-canvas, #F8FAFC);
      border-top: 1px solid var(--border-subtle, #E2E8F0);
      font-size: 0.75rem;
    }

    .pagination-info {
      color: var(--text-secondary, #64748B);
    }

    .pagination-controls {
      display: flex;
      align-items: center;
      gap: 0.5rem;
    }

    .btn-page {
      display: inline-flex;
      align-items: center;
      gap: 0.35rem;
      padding: 0.3rem 0.65rem;
      border-radius: var(--radius-sm, 4px);
      background-color: var(--color-white, #FFFFFF);
      border: 1px solid var(--border-subtle, #E2E8F0);
      font-size: 0.75rem;
      font-weight: 600;
      color: var(--text-primary, #0F172A);
      cursor: pointer;

      &:hover:not(:disabled) {
        background-color: var(--color-gray-100, #F1F5F9);
        border-color: var(--color-gray-300, #CBD5E1);
      }

      &:disabled {
        opacity: 0.5;
        cursor: not-allowed;
      }

      .page-icon {
        width: 13px;
        height: 13px;
      }
    }
  `]
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
