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
  RuntimeCapabilities
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
          Memoria RAM por Instancia: <span class="text-danger">*</span>
        </label>
        <div class="ram-presets-row">
          @for (preset of activeRamPresets(); track preset.mb) {
            <button 
              type="button" 
              class="btn-ram-preset" 
              [class.active]="hasUserConfiguredResources() && baseRamMB() === preset.mb" 
              (click)="setRam(preset.mb)"
            >
              <span class="preset-label">{{ preset.label }}</span>
              <span class="preset-desc">{{ preset.desc }}</span>
            </button>
          }
          <div class="custom-ram-wrapper">
            <input 
              type="text" 
              inputmode="numeric"
              class="form-control input-ram-custom font-mono" 
              [value]="formattedRamValue()" 
              (input)="onRamInput($event)"
              placeholder="1024"
              min="1" 
            />
            <span class="unit-tag">MB</span>
          </div>
        </div>

        @if (isRamBelowFloor()) {
          <div class="validation-message-alert alert-danger mt-2">
            <svg lucideAlertCircle class="w-4 h-4"></svg>
            <span>
              <strong>Hecho:</strong> Asignación actual ({{ baseRamMB() }} MB) inferior al piso requerido ({{ minimumFloorMB() }} MB).
              <strong>Causa:</strong> {{ floorAlertCause() }}
              <strong>Acción:</strong> {{ floorAlertAction() }}
            </span>
          </div>
        }

        @if (isRamExceedingHost()) {
          <div class="validation-message-alert alert-danger mt-2">
            <svg lucideAlertCircle class="w-4 h-4"></svg>
            <span>
              Excede la capacidad del host en {{ ramExcessMB() | number }} MB (máximo permitido: {{ maxAllowedRamMB() | number }} MB).
            </span>
          </div>
        } @else if (isRamExceedingCurrentFree()) {
          <div class="validation-message-alert alert-warning mt-2">
            <svg lucideAlertCircle class="w-4 h-4"></svg>
            <span>
              Aviso: La asignación supera la RAM libre actual ({{ hostFreeRamMB() | number }} MB). La plantilla será admitida pero la concurrencia dependerá de la carga del servidor.
            </span>
          </div>
        }
      </div>

      <!-- Perfil de Distribución de Recursos y Métrica Viva -->
      <div class="resource-profile-card">
        <div class="profile-header">
          <div class="profile-header-left">
            <span class="profile-title">Distribución de Recursos Estimada</span>
            <span class="badge-floor" [class.badge-floor-alert]="isRamBelowFloor()">
              Piso: {{ minimumFloorMB() }} MB ({{ floorReason() }})
            </span>
          </div>
          <span class="profile-badge">
            {{ capacityPluralLabel() }}
          </span>
        </div>
        <div class="memory-bar" [class.bar-overflow]="isRamExceedingHost()">
          @if (isRamExceedingHost()) {
            <div class="bar-segment bar-segment-overflow">
              <span>Excede la capacidad del host en {{ ramExcessMB() | number }} MB</span>
            </div>
          } @else {
            <div 
              class="bar-segment bar-editor" 
              [style.flex-grow]="resourceProfilePreview().editorBase" 
              [title]="(resourceProfilePreview().isJudge ? 'Runtime Sandbox: ' : 'Editor Web: ') + resourceProfilePreview().editorBase + ' MB'"
            >
              <span>{{ resourceProfilePreview().isJudge ? 'Sandbox (' + resourceProfilePreview().editorBase + ' MB)' : 'IDE (' + resourceProfilePreview().editorBase + ' MB)' }}</span>
            </div>
            @for (seg of resourceProfilePreview().satelliteSegments; track seg.engine) {
              <div 
                class="bar-segment bar-satellite" 
                [style.flex-grow]="seg.baseMB" 
                [title]="seg.label + ': ' + seg.baseMB + ' MB'"
              >
                <span>{{ seg.label }} ({{ seg.baseMB }} MB)</span>
              </div>
            }
            <div 
              class="bar-segment bar-usable" 
              [style.flex-grow]="resourceProfilePreview().usable" 
              title="Disponible para compilación / ejecución"
            >
              <span>Libre ({{ resourceProfilePreview().usable | number }} MB)</span>
            </div>
          }
        </div>
      </div>

      <!-- Bases de datos adicionales (Solo en IDE) -->
      @if (targetEnvironment() === 'IDE_PERSISTENTE') {
        <div class="form-group mt-4">
          <label class="form-label font-semibold" i18n="@@PU-32">
            Bases de datos adicionales (opcional):
          </label>
          <div class="services-toggle-grid">
            @for (svc of availableServices(); track svc.engine) {
              <div 
                class="service-card" 
                [class.selected]="isServiceSelected(svc.engine)"
                [class.disabled]="svc.isAvailable === false"
                [attr.aria-disabled]="svc.isAvailable === false"
                [title]="svc.isAvailable === false ? ('No disponible en este host: ' + (svc.description || 'motor no habilitado en el servidor')) : 'Haga clic para activar o desactivar este servicio'"
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
                    @if (svc.isAvailable === false) {
                      <span class="badge-service badge-unavailable" [title]="svc.description || 'No disponible en este host'">
                        No disponible en este host
                      </span>
                    } @else {
                      <span class="badge-service badge-optional" title="Servicio opcional que se aprovisionará de forma aislada">
                        Opcional
                      </span>
                    }
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
          <span>Los entornos de juez virtual no utilizan bases de datos adicionales.</span>
        </div>
      }
    </div>
  `,
  styleUrls: ['./step-resources.component.scss']
})
export class SolvStepResourcesComponent {
  readonly RELATIONAL_ENGINES = ['postgres', 'mysql'];
  readonly SATELLITE_BASE_RAM: Record<string, number> = {
    postgres: 128,
    mysql: 128,
    mongodb: 128,
    redis: 64
  };

  targetEnvironment = input<TargetEnvironment>('IDE_PERSISTENTE');
  baseRamMB = input<number>(1024);
  hasUserConfiguredResources = input<boolean>(true);
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

  maxAllowedRamMB = computed<number>(() => {
    const caps = this.runtimeCapabilities();
    if (caps?.max_allowed_ram_mb && caps.max_allowed_ram_mb > 0) {
      return caps.max_allowed_ram_mb;
    }
    const total = caps?.host_memory?.total_ram_mb ?? 8192;
    const derived = Math.floor(total * 0.75);
    return derived < 512 ? 512 : derived;
  });

  isRamExceedingHost = computed<boolean>(() => {
    return this.baseRamMB() > this.maxAllowedRamMB();
  });

  ramExcessMB = computed<number>(() => {
    return Math.max(0, this.baseRamMB() - this.maxAllowedRamMB());
  });

  selectedServiceSegments = computed<{ engine: string; label: string; baseMB: number }[]>(() => {
    if (this.targetEnvironment() === 'JUEZ_EFIMERO') return [];
    return this.selectedServices().map(s => {
      const match = this.availableServices().find(a => a.engine === s.engine);
      const label = match?.label ?? (s.engine === 'postgres' ? 'PostgreSQL' : s.engine === 'mysql' ? 'MySQL' : s.engine === 'mongodb' ? 'MongoDB' : s.engine === 'redis' ? 'Redis' : s.engine);
      return {
        engine: s.engine,
        label,
        baseMB: match?.base_ram_mb ?? this.SATELLITE_BASE_RAM[s.engine] ?? 128
      };
    });
  });

  totalSatelliteBaseMB = computed<number>(() => {
    return this.selectedServiceSegments().reduce((acc, curr) => acc + curr.baseMB, 0);
  });

  minimumFloorMB = computed<number>(() => {
    const isJudge = this.targetEnvironment() === 'JUEZ_EFIMERO';
    if (isJudge) {
      return 64;
    }
    const editorBase = this.runtimeCapabilities()?.editor_base_mb ?? 210;
    return editorBase + this.totalSatelliteBaseMB();
  });

  isRamBelowFloor = computed<boolean>(() => {
    return this.baseRamMB() < this.minimumFloorMB();
  });

  isRamTooLow = computed<boolean>(() => {
    return this.isRamBelowFloor();
  });

  floorReason = computed<string>(() => {
    const isJudge = this.targetEnvironment() === 'JUEZ_EFIMERO';
    if (isJudge) {
      return 'Sandbox CLI';
    }
    const editorBase = this.runtimeCapabilities()?.editor_base_mb ?? 210;
    const segments = this.selectedServiceSegments();
    if (segments.length === 0) {
      return `IDE base ${editorBase} MB`;
    }
    const dbLabels = segments.map(s => `${s.label} ${s.baseMB} MB`).join(' + ');
    return `IDE base ${editorBase} MB + ${dbLabels}`;
  });

  floorAlertCause = computed<string>(() => {
    const isJudge = this.targetEnvironment() === 'JUEZ_EFIMERO';
    if (isJudge) {
      return 'El sandbox de juez virtual requiere un mínimo de 64 MB para la inicialización del proceso.';
    }
    const editorBase = this.runtimeCapabilities()?.editor_base_mb ?? 210;
    const segments = this.selectedServiceSegments();
    if (segments.length === 0) {
      return `El editor web base (OpenVSCode Server) requiere al menos ${editorBase} MB para iniciar.`;
    }
    const dbLabels = segments.map(s => `${s.label} (${s.baseMB} MB)`).join(', ');
    return `El editor web requiere ${editorBase} MB y las bases de datos adicionales seleccionadas requieren ${this.totalSatelliteBaseMB()} MB (${dbLabels}).`;
  });

  floorAlertAction = computed<string>(() => {
    const min = this.minimumFloorMB();
    const presets = this.activeRamPresets();
    const candidate = presets.find(p => p.mb >= min);
    if (candidate) {
      return `Seleccione el preset de ${candidate.label} o configure al menos ${min} MB en el campo personalizado.`;
    }
    return `Configure al menos ${min} MB en el campo personalizado o desactive bases de datos adicionales.`;
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
        satelliteSegments: [] as { engine: string; label: string; baseMB: number }[],
        usable,
        estimatedCapacity: estimatedConcurrentEvaluations,
        isJudge: true
      };
    }

    const editorBase = caps?.editor_base_mb ?? 210;
    const satelliteSegments = this.selectedServiceSegments();
    const satBaseTotal = this.totalSatelliteBaseMB();
    const usable = Math.max(0, ram - editorBase - satBaseTotal);
    const perInstanceHostMB = ram + satBaseTotal;
    const estimatedStudents = Math.max(1, Math.floor(hostFreeRamMB / perInstanceHostMB));

    return {
      ram,
      editorBase,
      satelliteSegments,
      usable,
      estimatedCapacity: estimatedStudents,
      isJudge: false
    };
  });

  capacityPluralLabel = computed<string>(() => {
    const p = this.resourceProfilePreview();
    const count = p.estimatedCapacity;
    if (p.isJudge) {
      return count === 1 
        ? '≈ 1 evaluación concurrente estimada en este host' 
        : `≈ ${count.toLocaleString()} evaluaciones concurrentes estimadas en este host`;
    }
    return count === 1 
      ? 'Capacidad host: ~1 alumno simultáneo' 
      : `Capacidad host: ~${count.toLocaleString()} alumnos simultáneos`;
  });

  hostFreeRamMB = computed<number>(() => {
    const caps = this.runtimeCapabilities();
    return caps?.host_memory?.available_ram_mb ?? 2048;
  });

  isRamExceedingCurrentFree = computed<boolean>(() => {
    if (this.isRamExceedingHost()) return false;
    return this.baseRamMB() > this.hostFreeRamMB();
  });

  formattedRamValue = computed<string>(() => {
    const ram = this.baseRamMB();
    return ram ? ram.toLocaleString('es-ES') : '';
  });

  setRam(mb: number): void {
    this.baseRamMBChange.emit(mb);
  }

  onRamInput(event: Event): void {
    const inputEl = event.target as HTMLInputElement;
    const digits = inputEl.value.replace(/\D/g, '');
    const val = parseInt(digits, 10);
    if (!isNaN(val) && val >= 1) {
      this.baseRamMBChange.emit(val);
    }
  }

  isServiceSelected(engine: string): boolean {
    return this.selectedServices().some(s => s.engine === engine);
  }

  toggleService(service: AvailableSatelliteService): void {
    if (service.isAvailable === false) {
      return;
    }

    const isRelational = this.RELATIONAL_ENGINES.includes(service.engine);
    let current = [...this.selectedServices()];
    const isAlreadySelected = current.some(s => s.engine === service.engine);

    if (isAlreadySelected) {
      current = current.filter(s => s.engine !== service.engine);
    } else {
      if (isRelational) {
        current = current.filter(s => !this.RELATIONAL_ENGINES.includes(s.engine));
      }
      current.push({
        category: service.category,
        engine: service.engine,
        version: service.version
      });
    }
    this.selectedServicesChange.emit(current);
  }
}
