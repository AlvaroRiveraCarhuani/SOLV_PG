import { Component, EventEmitter, Input, Output, OnChanges, SimpleChanges, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import {
  LucideSparkles,
  LucideX,
  LucideRefreshCw,
  LucideEye
} from '@lucide/angular';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { TeacherCourseService } from '../../../../../services/teacher-course.service';

@Component({
  selector: 'generate-cases-modal',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideSparkles,
    LucideX,
    LucideRefreshCw,
    LucideEye,
    MachineDataDirective
  ],
  templateUrl: './generate-cases-modal.component.html',
  styleUrl: './generate-cases-modal.component.scss'
})
export class GenerateCasesModalComponent implements OnChanges {
  private courseService = inject(TeacherCourseService);

  @Input() isOpen = false;
  @Input() contract: any = null;

  @Output() casesGenerated = new EventEmitter<Array<{ input: string }>>();
  @Output() cancel = new EventEmitter<void>();

  readonly count = signal<number>(10);
  readonly seed = signal<number | null>(null);

  readonly previewCases = signal<Array<{ input: string }>>([]);
  readonly isLoadingPreview = signal<boolean>(false);
  readonly isGenerating = signal<boolean>(false);
  readonly error = signal<string | null>(null);

  ngOnChanges(changes: SimpleChanges): void {
    if ((changes['isOpen'] && this.isOpen) || (changes['contract'] && this.isOpen)) {
      if (this.contract) {
        this.fetchPreview();
      }
    }
  }

  onCountChange(val: number): void {
    const num = isNaN(val) ? 10 : Math.min(100, Math.max(1, Number(val)));
    this.count.set(num);
  }

  onSeedChange(val: string): void {
    if (val === '' || val === null || val === undefined) {
      this.seed.set(null);
    } else {
      const num = Number(val);
      this.seed.set(isNaN(num) ? null : num);
    }
    this.fetchPreview();
  }

  fetchPreview(): void {
    if (!this.contract) return;

    this.isLoadingPreview.set(true);
    this.error.set(null);

    const seedVal = this.seed() ?? undefined;
    this.courseService.generateCases(this.contract, 3, seedVal).subscribe({
      next: (res) => {
        this.previewCases.set(res.cases || []);
        this.isLoadingPreview.set(false);
      },
      error: (err) => {
        this.error.set(err?.error?.message || err?.message || 'Error al generar preview de casos');
        this.isLoadingPreview.set(false);
      }
    });
  }

  onConfirm(): void {
    if (!this.contract) return;

    this.isGenerating.set(true);
    this.error.set(null);

    const c = this.count();
    const seedVal = this.seed() ?? undefined;

    this.courseService.generateCases(this.contract, c, seedVal).subscribe({
      next: (res) => {
        this.isGenerating.set(false);
        this.casesGenerated.emit(res.cases || []);
        this.onClose();
      },
      error: (err) => {
        this.error.set(err?.error?.message || err?.message || 'Error al generar casos');
        this.isGenerating.set(false);
      }
    });
  }

  onClose(): void {
    this.cancel.emit();
  }

  onBackdropClick(event: MouseEvent): void {
    if ((event.target as HTMLElement).classList.contains('modal-backdrop')) {
      this.onClose();
    }
  }

  onKeyDown(event: KeyboardEvent): void {
    if (event.key === 'Escape') {
      this.onClose();
    }
  }
}
