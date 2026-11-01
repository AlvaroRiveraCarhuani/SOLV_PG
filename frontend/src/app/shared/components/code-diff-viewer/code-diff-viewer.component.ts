import {
  AfterViewInit,
  Component,
  ElementRef,
  OnChanges,
  OnDestroy,
  SimpleChanges,
  ViewChild,
  input
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { EditorView, basicSetup } from 'codemirror';
import { EditorState } from '@codemirror/state';
import { syntaxHighlighting } from '@codemirror/language';
import { unifiedMergeView } from '@codemirror/merge';
import {
  codeMirrorLanguageExtension,
  solvEditorTheme,
  solvHighlightStyle
} from '../code-editor/code-editor.component';

// Los nombres de clase de CodeMirror no son kebab-case; el tema se declara acá
// para que stylelint no los lea como selectores del componente.
const solvDiffTheme = EditorView.theme({
  '.cm-changedLine': {
    backgroundColor: 'var(--color-success-bg, var(--bg-surface-subtle))'
  },
  '.cm-changedText': {
    backgroundColor: 'var(--color-success-subtle, var(--bg-surface-active))',
    borderRadius: 'var(--radius-sm)'
  },
  '.cm-deletedChunk': {
    backgroundColor: 'var(--color-danger-bg, var(--bg-surface-subtle))',
    borderLeft: '2px solid var(--color-danger)'
  },
  '.cm-deletedText': {
    color: 'var(--color-danger)'
  },
  '.cm-mergeGutter': {
    color: 'var(--text-muted)'
  },
  '.cm-mergeControls': {
    display: 'none'
  }
});

/**
 * Diff nativo de solo lectura para SpeedGrader.
 *
 * Usa `unifiedMergeView` de CodeMirror: una sola columna con los bloques
 * eliminados intercalados y los agregados resaltados. El docente no acepta ni
 * rechaza cambios; solo lee la diferencia contra la base del ejercicio.
 *
 * El componente se reutiliza en las dos modalidades:
 * - Juez: `original` es el boilerplate del ejercicio y `modified` el código.
 * - IDE: `original` es el archivo base y `modified` el archivo seleccionado.
 */
@Component({
  selector: 'code-diff-viewer',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="diff-viewer-root">
      <div class="diff-legend" aria-hidden="true">
        <span class="legend-item legend-added"><span class="legend-mark">+</span> Código del alumno</span>
        <span class="legend-item legend-removed"><span class="legend-mark">−</span> Proyecto base</span>
      </div>
      <div #diffHost class="diff-host font-mono" [style.height]="height()"></div>
    </div>
  `,
  styleUrl: './code-diff-viewer.component.scss'
})
export class CodeDiffViewerComponent implements AfterViewInit, OnChanges, OnDestroy {
  readonly original = input.required<string>();
  readonly modified = input.required<string>();
  readonly language = input<string>('plaintext');
  readonly height = input<string>('520px');

  @ViewChild('diffHost', { static: true })
  private readonly hostRef!: ElementRef<HTMLDivElement>;

  private view?: EditorView;

  ngAfterViewInit(): void {
    this.render();
  }

  ngOnChanges(changes: SimpleChanges): void {
    if (!this.view) return;
    if (changes['original'] || changes['modified'] || changes['language']) {
      this.render();
    }
  }

  ngOnDestroy(): void {
    this.destroy();
  }

  private destroy(): void {
    this.view?.destroy();
    this.view = undefined;
  }

  private render(): void {
    this.destroy();
    const host = this.hostRef?.nativeElement;
    if (!host) return;

    const original = this.original() ?? '';
    const modified = this.modified() ?? '';

    const state = EditorState.create({
      doc: modified,
      extensions: [
        basicSetup,
        solvEditorTheme,
        solvDiffTheme,
        syntaxHighlighting(solvHighlightStyle),
        codeMirrorLanguageExtension(this.language()),
        EditorState.readOnly.of(true),
        EditorView.editable.of(false),
        unifiedMergeView({
          original,
          highlightChanges: true,
          gutter: true,
          syntaxHighlightDeletions: true,
          allowInlineDiffs: true,
          mergeControls: false
        })
      ]
    });

    this.view = new EditorView({ state, parent: host });
  }
}
