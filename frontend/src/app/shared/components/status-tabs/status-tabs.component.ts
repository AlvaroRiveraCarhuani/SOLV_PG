import { Component, ChangeDetectionStrategy, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';

export interface StatusTabItem {
  id: string;
  label: string;
  count?: number;
  badgeVariant?: 'default' | 'active' | 'warning' | 'error' | 'neutral' | string;
}

@Component({
  selector: 'status-tabs',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './status-tabs.component.html',
  styleUrl: './status-tabs.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class StatusTabsComponent {
  readonly tabs = input.required<StatusTabItem[]>();
  readonly activeTab = input.required<string>();

  readonly tabChange = output<string>();

  selectTab(tabId: string): void {
    if (tabId !== this.activeTab()) {
      this.tabChange.emit(tabId);
    }
  }
}
