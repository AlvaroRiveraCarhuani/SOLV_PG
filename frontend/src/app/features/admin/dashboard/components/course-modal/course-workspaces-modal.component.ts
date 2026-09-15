import { Component, input, output, signal, inject, effect, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { CourseLoadSummary, DockerContainerSummary } from '@core/models/admin.model';
import { AdminMetricsService } from '../../../services/admin-metrics.service';
import { StatusBadgeComponent } from '@shared/components/status-badge/status-badge.component';
import { LucideX, LucideSearch, LucideRotateCcw, LucidePause, LucideTerminal } from '@lucide/angular';

@Component({
  selector: 'solv-course-workspaces-modal',
  standalone: true,
  imports: [
    CommonModule, 
    FormsModule, 
    StatusBadgeComponent, 
    LucideX, 
    LucideSearch, 
    LucideRotateCcw, 
    LucidePause,
    LucideTerminal
  ],
  templateUrl: './course-workspaces-modal.component.html',
  styleUrl: './course-workspaces-modal.component.scss'
})
export class CourseWorkspacesModalComponent implements OnInit {
  private metricsService = inject(AdminMetricsService);

  course = input.required<CourseLoadSummary>();
  workspacesInput = input<DockerContainerSummary[]>([]);

  close = output<void>();
  restartWorkspace = output<string>();
  pauseWorkspace = output<string>();

  searchTerm = signal<string>('');
  statusFilter = signal<string>('all');
  workspaces = signal<DockerContainerSummary[]>([]);
  isLoading = signal<boolean>(false);

  private searchDebounceTimer?: ReturnType<typeof setTimeout>;

  constructor() {
    effect(() => {
      const c = this.course();
      if (c && c.id) {
        this.fetchWorkspaces();
      }
    });
  }

  ngOnInit(): void {
    this.fetchWorkspaces();
  }

  fetchWorkspaces(): void {
    const c = this.course();
    if (!c || !c.id) return;

    this.isLoading.set(true);
    this.metricsService.getCourseWorkspaces(c.id, this.searchTerm(), this.statusFilter()).subscribe({
      next: (list) => {
        this.isLoading.set(false);
        this.workspaces.set(list || []);
      },
      error: () => {
        this.isLoading.set(false);
        // Fallback al filtrado en memoria si hubiera problemas de red
        const term = this.searchTerm().toLowerCase().trim();
        const filter = this.statusFilter();
        const fallback = (this.workspacesInput() || []).filter(ws => {
          const matchCourse = ws.course_name.toLowerCase() === c.course_name.toLowerCase();
          if (!matchCourse) return false;
          if (filter !== 'all' && ws.status !== filter) return false;
          if (term) {
            return ws.student_name.toLowerCase().includes(term) ||
                   ws.student_email.toLowerCase().includes(term) ||
                   ws.id.toLowerCase().includes(term);
          }
          return true;
        });
        this.workspaces.set(fallback);
      }
    });
  }

  onSearchChange(term: string): void {
    this.searchTerm.set(term);
    if (this.searchDebounceTimer) clearTimeout(this.searchDebounceTimer);
    this.searchDebounceTimer = setTimeout(() => {
      this.fetchWorkspaces();
    }, 250);
  }

  onStatusChange(status: string): void {
    this.statusFilter.set(status);
    this.fetchWorkspaces();
  }

  getMemoryPercent(ws: DockerContainerSummary): number {
    if (!ws.memory_limit_mb || ws.memory_limit_mb <= 0) return 0;
    return Math.min(100, Math.round((ws.memory_used_mb / ws.memory_limit_mb) * 100));
  }

  getStudentInitials(name: string): string {
    if (!name) return 'ES';
    const parts = name.trim().split(/\s+/);
    if (parts.length >= 2) {
      return (parts[0][0] + parts[1][0]).toUpperCase();
    }
    return name.slice(0, 2).toUpperCase();
  }
}
