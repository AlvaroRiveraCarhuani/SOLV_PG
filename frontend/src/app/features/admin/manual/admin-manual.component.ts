import { Component, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { RouterModule } from '@angular/router';
import { DomSanitizer, SafeHtml } from '@angular/platform-browser';

@Component({
  selector: 'solv-admin-manual',
  standalone: true,
  imports: [CommonModule, RouterModule],
  templateUrl: './admin-manual.component.html',
  styleUrls: ['./admin-manual.component.scss']
})
export class AdminManualComponent implements OnInit {
  private http = inject(HttpClient);
  private sanitizer = inject(DomSanitizer);

  loading = signal<boolean>(true);
  error = signal<string | null>(null);
  renderedContent = signal<SafeHtml>('');

  ngOnInit(): void {
    this.loadManual();
  }

  loadManual(): void {
    this.loading.set(true);
    this.error.set(null);

    this.http.get('/docs/MANUAL_ADMIN.md', { responseType: 'text' }).subscribe({
      next: (markdown) => {
        const html = this.parseMarkdown(markdown);
        this.renderedContent.set(this.sanitizer.bypassSecurityTrustHtml(html));
        this.loading.set(false);
      },
      error: () => {
        this.error.set('No se pudo cargar el manual administrativo. Verifique la conexión.');
        this.loading.set(false);
      }
    });
  }

  private parseMarkdown(md: string): string {
    const lines = md.split('\n');
    let html = '';
    let inTable = false;
    let inList = false;
    let inCode = false;
    let codeContent = '';

    for (let i = 0; i < lines.length; i++) {
      const line = lines[i].trimEnd();

      // Fenced code blocks
      if (line.startsWith('```')) {
        if (!inCode) {
          if (inList) { html += '</ul>\n'; inList = false; }
          if (inTable) { html += '</tbody></table></div>\n'; inTable = false; }
          inCode = true;
          codeContent = '';
        } else {
          inCode = false;
          html += `<pre class="manual-code"><code>${this.escapeHtml(codeContent.trim())}</code></pre>\n`;
        }
        continue;
      }

      if (inCode) {
        codeContent += line + '\n';
        continue;
      }

      // Tables
      if (line.startsWith('|') && line.endsWith('|')) {
        if (inList) { html += '</ul>\n'; inList = false; }
        const cells = line.slice(1, -1).split('|').map(c => c.trim());
        
        // Separator row (| :--- | :--- |)
        if (cells.every(c => /^:?-+:?$/.test(c))) {
          continue;
        }

        if (!inTable) {
          inTable = true;
          html += '<div class="manual-table-wrapper"><table class="manual-table"><thead><tr>';
          for (const cell of cells) {
            html += `<th>${this.formatInline(cell)}</th>`;
          }
          html += '</tr></thead><tbody>\n';
        } else {
          html += '<tr>';
          for (const cell of cells) {
            html += `<td>${this.formatInline(cell)}</td>`;
          }
          html += '</tr>\n';
        }
        continue;
      } else if (inTable) {
        html += '</tbody></table></div>\n';
        inTable = false;
      }

      // Lists
      if (line.startsWith('- ') || line.startsWith('* ')) {
        if (!inList) {
          inList = true;
          html += '<ul class="manual-list">\n';
        }
        html += `<li>${this.formatInline(line.substring(2).trim())}</li>\n`;
        continue;
      } else if (inList && !line.startsWith('  ')) {
        html += '</ul>\n';
        inList = false;
      }

      // Headers
      if (line.startsWith('# ')) {
        html += `<h1 class="manual-h1">${this.formatInline(line.substring(2))}</h1>\n`;
      } else if (line.startsWith('## ')) {
        html += `<h2 class="manual-h2">${this.formatInline(line.substring(3))}</h2>\n`;
      } else if (line.startsWith('### ')) {
        html += `<h3 class="manual-h3">${this.formatInline(line.substring(4))}</h3>\n`;
      } else if (line.startsWith('#### ')) {
        html += `<h4 class="manual-h4">${this.formatInline(line.substring(5))}</h4>\n`;
      } else if (line === '---') {
        html += '<hr class="manual-hr" />\n';
      } else if (line.trim().length > 0) {
        html += `<p class="manual-p">${this.formatInline(line)}</p>\n`;
      }
    }

    if (inList) html += '</ul>\n';
    if (inTable) html += '</tbody></table></div>\n';
    if (inCode) html += `<pre class="manual-code"><code>${this.escapeHtml(codeContent.trim())}</code></pre>\n`;

    return html;
  }

  private formatInline(text: string): string {
    let out = this.escapeHtml(text);
    out = out.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');
    out = out.replace(/`([^`]+)`/g, '<code class="manual-inline-code">$1</code>');
    out = out.replace(/\$\$(.+?)\$\$/g, '<div class="manual-math">$1</div>');
    return out;
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
