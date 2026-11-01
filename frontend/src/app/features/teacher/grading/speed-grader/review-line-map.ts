import { ProjectFile, SubmissionComment } from '../../models/teacher.models';

/**
 * Traducción entre la línea global de `code` (la que persiste el backend) y la
 * línea local del archivo visible. Ambas son 1-based.
 *
 * `files[].start_line` es la primera línea de contenido del archivo dentro del
 * snapshot. Los comentarios siguen anclados a la línea global para no migrar la
 * tabla `submission_comments`.
 */
export function isLineInFile(file: ProjectFile, globalLine: number): boolean {
  if (file.line_count <= 0) return false;
  return globalLine >= file.start_line && globalLine < file.start_line + file.line_count;
}

export function globalToLocalLine(file: ProjectFile, globalLine: number): number | null {
  if (!isLineInFile(file, globalLine)) return null;
  return globalLine - file.start_line + 1;
}

export function localToGlobalLine(file: ProjectFile, localLine: number): number | null {
  if (localLine < 1 || localLine > file.line_count) return null;
  return file.start_line + localLine - 1;
}

export function fileForGlobalLine(files: ProjectFile[], globalLine: number): ProjectFile | null {
  return files.find((file) => isLineInFile(file, globalLine)) ?? null;
}

export function commentEndLine(comment: SubmissionComment): number {
  const end = comment.end_line_number ?? comment.line_number;
  return end >= comment.line_number ? end : comment.line_number;
}

/** Un comentario de bloque toca el archivo si su inicio o su cierre cae dentro. */
export function commentTouchesFile(file: ProjectFile, comment: SubmissionComment): boolean {
  if (isLineInFile(file, comment.line_number)) return true;
  return commentEndLine(comment) !== comment.line_number && isLineInFile(file, commentEndLine(comment));
}

/** Rango local del comentario dentro del archivo, recortado a sus límites. */
export function localRangeForComment(
  file: ProjectFile,
  comment: SubmissionComment
): { start: number; end: number } | null {
  if (!commentTouchesFile(file, comment)) return null;
  const start = globalToLocalLine(file, comment.line_number) ?? 1;
  const rawEnd = globalToLocalLine(file, commentEndLine(comment)) ?? file.line_count;
  return { start, end: Math.max(start, rawEnd) };
}

export function commentsForFile(
  comments: SubmissionComment[],
  file: ProjectFile
): SubmissionComment[] {
  return comments.filter((comment) => commentTouchesFile(file, comment));
}

/**
 * Comentarios que no caen dentro de ningún archivo visible: cabeceras del
 * snapshot, separadores o entregas sin archivos reconocibles. Se muestran como
 * comentarios generales, nunca anclados a un archivo.
 */
export function commentsOutsideFiles(
  comments: SubmissionComment[],
  files: ProjectFile[]
): SubmissionComment[] {
  return comments.filter((comment) => fileForGlobalLine(files, comment.line_number) === null);
}

/** Etiqueta compacta del bloque: L12 o L12–L18. */
export function commentRangeLabel(start: number, end: number): string {
  return end > start ? `L${start}–L${end}` : `L${start}`;
}
