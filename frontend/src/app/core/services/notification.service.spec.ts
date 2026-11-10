import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { signal } from '@angular/core';
import { AuthService } from '@core/services/auth.service';
import { NotificationItem, NotificationService } from './notification.service';

const ITEM: NotificationItem = {
  id: '11111111-1111-1111-1111-111111111111',
  tenant_id: 'tenant-1',
  recipient_user_id: 'user-1',
  channel: 'in_app',
  severity: 'critical',
  title: 'Falla de respaldo',
  message: 'Sin espacio en disco.',
  event_type: 'backup_failed',
  is_read: false,
  occurrence_count: 2,
  link: '/admin/configuracion',
  created_at: '2026-11-10T12:00:00Z'
};

describe('NotificationService', () => {
  let service: NotificationService;
  let httpMock: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        NotificationService,
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: AuthService,
          useValue: {
            currentUser: signal({ id: 'user-1' }),
            isAuthenticated: () => true
          }
        }
      ]
    });
    service = TestBed.inject(NotificationService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    service.stopPolling();
    httpMock.verify();
  });

  it('envía filtros y lee el total desde X-Total-Count', async () => {
    const pending = service.loadPage(2, { unreadOnly: true, severity: 'critical' });

    const req = httpMock.expectOne((r) => r.url === '/api/v1/notifications');
    expect(req.request.params.get('page')).toBe('2');
    expect(req.request.params.get('unread_only')).toBe('true');
    expect(req.request.params.get('severity')).toBe('critical');
    req.flush({ data: [ITEM] }, { headers: { 'X-Total-Count': '16' } });

    await pending;
    expect(service.items()).toEqual([ITEM]);
    expect(service.total()).toBe(16);
    expect(service.totalPages()).toBe(2);
  });

  it('carga los avisos recientes sin filtros', async () => {
    const pending = service.loadRecent();

    const req = httpMock.expectOne((r) => r.url === '/api/v1/notifications');
    expect(req.request.params.get('limit')).toBe('8');
    req.flush([ITEM]);

    await pending;
    expect(service.recent()).toEqual([ITEM]);
  });

  it('marca todas como leídas y limpia el contador', async () => {
    const countPending = service.refreshUnread();
    httpMock.expectOne('/api/v1/notifications/unread-count').flush({ data: { unread_count: 3 } });
    await countPending;

    const pagePending = service.loadPage(1, {});
    httpMock.expectOne((r) => r.url === '/api/v1/notifications').flush({ data: [ITEM] });
    await pagePending;

    const markPending = service.markAllRead();
    const req = httpMock.expectOne('/api/v1/notifications/mark-all-read');
    expect(req.request.method).toBe('POST');
    req.flush({ data: { marked_count: 3 } });
    await markPending;

    expect(service.unreadCount()).toBe(0);
    expect(service.items()[0].is_read).toBe(true);
  });

  it('marca un aviso individual y descuenta el contador', async () => {
    const countPending = service.refreshUnread();
    httpMock.expectOne('/api/v1/notifications/unread-count').flush({ data: { unread_count: 1 } });
    await countPending;

    const pagePending = service.loadPage(1, {});
    httpMock.expectOne((r) => r.url === '/api/v1/notifications').flush({ data: [ITEM] });
    await pagePending;

    const markPending = service.markRead(ITEM.id);
    const req = httpMock.expectOne(`/api/v1/notifications/${ITEM.id}/read`);
    expect(req.request.method).toBe('PATCH');
    req.flush({ data: { status: 'marked_as_read' } });
    await markPending;

    expect(service.items()[0].is_read).toBe(true);
    expect(service.unreadCount()).toBe(0);
  });
});
