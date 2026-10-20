import {
  Component,
  ElementRef,
  OnDestroy,
  OnInit,
  ViewChild,
  computed,
  effect,
  input,
  model
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { EditorView, basicSetup } from 'codemirror';
import { Compartment, EditorState } from '@codemirror/state';
import { keymap } from '@codemirror/view';
import { defaultKeymap, indentWithTab } from '@codemirror/commands';
import { python } from '@codemirror/lang-python';
import { cpp } from '@codemirror/lang-cpp';
import { java } from '@codemirror/lang-java';
import { javascript } from '@codemirror/lang-javascript';
import { json } from '@codemirror/lang-json';
import { StreamLanguage, HighlightStyle, syntaxHighlighting } from '@codemirror/language';
import { tags as t } from '@lezer/highlight';
import { dockerFile } from '@codemirror/legacy-modes/mode/dockerfile';
import { shell } from '@codemirror/legacy-modes/mode/shell';

export const solvHighlightStyle = HighlightStyle.define([
  { tag: t.keyword, color: 'var(--tenant-primary)', fontWeight: '600' },
  { tag: [t.name, t.deleted, t.character, t.propertyName, t.macroName], color: 'var(--text-primary)' },
  { tag: [t.function(t.variableName), t.labelName], color: 'var(--tenant-primary)' },
  { tag: [t.color, t.constant(t.name), t.standard(t.name)], color: 'var(--tenant-primary)' },
  { tag: [t.definition(t.name), t.separator], color: 'var(--text-primary)' },
  { tag: [t.typeName, t.className, t.number, t.changed, t.annotation, t.modifier, t.self, t.namespace], color: 'var(--tenant-primary)' },
  { tag: [t.operator, t.operatorKeyword, t.url, t.escape, t.regexp, t.link, t.special(t.string)], color: 'var(--tenant-primary)' },
  { tag: [t.meta, t.comment], color: 'var(--text-muted)', fontStyle: 'italic' },
  { tag: t.strong, fontWeight: 'bold' },
  { tag: t.emphasis, fontStyle: 'italic' },
  { tag: t.strikethrough, textDecoration: 'line-through' },
  { tag: t.link, color: 'var(--tenant-primary)', textDecoration: 'underline' },
  { tag: t.heading, fontWeight: 'bold', color: 'var(--text-primary)' },
  { tag: [t.atom, t.bool, t.special(t.variableName)], color: 'var(--tenant-primary)' },
  { tag: [t.processingInstruction, t.string, t.inserted], color: 'var(--color-success-dark, var(--color-success))' },
  { tag: t.invalid, color: 'var(--color-error)' }
]);

const solvEditorTheme = EditorView.theme({
  '&': {
    height: '100%',
    fontSize: 'var(--code-editor-font-size, var(--text-sm))',
    fontFamily: 'var(--font-mono)',
    backgroundColor: 'transparent',
    color: 'var(--text-primary)'
  },
  '.cm-content': {
    fontFamily: 'var(--font-mono)',
    padding: 'var(--space-2) 0',
    caretColor: 'var(--tenant-primary)'
  },
  '.cm-line': {
    padding: '0 var(--space-3)',
    lineHeight: 'var(--leading-relaxed, 1.6)'
  },
  '.cm-gutters': {
    backgroundColor: 'var(--bg-surface-subtle)',
    color: 'var(--text-muted)',
    borderRight: '1px solid var(--border-subtle)',
    paddingRight: 'var(--space-1)'
  },
  '.cm-lineNumbers .cm-gutterElement': {
    fontFamily: 'var(--font-mono)',
    fontSize: 'var(--text-xs)',
    minWidth: '32px',
    textAlign: 'right',
    padding: '0 var(--space-2)'
  },
  '.cm-activeLine': {
    backgroundColor: 'var(--bg-surface-active)'
  },
  '.cm-activeLineGutter': {
    backgroundColor: 'var(--bg-surface-active)',
    color: 'var(--tenant-primary)',
    fontWeight: '600'
  },
  '&.cm-focused': {
    outline: 'none'
  }
});

@Component({
  selector: 'code-editor',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div
      class="code-editor-root"
      [class.is-readonly]="readonly()"
      [style.--code-editor-font-size]="computedFontSize()"
    >
      <div
        #editorContainer
        class="editor-surface font-mono"
        [style.min-height]="minHeight()"
        [style.max-height]="maxHeight()"
      ></div>
    </div>
  `,
  styleUrl: './code-editor.component.scss'
})
export class CodeEditorComponent implements OnInit, OnDestroy {
  readonly value = model<string>('');
  readonly language = input<string>('python');
  readonly readonly = input<boolean>(false);
  readonly minHeight = input<string>('240px');
  readonly maxHeight = input<string>('600px');
  readonly fontSize = input<number | string | null>(null);

  readonly computedFontSize = computed(() => {
    const sz = this.fontSize();
    if (!sz) return null;
    return typeof sz === 'number' ? `${sz}px` : sz;
  });

  @ViewChild('editorContainer', { static: true })
  private readonly editorContainerRef!: ElementRef<HTMLDivElement>;

  private editorView?: EditorView;
  private readonly langCompartment = new Compartment();
  private readonly readOnlyCompartment = new Compartment();
  private isInternalUpdate = false;

  constructor() {
    // Sincronizar cambios externos de valor hacia CodeMirror
    effect(() => {
      const externalVal = this.value();
      if (!this.editorView || this.isInternalUpdate) {
        this.isInternalUpdate = false;
        return;
      }
      const currentDoc = this.editorView.state.doc.toString();
      if (externalVal !== currentDoc) {
        this.editorView.dispatch({
          changes: { from: 0, to: currentDoc.length, insert: externalVal }
        });
      }
    });

    // Sincronizar dinámicamente cambio de lenguaje
    effect(() => {
      const lang = this.language();
      if (!this.editorView) return;
      this.editorView.dispatch({
        effects: this.langCompartment.reconfigure(this.getLanguageExtension(lang))
      });
    });

    // Sincronizar modo lectura
    effect(() => {
      const isReadOnly = this.readonly();
      if (!this.editorView) return;
      this.editorView.dispatch({
        effects: this.readOnlyCompartment.reconfigure(EditorState.readOnly.of(isReadOnly))
      });
    });
  }

  ngOnInit(): void {
    this.initEditor();
  }

  ngOnDestroy(): void {
    if (this.editorView) {
      this.editorView.destroy();
      this.editorView = undefined;
    }
  }

  private initEditor(): void {
    const parent = this.editorContainerRef.nativeElement;
    const initialText = this.value() || '';

    const startState = EditorState.create({
      doc: initialText,
      extensions: [
        basicSetup,
        solvEditorTheme,
        syntaxHighlighting(solvHighlightStyle),
        keymap.of([indentWithTab, ...defaultKeymap]),
        this.langCompartment.of(this.getLanguageExtension(this.language())),
        this.readOnlyCompartment.of(EditorState.readOnly.of(this.readonly())),
        EditorView.updateListener.of((update) => {
          if (update.docChanged) {
            const newDoc = update.state.doc.toString();
            if (newDoc !== this.value()) {
              this.isInternalUpdate = true;
              this.value.set(newDoc);
            }
          }
        })
      ]
    });

    this.editorView = new EditorView({
      state: startState,
      parent
    });
  }

  private getLanguageExtension(lang: string) {
    const normalized = (lang || '').toLowerCase().trim();
    switch (normalized) {
      case 'python':
      case 'py':
        return python();
      case 'cpp':
      case 'c':
      case 'c++':
        return cpp();
      case 'java':
        return java();
      case 'javascript':
      case 'typescript':
      case 'js':
      case 'ts':
        return javascript({ typescript: normalized.includes('ts') });
      case 'json':
        return json();
      case 'dockerfile':
        return StreamLanguage.define(dockerFile);
      case 'shell':
      case 'sh':
      case 'bash':
        return StreamLanguage.define(shell);
      default:
        return [];
    }
  }
}
