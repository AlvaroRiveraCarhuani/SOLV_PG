import { DestroyRef, Injectable, computed, inject, signal } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { AuthService } from '@core/services/auth.service';

export type NotificationSeverity = 'info' | 'warning' | 'error' | 'critical';

/** Aviso proactivo tal como lo entrega GET /api/v1/notifications (ADR-034). */
export interface NotificationItem {
  id: string;
  tenant_id: string;
  recipient_user_id: string;
  channel: string;
  severity: NotificationSeverity;
  title: string;
  message: string;
  event_type?: string;
  is_read: boolean;
  read_at?: string | null;
  occurrence_count: number;
  link?: string;
  created_at: string;
}

export interface NotificationPageFilter {
  severity?: NotificationSeverity | 'all';
  unreadOnly?: boolean;
}

interface Wrapped<T> {
  data?: T;
}

/** Respuesta envuelta o cruda: el backend usa { data } en SendJSON. */
function unwrap<T>(resp: T | Wrapped<T> | null | undefined, fallback: T): T {
  if (resp === null || resp === undefined) return fallback;
  if (typeof resp === 'object' && 'data' in (resp as Wrapped<T>)) {
    return (resp as Wrapped<T>).data ?? fallback;
  }
  return resp as T;
}

const RECENT_LIMIT = 8;
export const NOTIFICATIONS_PAGE_SIZE = 15;
const POLL_INTERVAL_MS = 60_000;

@Injectable({ providedIn: 'root' })
export class NotificationService {
  private readonly http = inject(HttpClient);
  private readonly auth = inject(AuthService);

  /** Últimos avisos para la campana de la barra superior. */
  readonly recent = signal<NotificationItem[]>([]);
  /** Avisos de la página de avisos, con filtros y paginación. */
  readonly items = signal<NotificationItem[]>([]);
  readonly unreadCount = signal<number>(0);
  readonly total = signal<number>(0);
  readonly page = signal<number>(1);
  readonly loadingRecent = signal<boolean>(false);
  readonly loading = signal<boolean>(false);
  readonly error = signal<string | null>(null);

  readonly totalPages = computed(() =>
    Math.max(1, Math.ceil(this.total() / NOTIFICATIONS_PAGE_SIZE))
  );
  readonly hasUnread = computed(() => this.unreadCount() > 0);

  private pollTimer: ReturnType<typeof setInterval> | null = null;
  private visibilityHandler: (() => void) | null = null;
  private lastUserId: string | null = null;
  private pageFilter: NotificationPageFilter = {};

  constructor() {
    inject(DestroyRef).onDestroy(() => this.stopPolling());
  }

  /**
   * Arranca el sondeo del contador. Es idempotente y se reinicia si cambió el
   * usuario autenticado, para no mostrar avisos de otra sesión.
   */
  startPolling(): void {
    const userId = this.auth.currentUser()?.id ?? null;
    if (userId !== this.lastUserId) {
      this.lastUserId = userId;
      this.recent.set([]);
      this.items.set([]);
      this.unreadCount.set(0);
      if (userId) {
        void this.loadRecent();
      }
    }
    if (!userId) {
      this.stopPolling();
      return;
    }
    if (this.pollTimer) return;

    this.pollTimer = setInterval(() => void this.refreshUnread(), POLL_INTERVAL_MS);
    if (typeof document !== 'undefined') {
      this.visibilityHandler = () => {
        if (document.visibilityState === 'visible') {
          void this.refreshUnread();
        }
      };
      document.addEventListener('visibilitychange', this.visibilityHandler);
    }
  }

  stopPolling(): void {
    if (this.pollTimer) {
      clearInterval(this.pollTimer);
      this.pollTimer = null;
    }
    if (this.visibilityHandler && typeof document !== 'undefined') {
      document.removeEventListener('visibilitychange', this.visibilityHandler);
      this.visibilityHandler = null;
    }
  }

  async refreshUnread(): Promise<void> {
    if (!this.auth.isAuthenticated()) return;
    try {
      const resp = await firstValueFrom(
        this.http.get<Wrapped<{ unread_count: number }> | { unread_count: number }>(
          '/api/v1/notifications/unread-count',
          { withCredentials: true }
        )
      );
      const body = unwrap(resp, { unread_count: 0 });
      this.unreadCount.set(Math.max(0, body.unread_count ?? 0));
    } catch {
      // Un fallo de red no debe romper la barra: el contador conserva su valor.
    }
  }

  async loadRecent(): Promise<void> {
    if (!this.auth.isAuthenticated()) return;
    this.loadingRecent.set(true);
    try {
      const params = new HttpParams().set('page', 1).set('limit', RECENT_LIMIT);
      const resp = await firstValueFrom(
        this.http.get<Wrapped<NotificationItem[]> | NotificationItem[]>(
          '/api/v1/notifications',
          { params, withCredentials: true }
        )
      );
      this.recent.set(unwrap(resp, []));
      this.error.set(null);
    } catch {
      this.error.set('No se pudieron cargar los avisos.');
    } finally {
      this.loadingRecent.set(false);
    }
  }

  async loadPage(page = 1, filter: NotificationPageFilter = this.pageFilter): Promise<void> {
    if (!this.auth.isAuthenticated()) return;
    this.pageFilter = filter;
    this.loading.set(true);
    try {
      let params = new HttpParams()
        .set('page', Math.max(1, page))
        .set('limit', NOTIFICATIONS_PAGE_SIZE);
      if (filter.unreadOnly) {
        params = params.set('unread_only', 'true');
      }
      if (filter.severity && filter.severity !== 'all') {
        params = params.set('severity', filter.severity);
      }

      const resp = await firstValueFrom(
        this.http.get<Wrapped<NotificationItem[]> | NotificationItem[]>(
          '/api/v1/notifications',
          { params, withCredentials: true, observe: 'response' }
        )
      );
      const list = unwrap(resp.body, []);
      const totalHeader = Number(resp.headers.get('X-Total-Count') ?? '0');
      this.items.set(list);
      this.total.set(Number.isFinite(totalHeader) && totalHeader > 0 ? totalHeader : list.length);
      this.page.set(Math.max(1, page));
      this.error.set(null);
    } catch {
      this.error.set('No se pudieron cargar los avisos.');
    } finally {
      this.loading.set(false);
    }
  }

  async markRead(id: string): Promise<void> {
    const target = [...this.recent(), ...this.items()].find((n) => n.id === id);
    if (!target || target.is_read) return;

    try {
      await firstValueFrom(
        this.http.patch(`/api/v1/notifications/${encodeURIComponent(id)}/read`, {}, {
          withCredentials: true
        })
      );
      const now = new Date().toISOString();
      const mark = (list: NotificationItem[]) =>
        list.map((n) => (n.id === id ? { ...n, is_read: true, read_at: now } : n));
      this.recent.update(mark);
      this.items.update(mark);
      this.unreadCount.update((count) => Math.max(0, count - 1));
    } catch {
      this.error.set('No se pudo marcar el aviso como leído.');
    }
  }

  async markAllRead(): Promise<void> {
    if (this.unreadCount() === 0) return;
    try {
      await firstValueFrom(
        this.http.post('/api/v1/notifications/mark-all-read', {}, { withCredentials: true })
      );
      const now = new Date().toISOString();
      const mark = (list: NotificationItem[]) =>
        list.map((n) => (n.is_read ? n : { ...n, is_read: true, read_at: now }));
      this.recent.update(mark);
      this.items.update(mark);
      this.unreadCount.set(0);
    } catch {
      this.error.set('No se pudieron marcar los avisos como leídos.');
    }
  }
}
