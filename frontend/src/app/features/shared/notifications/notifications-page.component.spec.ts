import { ComponentFixture, TestBed } from '@angular/core/testing';
import { signal, computed, Signal, WritableSignal } from '@angular/core';
import { Mock, vi } from 'vitest';
import { provideRouter } from '@angular/router';
import { NotificationItem, NotificationService } from '@core/services/notification.service';
import { NotificationsPageComponent } from './notifications-page.component';

function notification(overrides: Partial<NotificationItem> = {}): NotificationItem {
  return {
    id: 'n-1',
    tenant_id: 'tenant-1',
    recipient_user_id: 'user-1',
    channel: 'in_app',
    severity: 'info',
    title: 'Evaluación publicada',
    message: 'Tu nota está disponible.',
    event_type: 'grade_published',
    is_read: false,
    occurrence_count: 1,
    link: '/student/history/sub-1',
    created_at: '2026-11-10T12:00:00Z',
    ...overrides
  };
}

describe('NotificationsPageComponent', () => {
  let fixture: ComponentFixture<NotificationsPageComponent>;
  let component: NotificationsPageComponent;
  let stub: {
    items: WritableSignal<NotificationItem[]>;
    loading: WritableSignal<boolean>;
    error: WritableSignal<string | null>;
    page: WritableSignal<number>;
    total: WritableSignal<number>;
    unreadCount: WritableSignal<number>;
    totalPages: Signal<number>;
    hasUnread: Signal<boolean>;
    loadPage: Mock;
    markAllRead: Mock;
    markRead: Mock;
  };

  beforeEach(async () => {
    const items = signal<NotificationItem[]>([notification()]);
    const total = signal(1);
    const unreadCount = signal(1);
    stub = {
      items,
      loading: signal(false),
      error: signal<string | null>(null),
      page: signal(1),
      total,
      unreadCount,
      totalPages: computed(() => 1),
      hasUnread: computed(() => unreadCount() > 0),
      loadPage: vi.fn(() => Promise.resolve()),
      markAllRead: vi.fn(() => Promise.resolve()),
      markRead: vi.fn(() => Promise.resolve())
    };

    await TestBed.configureTestingModule({
      imports: [NotificationsPageComponent],
      providers: [
        provideRouter([]),
        { provide: NotificationService, useValue: stub }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(NotificationsPageComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('carga la primera página al entrar', () => {
    expect(stub.loadPage).toHaveBeenCalledWith(1, { unreadOnly: false, severity: 'all' });
  });

  it('muestra severidad, enlace y ocurrencias agrupadas', () => {
    stub.items.set([notification({ severity: 'critical', occurrence_count: 2 })]);
    fixture.detectChanges();

    const text = (fixture.nativeElement as HTMLElement).textContent ?? '';
    expect(text).toContain('Crítico');
    expect(text).toContain('Evaluación publicada');
    expect(text).toContain('2 ocurrencias');
    expect(fixture.nativeElement.querySelector('.notification-action')).not.toBeNull();
  });

  it('pide solo los no leídos al cambiar de pestaña', () => {
    component.changeTab('unread');
    expect(component.activeTab()).toBe('unread');
    expect(stub.loadPage).toHaveBeenLastCalledWith(1, { unreadOnly: true, severity: 'all' });
  });

  it('marca todas como leídas', () => {
    component.markAllRead();
    expect(stub.markAllRead).toHaveBeenCalled();
  });
});
