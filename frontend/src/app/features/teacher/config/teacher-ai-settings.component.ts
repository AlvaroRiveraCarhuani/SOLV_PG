import { ChangeDetectionStrategy, Component, OnInit, computed, inject, signal } from '@angular/core';
import { LucideCpu, LucideKeyRound, LucideShieldCheck } from '@lucide/angular';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import {
  AIAvailabilityStatus,
  AIUserOverrideDTO,
  TeacherAiCopilotService
} from '../services/teacher-ai-copilot.service';
import { AiByokPanelComponent } from './ai-byok-panel/ai-byok-panel.component';

/**
 * Configuración de Inteligencia Artificial del docente.
 *
 * Expone, en un solo lugar, el motor institucional que la plataforma pone a
 * disposición del docente y la credencial personal opcional. La elección entre
 * ambos es explícita: no hay conmutación automática ante fallos.
 */
@Component({
  selector: 'teacher-ai-settings',
  standalone: true,
  imports: [AiByokPanelComponent, MachineDataDirective, LucideCpu, LucideKeyRound, LucideShieldCheck],
  templateUrl: './teacher-ai-settings.component.html',
  styleUrl: './teacher-ai-settings.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class TeacherAiSettingsComponent implements OnInit {
  private readonly copilotService = inject(TeacherAiCopilotService);

  readonly status = signal<AIAvailabilityStatus | null>(null);
  readonly settings = signal<AIUserOverrideDTO | null>(null);

  readonly institutionalEnabled = computed(() => this.settings()?.institutional_enabled === true);
  readonly institutionalProvider = computed(
    () => this.settings()?.institutional_provider_label ?? ''
  );
  readonly institutionalModel = computed(() => this.settings()?.institutional_model ?? '');
  readonly usingPersonalKey = computed(() => this.status()?.using_personal_key === true);

  ngOnInit(): void {
    this.refresh();
  }

  refresh(): void {
    this.copilotService.getStatus().subscribe({
      next: (status) => this.status.set(status),
      error: () => this.status.set({ enabled: false, provider: '', model: '' })
    });

    this.copilotService.getMySettings().subscribe({
      next: (settings) => this.settings.set(settings),
      error: () => this.settings.set(null)
    });
  }
}
