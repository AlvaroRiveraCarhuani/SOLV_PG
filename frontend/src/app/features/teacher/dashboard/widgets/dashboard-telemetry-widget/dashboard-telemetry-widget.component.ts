import { Component, ChangeDetectionStrategy, input, output, signal, computed, effect } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { 
  LucideSearch, 
  LucideLayoutGrid, 
  LucideList, 
  LucideRotateCcw, 
  LucideTerminal, 
  LucidePause, 
  LucidePlay, 
  LucideCpu, 
  LucideHardDrive, 
  LucideAlertCircle, 
  LucideAlertTriangle 
} from '@lucide/angular';
import { LiveWorkspaceSession } from '../../../models/teacher.models';
import { SparklineComponent } from '@shared/components/sparkline/sparkline.component';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';

@Component({
  selector: 'dashboard-telemetry-widget',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideSearch,
    LucideLayoutGrid,
    LucideList,
    LucideRotateCcw,
    LucideTerminal,
    LucidePause,
    LucidePlay,
    LucideCpu,
    LucideHardDrive,
    LucideAlertCircle,
    LucideAlertTriangle,
    SparklineComponent,
    DateTextPipe,
    MachineDataDirective
  ],
  templateUrl: './dashboard-telemetry-widget.component.html',
  styleUrl: './dashboard-telemetry-widget.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class DashboardTelemetryWidgetComponent {
  readonly sessions = input<LiveWorkspaceSession[]>([]);
  readonly isLoading = input<boolean>(false);
  readonly isPollingActive = input<boolean>(true);
  readonly pollingRateMs = input<number>(3000);

  readonly pollingRateChange = output<number>();
  readonly refresh = output<void>();
  readonly openTerminal = output<LiveWorkspaceSession>();

  readonly liveSearchTerm = signal<string>('');
  readonly liveViewMode = signal<'matrix' | 'table'>(
    (localStorage.getItem('solv_teacher_live_view_mode') as 'matrix' | 'table') || 'matrix'
  );

  constructor() {
    effect(() => {
      const mode = this.liveViewMode();
      try {
        localStorage.setItem('solv_teacher_live_view_mode', mode);
      } catch {}
    });
  }

  readonly filteredSessions = computed(() => {
    const term = this.liveSearchTerm().toLowerCase().trim();
    const list = this.sessions();
    if (!term) return list;
    return list.filter(s => 
      s.student_name.toLowerCase().includes(term) ||
      s.student_email.toLowerCase().includes(term) ||
      s.subject_name.toLowerCase().includes(term)
    );
  });

  getMemoryPercent(session: LiveWorkspaceSession): number {
    if (!session.memory_limit_mb || session.memory_limit_mb === 0) return 0;
    const used = session.memory_used_mb || 0;
    return Math.min(100, Math.round((used / session.memory_limit_mb) * 100));
  }

  getStudentInitials(name: string): string {
    if (!name) return 'ST';
    const parts = name.trim().split(/\s+/);
    if (parts.length === 1) return parts[0].substring(0, 2).toUpperCase();
    return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
  }

  setLiveViewMode(mode: 'matrix' | 'table'): void {
    this.liveViewMode.set(mode);
  }

  onSetPollingRate(rateMs: number): void {
    this.pollingRateChange.emit(rateMs);
  }
}
