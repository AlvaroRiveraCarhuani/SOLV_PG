import { TestCaseDTO } from '../../../../models/teacher.models';

export interface ValidationItem {
  valid: boolean;
  error?: string;
  loading?: boolean;
}

export function normalizeText(value: string): string {
  return value ? value.replace(/\r\n/g, '\n').replace(/\r/g, '\n') : '';
}

export function getWeightPercentage(totalWeight: number, weight?: number): string {
  const w = weight ?? 1.0;
  if (totalWeight <= 0) return '0.0%';
  const pct = (w / totalWeight) * 100;
  return `${pct.toFixed(1)}%`;
}

export function mapGeneratedCasesToDTOs(
  newCases: Array<{ input: string; output?: string | null; description?: string }>,
  currentLength: number
): TestCaseDTO[] {
  return newCases.map((c, i) => ({
    input: c.input,
    expected_output: c.output || '',
    description: c.description || `Caso de prueba #${currentLength + i + 1}`,
    visibility: 'hidden',
    weight: 1.0,
    is_hidden: true,
    is_sample: false,
    order_index: currentLength + i + 1
  }));
}

export function reindexValidationAfterRemoval(
  prev: Record<number, ValidationItem>,
  removedIndex: number
): Record<number, ValidationItem> {
  const next: Record<number, ValidationItem> = {};
  Object.keys(prev).forEach(key => {
    const k = Number(key);
    if (k < removedIndex) {
      next[k] = prev[k];
    } else if (k > removedIndex) {
      next[k - 1] = prev[k];
    }
  });
  return next;
}
