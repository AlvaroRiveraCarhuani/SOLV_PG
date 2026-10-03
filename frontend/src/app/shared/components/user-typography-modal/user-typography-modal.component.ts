import { Component, EventEmitter, Output, inject, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { UserTypographyService } from '@core/services/user-typography.service';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { LucideType, LucideRotateCcw, LucideCheck } from '@lucide/angular';

@Component({
  selector: 'user-typography-modal',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ModalShellComponent,
    FormFieldComponent,
    ComboboxComponent,
    MachineDataDirective,
    LucideType,
    LucideRotateCcw,
    LucideCheck
  ],
  templateUrl: './user-typography-modal.component.html',
  styleUrls: ['./user-typography-modal.component.scss']
})
export class UserTypographyModalComponent {
  private userTypoService = inject(UserTypographyService);

  @Output() close = new EventEmitter<void>();

  sansOptions = computed<ComboboxOption[]>(() => {
    return [
      { id: '', label: 'Predeterminada Institucional', value: '' },
      ...this.userTypoService.sansOptions.map(f => ({
        id: f.slug,
        label: f.name,
        value: f.slug
      }))
    ];
  });

  monoOptions = computed<ComboboxOption[]>(() => {
    return [
      { id: '', label: 'Predeterminada Institucional', value: '' },
      ...this.userTypoService.monoOptions.map(f => ({
        id: f.slug,
        label: f.name,
        value: f.slug
      }))
    ];
  });

  selectedSansSlug = computed(() => this.userTypoService.sansPreference() || '');
  selectedMonoSlug = computed(() => this.userTypoService.monoPreference() || '');

  selectedSansLabel = computed(() => {
    const slug = this.selectedSansSlug();
    if (!slug) return 'Predeterminada Institucional';
    const match = this.userTypoService.sansOptions.find(f => f.slug === slug);
    return match ? match.name : 'Predeterminada Institucional';
  });

  selectedMonoLabel = computed(() => {
    const slug = this.selectedMonoSlug();
    if (!slug) return 'Predeterminada Institucional';
    const match = this.userTypoService.monoOptions.find(f => f.slug === slug);
    return match ? match.name : 'Predeterminada Institucional';
  });

  onSansSelected(opt: ComboboxOption): void {
    this.userTypoService.setSansPreference(opt.value || null);
  }

  onMonoSelected(opt: ComboboxOption): void {
    this.userTypoService.setMonoPreference(opt.value || null);
  }

  resetToDefault(): void {
    this.userTypoService.resetToInstitutional();
  }
}
