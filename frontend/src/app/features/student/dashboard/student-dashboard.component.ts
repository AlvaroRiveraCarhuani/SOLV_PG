import { ChangeDetectionStrategy, Component, OnInit, computed, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { AuthService } from '@core/services/auth.service';
import { StudentService } from '@core/services/student.service';
import { VerdictBadgeComponent } from '@shared/components/verdict-badge/verdict-badge.component';
import { FieldMessageComponent } from '@shared/components/field-message/field-message.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { relativeDayLabel } from '@shared/utils/relative-day.util';
import {
  LucideBookOpen,
  LucideHistory,
  LucideInbox,
  LucideLaptop,
  LucideTerminal
} from '@lucide/angular';
import { StudentAulaService } from '../services/student-aula.service';
import { aulaStatusLabel, aulaStatusTone } from '../services/aula-status';

@Component({
  selector: 'student-dashboard',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    CommonModule,
    RouterLink,
    VerdictBadgeComponent,
    FieldMessageComponent,
    MachineDataDirective,
    DateTextPipe,
    LucideBookOpen,
    LucideHistory,
    LucideInbox,
    LucideLaptop,
    LucideTerminal
  ],
  templateUrl: './student-dashboard.component.html',
  styleUrl: './student-dashboard.component.scss'
})
export class StudentDashboardComponent implements OnInit {
  private readonly authService = inject(AuthService);
  private readonly studentService = inject(StudentService);
  private readonly aulasService = inject(StudentAulaService);

  readonly statusLabel = aulaStatusLabel;
  readonly statusTone = aulaStatusTone;

  readonly aulas = this.aulasService.aulas;
  readonly aulaLoadError = this.aulasService.error;
  readonly dueLoadFailed = this.studentService.dueLoadFailed;

  readonly studentFirstName = computed(() => {
    return this.authService.currentUser()?.first_name || 'Estudiante';
  });

  readonly timeGreeting = computed(() => {
    const hour = new Date().getHours();
    if (hour < 12) return 'Buenos días';
    if (hour < 19) return 'Buenas tardes';
    return 'Buenas noches';
  });

  readonly subjects = computed(() => {
    return this.studentService.dashboardData()?.subjects || [];
  });

  readonly topSubjects = computed(() => {
    return this.subjects().slice(0, 4);
  });

  readonly dueAssignments = computed(() => {
    return this.studentService.dueAssignments();
  });

  readonly dueCount = computed(() => this.dueAssignments().length);

  /** Máximo 4 actividades próximas */
  readonly dueSoon = computed(() => this.dueAssignments().slice(0, 4));

  readonly recentSubmissions = computed(() => {
    return this.studentService.dashboardData()?.recent_submissions || [];
  });

  readonly recent = computed(() => this.recentSubmissions().slice(0, 3));

  readonly primaryAula = computed(() => this.aulas()[0] ?? null);

  readonly otherAulaCount = computed(() => Math.max(this.aulas().length - 1, 0));

  readonly summary = computed(() => {
    if (this.dueLoadFailed()) {
      return 'No se pudieron consultar sus entregas pendientes.';
    }
    const clase = this.primaryAula();
    if (clase) {
      return `Tiene clase abierta de ${clase.subject_name}.`;
    }
    const pendientes = this.dueCount();
    if (pendientes === 1) {
      return 'Tiene 1 entrega próxima.';
    }
    if (pendientes > 1) {
      return `Tiene ${pendientes} entregas próximas.`;
    }
    return 'No tiene entregas pendientes.';
  });

  getUrgencyLabel(dueDateStr?: string | null): string {
    return relativeDayLabel(dueDateStr).label;
  }

  isExam(item: unknown): boolean {
    const it = item as { purpose?: string; type?: string; title?: string };
    return it?.purpose === 'exam' || it?.type === 'exam' || (it?.title?.toLowerCase().includes('examen') ?? false);
  }

  isIdeExercise(item: unknown): boolean {
    const it = item as { environment_type?: string; type?: string };
    return it?.environment_type === 'IDE_PERSISTENTE' || it?.type === 'project' || it?.type === 'ide';
  }

  ngOnInit(): void {
    void this.aulasService.load();
    this.studentService.loadDashboard();
  }
}
