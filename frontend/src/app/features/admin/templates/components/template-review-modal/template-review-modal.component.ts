import { Component, EventEmitter, Input, Output, signal, computed, HostListener, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { AdminTemplateItem, AdminTemplatesService, RamPresetSuggestion } from '../../../services/admin-templates.service';
import { 
  LucideX, 
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
    LucideX, 
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

  @Input({ required: true }) template!: AdminTemplateItem;
  @Output() approved = new EventEmitter<{ id: string; base_ram_mb: number }>();
  @Output() closed = new EventEmitter<void>();

  selectedRam = signal<number>(512);
  ramPresets = signal<RamPresetSuggestion[]>([]);
  isSubmitting = signal<boolean>(false);

  @HostListener('document:keydown.escape')
  onEscape(): void {
    this.closeModal();
  }

  hasLatestTag = computed(() => {
    const img = this.template?.docker_image?.toLowerCase().trim() || '';
    return img.endsWith(':latest') || (!img.includes(':') && !img.includes('@'));
  });

  ngOnInit(): void {
    if (this.template?.base_ram_mb && this.template.base_ram_mb > 0) {
      this.selectedRam.set(this.template.base_ram_mb);
    }

    const isJudge = this.template?.target_environment === 'JUEZ_EFIMERO';
    this.templatesService.getRuntimeCapabilities().subscribe({
      next: (caps) => {
        if (caps) {
          const suggestions = isJudge ? caps.judge_presets : caps.ide_presets;
          if (suggestions && suggestions.length > 0) {
            this.ramPresets.set(suggestions);
            return;
          }
        }
        this.ramPresets.set(this.getDefaultPresets(isJudge));
      },
      error: () => {
        this.ramPresets.set(this.getDefaultPresets(isJudge));
      }
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
    if (this.isSubmitting()) return;
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
