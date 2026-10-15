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
      default:
        return [];
    }
  }
}
