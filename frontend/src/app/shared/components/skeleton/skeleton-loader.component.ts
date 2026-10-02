import { Component, ChangeDetectionStrategy, input, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { SkeletonComponent } from './skeleton.component';

export type SkeletonVariant = 'cards' | 'table' | 'kpi' | 'code' | 'form' | 'list';

@Component({
  selector: 'skeleton-loader',
  standalone: true,
  imports: [CommonModule, SkeletonComponent],
  templateUrl: './skeleton-loader.component.html',
  styleUrl: './skeleton-loader.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class SkeletonLoaderComponent {
  readonly variant = input<SkeletonVariant>('cards');
  readonly count = input<number>(3);
  readonly rows = input<number>(4);
  readonly columns = input<number>(5);

  itemsArray = computed(() => Array.from({ length: this.count() }));
  rowsArray = computed(() => Array.from({ length: this.rows() }));
  columnsArray = computed(() => Array.from({ length: this.columns() }));
}
