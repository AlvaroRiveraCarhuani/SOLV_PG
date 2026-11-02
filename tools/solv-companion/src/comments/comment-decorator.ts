import * as vscode from 'vscode';
import { CommentAnchor, SubmissionComment } from '../domain/comment';

/**
 * Pinta los comentarios anclados en el margen de las líneas y los expone en el
 * hover. Los comentarios sin `file_path` o `local_line_number` no se anclan:
 * pertenecen a la rúbrica o a la modalidad juez.
 */
export class CommentDecorator implements vscode.Disposable {
  private readonly decorationType = vscode.window.createTextEditorDecorationType({
    isWholeLine: true,
    backgroundColor: new vscode.ThemeColor('solv.comment.background'),
    borderColor: new vscode.ThemeColor('solv.comment.border'),
    borderStyle: 'solid',
    borderWidth: '0 0 0 3px',
    overviewRulerColor: new vscode.ThemeColor('solv.comment.border'),
    overviewRulerLane: vscode.OverviewRulerLane.Left
  });

  private readonly hoverProvider = vscode.languages.registerHoverProvider(
    { scheme: 'file' },
    {
      provideHover: (document, position) => this.hoverAt(document, position)
    }
  );

  private readonly anchorsByDocument = new Map<string, CommentAnchor[]>();

  render(comments: SubmissionComment[]): void {
    for (const editor of vscode.window.visibleTextEditors) {
      this.applyToEditor(editor, comments);
    }
  }

  clearAll(): void {
    this.anchorsByDocument.clear();
    for (const editor of vscode.window.visibleTextEditors) {
      editor.setDecorations(this.decorationType, []);
    }
  }

  dispose(): void {
    this.hoverProvider.dispose();
    this.decorationType.dispose();
    this.anchorsByDocument.clear();
  }

  private applyToEditor(
    editor: vscode.TextEditor,
    comments: SubmissionComment[]
  ): void {
    const relativePath = normalize(vscode.workspace.asRelativePath(editor.document.uri, false));
    const absolutePath = normalize(editor.document.uri.fsPath);

    const anchors = comments
      .filter(comment => this.belongsTo(comment, relativePath, absolutePath))
      .map(comment => this.toAnchor(comment, editor.document))
      .filter((anchor): anchor is CommentAnchor => anchor !== null);

    this.anchorsByDocument.set(editor.document.uri.toString(), anchors);

    const decorations = anchors.map(anchor => ({
      range: new vscode.Range(anchor.startLine - 1, 0, anchor.endLine - 1, 0),
      hoverMessage: this.hoverFor(anchor.comment)
    }));
    editor.setDecorations(this.decorationType, decorations);
  }

  private belongsTo(
    comment: SubmissionComment,
    relativePath: string,
    absolutePath: string
  ): boolean {
    if (!comment.file_path) {
      return false;
    }
    const path = normalize(comment.file_path);
    return path === relativePath || absolutePath.endsWith(`/${path}`);
  }

  private toAnchor(
    comment: SubmissionComment,
    document: vscode.TextDocument
  ): CommentAnchor | null {
    const local = comment.local_line_number;
    if (!local || local < 1) {
      return null;
    }

    const span =
      comment.end_line_number && comment.end_line_number > comment.line_number
        ? comment.end_line_number - comment.line_number
        : 0;
    const start = Math.min(local, document.lineCount);
    const end = Math.min(start + span, document.lineCount);
    return { startLine: start, endLine: end, comment };
  }

  private hoverAt(
    document: vscode.TextDocument,
    position: vscode.Position
  ): vscode.Hover | undefined {
    const anchors = this.anchorsByDocument.get(document.uri.toString());
    if (!anchors || anchors.length === 0) {
      return undefined;
    }

    const line = position.line + 1;
    const hits = anchors.filter(
      anchor => line >= anchor.startLine && line <= anchor.endLine
    );
    if (hits.length === 0) {
      return undefined;
    }
    return new vscode.Hover(hits.map(hit => this.hoverFor(hit.comment)));
  }

  private hoverFor(comment: SubmissionComment): vscode.MarkdownString {
    const markdown = new vscode.MarkdownString();
    markdown.isTrusted = false;
    markdown.appendMarkdown('**');
    markdown.appendText(comment.author_name?.trim() || 'Docente');
    markdown.appendMarkdown('**\n\n');
    markdown.appendText(comment.comment);
    if (comment.end_line_number && comment.end_line_number > comment.line_number) {
      const lastLocal =
        (comment.local_line_number ?? comment.line_number) +
        (comment.end_line_number - comment.line_number);
      markdown.appendMarkdown(`\n\n_Líneas ${comment.local_line_number}–${lastLocal}_`);
    }
    return markdown;
  }
}

function normalize(path: string): string {
  return path.replace(/\\/g, '/').replace(/^\.\//, '');
}
