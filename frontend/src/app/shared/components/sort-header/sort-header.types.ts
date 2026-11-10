export type SortDirection = 'asc' | 'desc' | 'none';

export interface SortState<T = string> {
  column: T | null;
  direction: SortDirection;
}
