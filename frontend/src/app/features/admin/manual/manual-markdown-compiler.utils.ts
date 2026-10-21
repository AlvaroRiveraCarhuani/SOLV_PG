import { marked } from 'marked';
import katex from 'katex';
import hljs from 'highlight.js/lib/core';
import go from 'highlight.js/lib/languages/go';
import bash from 'highlight.js/lib/languages/bash';
import yaml from 'highlight.js/lib/languages/yaml';
import sql from 'highlight.js/lib/languages/sql';
import typescript from 'highlight.js/lib/languages/typescript';
import json from 'highlight.js/lib/languages/json';

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

export function escapeHtml(text: string): string {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;');
}

export function slugify(text: string): string {
  return text
    .toLowerCase()
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/(^-|-$)+/g, '');
}

export function extractCallouts(text: string): string {
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

export function highlightCode(code: string, lang: string): string {
  if (lang && hljs.getLanguage(lang)) {
    try {
      return hljs.highlight(code, { language: lang, ignoreIllegals: true }).value;
    } catch {
      // fall through to plain escape
    }
  }
  return escapeHtml(code);
}

export function highlightSearchTerms(html: string, query: string): string {
  if (!query) return html;
  const escaped = query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const regex = new RegExp(`(?![^<]*>)(${escaped})`, 'gi');
  return html.replace(regex, '<mark class="search-mark">$1</mark>');
}

export function compileMarkdown(rawText: string, searchQuery: string): { html: string; toc: TocItem[] } {
  const toc: TocItem[] = [];
  let processed = rawText;

  // Block KaTeX $$...$$
  processed = processed.replace(/\$\$([\s\S]+?)\$\$/g, (_, math) => {
    try {
      const rendered = katex.renderToString(math.trim(), { displayMode: true, throwOnError: false });
      return `<div class="katex-block-wrapper">${rendered}</div>`;
    } catch {
      return `<div class="katex-error">${escapeHtml(math)}</div>`;
    }
  });

  // Inline KaTeX $...$
  processed = processed.replace(/\$([^$\n]+?)\$/g, (_, math) => {
    try {
      return katex.renderToString(math.trim(), { displayMode: false, throwOnError: false });
    } catch {
      return escapeHtml(math);
    }
  });

  // Callouts
  processed = extractCallouts(processed);

  // Configure marked renderer
  const renderer = new marked.Renderer();

  // Headings with ID + copy anchor placeholder
  renderer.heading = ({ text, depth }) => {
    const plainText = text.replace(/<[^>]+>/g, '');
    const id = slugify(plainText);
    if (depth === 2 || depth === 3) {
      toc.push({ id, title: plainText, level: depth });
    }
    return `<h${depth} class="manual-h${depth}" id="${id}">${text}</h${depth}>`;
  };

  // Table renderer
  renderer.table = (token) => {
    const headerHtml = token.header
      .map((cell: { text: string; tokens?: unknown[] }) => {
        const cellHtml = cell.tokens
          ? (marked.parseInline(cell.tokens.map((t: unknown) => (t as { raw?: string }).raw ?? '').join('')) as string)
          : escapeHtml(cell.text);
        return `<th>${cellHtml}</th>`;
      })
      .join('');

    const bodyHtml = token.rows
      .map((row: Array<{ text: string; tokens?: unknown[] }>) => {
        const cells = row
          .map((cell) => {
            const cellHtml = cell.tokens
              ? (marked.parseInline(cell.tokens.map((t: unknown) => (t as { raw?: string }).raw ?? '').join('')) as string)
              : escapeHtml(cell.text);
            return `<td>${cellHtml}</td>`;
          })
          .join('');
        return `<tr>${cells}</tr>`;
      })
      .join('');

    return `<div class="manual-table-wrapper"><table class="manual-table"><thead><tr>${headerHtml}</tr></thead><tbody>${bodyHtml}</tbody></table></div>`;
  };

  // Code block renderer with highlight.js + mermaid passthrough
  let blockCounter = 0;
  renderer.code = ({ text, lang }) => {
    const cleanLang = (lang ?? '').trim().toLowerCase();
    blockCounter++;
    const blockId = `code-block-${blockCounter}`;

    if (cleanLang === 'mermaid') {
      return `<div class="mermaid-diagram-container" id="${blockId}" data-mermaid="${encodeURIComponent(text.trim())}"></div>`;
    }

    const highlighted = highlightCode(text.trim(), cleanLang);
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

  renderer.paragraph = ({ text }) => `<p class="manual-p">${text}</p>`;

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

  renderer.blockquote = ({ text }) => `<blockquote class="manual-blockquote">${text}</blockquote>`;
  renderer.strong = ({ text }) => `<strong class="manual-strong">${text}</strong>`;
  renderer.em = ({ text }) => `<em class="manual-em">${text}</em>`;
  renderer.codespan = ({ text }) => `<code class="manual-inline-code">${text}</code>`;
  renderer.hr = () => `<hr class="manual-hr">`;

  let rawHtml = marked.parse(processed, { renderer, async: false }) as string;

  if (searchQuery) {
    rawHtml = highlightSearchTerms(rawHtml, searchQuery);
  }

  return { html: rawHtml, toc };
}
