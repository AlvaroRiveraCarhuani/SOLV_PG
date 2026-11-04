import { ChangeDetectionStrategy, Component, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import {
  LucideBookOpen,
  LucideCalendarClock,
  LucideInbox,
  LucideRefreshCw,
  LucideUsers
} from '@lucide/angular';

import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';
import { FieldMessageComponent } from '@shared/components/field-message/field-message.component';

import {
  ClassAdvanceNoticeService,
  PendingClassAdvance
} from '../services/class-advance-notice.service';
import { StudentAulaService } from '../services/student-aula.service';
import { aulaStatusLabel, aulaStatusTone } from '../services/aula-status';
import { AdvanceConfirmModalComponent } from '../exercises/solver/advance-confirm-modal/advance-confirm-modal.component';

/**
 * Aula del alumno: la clase que el docente tiene abierta, la práctica asociada y
 * los avances de código que le llegaron. Es el punto donde el aviso del docente
 * (ADR-058) deja de ser una notificación suelta y aparece junto al aula.
 *
 * No inventa un aula: si no hay sesión abierta, lo dice y ofrece la lista de
 * prácticas. Un aula sin práctica se muestra igual, sin botón de avance, porque
 * en ese estado el docente tampoco puede emitir.
 */
@Component({
  selector: 'student-aula',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    CommonModule,
    RouterLink,
    DateTextPipe,
    MachineDataDirective,
    SkeletonLoaderComponent,
    FieldMessageComponent,
    AdvanceConfirmModalComponent,
    LucideBookOpen,
    LucideCalendarClock,
    LucideInbox,
    LucideRefreshCw,
    LucideUsers
  ],
  templateUrl: './student-aula.component.html',
  styleUrl: './student-aula.component.scss'
})
export class StudentAulaComponent implements OnInit {
  private readonly aulasService = inject(StudentAulaService);
  private readonly notices = inject(ClassAdvanceNoticeService);

  readonly aulas = this.aulasService.aulas;
  readonly isLoading = this.aulasService.loading;
  readonly loadError = this.aulasService.error;

  readonly selectedAdvance = signal<PendingClassAdvance | null>(null);

  /** Avances de una sesión concreta: el servicio consulta todos los pendientes. */
  pendingFor(sessionId: string): PendingClassAdvance[] {
    return this.notices.pending().filter(advance => advance.session_id === sessionId);
  }

  readonly statusLabel = aulaStatusLabel;
  readonly statusTone = aulaStatusTone;

  async ngOnInit(): Promise<void> {
    await this.aulasService.load();
    this.notices.start();
  }

  async refresh(): Promise<void> {
    await this.aulasService.load();
    await this.notices.refresh();
  }

  openAdvance(advance: PendingClassAdvance): void {
    this.selectedAdvance.set(advance);
  }

  closeAdvance(): void {
    this.selectedAdvance.set(null);
  }

  onApplied(): void {
    this.closeAdvance();
  }
}
