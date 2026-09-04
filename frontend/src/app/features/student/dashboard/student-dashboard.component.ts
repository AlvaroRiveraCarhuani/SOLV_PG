import { Component, OnInit, inject, computed, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';
import { AuthService } from '@core/services/auth.service';
import { StudentService, StudentSubjectItem, DueAssignment } from '@core/services/student.service';
import { StatusBadgeComponent } from '@shared/components/status-badge/status-badge.component';
import { 
  LucidePlay, 
  LucideRotateCw, 
  LucidePlus, 
  LucideClock, 
  LucideBookOpen 
} from '@lucide/angular';

@Component({
  selector: 'solv-student-dashboard',
  standalone: true,
  imports: [
    CommonModule, 
    RouterModule, 
    StatusBadgeComponent,
    LucidePlay, 
    LucideRotateCw, 
    LucidePlus, 
    LucideClock, 
    LucideBookOpen
  ],
  template: `
    <div class="dashboard-container">
      <!-- 1. HeaderGreeting: Saludo dinámico con prioridad (DASHBOARD.md) -->
      <section class="greeting-banner">
        <div class="greeting-content">
          <h1 class="greeting-title">
            {{ timeGreeting() }}, {{ studentFirstName() }}.
          </h1>
          <p class="greeting-subtitle">
            @if (dueCount() > 0) {
              Tienes <span class="highlight">{{ dueCount() }} entregas pendientes</span> para esta semana.
            } @else {
              Estás al día con todos tus laboratorios y evaluaciones prácticas.
            }
          </p>
        </div>
      </section>

      <!-- 2. Main Content Grid (66% Columna Central / 33% Columna Derecha) -->
      <div class="dashboard-grid">
        <!-- Columna Central (66% - Continuidad Técnica) -->
        <div class="col-central">
          <div class="card widget-labs">
            <div class="card-header">
              <div class="header-title">
                <svg lucideBookOpen class="header-icon"></svg>
                <h2>Mis Laboratorios Activos</h2>
              </div>
              <span class="badge-counter">{{ subjects().length }} Materias</span>
            </div>

            <div class="labs-list">
              @for (item of subjects(); track item.subject.id) {
                <div class="lab-item">
                  <div class="lab-main">
                    <div class="lab-meta">
                      <span class="subject-code">{{ item.subject.code }}</span>
                      <span class="subject-name">{{ item.subject.name }}</span>
                    </div>

                    <div class="lab-status">
                      @if (item.active_workspace) {
                        <solv-status-badge [status]="item.active_workspace.status" />
                        <span class="lab-details">
                          RAM: {{ item.active_workspace.memory_limit_mb }} MB &bull; {{ item.active_workspace.type }}
                        </span>
                      } @else {
                        <span class="no-workspace-text">Sin entorno instanciado</span>
                      }
                    </div>
                  </div>

                  <!-- Botón con Dualidad de Estado Absorbida (DASHBOARD.md / HU-FE-02) -->
                  <div class="lab-action">
                    @if (item.active_workspace) {
                      @switch (item.active_workspace.status) {
                        @case ('running') {
                          <button class="btn btn-running" (click)="openIDE(item)">
                            <svg lucidePlay class="btn-icon"></svg>
                            <span>Abrir IDE</span>
                          </button>
                        }
                        @case ('hibernated') {
                          <button class="btn btn-hibernated" (click)="resumeIDE(item)" [disabled]="isActionLoading()">
                            <svg lucideRotateCw class="btn-icon"></svg>
                            <span>Reanudar</span>
                          </button>
                        }
                        @default {
                          <button class="btn btn-neutral" (click)="resumeIDE(item)">
                            <svg lucideRotateCw class="btn-icon"></svg>
                            <span>Reiniciar</span>
                          </button>
                        }
                      }
                    } @else {
                      <button class="btn btn-primary" (click)="startNewWorkspace(item)" [disabled]="isActionLoading()">
                        <svg lucidePlus class="btn-icon"></svg>
                        <span>Iniciar Entorno</span>
                      </button>
                    }
                  </div>
                </div>
              } @empty {
                <div class="empty-state">
                  <p>No tienes materias matriculadas actualmente.</p>
                </div>
              }
            </div>
          </div>

          <!-- Widget: Mi Progreso Académico -->
          <div class="card widget-progress">
            <div class="card-header">
              <h3>Mi Progreso (Semestre Activo)</h3>
              <span class="badge-counter">85% Global</span>
            </div>
            <div class="progress-bars">
              <div class="progress-item">
                <div class="progress-label">
                  <span>Programación II (Algoritmia y Estructuras)</span>
                  <span class="progress-val">80%</span>
                </div>
                <div class="progress-track">
                  <div class="progress-fill" style="width: 80%;"></div>
                </div>
              </div>

              <div class="progress-item">
                <div class="progress-label">
                  <span>Sistemas Operativos (Procesos e Hilos)</span>
                  <span class="progress-val">65%</span>
                </div>
                <div class="progress-track">
                  <div class="progress-fill" style="width: 65%;"></div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Columna Derecha (33% - Urgencia Académica) -->
        <div class="col-derecha">
          <!-- Widget: Para Hoy / Entregas Próximas -->
          <div class="card widget-agenda">
            <div class="card-header">
              <div class="header-title">
                <svg lucideClock class="header-icon text-warning"></svg>
                <h3>Para Hoy &bull; Próximas Entregas</h3>
              </div>
            </div>

            <div class="agenda-list">
              @for (due of dueAssignments(); track due.exercise_id) {
                <div class="agenda-item">
                  <div class="agenda-urgency">
                    <span class="due-tag">Entrega</span>
                  </div>
                  <div class="agenda-body">
                    <span class="agenda-title">{{ due.title }}</span>
                    <span class="agenda-sub">{{ due.subject_name }} ({{ due.subject_code }})</span>
                    <span class="agenda-date">
                      <svg lucideClock class="date-icon"></svg>
                      {{ formatDate(due.due_date) }}
                    </span>
                  </div>
                </div>
              } @empty {
                <div class="empty-state">
                  <p>No hay entregas urgentes pendientes.</p>
                </div>
              }
            </div>
          </div>

          <!-- Widget: Accesos Recientes -->
          <div class="card widget-recent">
            <div class="card-header">
              <h3>Accesos Recientes</h3>
            </div>
            <ul class="recent-list">
              <li class="recent-item">
                <span class="recent-bullet"></span>
                <span class="recent-text">Lab #04: Árboles Binarios</span>
                <span class="recent-time">hace 2h</span>
              </li>
              <li class="recent-item">
                <span class="recent-bullet"></span>
                <span class="recent-text">Envío #12 &bull; Veredicto AC</span>
                <span class="recent-time">ayer</span>
              </li>
            </ul>
          </div>
        </div>
      </div>
    </div>
  `,
  styles: [`
    .dashboard-container {
      display: flex;
      flex-direction: column;
      gap: var(--space-6, 24px);
      max-width: 1300px;
      margin: 0 auto;
    }

    /* Greeting Banner */
    .greeting-banner {
      background: linear-gradient(135deg, var(--tenant-primary, #2563EB) 0%, var(--tenant-primary-hover, #1D4ED8) 100%);
      border-radius: var(--radius-xl, 12px);
      padding: var(--space-6, 24px) var(--space-8, 32px);
      color: #FFFFFF;
      box-shadow: 0 4px 12px -2px rgba(37, 99, 235, 0.2);
    }

    .greeting-title {
      font-size: var(--font-size-2xl, 24px);
      font-weight: 700;
      margin: 0 0 var(--space-1, 4px) 0;
      letter-spacing: -0.02em;
    }

    .greeting-subtitle {
      font-size: var(--font-size-sm, 14px);
      margin: 0;
      opacity: 0.92;

      .highlight {
        font-weight: 700;
        text-decoration: underline;
      }
    }

    /* Grid 66% / 33% */
    .dashboard-grid {
      display: grid;
      grid-template-columns: 2fr 1fr;
      gap: var(--space-6, 24px);

      @media (max-width: 1024px) {
        grid-template-columns: 1fr;
      }
    }

    .col-central, .col-derecha {
      display: flex;
      flex-direction: column;
      gap: var(--space-6, 24px);
    }

    /* Cards */
    .card {
      background-color: var(--bg-surface, #FFFFFF);
      border: 1px solid var(--border-subtle, #E2E8F0);
      border-radius: var(--radius-xl, 12px);
      padding: var(--space-5, 20px);
      box-shadow: 0 1px 3px 0 rgba(0, 0, 0, 0.04);
    }

    .card-header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      margin-bottom: var(--space-4, 16px);
      padding-bottom: var(--space-3, 12px);
      border-bottom: 1px solid var(--border-subtle, #E2E8F0);

      h2, h3 {
        margin: 0;
        font-size: var(--font-size-base, 16px);
        font-weight: 600;
        color: var(--text-primary, #0F172A);
      }

      .header-title {
        display: flex;
        align-items: center;
        gap: var(--space-2, 8px);
      }

      .header-icon {
        width: 20px;
        height: 20px;
        color: var(--tenant-primary, #2563EB);

        &.text-warning {
          color: var(--state-pending, #D97706);
        }
      }

      .badge-counter {
        font-size: 11px;
        font-weight: 600;
        padding: var(--space-1, 4px) var(--space-2-5, 10px);
        border-radius: var(--radius-full, 9999px);
        background-color: var(--bg-canvas, #F6F7F9);
        color: var(--text-secondary, #64748B);
      }
    }

    /* Labs List */
    .labs-list {
      display: flex;
      flex-direction: column;
      gap: var(--space-3, 12px);
    }

    .lab-item {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: var(--space-4, 16px);
      border-radius: var(--radius-lg, 8px);
      background-color: var(--bg-canvas, #F6F7F9);
      border: 1px solid var(--border-subtle, #E2E8F0);
      transition: all var(--transition-fast, 150ms ease);

      &:hover {
        border-color: var(--border-strong, #CBD5E1);
        background-color: #FFFFFF;
      }
    }

    .lab-main {
      display: flex;
      flex-direction: column;
      gap: var(--space-1-5, 6px);
    }

    .lab-meta {
      display: flex;
      align-items: center;
      gap: var(--space-2, 8px);
    }

    .subject-code {
      font-size: 11px;
      font-weight: 700;
      padding: 2px 6px;
      border-radius: var(--radius-sm, 4px);
      background-color: var(--tenant-primary-subtle, rgba(37, 99, 235, 0.1));
      color: var(--tenant-primary, #2563EB);
    }

    .subject-name {
      font-size: var(--font-size-sm, 14px);
      font-weight: 600;
      color: var(--text-primary, #0F172A);
    }

    .lab-status {
      display: flex;
      align-items: center;
      gap: var(--space-2-5, 10px);
    }

    .lab-details {
      font-size: 11px;
      color: var(--text-muted, #94A3B8);
    }

    .no-workspace-text {
      font-size: 12px;
      color: var(--text-muted, #94A3B8);
      font-style: italic;
    }

    /* Action Buttons */
    .btn {
      display: inline-flex;
      align-items: center;
      gap: var(--space-2, 8px);
      padding: var(--space-2, 8px) var(--space-4, 16px);
      border-radius: var(--radius-md, 6px);
      font-size: var(--font-size-xs, 12px);
      font-weight: 600;
      border: 1px solid transparent;
      cursor: pointer;
      transition: all var(--transition-fast, 150ms ease);

      .btn-icon {
        width: 14px;
        height: 14px;
      }

      &.btn-running {
        background-color: var(--state-running, #16A34A);
        color: #FFFFFF;

        &:hover {
          background-color: #15803D;
        }
      }

      &.btn-hibernated {
        background-color: var(--state-pending, #D97706);
        color: #FFFFFF;

        &:hover {
          background-color: #B45309;
        }
      }

      &.btn-primary {
        background-color: var(--tenant-primary, #2563EB);
        color: #FFFFFF;

        &:hover {
          background-color: var(--tenant-primary-hover, #1D4ED8);
        }
      }

      &.btn-neutral {
        background-color: var(--bg-surface, #FFFFFF);
        border-color: var(--border-subtle, #E2E8F0);
        color: var(--text-secondary, #64748B);

        &:hover {
          background-color: var(--bg-canvas, #F6F7F9);
          color: var(--text-primary, #0F172A);
        }
      }

      &:disabled {
        opacity: 0.6;
        cursor: not-allowed;
      }
    }

    /* Progress Widget */
    .progress-bars {
      display: flex;
      flex-direction: column;
      gap: var(--space-4, 16px);
    }

    .progress-item {
      display: flex;
      flex-direction: column;
      gap: var(--space-1-5, 6px);
    }

    .progress-label {
      display: flex;
      justify-content: space-between;
      font-size: var(--font-size-xs, 12px);
      font-weight: 500;
      color: var(--text-secondary, #64748B);

      .progress-val {
        font-weight: 700;
        color: var(--text-primary, #0F172A);
      }
    }

    .progress-track {
      height: 8px;
      background-color: var(--border-subtle, #E2E8F0);
      border-radius: var(--radius-full, 9999px);
      overflow: hidden;
    }

    .progress-fill {
      height: 100%;
      background-color: var(--tenant-primary, #2563EB);
      border-radius: var(--radius-full, 9999px);
      transition: width var(--transition-normal, 200ms ease);
    }

    /* Agenda Widget */
    .agenda-list {
      display: flex;
      flex-direction: column;
      gap: var(--space-3, 12px);
    }

    .agenda-item {
      display: flex;
      align-items: flex-start;
      gap: var(--space-3, 12px);
      padding: var(--space-3, 12px);
      border-radius: var(--radius-md, 6px);
      background-color: var(--bg-canvas, #F6F7F9);
      border: 1px solid var(--border-subtle, #E2E8F0);
    }

    .due-tag {
      font-size: 10px;
      font-weight: 700;
      text-transform: uppercase;
      padding: 2px 6px;
      border-radius: var(--radius-sm, 4px);
      background-color: var(--state-pending-bg, #FFFBEB);
      color: var(--state-pending-text, #B45309);
      border: 1px solid var(--state-pending-border, #FDE68A);
    }

    .agenda-body {
      display: flex;
      flex-direction: column;
      gap: 2px;
    }

    .agenda-title {
      font-size: var(--font-size-xs, 12px);
      font-weight: 600;
      color: var(--text-primary, #0F172A);
    }

    .agenda-sub {
      font-size: 11px;
      color: var(--text-muted, #94A3B8);
    }

    .agenda-date {
      display: flex;
      align-items: center;
      gap: 4px;
      font-size: 10px;
      font-weight: 500;
      color: var(--text-secondary, #64748B);
      margin-top: 4px;

      .date-icon {
        width: 12px;
        height: 12px;
      }
    }

    /* Recent Access */
    .recent-list {
      list-style: none;
      padding: 0;
      margin: 0;
      display: flex;
      flex-direction: column;
      gap: var(--space-2-5, 10px);
    }

    .recent-item {
      display: flex;
      align-items: center;
      gap: var(--space-2, 8px);
      font-size: var(--font-size-xs, 12px);
    }

    .recent-bullet {
      width: 6px;
      height: 6px;
      border-radius: 50%;
      background-color: var(--tenant-primary, #2563EB);
    }

    .recent-text {
      flex: 1;
      color: var(--text-secondary, #64748B);
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }

    .recent-time {
      font-size: 10px;
      color: var(--text-muted, #94A3B8);
    }

    .empty-state {
      padding: var(--space-4, 16px);
      text-align: center;
      color: var(--text-muted, #94A3B8);
      font-size: var(--font-size-xs, 12px);
    }
  `]
})
export class StudentDashboardComponent implements OnInit {
  authService = inject(AuthService);
  studentService = inject(StudentService);

  isActionLoading = signal<boolean>(false);

  studentFirstName = computed(() => {
    return this.authService.currentUser()?.first_name || 'Estudiante';
  });

  timeGreeting = computed(() => {
    const hour = new Date().getHours();
    if (hour < 12) return 'Buenos días';
    if (hour < 19) return 'Buenas tardes';
    return 'Buenas noches';
  });

  subjects = computed(() => {
    return this.studentService.dashboardData()?.subjects || [];
  });

  dueAssignments = computed(() => {
    return this.studentService.dueAssignments();
  });

  dueCount = computed(() => {
    return this.dueAssignments().length;
  });

  ngOnInit(): void {
    this.studentService.loadDashboard();
  }

  formatDate(dateStr: string): string {
    if (!dateStr) return 'Próximamente';
    const d = new Date(dateStr);
    return d.toLocaleDateString('es-BO', { weekday: 'short', day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' });
  }

  async openIDE(item: StudentSubjectItem): Promise<void> {
    if (item.active_workspace?.access_url) {
      window.open(item.active_workspace.access_url, '_blank');
    }
  }

  async resumeIDE(item: StudentSubjectItem): Promise<void> {
    this.isActionLoading.set(true);
    try {
      await this.studentService.startWorkspace(item.subject.id);
    } finally {
      this.isActionLoading.set(false);
    }
  }

  async startNewWorkspace(item: StudentSubjectItem): Promise<void> {
    this.isActionLoading.set(true);
    try {
      await this.studentService.startWorkspace(item.subject.id);
    } finally {
      this.isActionLoading.set(false);
    }
  }
}
