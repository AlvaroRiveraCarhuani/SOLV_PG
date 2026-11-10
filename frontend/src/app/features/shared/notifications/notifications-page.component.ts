import { ChangeDetectionStrategy, Component, OnInit, computed, inject, signal } from '@angular/core';
import { Router } from '@angular/router';
import {
  NOTIFICATIONS_PAGE_SIZE,
  NotificationItem,
  NotificationPageFilter,
  NotificationService,
  NotificationSeverity
} from '@core/services/notification.service';
import { ComboboxComponent, ComboboxOption } from '@shared/components/combobox/combobox.component';
import { PaginationBarComponent } from '@shared/components/pagination-bar/pagination-bar.component';
import { StatusTabItem, StatusTabsComponent } from '@shared/components/status-tabs/status-tabs.component';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';

type NotificationTab = 'all' | 'unread';
type SeverityFilter = NotificationSeverity | 'all';

@Component({
  selector: 'notifications-page',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    ComboboxComponent,
    PaginationBarComponent,
    StatusTabsComponent,
    DateTextPipe,
    MachineDataDirective
  ],
  templateUrl: './notifications-page.component.html',
  styleUrl: './notifications-page.component.scss'
})
export class NotificationsPageComponent implements OnInit {
  readonly notifService = inject(NotificationService);
  private readonly router = inject(Router);

  readonly activeTab = signal<NotificationTab>('all');
  readonly severity = signal<SeverityFilter>('all');

  readonly items = computed(() => this.notifService.items());
  readonly loading = computed(() => this.notifService.loading());
  readonly error = computed(() => this.notifService.error());
  readonly page = computed(() => this.notifService.page());
  readonly totalPages = computed(() => this.notifService.totalPages());
  readonly total = computed(() => this.notifService.total());

  readonly tabs = computed<StatusTabItem[]>(() => [
    { id: 'all', label: 'Todos' },
    {
      id: 'unread',
      label: 'Sin leer',
      count: this.notifService.unreadCount(),
      badgeVariant: 'warning'
    }
  ]);

  readonly severityOptions: ComboboxOption[] = [
    { id: 'all', label: 'Todas las severidades', value: 'all' },
    { id: 'info', label: 'Informativo', value: 'info' },
    { id: 'warning', label: 'Advertencia', value: 'warning' },
    { id: 'error', label: 'Error', value: 'error' },
    { id: 'critical', label: 'Crítico', value: 'critical' }
  ];

  readonly pageSize = NOTIFICATIONS_PAGE_SIZE;

  ngOnInit(): void {
    void this.notifService.loadPage(1, this.currentFilter());
  }

  changeTab(tab: string): void {
    this.activeTab.set(tab === 'unread' ? 'unread' : 'all');
    void this.notifService.loadPage(1, this.currentFilter());
  }

  changeSeverity(value: string): void {
    this.severity.set(this.isSeverity(value) ? value : 'all');
    void this.notifService.loadPage(1, this.currentFilter());
  }

  changePage(page: number): void {
    void this.notifService.loadPage(page, this.currentFilter());
  }

  markAllRead(): void {
    void this.notifService.markAllRead().then(() => {
      if (this.activeTab() === 'unread') {
        void this.notifService.loadPage(this.page(), this.currentFilter());
      }
    });
  }

  open(item: NotificationItem): void {
    void this.notifService.markRead(item.id);
    if (item.link) {
      void this.router.navigateByUrl(item.link);
    }
  }

  severityLabel(severity: NotificationSeverity): string {
    switch (severity) {
      case 'warning':
        return 'Advertencia';
      case 'error':
        return 'Error';
      case 'critical':
        return 'Crítico';
      case 'info':
      default:
        return 'Informativo';
    }
  }

  private currentFilter(): NotificationPageFilter {
    return {
      unreadOnly: this.activeTab() === 'unread',
      severity: this.severity()
    };
  }

  private isSeverity(value: string): value is SeverityFilter {
    return value === 'all' || value === 'info' || value === 'warning' || value === 'error' || value === 'critical';
  }
}
