import { Component, ChangeDetectionStrategy, computed, input } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'sparkline',
  standalone: true,
  imports: [CommonModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <svg 
      [attr.viewBox]="'0 0 ' + viewBoxWidth + ' ' + viewBoxHeight" 
      [style.width.px]="width()" 
      [style.height.px]="height()" 
      class="sparkline-svg"
      role="img"
      aria-hidden="true"
    >
      @if (data().length >= 2) {
        @if (filled() && fillPath()) {
          <path [attr.d]="fillPath()" class="sparkline-fill" [style.fill]="color()" />
        }
        <path [attr.d]="linePath()" class="sparkline-stroke" [style.stroke]="color()" />
        @if (lastPoint(); as pt) {
          <circle [attr.cx]="pt.x" [attr.cy]="pt.y" r="2.5" class="sparkline-dot" [style.fill]="color()" />
        }
      } @else {
        <line 
          x1="0" 
          [attr.y1]="viewBoxHeight / 2" 
          [attr.x2]="viewBoxWidth" 
          [attr.y2]="viewBoxHeight / 2" 
          class="sparkline-baseline"
          [style.stroke]="color()" 
        />
      }
    </svg>
  `,
  styleUrl: './sparkline.component.scss'
})
export class SparklineComponent {
  readonly data = input<number[]>([]);
  readonly color = input<string>('var(--tenant-primary)');
  readonly width = input<number>(72);
  readonly height = input<number>(20);
  readonly filled = input<boolean>(true);

  readonly viewBoxWidth = 100;
  readonly viewBoxHeight = 30;

  private coords = computed<{ x: number; y: number }[]>(() => {
    const list = this.data();
    if (!list || list.length === 0) return [];
    if (list.length === 1) {
      return [
        { x: 0, y: this.viewBoxHeight / 2 },
        { x: this.viewBoxWidth, y: this.viewBoxHeight / 2 }
      ];
    }

    const min = Math.min(...list);
    const max = Math.max(...list);
    const range = max === min ? 1 : max - min;
    const padding = 4;
    const availH = this.viewBoxHeight - padding * 2;
    const stepX = this.viewBoxWidth / (list.length - 1);

    return list.map((val, idx) => {
      const normY = (val - min) / range;
      // Invert Y because SVG coordinates have 0 at top
      const y = this.viewBoxHeight - padding - normY * availH;
      const x = idx * stepX;
      return { x: Math.round(x * 10) / 10, y: Math.round(y * 10) / 10 };
    });
  });

  linePath = computed<string>(() => {
    const points = this.coords();
    if (points.length < 2) return '';
    return points.reduce((acc, pt, idx) => {
      return idx === 0 ? `M ${pt.x} ${pt.y}` : `${acc} L ${pt.x} ${pt.y}`;
    }, '');
  });

  fillPath = computed<string>(() => {
    const points = this.coords();
    if (points.length < 2) return '';
    const line = this.linePath();
    const lastX = points[points.length - 1].x;
    const firstX = points[0].x;
    return `${line} L ${lastX} ${this.viewBoxHeight} L ${firstX} ${this.viewBoxHeight} Z`;
  });

  lastPoint = computed<{ x: number; y: number } | null>(() => {
    const points = this.coords();
    return points.length > 0 ? points[points.length - 1] : null;
  });
}
