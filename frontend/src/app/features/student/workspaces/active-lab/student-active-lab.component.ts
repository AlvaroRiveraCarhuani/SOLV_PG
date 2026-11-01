import { ChangeDetectionStrategy, Component, OnDestroy, OnInit, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { DomSanitizer, SafeResourceUrl } from '@angular/platform-browser';
import { LucideArrowLeft, LucideBookOpen, LucideExternalLink, LucideFileCode, LucideLock, LucidePause } from '@lucide/angular';
import { StudentService } from '@core/services/student.service';
import { StatusBadgeComponent } from '@shared/components/status-badge/status-badge.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import {
  ClassAdvanceNoticeService,
  PendingClassAdvance
} from '../../services/class-advance-notice.service';
import { AdvanceConfirmModalComponent } from './advance-confirm-modal/advance-confirm-modal.component';

@Component({
  selector: 'student-active-lab',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    CommonModule,
    RouterLink,
    StatusBadgeComponent,
    MachineDataDirective,
    DateTextPipe,
    AdvanceConfirmModalComponent,
    LucideArrowLeft,
    LucideBookOpen,
    LucideExternalLink,
    LucideFileCode,
    LucideLock,
    LucidePause
  ],
  templateUrl: './student-active-lab.component.html',
  styleUrl: './student-active-lab.component.scss'
})
export class StudentActiveLabComponent implements OnInit, OnDestroy {
  readonly studentService = inject(StudentService);
  readonly advances = inject(ClassAdvanceNoticeService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly sanitizer = inject(DomSanitizer);

  subjectId = signal('');
  isActionLoading = signal(false);
  selectedAdvance = signal<PendingClassAdvance | null>(null);
  isDismissing = signal(false);

  subjectItem = computed(() => {
    const id = this.subjectId();
    if (!id) {
      return undefined;
    }
    return this.studentService.dashboardData()?.subjects.find(item => item.subject.id === id);
  });

  subjectName = computed(() => this.subjectItem()?.subject.name ?? 'Laboratorio');
  subjectCode = computed(() => this.subjectItem()?.subject.code ?? '');
  activeWorkspace = computed(() => this.subjectItem()?.active_workspace);

  // Un examen cerrado deja el volumen en solo lectura: la pantalla deja de
  // ofrecer el editor y explica por qué (ADR-005 §5).
  isFrozen = computed(() => {
    const workspace = this.activeWorkspace();
    return !!workspace && (workspace.status === 'frozen' || !!workspace.frozen_at);
  });

  frozenAt = computed(() => this.activeWorkspace()?.frozen_at ?? null);

  rawUrl = computed(() => {
    const workspace = this.activeWorkspace();
    if (!workspace) {
      return '';
    }
    return workspace.access_url || workspace.url || '';
  });

  safeLabUrl = computed<SafeResourceUrl | null>(() => {
    const workspace = this.activeWorkspace();
    if (!workspace || workspace.status !== 'running') {
      return null;
    }
    const raw = workspace.access_url || workspace.url;
    return raw ? this.sanitizer.bypassSecurityTrustResourceUrl(raw) : null;
  });

  async startLab(): Promise<void> {
    const id = this.subjectId();
    if (!id) {
      return;
    }
    this.isActionLoading.set(true);
    try {
      await this.studentService.startWorkspace(id);
    } finally {
      this.isActionLoading.set(false);
    }
  }

  async pauseAndExit(): Promise<void> {
    const workspace = this.activeWorkspace();
    if (!workspace?.id) {
      await this.router.navigate(['/student/workspaces']);
      return;
    }
    this.isActionLoading.set(true);
    try {
      await this.studentService.pauseWorkspace(workspace.id);
      await this.router.navigate(['/student/workspaces']);
    } finally {
      this.isActionLoading.set(false);
    }
  }

  openAdvance(advance: PendingClassAdvance): void {
    this.selectedAdvance.set(advance);
  }

  closeAdvanceModal(): void {
    this.selectedAdvance.set(null);
  }

  async dismissAdvance(advance: PendingClassAdvance): Promise<void> {
    if (this.isDismissing()) {
      return;
    }
    this.isDismissing.set(true);
    try {
      await this.advances.dismiss(advance.id);
    } catch {
      // Si el descarte falla, el aviso sigue visible y se puede reintentar.
    } finally {
      this.isDismissing.set(false);
    }
  }

  async ngOnInit(): Promise<void> {
    const id = this.route.snapshot.paramMap.get('subjectId') ?? '';
    this.subjectId.set(id);
    this.advances.start();
    if (!this.studentService.dashboardData()) {
      await this.studentService.loadDashboard();
    }
  }

  ngOnDestroy(): void {
    this.advances.stop();
  }
}
