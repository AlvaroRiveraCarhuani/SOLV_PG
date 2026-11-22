import { Component, computed, inject, input, output, signal } from '@angular/core';
import { DomSanitizer, SafeHtml } from '@angular/platform-browser';
import { marked } from 'marked';
import katex from 'katex';
import { LucidePanelRightOpen } from '@lucide/angular';
import { DrawerComponent } from '@shared/components/drawer/drawer.component';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { ExerciseEditorStore } from '../../exercise-editor.store';

function escapeHtml(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

/** Render de la consigna con Markdown y fórmulas, igual que el editor. */
function renderStatement(raw: string): string {
  if (!raw.trim()) {
    return '';
  }

  let processed = raw.replace(/\$\$([\s\S]+?)\$\$/g, (_, math) => {
    try {
      return `<div class="katex-block">${katex.renderToString(math.trim(), { displayMode: true, throwOnError: false, output: 'mathml' })}</div>`;
    } catch {
      return `<pre>${escapeHtml(math)}</pre>`;
    }
  });

  processed = processed.replace(/\$([^$\n]+?)\$/g, (_, math) => {
    try {
      return katex.renderToString(math.trim(), { displayMode: false, throwOnError: false, output: 'mathml' });
    } catch {
      return `<code>${escapeHtml(math)}</code>`;
    }
  });

  try {
    return marked.parse(processed, { async: false }) as string;
  } catch {
    return `<p>${escapeHtml(raw)}</p>`;
  }
}

/** Hoja de consulta de solo lectura para las secciones 2 y 3. */
@Component({
  selector: 'consult-drawer',
  standalone: true,
  imports: [DrawerComponent, MachineDataDirective, LucidePanelRightOpen],
  template: `
    <drawer
      [open]="open()"
      [title]="store.isIde() ? 'Consigna del proyecto' : 'Enunciado del problema'"
      [subtitle]="store.metadata().title || 'Sin título'"
      width="440px"
      (closed)="closed.emit()"
    >
      <div class="consult-tabs" role="tablist" aria-label="Consulta rápida">
        <button
          type="button"
          class="consult-tab"
          role="tab"
          [class.is-active]="tab() === 'statement'"
          [attr.aria-selected]="tab() === 'statement'"
          (click)="tab.set('statement')"
        >
          {{ store.isIde() ? 'Consigna' : 'Enunciado' }}
        </button>
        @if (store.isIde()) {
          <button
            type="button"
            class="consult-tab"
            role="tab"
            [class.is-active]="tab() === 'rubric'"
            [attr.aria-selected]="tab() === 'rubric'"
            (click)="tab.set('rubric')"
          >
            Rúbrica
          </button>
        }
      </div>

      @if (tab() === 'statement') {
        @if (statementHtml()) {
          <div class="consult-markdown" [innerHTML]="statementHtml()"></div>
        } @else {
          <p class="consult-empty">Todavía no hay consigna redactada.</p>
        }
      } @else {
        <div class="rubric-summary">
          @if (hasCriteria()) {
            <div class="rubric-total-row">
              <span class="rubric-total-label">Total</span>
              <span class="rubric-total-badge font-mono" [class.is-balanced]="totalWeight() === 100" machineData>
                {{ totalWeight() }} %
              </span>
            </div>
            <span class="rubric-scale" aria-hidden="true">
              <span class="rubric-scale-fill" [style.width.%]="clampedTotal()" [class.is-balanced]="totalWeight() === 100"></span>
            </span>

            <ul class="rubric-list">
              @for (criterion of criteria(); track $index; let index = $index) {
                <li class="rubric-item">
                  <div class="rubric-item-head">
                    <span class="rubric-name">{{ index + 1 }}. {{ criterion.name }}</span>
                    <span class="rubric-weight font-mono" machineData>{{ criterion.weight }} %</span>
                  </div>
                  @if (criterion.description) {
                    <p class="rubric-desc">{{ criterion.description }}</p>
                  }
                </li>
              }
            </ul>
          } @else {
            <p class="consult-empty">
              Sin criterios definidos. La práctica se califica manualmente sobre 100 puntos.
            </p>
          }
        </div>
      }

      <footer class="consult-footer">
        <button type="button" class="btn-secondary btn-sm" (click)="editSection.emit()">
          <svg lucidePanelRightOpen class="btn-icon-sm" aria-hidden="true"></svg>
          Editar en la sección 1
        </button>
      </footer>
    </drawer>
  `,
  styleUrl: './consult-drawer.component.scss'
})
export class ConsultDrawerComponent {
  readonly store = inject(ExerciseEditorStore);
  private readonly sanitizer = inject(DomSanitizer);

  readonly open = input<boolean>(false);
  readonly closed = output<void>();
  readonly editSection = output<void>();

  readonly tab = signal<'statement' | 'rubric'>('statement');

  readonly statementHtml = computed<SafeHtml | null>(() => {
    const description = this.store.statement().description ?? '';
    if (!description.trim()) {
      return null;
    }
    return this.sanitizer.bypassSecurityTrustHtml(renderStatement(description));
  });

  readonly criteria = computed(() => this.store.rubric()?.criteria ?? []);
  readonly hasCriteria = computed(() => this.criteria().length > 0);

  readonly totalWeight = computed(() =>
    this.criteria().reduce((sum, criterion) => sum + (criterion.weight ?? 0), 0)
  );

  clampedTotal(): number {
    return Math.min(100, Math.max(0, this.totalWeight()));
  }
}
