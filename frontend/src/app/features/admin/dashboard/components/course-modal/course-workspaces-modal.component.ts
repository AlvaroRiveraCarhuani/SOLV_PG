import { Component, input, output, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { CourseLoadSummary, DockerContainerSummary } from '@core/models/admin.model';
import { StatusBadgeComponent } from '@shared/components/status-badge/status-badge.component';
import { LucideX, LucideSearch, LucideRotateCcw, LucidePause } from '@lucide/angular';

@Component({
  selector: 'solv-course-workspaces-modal',
  standalone: true,
  imports: [CommonModule, FormsModule, StatusBadgeComponent, LucideX, LucideSearch, LucideRotateCcw, LucidePause],
  template: `
    <div class="modal-backdrop" (click)="close.emit()">
      <div class="modal-dialog" (click)="$event.stopPropagation()">
        <!-- Cabecera del Modal -->
        <div class="modal-header">
          <div class="header-info">
            <h2 class="modal-title">Contenedores: {{ course().course_name }}</h2>
            <span class="modal-subtitle">Docente: {{ course().teacher_name }} &bull; {{ course().active_students }} alumnos en sesión</span>
          </div>
          <button class="btn-close" (click)="close.emit()" title="Cerrar ventana">
            <svg lucideX class="icon"></svg>
          </button>
        </div>

        <!-- Barra de Búsqueda y Filtros -->
        <div class="modal-filter-bar">
          <div class="search-box">
            <svg lucideSearch class="search-icon"></svg>
            <input 
              type="text" 
              placeholder="Buscar estudiante o ID (ej: Carlos, WS-089)..." 
              [ngModel]="searchTerm()" 
              (ngModelChange)="searchTerm.set($event)"
              class="search-input"
            />
          </div>

          <div class="filter-group">
            <label class="filter-label">Estado:</label>
            <select 
              [ngModel]="statusFilter()" 
              (ngModelChange)="statusFilter.set($event)"
              class="filter-select">
              <option value="all">Todos los estados</option>
              <option value="running">En ejecución (Running)</option>
              <option value="failed">Con fallos (OOM Killed)</option>
              <option value="hibernated">En pausa (Hibernated)</option>
            </select>
          </div>
        </div>

        <!-- Tabla de Workspaces de la Materia -->
        <div class="modal-table-wrapper">
          <table class="modal-table">
            <thead>
              <tr>
                <th>ID WORKSPACE</th>
                <th>ESTUDIANTE</th>
                <th>MEMORIA CONSUMIDA</th>
                <th>ESTADO</th>
                <th class="text-right">ACCIÓN</th>
              </tr>
            </thead>
            <tbody>
              @for (ws of filteredWorkspaces(); track ws.id) {
                <tr>
                  <td>
                    <span class="mono-id">{{ ws.id }}</span>
                  </td>
                  <td>
                    <div class="student-cell">
                      <span class="student-name">{{ ws.student_name }}</span>
                      <span class="student-email">{{ ws.student_email }}</span>
                    </div>
                  </td>
                  <td>
                    <span class="mono-mem">{{ ws.memory_used_mb }} / {{ ws.memory_limit_mb }} MB</span>
                  </td>
                  <td>
                    <solv-status-badge [status]="ws.status" />
                  </td>
                  <td class="text-right">
                    <div class="actions-wrapper">
                      @if (ws.status === 'failed') {
                        <button 
                          class="btn-action restart" 
                          (click)="restartWorkspace.emit(ws.id)"
                          title="Reiniciar contenedor caído">
                          <svg lucideRotateCcw class="btn-icon"></svg>
                          <span>Reiniciar Contenedor</span>
                        </button>
                      } @else if (ws.status === 'running') {
                        <button 
                          class="btn-action pause" 
                          (click)="pauseWorkspace.emit(ws.id)"
                          title="Pausar sesión activa">
                          <svg lucidePause class="btn-icon"></svg>
                          <span>Pausar</span>
                        </button>
                      }
                    </div>
                  </td>
                </tr>
              } @empty {
                <tr>
                  <td colspan="5" class="empty-state">
                    <span>No se encontraron contenedores para los criterios seleccionados.</span>
                  </td>
                </tr>
              }
            </tbody>
          </table>
        </div>

        <!-- Pie del Modal -->
        <div class="modal-footer">
          <button class="btn-secondary" (click)="close.emit()">
            <span>Cerrar</span>
          </button>
        </div>
      </div>
    </div>
  `,
  styles: [`
    .modal-backdrop {
      position: fixed;
      top: 0;
      left: 0;
      width: 100vw;
      height: 100vh;
      background-color: rgba(15, 23, 42, 0.5);
      backdrop-filter: blur(2px);
      display: flex;
      align-items: center;
      justify-content: center;
      z-index: 1000;
      padding: var(--space-4, 16px);
    }

    .modal-dialog {
      background-color: var(--bg-surface, #FFFFFF);
      border: 1px solid var(--border-subtle, #E2E8F0);
      border-radius: var(--radius-lg, 8px);
      box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.1), 0 8px 10px -6px rgba(0, 0, 0, 0.1);
      width: 100%;
      max-width: 860px;
      max-height: 85vh;
      display: flex;
      flex-direction: column;
      overflow: hidden;
      animation: modalEnter 150ms ease-out;
    }

    @keyframes modalEnter {
      from {
        opacity: 0;
        transform: scale(0.98);
      }
      to {
        opacity: 1;
        transform: scale(1);
      }
    }

    .modal-header {
      display: flex;
      justify-content: space-between;
      align-items: flex-start;
      padding: var(--space-4, 16px) var(--space-5, 20px);
      border-bottom: 1px solid var(--border-subtle, #E2E8F0);
    }

    .header-info {
      display: flex;
      flex-direction: column;
      gap: 2px;
    }

    .modal-title {
      font-size: 16px;
      font-weight: 700;
      color: var(--text-primary, #0F172A);
      margin: 0;
    }

    .modal-subtitle {
      font-size: 12px;
      color: var(--text-muted, #64748B);
    }

    .btn-close {
      background: transparent;
      border: none;
      cursor: pointer;
      padding: 4px;
      border-radius: 4px;
      color: var(--text-muted, #94A3B8);

      &:hover {
        background-color: #F1F5F9;
        color: #0F172A;
      }

      .icon {
        width: 18px;
        height: 18px;
      }
    }

    .modal-filter-bar {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 12px 20px;
      background-color: #F8FAFC;
      border-bottom: 1px solid var(--border-subtle, #E2E8F0);
      gap: 16px;
      flex-wrap: wrap;
    }

    .search-box {
      position: relative;
      flex: 1;
      min-width: 240px;
    }

    .search-icon {
      position: absolute;
      left: 10px;
      top: 50%;
      transform: translateY(-50%);
      width: 14px;
      height: 14px;
      color: var(--text-muted, #94A3B8);
    }

    .search-input {
      width: 100%;
      padding: 6px 10px 6px 30px;
      font-size: 12px;
      border: 1px solid var(--border-subtle, #CBD5E1);
      border-radius: var(--radius-md, 6px);
      outline: none;

      &:focus {
        border-color: var(--tenant-primary, #2563EB);
      }
    }

    .filter-group {
      display: flex;
      align-items: center;
      gap: 8px;
    }

    .filter-label {
      font-size: 12px;
      font-weight: 500;
      color: var(--text-secondary, #475569);
    }

    .filter-select {
      font-size: 12px;
      padding: 6px 10px;
      border: 1px solid var(--border-subtle, #CBD5E1);
      border-radius: var(--radius-md, 6px);
      background-color: #FFFFFF;
      outline: none;
    }

    .modal-table-wrapper {
      flex: 1;
      overflow-y: auto;
    }

    .modal-table {
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
        padding: 10px 16px;
        border-bottom: 1px solid var(--border-subtle, #F1F5F9);
        vertical-align: middle;
      }

      tr:hover td {
        background-color: #F8FAFC;
      }
    }

    .mono-id {
      font-family: 'JetBrains Mono', monospace;
      font-size: 11px;
      color: #334155;
      background-color: #F1F5F9;
      padding: 2px 6px;
      border-radius: 4px;
    }

    .student-cell {
      display: flex;
      flex-direction: column;
    }

    .student-name {
      font-weight: 500;
      color: var(--text-primary, #0F172A);
    }

    .student-email {
      font-size: 11px;
      color: var(--text-muted, #94A3B8);
    }

    .mono-mem {
      font-family: 'JetBrains Mono', monospace;
      font-size: 12px;
      color: var(--text-secondary, #475569);
    }

    .text-right {
      text-align: right;
    }

    .actions-wrapper {
      display: flex;
      justify-content: flex-end;
    }

    .btn-action {
      display: inline-flex;
      align-items: center;
      gap: 5px;
      padding: 4px 10px;
      border-radius: 4px;
      font-size: 11px;
      font-weight: 500;
      cursor: pointer;
      transition: all 150ms ease;

      .btn-icon {
        width: 12px;
        height: 12px;
      }

      &.restart {
        background-color: #FEE2E2;
        border: 1px solid #FCA5A5;
        color: #991B1B;

        &:hover {
          background-color: #FCA5A5;
        }
      }

      &.pause {
        background-color: #F1F5F9;
        border: 1px solid #CBD5E1;
        color: #334155;

        &:hover {
          background-color: #E2E8F0;
        }
      }
    }

    .empty-state {
      text-align: center;
      padding: 32px 16px;
      color: var(--text-muted, #94A3B8);
    }

    .modal-footer {
      display: flex;
      justify-content: flex-end;
      padding: 12px 20px;
      border-top: 1px solid var(--border-subtle, #E2E8F0);
      background-color: #F8FAFC;
    }

    .btn-secondary {
      padding: 6px 14px;
      background-color: #FFFFFF;
      border: 1px solid var(--border-subtle, #CBD5E1);
      border-radius: var(--radius-md, 6px);
      font-size: 12px;
      font-weight: 500;
      color: var(--text-secondary, #475569);
      cursor: pointer;

      &:hover {
        background-color: #F1F5F9;
      }
    }
  `]
})
export class CourseWorkspacesModalComponent {
  course = input.required<CourseLoadSummary>();
  workspaces = input.required<DockerContainerSummary[]>();

  close = output<void>();
  restartWorkspace = output<string>();
  pauseWorkspace = output<string>();

  searchTerm = signal<string>('');
  statusFilter = signal<string>('all');

  filteredWorkspaces = computed(() => {
    const term = this.searchTerm().toLowerCase().trim();
    const filter = this.statusFilter();
    
    return this.workspaces().filter(ws => {
      // Filtrar por curso
      const matchCourse = ws.course_name.toLowerCase() === this.course().course_name.toLowerCase();
      if (!matchCourse) return false;

      // Filtrar por estado
      if (filter !== 'all' && ws.status !== filter) return false;

      // Filtrar por término
      if (term) {
        const matchTerm = ws.student_name.toLowerCase().includes(term) ||
                          ws.student_email.toLowerCase().includes(term) ||
                          ws.id.toLowerCase().includes(term);
        if (!matchTerm) return false;
      }
      return true;
    });
  });
}
