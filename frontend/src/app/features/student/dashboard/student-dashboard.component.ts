import { Component, OnInit, inject, computed, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';
import { AuthService } from '@core/services/auth.service';
import { StudentService, StudentSubjectItem, DueAssignment } from '@core/services/student.service';
import { StatusBadgeComponent } from '@shared/components/status-badge/status-badge.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { formatSolvDate } from '@shared/pipes/date-text.pipe';
import { 
  LucidePlay, 
  LucideRotateCw, 
  LucidePlus, 
  LucideClock, 
  LucideBookOpen 
} from '@lucide/angular';

@Component({
  selector: 'student-dashboard',
  standalone: true,
  imports: [
    CommonModule, 
    RouterModule, 
    StatusBadgeComponent,
    MachineDataDirective,
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
                      <span class="subject-code" machineData>{{ item.subject.code }}</span>
                      <span class="subject-name">{{ item.subject.name }}</span>
                    </div>

                    <div class="lab-status">
                      @if (item.active_workspace) {
                        <status-badge [status]="item.active_workspace.status" />
                        <span class="lab-details">
                          RAM: <span machineData>{{ item.active_workspace.memory_limit_mb }} MB</span> &bull; {{ item.active_workspace.type }}
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
                    <span class="agenda-date font-date">
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
  styleUrl: './student-dashboard.component.scss',
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
    if (!dateStr || dateStr.trim() === '') return 'Próximamente';
    return formatSolvDate(dateStr, 'datetime') ?? dateStr;
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
