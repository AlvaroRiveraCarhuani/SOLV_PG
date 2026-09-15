import { Component, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { AdminTeachersService } from '../services/admin-teachers.service';
import { TeacherInviteModalComponent } from './components/teacher-invite-modal/teacher-invite-modal.component';
import { TeacherItem, TeacherInvitationPayload } from '@core/models/admin.model';
import { 
  LucideUserPlus, 
  LucideSearch, 
  LucideLock, 
  LucideCheckCircle, 
  LucideClock, 
  LucideAlertCircle, 
  LucideSend, 
  LucideRotateCcw,
  LucideMoreVertical
} from '@lucide/angular';

@Component({
  selector: 'solv-admin-teachers',
  standalone: true,
  imports: [
    CommonModule, 
    FormsModule, 
    TeacherInviteModalComponent,
    LucideUserPlus, 
    LucideSearch, 
    LucideLock, 
    LucideCheckCircle, 
    LucideClock, 
    LucideAlertCircle, 
    LucideSend, 
    LucideRotateCcw,
    LucideMoreVertical
  ],
  template: `
    <div class="teachers-page">
      <!-- Header Superior -->
      <header class="page-header">
        <div class="header-left">
          <h1 class="page-title">Gestión y Alta de Docentes</h1>
          <p class="page-subtitle">Ciclo de vida de identidades y emisión de tokens transaccionales (ADR-022 / ADR-025).</p>
        </div>

        <div class="header-actions">
          <button class="btn-primary" (click)="showInviteModal.set(true)">
            <svg lucideUserPlus class="btn-icon"></svg>
            <span>+ Invitar Profesor</span>
          </button>
        </div>
      </header>

      <!-- Tabla y Barra de Filtros -->
      <div class="table-card">
        <div class="filter-toolbar">
          <div class="search-box">
            <svg lucideSearch class="search-icon"></svg>
            <input 
              type="text" 
              placeholder="Buscar por nombre o email..." 
              [ngModel]="searchTerm()" 
              (ngModelChange)="searchTerm.set($event)"
              class="search-input"
            />
          </div>

          <div class="filters-group">
            <div class="filter-item">
              <label class="filter-label">Estado:</label>
              <select 
                [ngModel]="statusFilter()" 
                (ngModelChange)="statusFilter.set($event)"
                class="filter-select">
                <option value="all">Todos los estados</option>
                <option value="active">Activo</option>
                <option value="pending">Pendiente (72h)</option>
                <option value="expired">Expirado</option>
              </select>
            </div>

            <div class="filter-item">
              <label class="filter-label">Origen:</label>
              <select 
                [ngModel]="originFilter()" 
                (ngModelChange)="originFilter.set($event)"
                class="filter-select">
                <option value="all">Todos los orígenes</option>
                <option value="manual">Manual</option>
                <option value="gclassroom">Google Classroom</option>
              </select>
            </div>
          </div>
        </div>

        <div class="table-wrapper">
          <table class="teachers-table">
            <thead>
              <tr>
                <th>NOMBRE Y CORREO</th>
                <th>ORIGEN</th>
                <th>ESTADO</th>
                <th>INVITADO</th>
                <th class="text-right">ACCIÓN</th>
              </tr>
            </thead>
            <tbody>
              @for (teacher of filteredTeachers(); track teacher.id) {
                <tr>
                  <td>
                    <div class="teacher-cell">
                      <div class="teacher-avatar">
                        {{ getInitials(teacher.full_name) }}
                      </div>
                      <div class="teacher-meta">
                        <span class="teacher-name">{{ teacher.full_name }}</span>
                        <span class="teacher-email">{{ teacher.email }}</span>
                      </div>
                    </div>
                  </td>
                  <td>
                    @if (teacher.origin === 'gclassroom') {
                      <span class="origin-badge gclassroom" title="Sincronizado vía Google Classroom API (ADR-022)">
                        <svg lucideLock class="origin-icon"></svg>
                        <span>GClassroom</span>
                      </span>
                    } @else {
                      <span class="origin-badge manual">Manual</span>
                    }
                  </td>
                  <td>
                    @switch (teacher.status) {
                      @case ('active') {
                        <span class="status-pill active">
                          <svg lucideCheckCircle class="pill-icon"></svg>
                          <span>Activo</span>
                        </span>
                      }
                      @case ('pending') {
                        <span class="status-pill pending" title="Enlace válido por 72 horas">
                          <svg lucideClock class="pill-icon"></svg>
                          <span>Pendiente</span>
                        </span>
                      }
                      @case ('expired') {
                        <span class="status-pill expired" title="Token vencido">
                          <svg lucideAlertCircle class="pill-icon"></svg>
                          <span>Expirado</span>
                        </span>
                      }
                    }
                  </td>
                  <td>
                    <span class="date-text">{{ teacher.invited_at }}</span>
                  </td>
                  <td class="text-right">
                    @if (teacher.status === 'pending') {
                      <button 
                        class="btn-row-action resend" 
                        (click)="resendInvitation(teacher.id)"
                        title="Reenviar enlace por correo">
                        <svg lucideSend class="btn-icon"></svg>
                        <span>Reenviar</span>
                      </button>
                    } @else if (teacher.status === 'expired') {
                      <button 
                        class="btn-row-action renew" 
                        (click)="renewInvitation(teacher.id)"
                        title="Emitir nuevo token transaccional (ADR-025)">
                        <svg lucideRotateCcw class="btn-icon"></svg>
                        <span>Renovar</span>
                      </button>
                    } @else {
                      <button class="btn-icon-more" title="Opciones del docente">
                        <svg lucideMoreVertical class="icon"></svg>
                      </button>
                    }
                  </td>
                </tr>
              } @empty {
                <tr>
                  <td colspan="5" class="empty-state">
                    <span>No se encontraron docentes registrados o que coincidan con la búsqueda.</span>
                  </td>
                </tr>
              }
            </tbody>
          </table>
        </div>
      </div>

      <!-- Modal de Invitación -->
      @if (showInviteModal()) {
        <solv-teacher-invite-modal 
          (close)="showInviteModal.set(false)"
          (submit)="onInviteSubmitted($event)"
        />
      }
    </div>
  `,
  styles: [`
    .teachers-page {
      display: flex;
      flex-direction: column;
      gap: var(--space-4, 16px);
      max-width: 1440px;
      margin: 0 auto;
    }

    .page-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      flex-wrap: wrap;
      gap: 16px;
    }

    .page-title {
      font-size: 20px;
      font-weight: 700;
      color: var(--text-primary, #0F172A);
      margin: 0;
    }

    .page-subtitle {
      font-size: 13px;
      color: var(--text-muted, #64748B);
      margin: 2px 0 0 0;
    }

    .btn-primary {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 8px 16px;
      background-color: var(--tenant-primary, #2563EB);
      border: 1px solid transparent;
      border-radius: var(--radius-md, 6px);
      font-size: 13px;
      font-weight: 600;
      color: #FFFFFF;
      cursor: pointer;
      transition: all 150ms ease;

      &:hover {
        opacity: 0.92;
      }

      .btn-icon {
        width: 15px;
        height: 15px;
      }
    }

    .table-card {
      background-color: var(--bg-surface, #FFFFFF);
      border: 1px solid var(--border-subtle, #E2E8F0);
      border-radius: var(--radius-lg, 8px);
      overflow: hidden;
      display: flex;
      flex-direction: column;
    }

    .filter-toolbar {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 14px 20px;
      border-bottom: 1px solid var(--border-subtle, #E2E8F0);
      gap: 16px;
      flex-wrap: wrap;
    }

    .search-box {
      position: relative;
      flex: 1;
      min-width: 260px;
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

      &:focus {
        border-color: var(--tenant-primary, #2563EB);
      }
    }

    .filters-group {
      display: flex;
      align-items: center;
      gap: 16px;
    }

    .filter-item {
      display: flex;
      align-items: center;
      gap: 8px;
    }

    .filter-label {
      font-size: 12px;
      font-weight: 600;
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

    .table-wrapper {
      overflow-x: auto;
    }

    .teachers-table {
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

    .teacher-cell {
      display: flex;
      align-items: center;
      gap: 12px;
    }

    .teacher-avatar {
      width: 32px;
      height: 32px;
      border-radius: 50%;
      background-color: var(--tenant-primary-subtle, rgba(37, 99, 235, 0.08));
      color: var(--tenant-primary, #2563EB);
      font-size: 11px;
      font-weight: 700;
      display: flex;
      align-items: center;
      justify-content: center;
      flex-shrink: 0;
    }

    .teacher-meta {
      display: flex;
      flex-direction: column;
    }

    .teacher-name {
      font-weight: 600;
      color: var(--text-primary, #0F172A);
    }

    .teacher-email {
      font-size: 11px;
      color: var(--text-muted, #64748B);
    }

    .origin-badge {
      display: inline-flex;
      align-items: center;
      gap: 4px;
      padding: 2px 8px;
      border-radius: 4px;
      font-size: 11px;
      font-weight: 500;

      &.manual {
        background-color: #F1F5F9;
        color: #475569;
      }

      &.gclassroom {
        background-color: #EFF6FF;
        color: #1D4ED8;
        border: 1px solid #BFDBFE;

        .origin-icon {
          width: 11px;
          height: 11px;
        }
      }
    }

    .status-pill {
      display: inline-flex;
      align-items: center;
      gap: 5px;
      padding: 3px 9px;
      border-radius: var(--radius-full, 9999px);
      font-size: 11px;
      font-weight: 600;

      .pill-icon {
        width: 12px;
        height: 12px;
      }

      &.active {
        background-color: #DCFCE7;
        color: #15803D;
      }

      &.pending {
        background-color: #FEF3C7;
        color: #B45309;
      }

      &.expired {
        background-color: #FEE2E2;
        color: #B91C1C;
      }
    }

    .date-text {
      font-size: 12px;
      color: var(--text-secondary, #64748B);
    }

    .text-right {
      text-align: right;
    }

    .btn-row-action {
      display: inline-flex;
      align-items: center;
      gap: 4px;
      padding: 4px 10px;
      border-radius: 4px;
      font-size: 11px;
      font-weight: 600;
      cursor: pointer;
      transition: all 150ms ease;

      .btn-icon {
        width: 12px;
        height: 12px;
      }

      &.resend {
        background-color: #F8FAFC;
        border: 1px solid #CBD5E1;
        color: #334155;

        &:hover {
          background-color: #E2E8F0;
        }
      }

      &.renew {
        background-color: #FEE2E2;
        border: 1px solid #FCA5A5;
        color: #991B1B;

        &:hover {
          background-color: #FCA5A5;
        }
      }
    }

    .btn-icon-more {
      background: transparent;
      border: none;
      cursor: pointer;
      padding: 6px;
      border-radius: 4px;
      color: var(--text-muted, #94A3B8);

      &:hover {
        background-color: #F1F5F9;
        color: #0F172A;
      }

      .icon {
        width: 15px;
        height: 15px;
      }
    }

    .empty-state {
      text-align: center;
      padding: 32px 16px;
      color: var(--text-muted, #94A3B8);
    }
  `]
})
export class AdminTeachersComponent {
  private teachersService = inject(AdminTeachersService);

  teachers = this.teachersService.teachers;
  isLoading = this.teachersService.isLoading;

  showInviteModal = signal<boolean>(false);
  searchTerm = signal<string>('');
  statusFilter = signal<string>('all');
  originFilter = signal<string>('all');

  filteredTeachers = computed(() => {
    const term = this.searchTerm().toLowerCase().trim();
    const status = this.statusFilter();
    const origin = this.originFilter();

    return this.teachers().filter(t => {
      if (status !== 'all' && t.status !== status) return false;
      if (origin !== 'all' && t.origin !== origin) return false;
      if (term) {
        const match = t.full_name.toLowerCase().includes(term) ||
                      t.email.toLowerCase().includes(term);
        if (!match) return false;
      }
      return true;
    });
  });

  getInitials(name: string): string {
    return name
      .split(' ')
      .slice(0, 2)
      .map(w => w[0])
      .join('')
      .toUpperCase();
  }

  resendInvitation(id: string): void {
    this.teachersService.resendInvitation(id);
  }

  renewInvitation(id: string): void {
    this.teachersService.renewInvitation(id);
  }

  onInviteSubmitted(payload: TeacherInvitationPayload): void {
    this.teachersService.inviteTeacher(payload);
    this.showInviteModal.set(false);
  }
}
