import { Component, EventEmitter, Input, Output, signal, computed, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { AdminTemplateItem, AdminTemplatesService, RamPresetSuggestion, RuntimeCapabilities } from '../../../services/admin-templates.service';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import {
  LucideHardDrive,
  LucideCheckCircle2,
  LucideLayers,
  LucideInfo,
  LucideAlertTriangle
} from '@lucide/angular';

@Component({
  selector: 'template-review-modal',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    MachineDataDirective,
    ModalShellComponent,
    LucideHardDrive,
    LucideCheckCircle2,
    LucideLayers,
    LucideInfo,
    LucideAlertTriangle
  ],
  templateUrl: './template-review-modal.component.html',
  styleUrls: ['./template-review-modal.component.scss']
})
export class TemplateReviewModalComponent implements OnInit {
  private templatesService = inject(AdminTemplatesService);

  readonly templateItem = signal<AdminTemplateItem | null>(null);

  @Input({ required: true })
  set template(value: AdminTemplateItem) {
    this.templateItem.set(value);
  }
  get template(): AdminTemplateItem {
    return this.templateItem()!;
  }
  @Output() approved = new EventEmitter<{ id: string; base_ram_mb: number }>();
  @Output() closed = new EventEmitter<void>();

  selectedRam = signal<number>(512);
  ramPresets = signal<RamPresetSuggestion[]>([]);
  isSubmitting = signal<boolean>(false);
  runtimeCapabilities = signal<RuntimeCapabilities | null>(null);

  isDual = computed<boolean>(() => {
    return this.templateItem()?.target_environment === 'AMBOS';
  });

  isJudge = computed<boolean>(() => {
    const env = this.templateItem()?.target_environment;
    return env === 'JUEZ_EFIMERO' || env === 'JUEZ_VIRTUAL';
  });

  derivedJudgeRamMB = computed<number>(() => {
    return this.selectedRam() >= 1024 ? 256 : 128;
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

  hostFreeRamMB = computed<number>(() => {
    return this.runtimeCapabilities()?.host_memory?.available_ram_mb ?? 2048;
  });

  minimumFloorMB = computed<number>(() => {
    if (this.isJudge()) return 64;
    let floor = this.runtimeCapabilities()?.editor_base_mb ?? 210;
    const services = this.templateItem()?.services_config?.services || [];
    for (const svc of services) {
      const engine = (svc.engine || '').toLowerCase().trim();
      if (engine === 'postgres') {
        floor += 50;
      } else if (engine === 'mysql' || engine === 'mongodb') {
        floor += 128;
      } else if (engine === 'redis') {
        floor += 64;
      }
    }
    return floor;
  });

  isRamBelowFloor = computed<boolean>(() => {
    return this.selectedRam() < this.minimumFloorMB();
  });

  isRamExceedingHost = computed<boolean>(() => {
    return this.selectedRam() > this.maxAllowedRamMB();
  });

  ramExcessMB = computed<number>(() => {
    return Math.max(0, this.selectedRam() - this.maxAllowedRamMB());
  });

  isRamExceedingCurrentFree = computed<boolean>(() => {
    if (this.isRamExceedingHost()) return false;
    return this.selectedRam() > this.hostFreeRamMB();
  });

  hasLatestTag = computed(() => {
    const img = this.templateItem()?.docker_image?.toLowerCase().trim() || '';
    return img.endsWith(':latest') || (!img.includes(':') && !img.includes('@'));
  });

  canApprove = computed<boolean>(() => {
    if (this.isSubmitting()) return false;
    if (this.hasLatestTag()) return false;
    if (this.isRamBelowFloor()) return false;
    if (this.isRamExceedingHost()) return false;
    return this.selectedRam() > 0;
  });

  ngOnInit(): void {
    if (this.template?.base_ram_mb && this.template.base_ram_mb > 0) {
      this.selectedRam.set(this.template.base_ram_mb);
    }

    const isJudge = this.isJudge();
    this.ramPresets.set(this.getDefaultPresets(isJudge));
    this.templatesService.getRuntimeCapabilities().subscribe({
      next: (caps) => {
        if (caps) {
          this.runtimeCapabilities.set(caps);
          const suggestions = isJudge ? caps.judge_presets : caps.ide_presets;
          if (suggestions && suggestions.length > 0) {
            this.ramPresets.set(suggestions);
            return;
          }
        }
      },
      error: () => {}
    });
  }

  private getDefaultPresets(isJudge: boolean): RamPresetSuggestion[] {
    return isJudge
      ? [
          { mb: 128, label: '128 MB', desc: 'Ultra-ligera' },
          { mb: 256, label: '256 MB', desc: 'Recomendada' },
          { mb: 512, label: '512 MB', desc: 'Completa' }
        ]
      : [
          { mb: 512, label: '512 MB', desc: 'Ligera' },
          { mb: 1024, label: '1 GB', desc: 'Estándar' },
          { mb: 2048, label: '2 GB', desc: 'Intensiva' },
          { mb: 4096, label: '4 GB', desc: 'Datos & IA' }
        ];
  }

  setRam(mb: number): void {
    this.selectedRam.set(mb);
  }

  confirmApprove(): void {
    if (this.isSubmitting() || !this.canApprove()) return;
    this.isSubmitting.set(true);
    this.approved.emit({
      id: this.template.id,
      base_ram_mb: this.selectedRam()
    });
  }

  closeModal(): void {
    if (this.isSubmitting()) return;
    this.closed.emit();
  }
}
