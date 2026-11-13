import { Component, OnInit, OnDestroy, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import {
  LucideAlertCircle,
  LucideCheck,
  LucideClock,
  LucideInbox,
  LucideMessageSquare
} from '@lucide/angular';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { SkeletonLoaderComponent } from '@shared/components/skeleton/skeleton-loader.component';
import { ContextHelpService } from '@shared/help/context-help.service';

const REQUESTS_HELP_KEY = 'environment-requests';
import { EnvironmentLabelPipe } from '@shared/pipes/environment-label.pipe';
import { EnvironmentRequestService } from '@features/teacher/services/environment-request.service';
import {
  ENVIRONMENT_REQUEST_STATUS_LABEL,
  ENVIRONMENT_REQUEST_STATUS_TONE,
  EnvironmentRequest,
  EnvironmentRequestStatus
} from '@features/teacher/templates/environment-request.models';
import { AdminTemplatesService } from '../services/admin-templates.service';
import { AdminTemplateItem } from '../models/admin-templates.models';

/**
 * Cola de solicitudes de entorno del docente.
 *
 * La ficha llega armada: piezas reconocidas, las que faltan en el catálogo y el
 * uso declarado. Aprobar exige enlazar la plantilla que resuelve el pedido,
 * porque el docente necesita algo que pueda usar, no un cambio de estado.
 */
@Component({
  selector: 'admin-environment-requests',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    LucideAlertCircle,
    LucideCheck,
    LucideClock,
    LucideInbox,
    LucideMessageSquare,
    DateTextPipe,
    SkeletonLoaderComponent,
    EnvironmentLabelPipe
  ],
  templateUrl: './admin-environment-requests.component.html',
  styleUrl: './admin-environment-requests.component.scss'
})
export class AdminEnvironmentRequestsComponent implements OnInit, OnDestroy {
  private requestService = inject(EnvironmentRequestService);
  private readonly help = inject(ContextHelpService);
  private templatesService = inject(AdminTemplatesService);

  requests = signal<EnvironmentRequest[]>([]);
  templates = signal<AdminTemplateItem[]>([]);
  isLoading = signal<boolean>(true);
  filter = signal<'pending' | 'answered'>('pending');

  selected = signal<EnvironmentRequest | null>(null);
  selectedTemplateId = signal<string>('');
  reviewNote = signal<string>('');
  isSaving = signal<boolean>(false);
  error = signal<string | null>(null);
  notice = signal<string | null>(null);

  readonly pending = computed(() =>
    this.requests().filter(item => item.status === 'EN_REVISION')
  );
  readonly answered = computed(() =>
    this.requests().filter(item => item.status !== 'EN_REVISION')
  );
  readonly visible = computed(() => (this.filter() === 'pending' ? this.pending() : this.answered()));

  ngOnInit(): void {
    this.help.registerScreen(REQUESTS_HELP_KEY);
    this.load();
    this.templatesService.getTemplates('approved').subscribe({
      next: list => this.templates.set(list || []),
      error: () => this.templates.set([])
    });
  }

  load(): void {
    this.isLoading.set(true);
    this.requestService.listForReview().subscribe({
      next: list => {
        this.requests.set(list || []);
        this.isLoading.set(false);
      },
      error: () => {
        this.requests.set([]);
        this.isLoading.set(false);
      }
    });
  }

  setFilter(value: 'pending' | 'answered'): void {
    this.filter.set(value);
  }

  statusLabel(status: EnvironmentRequestStatus): string {
    return ENVIRONMENT_REQUEST_STATUS_LABEL[status] || status;
  }

  statusTone(status: EnvironmentRequestStatus): string {
    return ENVIRONMENT_REQUEST_STATUS_TONE[status] || 'neutral';
  }

  usesLabel(request: EnvironmentRequest): string {
    const labels: string[] = [];
    if (request.uses?.includes('practice')) labels.push('Practicar en clase');
    if (request.uses?.includes('grade')) labels.push('Entregar y calificar');
    return labels.join(' · ') || 'Sin uso declarado';
  }

  ngOnDestroy(): void {
    this.help.unregisterScreen(REQUESTS_HELP_KEY);
  }

  openReview(request: EnvironmentRequest): void {
    this.selected.set(request);
    this.selectedTemplateId.set('');
    this.reviewNote.set('');
    this.error.set(null);
  }

  closeReview(): void {
    this.selected.set(null);
    this.error.set(null);
  }

  approve(): void {
    if (!this.selectedTemplateId()) {
      this.error.set('Elige la plantilla que resuelve esta solicitud.');
      return;
    }
    this.submit({
      status: 'APROBADA',
      review_note: this.reviewNote(),
      resolved_template_id: this.selectedTemplateId()
    });
  }

  requestChanges(): void {
    if (this.reviewNote().trim().length < 5) {
      this.error.set('Explica qué falta para poder continuar.');
      return;
    }
    this.submit({ status: 'NECESITA_CAMBIO', review_note: this.reviewNote() });
  }

  reject(): void {
    if (this.reviewNote().trim().length < 5) {
      this.error.set('Explica por qué no es viable.');
      return;
    }
    this.submit({ status: 'NO_VIABLE', review_note: this.reviewNote() });
  }

  private submit(payload: {
    status: EnvironmentRequestStatus;
    review_note: string;
    resolved_template_id?: string;
  }): void {
    const request = this.selected();
    if (!request) {
      return;
    }
    this.isSaving.set(true);
    this.error.set(null);
    this.requestService.review(request.id, payload).subscribe({
      next: updated => {
        this.isSaving.set(false);
        this.requests.update(list => list.map(item => (item.id === updated.id ? updated : item)));
        this.notice.set(`Solicitud «${updated.title}» actualizada: ${this.statusLabel(updated.status)}.`);
        this.closeReview();
        this.filter.set(payload.status === 'APROBADA' ? 'answered' : 'pending');
      },
      error: err => {
        this.isSaving.set(false);
        this.error.set(err?.error?.message || 'No pudimos registrar la respuesta.');
      }
    });
  }
}
