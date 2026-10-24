import { ChangeDetectionStrategy, Component, OnInit, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router, RouterLink } from '@angular/router';
import { StudentService, StudentSubjectItem } from '@core/services/student.service';
import { StatusBadgeComponent } from '@shared/components/status-badge/status-badge.component';
import { SearchBarComponent } from '@shared/components/search-bar/search-bar.component';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { parseDateValue, formatSolvDate } from '@shared/pipes/date-text.pipe';

@Component({
  selector: 'student-workspaces',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    CommonModule,
    RouterLink,
    StatusBadgeComponent,
    SearchBarComponent,
    SkeletonLoaderComponent,
    MachineDataDirective
  ],
  templateUrl: './student-workspaces.component.html',
  styleUrl: './student-workspaces.component.scss'
})
export class StudentWorkspacesComponent implements OnInit {
  readonly studentService = inject(StudentService);
  private readonly router = inject(Router);

  searchQuery = signal('');
  isActionLoading = signal(false);

  subjects = computed(() => this.studentService.dashboardData()?.subjects ?? []);
  isLoading = computed(() => this.studentService.loading());

  filteredSubjects = computed(() => {
    const query = this.searchQuery().trim().toLowerCase();
    if (!query) {
      return this.subjects();
    }
    return this.subjects().filter(item =>
      item.subject.name.toLowerCase().includes(query) ||
      item.subject.code.toLowerCase().includes(query)
    );
  });

  labActionLabel(item: StudentSubjectItem): string {
    return item.active_workspace ? 'Reanudar laboratorio' : 'Iniciar laboratorio';
  }

  async openOrStartLab(subjectId: string): Promise<void> {
    this.isActionLoading.set(true);
    try {
      await this.studentService.startWorkspace(subjectId);
      await this.router.navigate(['/student/workspaces', subjectId]);
    } finally {
      this.isActionLoading.set(false);
    }
  }

  async pauseLab(item: StudentSubjectItem): Promise<void> {
    const workspaceId = item.active_workspace?.id;
    if (!workspaceId) {
      return;
    }
    this.isActionLoading.set(true);
    try {
      await this.studentService.pauseWorkspace(workspaceId);
    } finally {
      this.isActionLoading.set(false);
    }
  }

  relativeTime(value: string | null | undefined): string {
    if (!value) {
      return '';
    }
    const date = parseDateValue(value);
    if (!date) {
      return String(value);
    }
    const diffMinutes = Math.floor((Date.now() - date.getTime()) / 60000);
    if (diffMinutes < 1) {
      return 'ahora';
    }
    if (diffMinutes < 60) {
      return `hace ${diffMinutes} min`;
    }
    const diffHours = Math.floor(diffMinutes / 60);
    if (diffHours < 24) {
      return `hace ${diffHours} h`;
    }
    const diffDays = Math.floor(diffHours / 24);
    if (diffDays < 30) {
      return `hace ${diffDays} d`;
    }
    return formatSolvDate(value, 'daymonthyear') ?? '';
  }

  async ngOnInit(): Promise<void> {
    if (!this.studentService.dashboardData()) {
      await this.studentService.loadDashboard();
    }
  }
}
