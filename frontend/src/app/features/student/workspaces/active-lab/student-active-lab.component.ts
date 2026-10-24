import { ChangeDetectionStrategy, Component, OnInit, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { DomSanitizer, SafeResourceUrl } from '@angular/platform-browser';
import { LucideArrowLeft, LucideBookOpen, LucideExternalLink, LucidePause } from '@lucide/angular';
import { StudentService } from '@core/services/student.service';
import { StatusBadgeComponent } from '@shared/components/status-badge/status-badge.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';

@Component({
  selector: 'student-active-lab',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    CommonModule,
    RouterLink,
    StatusBadgeComponent,
    MachineDataDirective,
    LucideArrowLeft,
    LucideBookOpen,
    LucideExternalLink,
    LucidePause
  ],
  templateUrl: './student-active-lab.component.html',
  styleUrl: './student-active-lab.component.scss'
})
export class StudentActiveLabComponent implements OnInit {
  readonly studentService = inject(StudentService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly sanitizer = inject(DomSanitizer);

  subjectId = signal('');
  isActionLoading = signal(false);

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

  async ngOnInit(): Promise<void> {
    const id = this.route.snapshot.paramMap.get('subjectId') ?? '';
    this.subjectId.set(id);
    if (!this.studentService.dashboardData()) {
      await this.studentService.loadDashboard();
    }
  }
}
