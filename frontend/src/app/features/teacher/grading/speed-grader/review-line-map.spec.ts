import { describe, expect, it } from 'vitest';
import { ProjectFile, SubmissionComment } from '../../models/teacher.models';
import {
  commentRangeLabel,
  commentsForFile,
  commentsOutsideFiles,
  fileForGlobalLine,
  globalToLocalLine,
  isLineInFile,
  localRangeForComment,
  localToGlobalLine
} from './review-line-map';

const appFile: ProjectFile = {
  path: 'src/app.js',
  language: 'javascript',
  content: 'line one\nline two\nline three\n',
  start_line: 2,
  line_count: 3
};

const emptyFile: ProjectFile = {
  path: 'src/empty.js',
  language: 'javascript',
  content: '',
  start_line: 10,
  line_count: 0
};

function comment(id: string, lineNumber: number, endLineNumber?: number): SubmissionComment {
  return {
    id,
    submission_id: 'sub-1',
    author_id: 'teacher-1',
    author_name: 'Docente',
    line_number: lineNumber,
    ...(endLineNumber !== undefined ? { end_line_number: endLineNumber } : {}),
    comment: `Comentario ${id}`,
    created_at: '2026-11-01T10:00:00Z'
  };
}

describe('review-line-map', () => {
  it('maps the first and last local line to the global snapshot', () => {
    expect(appFile.start_line).toBe(2);
    expect(globalToLocalLine(appFile, 2)).toBe(1);
    expect(globalToLocalLine(appFile, 4)).toBe(3);
    expect(localToGlobalLine(appFile, 1)).toBe(2);
    expect(localToGlobalLine(appFile, 3)).toBe(4);
  });

  it('rejects lines outside the file range', () => {
    expect(globalToLocalLine(appFile, 1)).toBeNull();
    expect(globalToLocalLine(appFile, 5)).toBeNull();
    expect(localToGlobalLine(appFile, 0)).toBeNull();
    expect(localToGlobalLine(appFile, 4)).toBeNull();
    expect(isLineInFile(appFile, 1)).toBe(false);
  });

  it('treats an empty file as having no commentable lines', () => {
    expect(isLineInFile(emptyFile, 10)).toBe(false);
    expect(globalToLocalLine(emptyFile, 10)).toBeNull();
    expect(localToGlobalLine(emptyFile, 1)).toBeNull();
  });

  it('finds the file that owns a global line', () => {
    const files = [appFile, { ...emptyFile, start_line: 8 }];
    expect(fileForGlobalLine(files, 3)?.path).toBe('src/app.js');
    expect(fileForGlobalLine(files, 1)).toBeNull();
    expect(fileForGlobalLine([], 3)).toBeNull();
  });

  it('separates file comments from header or general comments', () => {
    const comments = [comment('header', 1), comment('inside', 3), comment('after', 9)];

    expect(commentsForFile(comments, appFile).map((c) => c.id)).toEqual(['inside']);
    expect(commentsOutsideFiles(comments, [appFile]).map((c) => c.id)).toEqual(['header', 'after']);
  });

  it('maps a multiline comment to its local range', () => {
    const block = comment('block', 2, 4);

    expect(localRangeForComment(appFile, block)).toEqual({ start: 1, end: 3 });
    expect(localRangeForComment(appFile, comment('single', 3))).toEqual({ start: 2, end: 2 });
    expect(localRangeForComment(appFile, comment('header', 1))).toBeNull();
  });

  it('prefers the stored file anchor over the global line', () => {
    const anchored: SubmissionComment = {
      ...comment('anchored', 99, 101),
      file_path: 'src/app.js',
      local_line_number: 2
    };

    expect(commentsForFile([anchored], appFile).map((c) => c.id)).toEqual(['anchored']);
    expect(localRangeForComment(appFile, anchored)).toEqual({ start: 2, end: 3 });
    expect(commentsOutsideFiles([anchored], [appFile])).toEqual([]);
  });

  it('shows an anchored comment as general when its file is gone', () => {
    const anchored: SubmissionComment = {
      ...comment('orphan', 3),
      file_path: 'src/removed.js',
      local_line_number: 1
    };

    expect(commentsForFile([anchored], appFile)).toEqual([]);
    expect(commentsOutsideFiles([anchored], [appFile]).map((c) => c.id)).toEqual(['orphan']);
  });

  it('labels single lines and blocks differently', () => {
    expect(commentRangeLabel(12, 12)).toBe('L12');
    expect(commentRangeLabel(12, 18)).toBe('L12–L18');
  });
});
