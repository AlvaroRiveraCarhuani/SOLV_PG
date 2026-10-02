export interface DiffLine {
  type: 'added' | 'removed' | 'unchanged';
  oldLineNumber?: number;
  newLineNumber?: number;
  content: string;
}

export function computeLineDiff(originalCode: string, modifiedCode: string): DiffLine[] {
  const origLines = (originalCode || '').split('\n');
  const modLines = (modifiedCode || '').split('\n');

  // If original is empty, all modified lines are added
  if (!originalCode.trim()) {
    return modLines.map((line, idx) => ({
      type: 'added',
      newLineNumber: idx + 1,
      content: line
    }));
  }

  const result: DiffLine[] = [];
  let i = 0;
  let j = 0;

  while (i < origLines.length || j < modLines.length) {
    if (i < origLines.length && j < modLines.length) {
      if (origLines[i] === modLines[j]) {
        result.push({
          type: 'unchanged',
          oldLineNumber: i + 1,
          newLineNumber: j + 1,
          content: modLines[j]
        });
        i++;
        j++;
      } else {
        // Look ahead to see if it's an insertion
        const nextMatchInMod = modLines.indexOf(origLines[i], j);
        if (nextMatchInMod !== -1 && nextMatchInMod - j < 5) {
          while (j < nextMatchInMod) {
            result.push({
              type: 'added',
              newLineNumber: j + 1,
              content: modLines[j]
            });
            j++;
          }
        } else {
          // Changed or added
          result.push({
            type: 'added',
            newLineNumber: j + 1,
            content: modLines[j]
          });
          j++;
          i++;
        }
      }
    } else if (j < modLines.length) {
      result.push({
        type: 'added',
        newLineNumber: j + 1,
        content: modLines[j]
      });
      j++;
    } else {
      result.push({
        type: 'removed',
        oldLineNumber: i + 1,
        content: origLines[i]
      });
      i++;
    }
  }

  return result;
}
