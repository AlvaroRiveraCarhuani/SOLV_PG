import {
  Component,
  OnInit,
  AfterViewInit,
  OnDestroy,
  inject,
  signal,
  computed,
  effect,
  HostListener,
  ElementRef
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { RouterModule } from '@angular/router';
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
import { marked } from 'marked';
import katex from 'katex';
import mermaid from 'mermaid';
import hljs from 'highlight.js/lib/core';
import go from 'highlight.js/lib/languages/go';
import bash from 'highlight.js/lib/languages/bash';
import yaml from 'highlight.js/lib/languages/yaml';
import sql from 'highlight.js/lib/languages/sql';
import typescript from 'highlight.js/lib/languages/typescript';
import json from 'highlight.js/lib/languages/json';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';

// Register only the languages used in SOLV docs (tree-shaking friendly)
hljs.registerLanguage('go', go);
hljs.registerLanguage('bash', bash);
hljs.registerLanguage('sh', bash);
hljs.registerLanguage('yaml', yaml);
hljs.registerLanguage('yml', yaml);
hljs.registerLanguage('sql', sql);
hljs.registerLanguage('typescript', typescript);
hljs.registerLanguage('ts', typescript);
hljs.registerLanguage('json', json);

export interface TocItem {
  id: string;
  title: string;
  level: number;
}

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
export class AdminManualComponent implements OnInit, AfterViewInit, OnDestroy {
  private http = inject(HttpClient);
  private sanitizer = inject(DomSanitizer);
  private elementRef = inject(ElementRef);

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
    const { html, toc } = this.compileMarkdown(md, query);
    return {
      html: this.sanitizer.bypassSecurityTrustHtml(html),
      toc
    };
  });

  renderedContent = computed<SafeHtml>(() => this.parsedData().html);

  tocItems = computed<TocItem[]>(() => {
    if (this.searchQuery().trim()) {
      const { toc } = this.compileMarkdown(this.rawMarkdown(), '');
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
      const _ = this.renderedContent();
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

  ngOnDestroy(): void {
    // nothing to clean up
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
      if (h.querySelector('[data-anchor-btn]')) return; // already attached
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

  private slugify(text: string): string {
    return text
      .toLowerCase()
      .normalize('NFD')
      .replace(/[\u0300-\u036f]/g, '')
      .replace(/[^a-z0-9]+/g, '-')
      .replace(/(^-|-$)+/g, '');
  }

  // FIX 4: Hardened callout regex — handles optional spaces, works multiline robustly
  private extractCallouts(text: string): string {
    return text.replace(
      /^>\s*\[!(NOTE|WARNING|DANGER|TIP)\][^\n]*\n((?:>[ \t]?[^\n]*\n?)*)/gm,
      (_, type, body) => {
        let calloutType = 'info';
        let calloutTitle = 'Nota';
        if (type === 'WARNING') { calloutType = 'warning'; calloutTitle = 'Advertencia'; }
        if (type === 'DANGER')  { calloutType = 'danger';  calloutTitle = 'Riesgo Crítico'; }
        if (type === 'TIP')     { calloutType = 'success'; calloutTitle = 'Recomendación'; }

        const cleanBody = body.replace(/^>[ \t]?/gm, '').trim();
        return [
          `<div class="manual-callout callout-${calloutType}">`,
          `<div class="callout-header"><strong class="callout-title">${calloutTitle}</strong></div>`,
          `<div class="callout-content">${cleanBody}</div>`,
          `</div>\n`
        ].join('');
      }
    );
  }

  // FIX 5: highlight.js syntax coloring
  private highlightCode(code: string, lang: string): string {
    if (lang && hljs.getLanguage(lang)) {
      try {
        return hljs.highlight(code, { language: lang, ignoreIllegals: true }).value;
      } catch {
        // fall through to plain escape
      }
    }
    return this.escapeHtml(code);
  }

  // Search term highlight — wraps matches in <mark class="search-mark">
  private highlightSearchTerms(html: string, query: string): string {
    if (!query) return html;
    const escaped = query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    const regex = new RegExp(`(?![^<]*>)(${escaped})`, 'gi');
    return html.replace(regex, '<mark class="search-mark">$1</mark>');
  }

  private compileMarkdown(rawText: string, searchQuery: string): { html: string; toc: TocItem[] } {
    const toc: TocItem[] = [];
    let processed = rawText;

    // FIX 1 (pre-pass): block KaTeX $$...$$
    processed = processed.replace(/\$\$([\s\S]+?)\$\$/g, (_, math) => {
      try {
        const rendered = katex.renderToString(math.trim(), { displayMode: true, throwOnError: false });
        return `<div class="katex-block-wrapper">${rendered}</div>`;
      } catch {
        return `<div class="katex-error">${this.escapeHtml(math)}</div>`;
      }
    });

    // Inline KaTeX $...$
    processed = processed.replace(/\$([^\$\n]+?)\$/g, (_, math) => {
      try {
        return katex.renderToString(math.trim(), { displayMode: false, throwOnError: false });
      } catch {
        return this.escapeHtml(math);
      }
    });

    // FIX 4: hardened callout extraction (before marked so blockquotes are replaced)
    processed = this.extractCallouts(processed);

    // Configure marked renderer
    const renderer = new marked.Renderer();

    // Headings with ID + copy anchor placeholder
    renderer.heading = ({ text, depth }) => {
      const plainText = text.replace(/<[^>]+>/g, '');
      const id = this.slugify(plainText);
      if (depth === 2 || depth === 3) {
        toc.push({ id, title: plainText, level: depth });
      }
      return `<h${depth} class="manual-h${depth}" id="${id}">${text}</h${depth}>`;
    };

    // FIX 1: table renderer using marked v18 token API
    renderer.table = (token) => {
      // token: { header: Tokens.TableCell[], rows: Tokens.TableCell[][], align: ... }
      const headerHtml = token.header
        .map((cell: { text: string; tokens?: unknown[] }) => {
          const cellHtml = cell.tokens
            ? (marked.parseInline(cell.tokens.map((t: unknown) => (t as { raw?: string }).raw ?? '').join('')) as string)
            : this.escapeHtml(cell.text);
          return `<th>${cellHtml}</th>`;
        })
        .join('');

      const bodyHtml = token.rows
        .map((row: Array<{ text: string; tokens?: unknown[] }>) => {
          const cells = row
            .map((cell) => {
              const cellHtml = cell.tokens
                ? (marked.parseInline(cell.tokens.map((t: unknown) => (t as { raw?: string }).raw ?? '').join('')) as string)
                : this.escapeHtml(cell.text);
              return `<td>${cellHtml}</td>`;
            })
            .join('');
          return `<tr>${cells}</tr>`;
        })
        .join('');

      return `<div class="manual-table-wrapper"><table class="manual-table"><thead><tr>${headerHtml}</tr></thead><tbody>${bodyHtml}</tbody></table></div>`;
    };

    // FIX 5: code renderer with highlight.js + mermaid passthrough
    let blockCounter = 0;
    renderer.code = ({ text, lang }) => {
      const cleanLang = (lang ?? '').trim().toLowerCase();
      blockCounter++;
      const blockId = `code-block-${blockCounter}`;

      if (cleanLang === 'mermaid') {
        return `<div class="mermaid-diagram-container" id="${blockId}" data-mermaid="${encodeURIComponent(text.trim())}"></div>`;
      }

      const highlighted = this.highlightCode(text.trim(), cleanLang);
      const encoded = encodeURIComponent(text.trim());
      const displayLang = cleanLang ? cleanLang.toUpperCase() : 'CÓDIGO';

      return `
        <div class="manual-code-wrapper" id="${blockId}">
          <div class="code-toolbar">
            <span class="code-lang-tag">${displayLang}</span>
            <button type="button" class="btn-copy-code" data-copy-btn data-code="${encoded}">
              <span class="btn-copy-label">Copiar</span>
            </button>
          </div>
          <pre class="manual-code"><code class="hljs lang-${cleanLang}">${highlighted}</code></pre>
        </div>
      `;
    };

    // FIX 2: prose renderers using design system tokens (no bare browser defaults)
    renderer.paragraph = ({ text }) =>
      `<p class="manual-p">${text}</p>`;

    renderer.list = (token) => {
      const tag = token.ordered ? 'ol' : 'ul';
      const items = token.items
        .map((item: { text?: string; tokens?: unknown[] }) => {
          const content = item.tokens
            ? (marked.parse(item.tokens.map((t: unknown) => (t as { raw?: string }).raw ?? '').join(''), { async: false }) as string)
            : (item.text ?? '');
          return `<li>${content}</li>`;
        })
        .join('');
      return `<${tag} class="manual-list">${items}</${tag}>`;
    };

    renderer.link = ({ href, title, text }) => {
      const titleAttr = title ? ` title="${title}"` : '';
      const external = href?.startsWith('http') ? ' target="_blank" rel="noopener noreferrer"' : '';
      return `<a href="${href ?? '#'}"${titleAttr}${external} class="manual-link">${text}</a>`;
    };

    renderer.blockquote = ({ text }) =>
      `<blockquote class="manual-blockquote">${text}</blockquote>`;

    renderer.strong = ({ text }) =>
      `<strong class="manual-strong">${text}</strong>`;

    renderer.em = ({ text }) =>
      `<em class="manual-em">${text}</em>`;

    renderer.codespan = ({ text }) =>
      `<code class="manual-inline-code">${text}</code>`;

    renderer.hr = () => `<hr class="manual-hr">`;

    let rawHtml = marked.parse(processed, { renderer, async: false }) as string;

    // Search highlight pass (after HTML generation, outside tags only)
    if (searchQuery) {
      rawHtml = this.highlightSearchTerms(rawHtml, searchQuery);
    }

    return { html: rawHtml, toc };
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
        el.innerHTML = `<pre class="mermaid-error"><code>${this.escapeHtml(code)}</code></pre>`;
      }
    });
  }

  private escapeHtml(text: string): string {
    return text
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#039;');
  }
}
