import { Component, EventEmitter, Input, Output, signal, computed, inject, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { AdminTemplateItem, AdminTemplatesService, RamPresetSuggestion, RuntimeCapabilities } from '../../../services/admin-templates.service';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import {
  LucideHardDrive,
  LucideLayers,
  LucideSave,
  LucideAlertCircle
} from '@lucide/angular';

@Component({
  selector: 'template-edit-modal',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ModalShellComponent,
    FormFieldComponent,
    LucideHardDrive,
    LucideLayers,
    LucideSave,
    LucideAlertCircle
  ],
  templateUrl: './template-edit-modal.component.html',
  styleUrls: ['./template-edit-modal.component.scss']
})
export class TemplateEditModalComponent implements OnInit {
  private templatesService = inject(AdminTemplatesService, { optional: true });

  @Input({ required: true }) template!: AdminTemplateItem;
  @Output() saved = new EventEmitter<{ id: string; base_ram_mb: number; description?: string }>();
  @Output() closed = new EventEmitter<void>();

  selectedRam = signal<number>(512);
  description = signal<string>('');
  isSubmitting = signal<boolean>(false);

  runtimeCapabilities = signal<RuntimeCapabilities | null>(null);
  ramPresets = signal<RamPresetSuggestion[]>([]);

  isJudge = computed<boolean>(() => {
    return (this.template?.target_environment || 'IDE_PERSISTENTE') === 'JUEZ_EFIMERO';
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
    return this.runtimeCapabilities()?.editor_base_mb ?? 256;
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

  canSave = computed<boolean>(() => {
    if (this.isSubmitting()) return false;
    if (this.isRamBelowFloor()) return false;
    if (this.isRamExceedingHost()) return false;
    return this.selectedRam() > 0;
  });

  invalidReason = computed<string>(() => {
    if (this.isRamExceedingHost()) {
      return `RAM supera la capacidad del host (${this.maxAllowedRamMB()} MB)`;
    }
    if (this.isRamBelowFloor()) {
      return `RAM inferior al mínimo permitido (${this.minimumFloorMB()} MB)`;
    }
    return '';
  });

  formattedRamValue = computed<string>(() => {
    const ram = this.selectedRam();
    return ram ? String(ram) : '';
  });

  ngOnInit(): void {
    if (this.template) {
      this.selectedRam.set(this.template.base_ram_mb || 512);
      this.description.set(this.template.description || '');
    }
    const judge = (this.template?.target_environment || 'IDE_PERSISTENTE') === 'JUEZ_EFIMERO';
    this.ramPresets.set(this.getDefaultPresets(judge));
    this.templatesService?.getRuntimeCapabilities().subscribe({
      next: (caps) => {
        if (caps) {
          this.runtimeCapabilities.set(caps);
          const suggestions = judge ? caps.judge_presets : caps.ide_presets;
          if (suggestions && suggestions.length > 0) {
            this.ramPresets.set(suggestions);
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

  onCustomRamInput(event: Event): void {
    const inputEl = event.target as HTMLInputElement;
    const digits = inputEl.value.replace(/\D/g, '');
    const val = parseInt(digits, 10);
    if (!isNaN(val) && val >= 1) {
      this.selectedRam.set(val);
    }
  }

  confirmSave(): void {
    if (!this.canSave()) return;
    this.isSubmitting.set(true);
    this.saved.emit({
      id: this.template.id,
      base_ram_mb: this.selectedRam(),
      description: this.description().trim()
    });
  }

  closeModal(): void {
    if (this.isSubmitting()) return;
    this.closed.emit();
  }
}
