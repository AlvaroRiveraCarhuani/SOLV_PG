import {
  Component,
  OnInit,
  AfterViewInit,
  inject,
  signal,
  computed,
  effect,
  HostListener,
  ElementRef
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
import {
  TocItem,
  compileMarkdown,
  escapeHtml
} from './manual-markdown-compiler.utils';

export type { TocItem };

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
  private elementRef = inject(ElementRef);
  private route = inject(ActivatedRoute);

  loading = signal<boolean>(true);
  error = signal<string | null>(null);
  rawMarkdown = signal<string>('');
  searchQuery = signal<string>('');
  activeSectionId = signal<string>('');
  showToc = signal<boolean>(true);
  showBackToTop = signal<boolean>(false);
  readingProgress = signal<number>(0);

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
      return '# Sin resultados\n\nNo se encontraron secciones que coincidan con su búsqueda.';
    }

    return matched.join('\n\n---\n\n');
  });

  parsedData = computed<{ html: SafeHtml; toc: TocItem[] }>(() => {
    const md = this.filteredMarkdown();
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
      const { toc } = compileMarkdown(this.rawMarkdown(), '');
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
    mermaid.initialize({
      startOnLoad: false,
      theme: 'neutral',
      securityLevel: 'loose',
      fontFamily: 'Inter, system-ui, -apple-system, sans-serif'
    });

    effect(() => {
      // Read tracked by the effect so content changes re-run the rendering pass.
      this.renderedContent();
      setTimeout(() => {
        this.renderMermaidDiagrams();
        this.attachHeadingAnchors();
      }, 50);
    });
  }

  ngOnInit(): void {
    this.loadManual();
  }

  ngAfterViewInit(): void {
    this.setupScrollSpy();
    this.renderMermaidDiagrams();
  }

  toggleToc(): void {
    this.showToc.update((v) => !v);
  }

  onSectionSelected(option: ComboboxOption): void {
    if (option?.value) {
      this.scrollToSection(option.value);
    }
  }

  @HostListener('window:scroll', [])
  onWindowScroll(): void {
    const yOffset = window.pageYOffset || document.documentElement.scrollTop;
    this.showBackToTop.set(yOffset > 300);
    this.updateActiveSectionOnScroll();
    this.updateReadingProgress();
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
    window.scrollTo({ top: 0, behavior: 'smooth' });
  }

  clearSearch(): void {
    this.searchQuery.set('');
  }

  private onMarkdownLoaded(markdown: string): void {
    this.rawMarkdown.set(markdown);
    this.loading.set(false);
    setTimeout(() => {
      this.setupScrollSpy();
      this.renderMermaidDiagrams();
      this.attachHeadingAnchors();
      const fragment = this.route.snapshot.fragment || window.location.hash?.replace('#', '');
      if (fragment) {
        setTimeout(() => this.scrollToSection(fragment), 150);
      }
    }, 100);
  }

  private updateReadingProgress(): void {
    const el = document.documentElement;
    const scrollTop = el.scrollTop || document.body.scrollTop;
    const scrollHeight = el.scrollHeight - el.clientHeight;
    const progress = scrollHeight > 0 ? Math.min(100, (scrollTop / scrollHeight) * 100) : 0;
    this.readingProgress.set(Math.round(progress));
  }

  private updateActiveSectionOnScroll(): void {
    const headings = Array.from(
      this.elementRef.nativeElement.querySelectorAll('.manual-content-card h2, .manual-content-card h3')
    ) as HTMLElement[];

    const scrollPos = (window.pageYOffset || document.documentElement.scrollTop) + 120;

    for (let i = headings.length - 1; i >= 0; i--) {
      const heading = headings[i];
      if (heading.offsetTop <= scrollPos) {
        if (heading.id && heading.id !== this.activeSectionId()) {
          this.activeSectionId.set(heading.id);
        }
        break;
      }
    }
  }

  private setupScrollSpy(): void {
    const headings = this.elementRef.nativeElement.querySelectorAll('.manual-content-card h2');
    if (headings.length > 0 && !this.activeSectionId()) {
      this.activeSectionId.set(headings[0].id);
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

  private renderMermaidDiagrams(): void {
    const containers = this.elementRef.nativeElement.querySelectorAll(
      '.mermaid-diagram-container:not([data-rendered])'
    );
    if (!containers || containers.length === 0) return;

    containers.forEach(async (el: HTMLElement, index: number) => {
      const rawCode = el.getAttribute('data-mermaid');
      if (!rawCode) return;

      const code = decodeURIComponent(rawCode);
      const uniqueId = `mermaid-svg-${Date.now()}-${index}`;

      try {
        const { svg } = await mermaid.render(uniqueId, code);
        el.innerHTML = svg;
        el.setAttribute('data-rendered', 'true');
      } catch (err) {
        console.error('Error rendering Mermaid diagram:', err);
        el.innerHTML = `<pre class="mermaid-error"><code>${escapeHtml(code)}</code></pre>`;
      }
    });
  }
}
