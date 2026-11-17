import {
  ChangeDetectionStrategy,
  Component,
  OnDestroy,
  OnInit,
  computed,
  inject,
  signal,
  viewChild
} from '@angular/core';
import {
  LucideCpu,
  LucideKeyRound,
  LucidePauseCircle,
  LucideShieldCheck
} from '@lucide/angular';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { ContextHelpService } from '@shared/help/context-help.service';
import { ProductTourService } from '@shared/help/product-tour.service';
import { InteractiveTutorialService } from '@shared/help/interactive-tutorial.service';
import {
  AIAvailabilityStatus,
  AIUserOverrideDTO,
  TeacherAiCopilotService
} from '../services/teacher-ai-copilot.service';
import { AiByokPanelComponent } from './ai-byok-panel/ai-byok-panel.component';
import {
  buildTeacherAiTour,
  buildTeacherAiTutorial
} from './teacher-ai-help.definitions';

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
  imports: [
    AiByokPanelComponent,
    MachineDataDirective,
    LucideCpu,
    LucideKeyRound,
    LucidePauseCircle,
    LucideShieldCheck
  ],
  templateUrl: './teacher-ai-settings.component.html',
  styleUrl: './teacher-ai-settings.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class TeacherAiSettingsComponent implements OnInit, OnDestroy {
  private readonly copilotService = inject(TeacherAiCopilotService);
  private readonly help = inject(ContextHelpService);
  private readonly tour = inject(ProductTourService);
  private readonly tutorial = inject(InteractiveTutorialService);

  readonly byokPanel = viewChild(AiByokPanelComponent);

  readonly status = signal<AIAvailabilityStatus | null>(null);
  readonly settings = signal<AIUserOverrideDTO | null>(null);

  readonly institutionalEnabled = computed(() => this.settings()?.institutional_enabled === true);
  readonly institutionalProvider = computed(
    () => this.settings()?.institutional_provider_label ?? ''
  );
  readonly institutionalModel = computed(() => this.settings()?.institutional_model ?? '');
  readonly usingPersonalKey = computed(() => this.status()?.using_personal_key === true);

  /**
   * El motor institucional sigue existiendo aunque el docente use su clave
   * personal, por eso el distintivo deja de decir "HABILITADO": informa que
   * está en pausa para que nadie crea que se sigue consumiendo.
   */
  readonly institutionalBadgeLabel = computed(() => {
    if (!this.institutionalEnabled()) {
      return 'DESHABILITADO';
    }
    return this.usingPersonalKey() ? 'EN PAUSA' : 'HABILITADO';
  });

  ngOnInit(): void {
    this.refresh();
    this.help.registerScreen('teacher-ai-settings');
    this.help.registerGuides('teacher-ai-settings', {
      tour: {
        label: 'Recorrido por Mi Inteligencia Artificial',
        hint: 'Explicación de cada bloque y diferencia de cuotas',
        run: () => this.tour.start(buildTeacherAiTour())
      },
      tutorial: {
        label: 'Configurar mi clave personal',
        hint: 'Acompaña la activación de su clave y la prueba de conexión',
        run: () => {
          const panel = this.byokPanel();
          if (panel) {
            this.tutorial.start(buildTeacherAiTutorial(panel.tutorialContext()));
          }
        }
      }
    });
  }

  ngOnDestroy(): void {
    this.help.unregisterGuides('teacher-ai-settings');
    this.help.unregisterScreen('teacher-ai-settings');
    this.tutorial.exit();
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
