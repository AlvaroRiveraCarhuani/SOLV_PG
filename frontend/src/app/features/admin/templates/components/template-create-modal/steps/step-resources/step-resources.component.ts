import { Component, input, output, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { 
  LucideHelpCircle, 
  LucideAlertCircle, 
  LucideDatabase, 
  LucideCheck, 
  LucideInfo 
} from '@lucide/angular';
import { 
  TargetEnvironment, 
  AvailableSatelliteService, 
  ServiceRequirement,
  RuntimeCapabilities,
  HostCapacityInfo
} from '../../../../../services/admin-templates.service';

export interface RamPreset {
  mb: number;
  label: string;
  desc: string;
}

export const RAM_PRESETS_IDE: RamPreset[] = [
  { mb: 512, label: '512 MB', desc: 'Ligera (C/Go)' },
  { mb: 1024, label: '1 GB', desc: 'Estándar (Web/Python)' },
  { mb: 2048, label: '2 GB', desc: 'Intensiva (Java/ML)' },
  { mb: 4096, label: '4 GB', desc: 'Datos & IA' }
];

export const RAM_PRESETS_JUDGE: RamPreset[] = [
  { mb: 128, label: '128 MB', desc: 'Ultra-ligera (C/C++)' },
  { mb: 256, label: '256 MB', desc: 'Recomendada (Python/Go)' },
  { mb: 512, label: '512 MB', desc: 'Completa (Java/JVM)' }
];

@Component({
  selector: 'solv-step-resources',
  standalone: true,
  imports: [
    CommonModule, 
    FormsModule, 
    LucideHelpCircle, 
    LucideAlertCircle, 
    LucideDatabase, 
    LucideCheck, 
    LucideInfo
  ],
  template: `
    <div class="resources-step-container">
      <div class="step-header-with-help mb-3">
        <h4 class="step-section-title" id="step-title-resources" tabindex="-1" i18n="@@PU-11-HEADING">
          Recursos de hardware y servicios
        </h4>
        <button 
          type="button" 
          class="btn-step-help" 
          (click)="helpRequested.emit()" 
          title="Ayuda contextual del paso" 
          aria-label="Ayuda contextual del paso"
          i18n-aria-label="@@AY-02"
        >
          <svg lucideHelpCircle class="w-4 h-4"></svg>
        </button>
      </div>

      <!-- Asignación de RAM Base -->
      <div class="form-group">
        <label class="form-label font-semibold">
          Memoria RAM por Instancia (MB): <span class="text-danger">*</span>
        </label>
        <div class="ram-presets-row">
          @for (preset of activeRamPresets(); track preset.mb) {
            <button 
              type="button" 
              class="btn-ram-preset" 
              [class.active]="baseRamMB() === preset.mb" 
              (click)="setRam(preset.mb)"
            >
              <span class="preset-label">{{ preset.label }}</span>
              <span class="preset-desc">{{ preset.desc }}</span>
            </button>
          }
          <div class="custom-ram-wrapper">
            <input 
              type="number" 
              class="form-control input-ram-custom font-mono" 
              [value]="baseRamMB()" 
              (input)="onRamInput($event)"
              [min]="targetEnvironment() === 'JUEZ_EFIMERO' ? 64 : 256" 
              max="8192" 
              step="64" 
            />
            <span class="unit-tag">MB</span>
          </div>
        </div>

        @if (isRamTooLow()) {
          <div class="validation-message-alert alert-danger mt-2">
            <svg lucideAlertCircle class="w-4 h-4"></svg>
            <span>
              {{ targetEnvironment() === 'JUEZ_EFIMERO' ? 'El mínimo recomendado es 64 MB para el sandbox de juez.' : 'El mínimo recomendado es 256 MB para alojar el editor web y el runtime.' }}
            </span>
          </div>
        }
      </div>

      <!-- Perfil de Distribución de Recursos y Métrica Viva -->
      <div class="resource-profile-card">
        <div class="profile-header">
          <span class="profile-title">Distribución de Recursos Estimada</span>
          <span class="profile-badge">
            @if (resourceProfilePreview().isJudge) {
              <span>≈ {{ resourceProfilePreview().estimatedCapacity }} evaluaciones concurrentes estimadas en este host</span>
            } @else {
              <span>Capacidad host: ~{{ resourceProfilePreview().estimatedCapacity }} alumnos simultáneos</span>
            }
          </span>
        </div>
        <div class="memory-bar">
          <div 
            class="bar-segment bar-editor" 
            [style.flex-grow]="resourceProfilePreview().editorBase" 
            [title]="(resourceProfilePreview().isJudge ? 'Runtime Sandbox: ' : 'Editor Web: ') + resourceProfilePreview().editorBase + ' MB'"
          >
            <span>{{ resourceProfilePreview().isJudge ? 'Sandbox (32 MB)' : 'IDE (210 MB)' }}</span>
          </div>
          <div 
            class="bar-segment bar-usable" 
            [style.flex-grow]="resourceProfilePreview().usable" 
            title="Disponible para compilación / ejecución"
          >
            <span>Libre ({{ resourceProfilePreview().usable }} MB)</span>
          </div>
        </div>
      </div>

      <!-- Servicios Satélite Adicionales (Solo en IDE) -->
      @if (targetEnvironment() === 'IDE_PERSISTENTE') {
        <div class="form-group mt-4">
          <label class="form-label font-semibold">
            Servicios Satélite Opcionales (Bases de Datos Aisladas):
          </label>
          <div class="services-toggle-grid">
            @for (svc of availableServices(); track svc.engine) {
              <div 
                class="service-card" 
                [class.selected]="isServiceSelected(svc.engine)"
                (click)="toggleService(svc)"
              >
                <div class="service-header">
                  <div class="service-title-wrap">
                    <svg lucideDatabase class="w-4 h-4 text-primary"></svg>
                    <strong class="service-name">{{ svc.label }}</strong>
                  </div>
                  <div class="service-status-wrap">
                    @if (isServiceSelected(svc.engine)) {
                      <svg lucideCheck class="w-4 h-4 text-success mr-1"></svg>
                    }
                    <span class="badge-satellite badge-optional">
                      Opcional
                    </span>
                  </div>
                </div>
                <p class="service-desc">{{ svc.description }}</p>
                <div class="service-footer">
                  <span class="badge-tag">v{{ svc.version || '16' }}</span>
                  <span class="badge-ram">{{ svc.category }}</span>
                </div>
              </div>
            }
          </div>
        </div>
      } @else {
        <div class="judge-services-disabled-banner mt-3" i18n="@@PU-14">
          <svg lucideInfo class="w-4 h-4 text-muted mr-2"></svg>
          <span>Los entornos de juez virtual no utilizan servicios satélite desacoplados.</span>
        </div>
      }
    </div>
  `,
  styleUrls: ['./step-resources.component.scss']
})
export class SolvStepResourcesComponent {
  targetEnvironment = input<TargetEnvironment>('IDE_PERSISTENTE');
  baseRamMB = input<number>(1024);
  availableServices = input<AvailableSatelliteService[]>([]);
  selectedServices = input<ServiceRequirement[]>([]);
  runtimeCapabilities = input<RuntimeCapabilities | null>(null);

  baseRamMBChange = output<number>();
  selectedServicesChange = output<ServiceRequirement[]>();
  helpRequested = output<void>();
  advance = output<void>();

  activeRamPresets = computed<RamPreset[]>(() => {
    return this.targetEnvironment() === 'JUEZ_EFIMERO' ? RAM_PRESETS_JUDGE : RAM_PRESETS_IDE;
  });

  isRamTooLow = computed<boolean>(() => {
    const isJudge = this.targetEnvironment() === 'JUEZ_EFIMERO';
    const minRam = isJudge ? 64 : 256;
    return this.baseRamMB() < minRam;
  });

  resourceProfilePreview = computed(() => {
    const isJudge = this.targetEnvironment() === 'JUEZ_EFIMERO';
    const ram = this.baseRamMB() || (isJudge ? 256 : 512);
    const caps = this.runtimeCapabilities();
    const hostFreeRamMB = caps?.host_memory?.available_ram_mb ?? (caps?.host_memory?.total_ram_mb ? caps.host_memory.total_ram_mb * 0.20 : 1024);

    if (isJudge) {
      const runtimeBase = caps?.runtime_base_mb ?? 32;
      const usable = Math.max(0, ram - runtimeBase);
      const estimatedConcurrentEvaluations = Math.max(1, Math.floor(hostFreeRamMB / ram));
      return {
        ram,
        editorBase: runtimeBase,
        usable,
        estimatedCapacity: estimatedConcurrentEvaluations,
        isJudge: true
      };
    }

    const editorBase = caps?.editor_base_mb ?? 210;
    const usable = Math.max(0, ram - editorBase);
    const estimatedStudents = Math.max(1, Math.floor(hostFreeRamMB / ram));
    return {
      ram,
      editorBase,
      usable,
      estimatedCapacity: estimatedStudents,
      isJudge: false
    };
  });

  setRam(mb: number): void {
    this.baseRamMBChange.emit(mb);
  }

  onRamInput(event: Event): void {
    const val = parseInt((event.target as HTMLInputElement).value, 10);
    if (!isNaN(val)) {
      this.baseRamMBChange.emit(val);
    }
  }

  isServiceSelected(engine: string): boolean {
    return this.selectedServices().some(s => s.engine === engine);
  }

  toggleService(service: AvailableSatelliteService): void {
    const current = [...this.selectedServices()];
    const idx = current.findIndex(s => s.engine === service.engine);
    if (idx >= 0) {
      current.splice(idx, 1);
    } else {
      current.push({
        category: service.category,
        engine: service.engine,
        version: service.version
      });
    }
    this.selectedServicesChange.emit(current);
  }
}
