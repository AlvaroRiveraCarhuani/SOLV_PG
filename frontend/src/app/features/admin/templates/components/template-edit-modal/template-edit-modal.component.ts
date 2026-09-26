import { Component, EventEmitter, Input, Output, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { AdminTemplateItem } from '../../../services/admin-templates.service';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import {
  LucideHardDrive,
  LucideLayers,
  LucideSave
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
    LucideSave
  ],
  templateUrl: './template-edit-modal.component.html',
  styleUrls: ['./template-edit-modal.component.scss']
})
export class TemplateEditModalComponent implements OnInit {
  @Input({ required: true }) template!: AdminTemplateItem;
  @Output() saved = new EventEmitter<{ id: string; base_ram_mb: number; description?: string }>();
  @Output() closed = new EventEmitter<void>();

  selectedRam = signal<number>(512);
  description = signal<string>('');
  isSubmitting = signal<boolean>(false);

  ngOnInit(): void {
    if (this.template) {
      this.selectedRam.set(this.template.base_ram_mb || 512);
      this.description.set(this.template.description || '');
    }
  }

  setRam(mb: number): void {
    this.selectedRam.set(mb);
  }

  confirmSave(): void {
    if (this.isSubmitting()) return;
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
