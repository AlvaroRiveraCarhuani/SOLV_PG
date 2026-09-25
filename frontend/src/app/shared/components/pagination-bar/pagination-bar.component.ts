import { Component, ChangeDetectionStrategy, input, output, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { LucideChevronLeft, LucideChevronRight } from '@lucide/angular';

@Component({
  selector: 'pagination-bar',
  standalone: true,
  imports: [CommonModule, LucideChevronLeft, LucideChevronRight],
  templateUrl: './pagination-bar.component.html',
  styleUrl: './pagination-bar.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class PaginationBarComponent {
  readonly currentPage = input.required<number>();
  readonly totalPages = input.required<number>();
  readonly totalItems = input<number>();
  readonly itemLabel = input<string>('elementos');
  readonly showPageNumbers = input<boolean>(true);
  readonly from = input<number>();
  readonly to = input<number>();

  readonly pageChange = output<number>();

  readonly displayedPages = computed<number[]>(() => {
    const total = this.totalPages();
    const current = this.currentPage();

    if (total <= 7) {
      return Array.from({ length: total }, (_, i) => i + 1);
    }

    const pages = new Set<number>();
    pages.add(1);
    pages.add(total);

    for (let i = Math.max(1, current - 2); i <= Math.min(total, current + 2); i++) {
      pages.add(i);
    }

    return Array.from(pages).sort((a, b) => a - b);
  });

  goToPage(page: number): void {
    if (page >= 1 && page <= this.totalPages() && page !== this.currentPage()) {
      this.pageChange.emit(page);
    }
  }
}
