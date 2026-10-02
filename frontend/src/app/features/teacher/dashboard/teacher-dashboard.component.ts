import { Component, OnInit, inject, signal, computed, effect } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { 
  LucideBookOpen, 
  LucideArrowRight, 
  LucideCheckCircle2, 
  LucideAlertTriangle, 
  LucideClock, 
  LucideCheckCircle, 
  LucideFlame, 
  LucideShieldAlert,
  LucideTerminal
} from '@lucide/angular';
import { TeacherDashboardService } from '../services/teacher-dashboard.service';
import { TeacherLiveService } from '../services/teacher-live.service';
import { LiveWorkspaceSession } from '../models/teacher.models';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { ViewSwitcherComponent, ViewMode } from '@shared/components/view-switcher/view-switcher.component';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';
import { ShadowTerminalModalComponent } from './shadow-terminal-modal/shadow-terminal-modal.component';

@Component({
  selector: 'teacher-dashboard',
  standalone: true,
  imports: [
    CommonModule, 
    RouterLink, 
    FormsModule,
    LucideBookOpen, 
    LucideArrowRight, 
    LucideCheckCircle2, 
    LucideAlertTriangle, 
    LucideClock, 
    LucideCheckCircle, 
    LucideFlame, 
    LucideShieldAlert,
    LucideTerminal,
    DateTextPipe, 
    MachineDataDirective,
    ViewSwitcherComponent,
    SkeletonLoaderComponent,
    ShadowTerminalModalComponent
  ],
  templateUrl: './teacher-dashboard.component.html',
  styleUrl: './teacher-dashboard.component.scss'
})
export class TeacherDashboardComponent implements OnInit {
  private dashboardService = inject(TeacherDashboardService);
  private liveService = inject(TeacherLiveService);

  courses = this.dashboardService.courses;
  attention = this.dashboardService.attention;
  periods = this.dashboardService.periods;
  isLoading = this.dashboardService.isLoading;
  liveSessions = this.liveService.liveSessions;

  viewMode = signal<ViewMode>((localStorage.getItem('solv_teacher_view_mode') as ViewMode) || 'cards');
  selectedPeriodId = signal<string>('');
  selectedLiveSession = signal<LiveWorkspaceSession | null>(null);

  constructor() {
    effect(() => {
      const mode = this.viewMode();
      try {
        localStorage.setItem('solv_teacher_view_mode', mode);
      } catch {}
    });
  }

  criticalCount = computed(() => this.attention()?.critical?.length ?? 0);
  warningCount = computed(() => this.attention()?.warning?.length ?? 0);
  standardCount = computed(() => this.attention()?.standard?.length ?? 0);

  greetingMessage = computed(() => {
    const critical = this.criticalCount();
    if (critical > 0) {
      return {
        type: 'critical',
        text: `Atención: ${critical} contenedor(es) excedieron el límite de memoria en tus cursos.`
      };
    }
    const standard = this.standardCount();
    if (standard > 0) {
      return {
        type: 'warning',
        text: `Tenés ${standard} entrega(s) pendiente(s) de revisión.`
      };
    }
    return {
      type: 'info',
      text: `Panel docente activo · ${this.courses().length} materia(s) asignadas en el semestre en curso.`
    };
  });

  ngOnInit(): void {
    this.dashboardService.getAcademicPeriods().subscribe({
      next: (periodsList) => {
        const active = periodsList.find(p => p.is_active) || periodsList[0];
        if (active) {
          this.selectedPeriodId.set(active.id);
        }
        this.loadDashboard();
      },
      error: () => this.loadDashboard()
    });
    this.loadLiveSessions();
  }

  onPeriodChange(periodId: string): void {
    this.selectedPeriodId.set(periodId);
    this.loadDashboard();
  }

  loadLiveSessions(): void {
    this.liveService.loadLiveSessions().subscribe();
  }

  openShadowTerminal(session: LiveWorkspaceSession): void {
    this.selectedLiveSession.set(session);
  }

  closeShadowTerminal(): void {
    this.selectedLiveSession.set(null);
  }

  private loadDashboard(): void {
    this.dashboardService.loadDashboardData(this.selectedPeriodId()).subscribe();
  }
}

