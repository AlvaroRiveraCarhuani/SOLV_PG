import {
  Component,
  OnInit,
  AfterViewInit,
  inject,
  signal,
  computed,
  effect,
  ElementRef,
  DestroyRef,
  HostListener
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { RouterModule, ActivatedRoute } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { DomSanitizer, SafeHtml } from '@angular/platform-browser';
import {
  LucideSearch,
  LucideArrowUp,
  LucideEye,
  LucideEyeOff,
  LucideX,
  LucideRefreshCw
} from '@lucide/angular';
import mermaid from 'mermaid';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { ThemeService } from '@core/services/theme.service';
import {
  TocItem,
  compileMarkdown,
  escapeHtml,
  stripDocumentTitle
} from './manual-markdown-compiler.utils';

export type { TocItem };

/** Holgura superior (en px) para decidir qué título está activo al desplazarse. */
const ACTIVE_HEADING_OFFSET = 140;

@Component({
  selector: 'admin-manual',
  standalone: true,
  imports: [
    CommonModule,
    RouterModule,
    FormsModule,
    ComboboxComponent,
    LucideSearch,
    LucideArrowUp,
    LucideEye,
    LucideEyeOff,
    LucideX,
    LucideRefreshCw
  ],
  templateUrl: './admin-manual.component.html',
  styleUrls: ['./admin-manual.component.scss']
})
export class AdminManualComponent implements OnInit, AfterViewInit {
  private http = inject(HttpClient);
  private sanitizer = inject(DomSanitizer);
  private elementRef = inject<ElementRef<HTMLElement>>(ElementRef);
  private route = inject(ActivatedRoute);
  private destroyRef = inject(DestroyRef);
  private theme = inject(ThemeService);

  loading = signal<boolean>(true);
  error = signal<string | null>(null);
  rawMarkdown = signal<string>('');
  searchQuery = signal<string>('');
  activeSectionId = signal<string>('');
  showToc = signal<boolean>(true);
  showBackToTop = signal<boolean>(false);
  readingProgress = signal<number>(0);

  private scrollTarget: HTMLElement | Window | null = null;
  private detachScroll: (() => void) | null = null;
  private scrollFrame = 0;
  private tocFrame = 0;
  private mermaidScheduled = false;
  private mermaidRun = 0;
  private mermaidChain: Promise<void> = Promise.resolve();

  filteredMarkdown = computed<string>(() => {
    const raw = this.rawMarkdown();
    const query = this.searchQuery().trim().toLowerCase();
    if (!query || !raw) return raw;

    const sections = raw.split(/\n(?=## )/);
    const matched: string[] = [];

    for (const sec of sections) {
      if (sec.toLowerCase().includes(query)) {
        matched.push(sec);
      }
    }

    if (matched.length === 0) {
      return '## Sin resultados\n\nNo se encontraron secciones que coincidan con su búsqueda.';
    }

    return matched.join('\n\n---\n\n');
  });

  parsedData = computed<{ html: SafeHtml; toc: TocItem[] }>(() => {
    const md = stripDocumentTitle(this.filteredMarkdown());
    const query = this.searchQuery().trim();
    const { html, toc } = compileMarkdown(md, query);
    return {
      html: this.sanitizer.bypassSecurityTrustHtml(html),
      toc
    };
  });

  renderedContent = computed<SafeHtml>(() => this.parsedData().html);

  tocItems = computed<TocItem[]>(() => {
    if (this.searchQuery().trim()) {
      const { toc } = compileMarkdown(stripDocumentTitle(this.rawMarkdown()), '');
      return toc;
    }
    return this.parsedData().toc;
  });

  topLevelSections = computed<TocItem[]>(() =>
    this.tocItems().filter((item) => item.level === 2)
  );

  sectionComboboxOptions = computed<ComboboxOption[]>(() =>
    this.topLevelSections().map((sec) => ({
      id: sec.id,
      label: sec.title,
      value: sec.id
    }))
  );

  constructor() {
    // Re-render del diagrama cuando cambia el HTML compilado o el tema activo.
    effect(() => {
      this.renderedContent();
      this.theme.isDark();
      this.scheduleMermaidRender();
    });

    // Mantiene visible en el índice el ítem de la sección activa.
    effect(() => {
      this.activeSectionId();
      this.scheduleActiveTocScroll();
    });

    this.destroyRef.onDestroy(() => {
      this.detachScroll?.();
      this.detachScroll = null;
      if (typeof cancelAnimationFrame === 'function') {
        cancelAnimationFrame(this.scrollFrame);
        cancelAnimationFrame(this.tocFrame);
      }
    });
  }

  ngOnInit(): void {
    this.loadManual();
  }

  ngAfterViewInit(): void {
    this.attachScrollTracking();
    this.scheduleMermaidRender();
  }

  toggleToc(): void {
    this.showToc.update((v) => !v);
  }

  onSectionSelected(option: ComboboxOption): void {
    if (option?.value) {
      this.scrollToSection(option.value);
    }
  }

  @HostListener('click', ['$event'])
  onComponentClick(event: MouseEvent): void {
    const target = event.target as HTMLElement;

    // Copy code button
    const copyBtn = target.closest('[data-copy-btn]') as HTMLElement;
    if (copyBtn) {
      const encoded = copyBtn.getAttribute('data-code');
      if (encoded) {
        navigator.clipboard.writeText(decodeURIComponent(encoded)).then(() => {
          const label = copyBtn.querySelector('.btn-copy-label');
          if (label) {
            const orig = label.textContent;
            label.textContent = '¡Copiado!';
            copyBtn.classList.add('copied');
            setTimeout(() => {
              label.textContent = orig;
              copyBtn.classList.remove('copied');
            }, 2000);
          }
        });
      }
    }

    // Heading anchor copy
    const anchorBtn = target.closest('[data-anchor-btn]') as HTMLElement;
    if (anchorBtn) {
      const id = anchorBtn.getAttribute('data-anchor-id');
      if (id) {
        const url = `${window.location.href.split('#')[0]}#${id}`;
        navigator.clipboard.writeText(url).then(() => {
          anchorBtn.classList.add('anchor-copied');
          setTimeout(() => anchorBtn.classList.remove('anchor-copied'), 1500);
        });
      }
    }
  }

  loadManual(): void {
    this.loading.set(true);
    this.error.set(null);

    this.http.get('/api/v1/admin/manual', { responseType: 'text' }).subscribe({
      next: (markdown) => this.onMarkdownLoaded(markdown),
      error: () => {
        this.http.get('/docs/MANUAL_ADMIN.md', { responseType: 'text' }).subscribe({
          next: (markdown) => this.onMarkdownLoaded(markdown),
          error: () => {
            this.error.set('No se pudo cargar el manual administrativo. Verifique la conexión.');
            this.loading.set(false);
          }
        });
      }
    });
  }

  scrollToSection(id: string): void {
    const target = document.getElementById(id);
    if (target) {
      target.scrollIntoView({ behavior: 'smooth', block: 'start' });
      this.activeSectionId.set(id);
    }
  }

  scrollToTop(): void {
    const target = this.scrollTarget ?? window;
    if (target === window) {
      window.scrollTo({ top: 0, behavior: 'smooth' });
      return;
    }
    (target as HTMLElement).scrollTo({ top: 0, behavior: 'smooth' });
  }

  clearSearch(): void {
    this.searchQuery.set('');
  }

  private onMarkdownLoaded(markdown: string): void {
    this.rawMarkdown.set(markdown);
    this.loading.set(false);

    const afterRender = () => {
      this.attachScrollTracking();
      this.attachHeadingAnchors();
      this.scheduleMermaidRender();
      const fragment = this.route.snapshot.fragment || window.location.hash?.replace('#', '');
      if (fragment) {
        setTimeout(() => this.scrollToSection(fragment), 150);
      }
    };

    if (typeof requestAnimationFrame === 'function') {
      requestAnimationFrame(afterRender);
    } else {
      setTimeout(afterRender, 0);
    }
  }

  // ─── Scroll container real ─────────────────────────────────────────────────

  /**
   * La ventana nunca scrollea en el layout administrativo (`html, body` usan
   * `overflow: hidden`); el scroll vive en `.shell-content`. Se resuelve el
   * primer ancestro con overflow propio y se cae a `window` si no existe.
   */
  private resolveScrollTarget(): HTMLElement | Window {
    let node: HTMLElement | null = this.elementRef.nativeElement.parentElement;
    while (node) {
      const overflowY = getComputedStyle(node).overflowY;
      if (overflowY === 'auto' || overflowY === 'scroll') {
        return node;
      }
      node = node.parentElement;
    }
    return window;
  }

  private attachScrollTracking(): void {
    this.detachScroll?.();
    this.detachScroll = null;

    const target = this.resolveScrollTarget();
    this.scrollTarget = target;

    const handler = () => this.scheduleScrollStateUpdate();
    const listener = handler as EventListener;
    (target as EventTarget).addEventListener('scroll', listener, { passive: true });
    this.detachScroll = () => {
      (target as EventTarget).removeEventListener('scroll', listener);
    };

    this.updateScrollState();
  }

  private scheduleScrollStateUpdate(): void {
    if (typeof requestAnimationFrame !== 'function') {
      this.updateScrollState();
      return;
    }
    if (this.scrollFrame) return;
    this.scrollFrame = requestAnimationFrame(() => {
      this.scrollFrame = 0;
      this.updateScrollState();
    });
  }

  private updateScrollState(): void {
    const target = this.scrollTarget ?? window;
    const isWindow = target === window;
    const el = isWindow
      ? (document.scrollingElement as HTMLElement | null) ?? document.documentElement
      : (target as HTMLElement);

    const scrollTop = isWindow
      ? window.pageYOffset || document.documentElement.scrollTop || el.scrollTop
      : el.scrollTop;
    const scrollHeight = isWindow ? document.documentElement.scrollHeight : el.scrollHeight;
    const clientHeight = isWindow ? window.innerHeight : el.clientHeight;
    const scrollable = scrollHeight - clientHeight;

    this.showBackToTop.set(scrollTop > 300);
    this.readingProgress.set(
      scrollable > 0 ? Math.min(100, Math.max(0, Math.round((scrollTop / scrollable) * 100))) : 0
    );

    this.updateActiveSection(el, isWindow, scrollTop, scrollHeight, clientHeight);
  }

  private updateActiveSection(
    el: HTMLElement,
    isWindow: boolean,
    scrollTop: number,
    scrollHeight: number,
    clientHeight: number
  ): void {
    const headings = Array.from(
      this.elementRef.nativeElement.querySelectorAll<HTMLElement>(
        '.manual-content-card h2[id], .manual-content-card h3[id]'
      )
    );
    if (headings.length === 0) return;

    const containerTop = isWindow ? 0 : el.getBoundingClientRect().top;
    const threshold = containerTop + ACTIVE_HEADING_OFFSET;
    let current = headings[0].id;

    for (const heading of headings) {
      if (heading.getBoundingClientRect().top <= threshold) {
        current = heading.id;
      } else {
        break;
      }
    }

    // Al llegar al final, la última sección es la activa aunque su título no
    // haya alcanzado el umbral.
    if (scrollHeight - clientHeight - scrollTop <= 4) {
      current = headings[headings.length - 1].id;
    }

    if (current && current !== this.activeSectionId()) {
      this.activeSectionId.set(current);
    }
  }

  // ─── Auto-scroll del índice ────────────────────────────────────────────────

  private scheduleActiveTocScroll(): void {
    if (typeof requestAnimationFrame !== 'function') return;
    cancelAnimationFrame(this.tocFrame);
    this.tocFrame = requestAnimationFrame(() => this.scrollActiveTocItemIntoView());
  }

  private scrollActiveTocItemIntoView(): void {
    const card = this.elementRef.nativeElement.querySelector<HTMLElement>('.toc-sticky-card');
    const link = card?.querySelector<HTMLElement>('.toc-link.active');
    if (!card || !link) return;

    const cardRect = card.getBoundingClientRect();
    const linkRect = link.getBoundingClientRect();
    const headerHeight =
      card.querySelector<HTMLElement>('.toc-header')?.getBoundingClientRect().height ?? 0;
    const margin = 8;
    const visibleTop = cardRect.top + headerHeight + margin;
    const visibleBottom = cardRect.bottom - margin;

    if (linkRect.top < visibleTop) {
      card.scrollTop -= visibleTop - linkRect.top;
    } else if (linkRect.bottom > visibleBottom) {
      card.scrollTop += linkRect.bottom - visibleBottom;
    }
  }

  private attachHeadingAnchors(): void {
    const headings = this.elementRef.nativeElement.querySelectorAll(
      '.manual-content-card h2[id], .manual-content-card h3[id]'
    ) as NodeListOf<HTMLElement>;

    headings.forEach((h) => {
      if (h.querySelector('[data-anchor-btn]')) return;
      const btn = document.createElement('button');
      btn.type = 'button';
      btn.className = 'heading-anchor-btn';
      btn.setAttribute('data-anchor-btn', 'true');
      btn.setAttribute('data-anchor-id', h.id);
      btn.setAttribute('title', 'Copiar enlace a esta sección');
      btn.innerHTML = `<svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>`;
      h.appendChild(btn);
    });
  }

  // ─── Mermaid ───────────────────────────────────────────────────────────────

  /**
   * Un solo pase por cambio de contenido o de tema. El doble
   * `requestAnimationFrame` espera a que el `[innerHTML]` esté pintado antes de
   * medir y renderizar el diagrama.
   */
  private scheduleMermaidRender(): void {
    if (this.mermaidScheduled) return;

    if (typeof requestAnimationFrame !== 'function') {
      this.renderMermaidDiagrams();
      return;
    }

    this.mermaidScheduled = true;
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        this.mermaidScheduled = false;
        this.renderMermaidDiagrams();
      });
    });
  }

  /** Encola el render para que nunca haya dos `mermaid.render` en paralelo. */
  private renderMermaidDiagrams(): void {
    const run = ++this.mermaidRun;
    this.mermaidChain = this.mermaidChain
      .catch(() => undefined)
      .then(() => this.paintMermaidDiagrams(run));
  }

  private async paintMermaidDiagrams(run: number): Promise<void> {
    // Un pase más nuevo ya está en cola: éste deja de ser relevante.
    if (run !== this.mermaidRun) return;

    const containers = Array.from(
      this.elementRef.nativeElement.querySelectorAll<HTMLElement>('.mermaid-diagram-container')
    );
    if (containers.length === 0) return;

    mermaid.initialize({
      startOnLoad: false,
      theme: this.theme.isDark() ? 'dark' : 'neutral',
      securityLevel: 'loose',
      fontFamily: this.resolveMermaidFontFamily()
    });

    for (let index = 0; index < containers.length; index++) {
      if (run !== this.mermaidRun) return;

      const el = containers[index];
      const rawCode = el.getAttribute('data-mermaid');
      if (!rawCode) continue;

      const code = decodeURIComponent(rawCode);

      try {
        const { svg } = await mermaid.render(`mermaid-svg-${run}-${index}`, code);
        el.innerHTML = svg;
        el.setAttribute('data-rendered', 'true');
      } catch (err) {
        console.error('Error rendering Mermaid diagram:', err);
        el.innerHTML = `<pre class="mermaid-error"><code>${escapeHtml(code)}</code></pre>`;
        el.setAttribute('data-rendered', 'true');
      }
    }
  }

  private resolveMermaidFontFamily(): string {
    const token = getComputedStyle(document.documentElement)
      .getPropertyValue('--font-sans')
      .trim();
    return token || 'system-ui, sans-serif';
  }
}
