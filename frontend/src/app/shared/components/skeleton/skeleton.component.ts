import { Component, ChangeDetectionStrategy, input } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'skeleton',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div 
      class="skeleton-primitive"
      [style.width]="width()"
      [style.height]="height()"
      [style.border-radius]="radius()"
      [class.pulse]="animation() === 'pulse'"
      [class.shimmer]="animation() === 'shimmer'"
    ></div>
  `,
  styleUrl: './skeleton.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class SkeletonComponent {
  readonly width = input<string>('100%');
  readonly height = input<string>('1rem');
  readonly radius = input<string>('var(--radius-sm, 4px)');
  readonly animation = input<'shimmer' | 'pulse' | 'none'>('shimmer');
}
