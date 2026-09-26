import { Component, EventEmitter, Input, Output, signal, inject, OnInit, HostListener } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { AdminTemplateItem, TemplateCategory, AdminTemplatesService } from '../../../services/admin-templates.service';
import { LucideX, LucideSparkles, LucideAlertCircle } from '@lucide/angular';

@Component({
  selector: 'template-promote-modal',
  standalone: true,
  imports: [CommonModule, FormsModule, LucideX, LucideSparkles, LucideAlertCircle],
  templateUrl: './template-promote-modal.component.html',
  styleUrls: ['./template-promote-modal.component.scss']
})
export class TemplatePromoteModalComponent implements OnInit {
  private templatesService = inject(AdminTemplatesService);

  @Input({ required: true }) template!: AdminTemplateItem;
  @Output() promoted = new EventEmitter<{ id: string; name: string; category_id?: string; description?: string }>();
  @Output() closed = new EventEmitter<void>();

  modelName = signal<string>('');
  categoryId = signal<string | null>(null);
  description = signal<string>('');
  categories = signal<TemplateCategory[]>([]);
  isPromoting = signal<boolean>(false);
  errorMsg = signal<string | null>(null);

  ngOnInit(): void {
    this.modelName.set(this.template.name);
    this.categoryId.set(this.template.category_id || null);
    this.description.set(this.template.description || '');

    this.templatesService.getCategories().subscribe({
      next: (cats) => this.categories.set(cats || []),
      error: () => {}
    });
  }

  @HostListener('document:keydown.escape')
  onEscape(): void {
    this.closeModal();
  }

  confirm(): void {
    const name = this.modelName().trim();
    if (!name || this.isPromoting()) return;

    this.isPromoting.set(true);
    this.errorMsg.set(null);

    this.templatesService.promoteToModel(this.template.id, {
      name,
      category_id: this.categoryId() || undefined,
      description: this.description().trim()
    }).subscribe({
      next: () => {
        this.isPromoting.set(false);
        this.promoted.emit({
          id: this.template.id,
          name,
          category_id: this.categoryId() || undefined,
          description: this.description().trim()
        });
      },
      error: (err) => {
        this.isPromoting.set(false);
        const msg = err.error?.message || err.error?.error || 'No se pudo promover la plantilla a modelo.';
        this.errorMsg.set(msg);
      }
    });
  }

  closeModal(): void {
    if (this.isPromoting()) return;
    this.closed.emit();
  }
}
