import { Component, input, output, HostListener } from '@angular/core';
import { CommonModule } from '@angular/common';
import { AdminStudentItem, AdminStudentCourseItem } from '../../../services/admin-students.service';
import { 
  LucideBookOpen, 
  LucideX, 
  LucideCpu, 
  LucideAlertTriangle, 
  LucideUserCheck, 
  LucideCalendar,
  LucideRotateCcw,
  LucidePause
} from '@lucide/angular';

@Component({
  selector: 'student-courses-modal',
  standalone: true,
  imports: [
    CommonModule, 
    LucideBookOpen, 
    LucideX, 
    LucideCpu, 
    LucideAlertTriangle, 
    LucideUserCheck, 
    LucideCalendar,
    LucideRotateCcw,
    LucidePause
  ],
  templateUrl: './student-courses-modal.component.html',
  styleUrls: ['./student-courses-modal.component.scss']
})
export class StudentCoursesModalComponent {
  student = input.required<AdminStudentItem>();
  courses = input.required<AdminStudentCourseItem[]>();
  isLoading = input<boolean>(false);
  actionInProgressId = input<string | null>(null);

  close = output<void>();
  restartWorkspace = output<string>();
  pauseWorkspace = output<string>();

  @HostListener('document:keydown.escape')
  handleEscape(): void {
    this.close.emit();
  }

  formatDate(dateStr?: string | null): string {
    if (!dateStr) return 'Sin fecha';
    try {
      const d = new Date(dateStr);
      return d.toLocaleDateString('es-ES', { day: '2-digit', month: 'short', year: 'numeric' });
    } catch {
      return dateStr;
    }
  }

  getWorkspaceBadgeClass(status?: string | null): string {
    if (!status) return 'status-none';
    switch (status.toLowerCase()) {
      case 'running':
        return 'status-running';
      case 'hibernated':
        return 'status-hibernated';
      case 'failed':
      case 'oom_killed':
        return 'status-failed';
      default:
        return 'status-pending';
    }
  }

  getWorkspaceLabel(status?: string | null): string {
    if (!status) return 'Sin contenedor';
    switch (status.toLowerCase()) {
      case 'running':
        return 'Activo (En ejecución)';
      case 'hibernated':
        return 'Hibernado';
      case 'failed':
      case 'oom_killed':
        return 'Detenido / OOM-Killed';
      default:
        return status;
    }
  }
}
