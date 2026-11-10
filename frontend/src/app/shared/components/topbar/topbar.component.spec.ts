import { ComponentFixture, TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { Router, provideRouter } from '@angular/router';
import { Mock, vi } from 'vitest';
import { AuthService } from '@core/services/auth.service';
import { TenantService } from '@core/services/tenant.service';
import { NotificationItem, NotificationService } from '@core/services/notification.service';
import { TopbarComponent } from './topbar.component';

const NOTICE: NotificationItem = {
  id: 'n-1',
  tenant_id: 'tenant-1',
  recipient_user_id: 'user-1',
  channel: 'in_app',
  severity: 'warning',
  title: 'Entrega por vencer',
  message: 'Tu práctica vence en 12 horas.',
  event_type: 'deadline_reminder',
  is_read: false,
  occurrence_count: 1,
  link: '/student/exercises/ex-1',
  created_at: '2026-11-10T12:00:00Z'
};

describe('TopbarComponent notifications', () => {
  let fixture: ComponentFixture<TopbarComponent>;
  let component: TopbarComponent;
  let notifStub: {
    recent: ReturnType<typeof signal<NotificationItem[]>>;
    unreadCount: ReturnType<typeof signal<number>>;
    loadingRecent: ReturnType<typeof signal<boolean>>;
    hasUnread: ReturnType<typeof signal<boolean>>;
    startPolling: Mock;
    stopPolling: Mock;
    refreshUnread: Mock;
    loadRecent: Mock;
    markAllRead: Mock;
    markRead: Mock;
  };

  beforeEach(async () => {
    notifStub = {
      recent: signal([NOTICE]),
      unreadCount: signal(3),
      loadingRecent: signal(false),
      hasUnread: signal(true),
      startPolling: vi.fn(),
      stopPolling: vi.fn(),
      refreshUnread: vi.fn(() => Promise.resolve()),
      loadRecent: vi.fn(() => Promise.resolve()),
      markAllRead: vi.fn(() => Promise.resolve()),
      markRead: vi.fn(() => Promise.resolve())
    };

    await TestBed.configureTestingModule({
      imports: [TopbarComponent],
      providers: [
        provideRouter([]),
        {
          provide: AuthService,
          useValue: {
            currentUser: signal({
              id: 'user-1',
              first_name: 'ana',
              last_name: 'perez',
              email: 'ana@uab.edu.bo',
              role: 'student',
              avatar_url: ''
            }),
            logout: vi.fn()
          }
        },
        {
          provide: TenantService,
          useValue: { config: signal(null), tenantInitials: signal('SO') }
        },
        { provide: NotificationService, useValue: notifStub }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(TopbarComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('muestra el contador real y no usa localStorage', () => {
    const badge = fixture.nativeElement.querySelector('.notif-badge');
    expect(badge?.textContent.trim()).toBe('3');
    expect(notifStub.startPolling).toHaveBeenCalled();
  });

  it('abre el desplegable con los avisos del backend', () => {
    component.toggleNotifMenu();
    fixture.detectChanges();

    const text = (fixture.nativeElement as HTMLElement).textContent ?? '';
    expect(text).toContain('Entrega por vencer');
    expect(text).toContain('Ver todos los avisos');
    expect(fixture.nativeElement.querySelector('a.notif-see-all')?.getAttribute('href')).toBe(
      '/student/notifications'
    );
  });

  it('marca leído y navega al enlace interno del aviso', () => {
    const navigate = vi.spyOn(TestBed.inject(Router), 'navigateByUrl');
    component.openNotification(NOTICE);

    expect(notifStub.markRead).toHaveBeenCalledWith('n-1');
    expect(navigate).toHaveBeenCalledWith('/student/exercises/ex-1');
  });
});
