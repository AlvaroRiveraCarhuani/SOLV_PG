import { ChangeDetectionStrategy, Component, OnInit, computed, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { AuthService } from '@core/services/auth.service';
import { StudentService } from '@core/services/student.service';
import { VerdictBadgeComponent } from '@shared/components/verdict-badge/verdict-badge.component';
import { FieldMessageComponent } from '@shared/components/field-message/field-message.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import {
  LucideArrowRight,
  LucideBookOpen,
  LucideHistory
} from '@lucide/angular';
import { StudentAulaService } from '../services/student-aula.service';
import { aulaStatusLabel, aulaStatusTone } from '../services/aula-status';

/**
 * Inicio del alumno.
 *
 * No es un tablero de widgets: son las tres cosas que el alumno necesita ver al
 * entrar —la clase que está abierta, lo que entrega pronto y el avance de sus
 * materias— y el acceso a su historial. El entorno ya no se muestra acá porque
 * es de una práctica, no de la materia: se abre desde la práctica o desde el
 * aula (ADR-059).
 */
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
    LucideArrowRight,
    LucideBookOpen,
    LucideHistory
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

  readonly dueAssignments = computed(() => {
    return this.studentService.dueAssignments();
  });

  readonly dueCount = computed(() => this.dueAssignments().length);

  /** La portada resume; la lista completa vive en su vista. */
  readonly dueSoon = computed(() => this.dueAssignments().slice(0, 4));

  readonly recentSubmissions = computed(() => {
    return this.studentService.dashboardData()?.recent_submissions || [];
  });

  readonly recent = computed(() => this.recentSubmissions().slice(0, 3));

  readonly primaryAula = computed(() => this.aulas()[0] ?? null);

  readonly otherAulaCount = computed(() => Math.max(this.aulas().length - 1, 0));

  readonly globalProgressPercent = computed(() => {
    const items = this.subjects();
    if (items.length === 0) return 0;
    const total = items.reduce((acc, item) => acc + (item.total_exercises ?? 0), 0);
    const completed = items.reduce((acc, item) => acc + (item.completed_exercises ?? 0), 0);
    if (total === 0) return 0;
    return Math.round((completed * 100) / total);
  });

  readonly summary = computed(() => {
    if (this.dueLoadFailed()) {
      return 'No pudimos revisar tus entregas pendientes.';
    }
    const clase = this.primaryAula();
    if (clase) {
      return `Tienes clase abierta de ${clase.subject_name}.`;
    }
    const pendientes = this.dueCount();
    if (pendientes === 1) {
      return 'Tienes 1 entrega próxima.';
    }
    if (pendientes > 1) {
      return `Tienes ${pendientes} entregas próximas.`;
    }
    return 'No tienes entregas pendientes.';
  });

  ngOnInit(): void {
    void this.aulasService.load();
    this.studentService.loadDashboard();
  }
}
