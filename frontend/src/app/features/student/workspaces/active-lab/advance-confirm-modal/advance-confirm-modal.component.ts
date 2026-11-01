import { ChangeDetectionStrategy, Component, OnInit, inject, input, output, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { LucideDownload, LucideFileCode, LucideRefreshCw } from '@lucide/angular';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import {
  AdvanceApplyResult,
  AdvanceFilePreview,
  ClassAdvanceNoticeService,
  PendingClassAdvance
} from '../../../services/class-advance-notice.service';

/**
 * Confirmación del avance del docente en el laboratorio del alumno.
 *
 * Primero muestra qué archivos se agregan y cuáles se reemplazan; recién
 * después aplica. El servidor respalda lo que reemplaza, así que la pantalla
 * lo dice antes de escribir y deja constancia del resultado después.
 */
@Component({
  selector: 'advance-confirm-modal',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    CommonModule,
    ModalShellComponent,
    MachineDataDirective,
    LucideDownload,
    LucideFileCode,
    LucideRefreshCw
  ],
  templateUrl: './advance-confirm-modal.component.html',
  styleUrl: './advance-confirm-modal.component.scss'
})
export class AdvanceConfirmModalComponent implements OnInit {
  private readonly notices = inject(ClassAdvanceNoticeService);

  readonly advance = input.required<PendingClassAdvance>();

  readonly closed = output<void>();
  readonly applied = output<AdvanceApplyResult>();

  readonly isLoading = signal(true);
  readonly preview = signal<AdvanceFilePreview | null>(null);
  readonly loadError = signal<string | null>(null);
  readonly isApplying = signal(false);
  readonly applyError = signal<string | null>(null);
  readonly result = signal<AdvanceApplyResult | null>(null);

  ngOnInit(): void {
    void this.loadPreview();
  }

  async loadPreview(): Promise<void> {
    this.isLoading.set(true);
    this.loadError.set(null);
    try {
      this.preview.set(await this.notices.preview(this.advance().id));
    } catch (err) {
      this.preview.set(null);
      this.loadError.set(ClassAdvanceNoticeService.reason(err, 'No se pudieron leer los archivos del avance.'));
    } finally {
      this.isLoading.set(false);
    }
  }

  async apply(): Promise<void> {
    if (this.isApplying() || this.result()) {
      return;
    }
    this.isApplying.set(true);
    this.applyError.set(null);
    try {
      const result = await this.notices.apply(this.advance().id);
      this.result.set(result);
      this.applied.emit(result);
    } catch (err) {
      this.applyError.set(ClassAdvanceNoticeService.reason(err, 'No se pudo aplicar el avance.'));
    } finally {
      this.isApplying.set(false);
    }
  }

  onClose(): void {
    if (this.isApplying()) {
      return;
    }
    this.closed.emit();
  }
}
