import { Component, OnInit, inject, input, signal, computed, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router, RouterLink } from '@angular/router';
import { TenantService } from '@core/services/tenant.service';
import { AuthService } from '@core/services/auth.service';
import { ContextHelpService } from '@shared/help/context-help.service';
import {
  NotificationItem,
  NotificationService
} from '@core/services/notification.service';
import { UserTypographyModalComponent } from '@shared/components/user-typography-modal/user-typography-modal.component';
import { DismissibleDirective } from '@shared/directives/dismissible.directive';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { LucidePanelLeft, LucideBell, LucideLogOut, LucideUser, LucideType, LucideCircleHelp } from '@lucide/angular';

@Component({
  selector: 'topbar',
  standalone: true,
  imports: [
    CommonModule,
    RouterLink,
    LucidePanelLeft,
    LucideBell,
    LucideLogOut,
    LucideUser,
    LucideType,
    LucideCircleHelp,
    UserTypographyModalComponent,
    DismissibleDirective,
    DateTextPipe,
    MachineDataDirective
  ],
  template: `
    <header class="topbar">
      <div class="topbar-left">
        <button
          class="btn-icon"
          (click)="toggleSidebar.emit()"
          title="Alternar Menú Lateral"
          aria-label="Alternar Menú Lateral">
          <svg lucidePanelLeft class="icon"></svg>
        </button>

        <div class="brand">
          @if (tenantService.config()?.logo_url && !logoFailed()) {
            <img
              [src]="tenantService.config()?.logo_url"
              [alt]="tenantService.config()?.institution_name || 'Logo'"
              class="brand-logo"
              (error)="logoFailed.set(true)"
            />
          } @else {
            <div class="brand-shield">{{ tenantService.tenantInitials() }}</div>
          }
          <div class="brand-info">
            <span class="brand-name">{{ tenantService.config()?.institution_name || 'Plataforma SOLV' }}</span>
            <span class="brand-system">SOLV &bull; Laboratorios Virtuales</span>
          </div>
        </div>
      </div>

      <div class="topbar-right">
        @if (showHelpButton() && help.hasContent()) {
          <button
            type="button"
            class="btn-icon"
            (click)="openHelp($event)"
            aria-label="Abrir el centro de ayuda"
            [attr.aria-expanded]="help.isOpen()"
          >
            <svg lucideCircleHelp class="icon"></svg>
          </button>
        }

        <!-- Campana de avisos (ADR-034) -->
        <div class="notif-wrapper">
          <button #notifBtn class="btn-icon" (click)="toggleNotifMenu()" title="Avisos" aria-label="Avisos">
            <svg lucideBell class="icon"></svg>
            @if (notifService.unreadCount() > 0) {
              <span class="notif-badge">{{ unreadBadge() }}</span>
            }
          </button>

          @if (notifMenuOpen()) {
            <div
              class="notif-dropdown animate-dropdown"
              dismissible
              [dismissExclude]="notifBtn"
              (dismiss)="notifMenuOpen.set(false)"
            >
              <div class="notif-header">
                <span class="notif-heading">Avisos</span>
                @if (notifService.unreadCount() > 0) {
                  <button class="btn-mark-read" (click)="notifService.markAllRead()">
                    Marcar todas como leídas
                  </button>
                }
              </div>
              <div class="notif-list">
                @for (item of notifService.recent(); track item.id) {
                  <button
                    [class]="notificationClass(item)"
                    (click)="openNotification(item)"
                  >
                    <span class="notif-title-row">
                      <span class="notif-title">{{ item.title }}</span>
                      @if (!item.is_read) {
                        <span class="notif-unread-dot" aria-label="Sin leer"></span>
                      }
                    </span>
                    <span class="notif-desc">{{ item.message }}</span>
                    <span class="notif-meta">
                      <time [attr.datetime]="item.created_at">{{ item.created_at | dateText: 'compact-datetime' }}</time>
                      @if (item.occurrence_count > 1) {
                        <span machineData class="notif-repeat">&times;{{ item.occurrence_count }}</span>
                      }
                    </span>
                  </button>
                } @empty {
                  <div class="notif-empty">
                    @if (notifService.loadingRecent()) {
                      Cargando avisos…
                    } @else {
                      No hay avisos por ahora.
                    }
                  </div>
                }
              </div>
              <div class="notif-footer">
                <a class="notif-see-all" [routerLink]="notificationsRoute()" (click)="notifMenuOpen.set(false)">
                  Ver todos los avisos
                </a>
              </div>
            </div>
          }
        </div>

        <!-- Perfil del Usuario -->
        <div class="profile-wrapper">
          <div #profileBtn class="profile-trigger" (click)="toggleProfileMenu()">
            <div class="profile-avatar">
              @if (authService.currentUser()?.avatar_url) {
                <img [src]="authService.currentUser()?.avatar_url" alt="Avatar" />
              } @else {
                <svg lucideUser class="avatar-fallback"></svg>
              }
            </div>
            <div class="profile-details">
              <span class="profile-name">
                {{ formatTitleCase(authService.currentUser()?.first_name) || 'Usuario' }}
              </span>
              <span class="profile-role">{{ roleLabel() }}</span>
            </div>
          </div>

          @if (profileMenuOpen()) {
            <div
              class="profile-dropdown animate-dropdown"
              dismissible
              [dismissExclude]="profileBtn"
              (dismiss)="profileMenuOpen.set(false)"
            >
              <div class="dropdown-header">
                <span class="user-fullname">
                  {{ formatTitleCase((authService.currentUser()?.first_name || '') + ' ' + (authService.currentUser()?.last_name || '')) }}
                </span>
                <span class="user-email">{{ authService.currentUser()?.email }}</span>
              </div>
              <div class="dropdown-divider"></div>
              <button class="dropdown-item" (click)="openTypographyModal()">
                <svg lucideType class="dropdown-icon"></svg>
                <span>Tipografía</span>
              </button>
              <div class="dropdown-divider"></div>
              <button class="dropdown-item logout" (click)="logout()">
                <svg lucideLogOut class="dropdown-icon"></svg>
                <span>Cerrar Sesión</span>
              </button>
            </div>
          }
        </div>
      </div>

      @if (typographyModalOpen()) {
        <user-typography-modal (close)="typographyModalOpen.set(false)" />
      }
    </header>
  `,
  styleUrl: './topbar.component.scss',
})
export class TopbarComponent implements OnInit {
  tenantService = inject(TenantService);
  authService = inject(AuthService);
  notifService = inject(NotificationService);
  private router = inject(Router);

  toggleSidebar = output<void>();

  /** El centro de ayuda sólo se ofrece en el área administrativa. */
  showHelpButton = input<boolean>(false);

  readonly help = inject(ContextHelpService);

  logoFailed = signal<boolean>(false);
  profileMenuOpen = signal<boolean>(false);
  notifMenuOpen = signal<boolean>(false);
  typographyModalOpen = signal<boolean>(false);

  unreadBadge = computed(() => {
    const count = this.notifService.unreadCount();
    return count > 99 ? '99+' : String(count);
  });

  ngOnInit(): void {
    this.notifService.startPolling();
    void this.notifService.refreshUnread();
  }

  roleLabel = () => {
    const role = this.authService.currentUser()?.role;
    if (role === 'admin') return 'Administrador';
    if (role === 'teacher') return 'Docente';
    return 'Estudiante';
  };

  notificationClass(item: NotificationItem): string {
    return `notif-item severity-${item.severity}${item.is_read ? ' read' : ''}`;
  }

  notificationsRoute(): string {
    const role = this.authService.currentUser()?.role;
    if (role === 'admin') return '/admin/notifications';
    if (role === 'teacher') return '/teacher/notifications';
    return '/student/notifications';
  }

  openHelp(event: MouseEvent): void {
    this.help.toggle(event.currentTarget as HTMLElement);
  }

  toggleProfileMenu(): void {
    this.profileMenuOpen.update(v => !v);
    if (this.profileMenuOpen()) {
      this.notifMenuOpen.set(false);
    }
  }

  toggleNotifMenu(): void {
    const nextState = !this.notifMenuOpen();
    this.notifMenuOpen.set(nextState);
    if (nextState) {
      this.profileMenuOpen.set(false);
      void this.notifService.loadRecent();
      void this.notifService.refreshUnread();
    }
  }

  openNotification(item: NotificationItem): void {
    this.notifMenuOpen.set(false);
    void this.notifService.markRead(item.id);
    if (item.link) {
      void this.router.navigateByUrl(item.link);
    }
  }

  closeAllMenus(): void {
    this.notifMenuOpen.set(false);
    this.profileMenuOpen.set(false);
  }

  formatTitleCase(text?: string | null): string {
    if (!text) return '';
    return text
      .toLowerCase()
      .split(' ')
      .filter(Boolean)
      .map(word => word.charAt(0).toUpperCase() + word.slice(1))
      .join(' ');
  }

  openTypographyModal(): void {
    this.profileMenuOpen.set(false);
    this.typographyModalOpen.set(true);
  }

  logout(): void {
    this.authService.logout();
    this.router.navigate(['/login']);
  }
}
