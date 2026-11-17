import {
  ChangeDetectionStrategy,
  Component,
  OnInit,
  computed,
  inject,
  input,
  output,
  signal,
  viewChild
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import {
  LucideAlertCircle,
  LucideCheck,
  LucideLoader2,
  LucideSearch,
  LucideTrash2,
  LucideUserPlus,
  LucideUsers,
  LucideX
} from '@lucide/angular';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { ConfirmModalComponent } from '@shared/components/confirm-modal/confirm-modal.component';
import { RosterUploadComponent } from '@shared/components/roster-upload/roster-upload.component';
import { SearchBarComponent } from '@shared/components/search-bar/search-bar.component';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { ParsedRosterResult } from '@core/models/admin.model';
import {
  CourseAvailableStudent,
  CourseRosterService,
  CourseStudent
} from '@core/services/course-roster.service';

type EnrollTab = 'individual' | 'roster';

/**
 * Padrón de una materia. Es el mismo componente para la administración y para
 * el docente titular: muestra quiénes están matriculados, de dónde vino cada
 * matrícula y permite altas individuales, carga de planilla y bajas.
 */
@Component({
  selector: 'course-students-panel',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ModalShellComponent,
    ConfirmModalComponent,
    RosterUploadComponent,
    SearchBarComponent,
    DateTextPipe,
    MachineDataDirective,
    LucideAlertCircle,
    LucideCheck,
    LucideLoader2,
    LucideSearch,
    LucideTrash2,
    LucideUserPlus,
    LucideUsers,
    LucideX
  ],
  templateUrl: './course-students-panel.component.html',
  styleUrl: './course-students-panel.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class CourseStudentsPanelComponent implements OnInit {
  private readonly rosterService = inject(CourseRosterService);

  /** Materia cuyo padrón se gestiona. */
  readonly subjectId = input.required<string>();
  /** Etiqueta del botón principal; la administración y el docente la comparten. */
  readonly enrollLabel = input<string>('Inscribir estudiante');

  /** Notifica el total de matriculados tras cada alta o baja. */
  readonly changed = output<number>();

  private readonly rosterUpload = viewChild(RosterUploadComponent);

  readonly students = signal<CourseStudent[]>([]);
  readonly isLoading = signal(true);
  readonly error = signal<string | null>(null);
  readonly query = signal('');

  readonly showEnrollModal = signal(false);
  readonly enrollTab = signal<EnrollTab>('individual');

  readonly availableQuery = signal('');
  readonly availableStudents = signal<CourseAvailableStudent[]>([]);
  readonly selectedStudent = signal<CourseAvailableStudent | null>(null);
  readonly isSearching = signal(false);
  readonly isEnrolling = signal(false);
  readonly hasSearched = signal(false);

  readonly rosterEmails = signal<string[]>([]);
  readonly isSubmittingRoster = signal(false);

  readonly feedbackMessage = signal<string | null>(null);
  readonly feedbackOk = signal<boolean | null>(null);

  readonly studentToRemove = signal<CourseStudent | null>(null);
  readonly removingStudentId = signal<string | null>(null);

  readonly filteredStudents = computed(() => {
    const term = this.query().trim().toLowerCase();
    if (!term) {
      return this.students();
    }
    return this.students().filter(
      (student) =>
        student.full_name.toLowerCase().includes(term) ||
        student.email.toLowerCase().includes(term)
    );
  });

  readonly officialCount = computed(
    () => this.students().filter((student) => student.source_system === 'institutional_roster').length
  );

  readonly courseCount = computed(() => this.students().length - this.officialCount());

  private searchTimeout: ReturnType<typeof setTimeout> | null = null;
  private initialized = false;

  ngOnInit(): void {
    this.load();
  }

  load(): void {
    this.isLoading.set(true);
    this.error.set(null);
    this.rosterService.listStudents(this.subjectId()).subscribe({
      next: (students) => {
        this.students.set(students);
        this.isLoading.set(false);
        // La primera lectura la conoce el contenedor por su propio resumen; el
        // output queda para confirmar altas y bajas, no para escribir señales
        // ajenas durante la detección inicial.
        if (this.initialized) {
          this.changed.emit(students.length);
        }
        this.initialized = true;
      },
      error: () => {
        this.isLoading.set(false);
        this.error.set('No se pudo cargar el padrón de la materia.');
      }
    });
  }

  onQueryChange(value: string): void {
    this.query.set(value);
  }

  initials(fullName: string): string {
    const parts = fullName.trim().split(/\s+/).filter(Boolean);
    if (parts.length === 0) {
      return '?';
    }
    const first = parts[0]?.charAt(0) ?? '';
    const last = parts.length > 1 ? parts[parts.length - 1].charAt(0) : '';
    return (first + last).toUpperCase();
  }

  sourceLabel(source: string): string {
    switch (source) {
      case 'institutional_roster':
        return 'Nómina oficial';
      case 'teacher_manual':
        return 'Alta de cátedra';
      case 'manual':
        return 'Manual';
      default:
        return source || 'Manual';
    }
  }

  sourceVariant(source: string): 'official' | 'course' {
    return source === 'institutional_roster' ? 'official' : 'course';
  }

  workspaceLabel(status: string): string {
    switch (status) {
      case 'running':
        return 'Entorno activo';
      case 'hibernated':
        return 'Hibernado';
      case 'pending':
        return 'Preparando';
      case 'failed':
      case 'oom_killed':
        return 'Con fallo';
      default:
        return 'Sin entorno';
    }
  }

  workspaceVariant(status: string): string {
    switch (status) {
      case 'running':
        return 'running';
      case 'pending':
        return 'pending';
      case 'failed':
      case 'oom_killed':
        return 'failed';
      default:
        return 'neutral';
    }
  }

  openEnrollModal(tab: EnrollTab = 'individual'): void {
    this.enrollTab.set(tab);
    this.feedbackMessage.set(null);
    this.feedbackOk.set(null);
    this.showEnrollModal.set(true);
  }

  closeEnrollModal(): void {
    this.showEnrollModal.set(false);
    this.availableQuery.set('');
    this.availableStudents.set([]);
    this.selectedStudent.set(null);
    this.hasSearched.set(false);
    this.rosterEmails.set([]);
    this.rosterUpload()?.reset();
  }

  setEnrollTab(tab: EnrollTab): void {
    this.enrollTab.set(tab);
    this.feedbackMessage.set(null);
    this.feedbackOk.set(null);
  }

  onAvailableSearchChange(value: string): void {
    this.availableQuery.set(value);
    this.selectedStudent.set(null);
    this.feedbackMessage.set(null);

    if (this.searchTimeout) {
      clearTimeout(this.searchTimeout);
    }
    const trimmed = value.trim();
    if (trimmed.length < 2) {
      this.availableStudents.set([]);
      this.hasSearched.set(false);
      this.isSearching.set(false);
      return;
    }
    this.isSearching.set(true);
    this.searchTimeout = setTimeout(() => this.performAvailableSearch(trimmed), 300);
  }

  private performAvailableSearch(query: string): void {
    this.rosterService.searchAvailableStudents(this.subjectId(), query).subscribe({
      next: (students) => {
        this.availableStudents.set(students);
        this.hasSearched.set(true);
        this.isSearching.set(false);
      },
      error: () => {
        this.availableStudents.set([]);
        this.hasSearched.set(true);
        this.isSearching.set(false);
      }
    });
  }

  selectStudent(student: CourseAvailableStudent): void {
    this.selectedStudent.set(student);
  }

  confirmEnroll(): void {
    const student = this.selectedStudent();
    if (!student || this.isEnrolling()) {
      return;
    }
    this.isEnrolling.set(true);
    this.feedbackMessage.set(null);

    this.rosterService.enrollStudent(this.subjectId(), student.id).subscribe({
      next: () => {
        this.isEnrolling.set(false);
        this.feedbackOk.set(true);
        this.feedbackMessage.set(`${student.first_name} ${student.last_name} quedó matriculado en la materia.`);
        this.selectedStudent.set(null);
        this.availableStudents.update((list) => list.filter((item) => item.id !== student.id));
        this.load();
      },
      error: (err: unknown) => {
        this.isEnrolling.set(false);
        this.feedbackOk.set(false);
        this.feedbackMessage.set(this.resolveError(err, 'No se pudo matricular al estudiante.'));
      }
    });
  }

  onRosterParsed(parsed: ParsedRosterResult): void {
    this.rosterEmails.set(parsed.valid_emails ?? []);
    this.feedbackMessage.set(null);
  }

  submitRoster(): void {
    const emails = this.rosterEmails();
    if (emails.length === 0 || this.isSubmittingRoster()) {
      return;
    }
    this.isSubmittingRoster.set(true);
    this.feedbackMessage.set(null);

    this.rosterService.enrollRoster(this.subjectId(), emails).subscribe({
      next: (result) => {
        this.isSubmittingRoster.set(false);
        this.feedbackOk.set(true);
        const enrolled = result?.enrolled_students ?? 0;
        const created = result?.created_students ?? 0;
        this.feedbackMessage.set(
          `Nómina aplicada: ${enrolled} matrículas nuevas y ${created} cuentas creadas.`
        );
        this.rosterEmails.set([]);
        this.rosterUpload()?.reset();
        this.load();
      },
      error: (err: unknown) => {
        this.isSubmittingRoster.set(false);
        this.feedbackOk.set(false);
        this.feedbackMessage.set(this.resolveError(err, 'No se pudo procesar la nómina.'));
      }
    });
  }

  requestRemove(student: CourseStudent): void {
    this.studentToRemove.set(student);
  }

  cancelRemove(): void {
    this.studentToRemove.set(null);
  }

  executeRemove(student: CourseStudent): void {
    this.removingStudentId.set(student.student_id);
    this.rosterService.removeStudent(this.subjectId(), student.student_id).subscribe({
      next: () => {
        this.removingStudentId.set(null);
        this.studentToRemove.set(null);
        this.feedbackOk.set(true);
        this.feedbackMessage.set(`${student.full_name} fue dado de baja de la materia.`);
        this.load();
      },
      error: (err: unknown) => {
        this.removingStudentId.set(null);
        this.studentToRemove.set(null);
        this.feedbackOk.set(false);
        this.feedbackMessage.set(this.resolveError(err, 'No se pudo dar de baja al estudiante.'));
      }
    });
  }

  private resolveError(err: unknown, fallback: string): string {
    const body = (err as { error?: { error?: string; message?: string } } | null)?.error;
    return body?.message || body?.error || fallback;
  }
}
