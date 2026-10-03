import { Component, inject, signal, computed, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { TenantService } from '@core/services/tenant.service';
import { AuthService } from '@core/services/auth.service';
import { UserTypographyModalComponent } from '@shared/components/user-typography-modal/user-typography-modal.component';
import { DismissibleDirective } from '@shared/directives/dismissible.directive';
import { LucidePanelLeft, LucideBell, LucideLogOut, LucideUser, LucideType } from '@lucide/angular';

@Component({
  selector: 'topbar',
  standalone: true,
  imports: [
    CommonModule, 
    LucidePanelLeft, 
    LucideBell, 
    LucideLogOut, 
    LucideUser, 
    LucideType,
    UserTypographyModalComponent,
    DismissibleDirective
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
        <!-- Campana de Notificaciones Proactivas -->
        <div class="notif-wrapper">
          <button #notifBtn class="btn-icon" (click)="toggleNotifMenu()" title="Notificaciones" aria-label="Notificaciones">
            <svg lucideBell class="icon"></svg>
            @if (unreadCount() > 0) {
              <span class="notif-badge">{{ unreadCount() }}</span>
            }
          </button>

          @if (notifMenuOpen()) {
            <div 
              class="notif-dropdown"
              dismissible
              [dismissExclude]="notifBtn"
              (dismiss)="notifMenuOpen.set(false)"
            >
              <div class="notif-header">
                <span class="notif-heading">Avisos del Sistema</span>
                @if (notifications().length > 0) {
                  <button class="btn-clear-notifs" (click)="clearNotifs()">Limpiar historial</button>
                }
              </div>
              <div class="notif-list">
                @for (item of notifications(); track item.id) {
                  <div class="notif-item" [class.read]="item.read">
                    <div class="notif-title-row">
                      <span class="notif-title">{{ item.title }}</span>
                      <span class="notif-time">{{ item.time }}</span>
                    </div>
                    <span class="notif-desc">{{ item.desc }}</span>
                  </div>
                } @empty {
                  <div class="notif-empty">No hay avisos pendientes en el clúster.</div>
                }
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
              class="profile-dropdown"
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
export class TopbarComponent {
  tenantService = inject(TenantService);
  authService = inject(AuthService);
  private router = inject(Router);

  toggleSidebar = output<void>();

  logoFailed = signal<boolean>(false);
  profileMenuOpen = signal<boolean>(false);
  notifMenuOpen = signal<boolean>(false);
  typographyModalOpen = signal<boolean>(false);
  private loadInitialNotifs() {
    if (typeof window !== 'undefined' && localStorage.getItem('solv_notifs_cleared') === 'true') {
      return [];
    }
    const isRead1 = typeof window !== 'undefined' && localStorage.getItem('solv_notif_1_read') === 'true';
    const isRead2 = typeof window !== 'undefined' && localStorage.getItem('solv_notif_2_read') === 'true';
    return [
      {
        id: 'notif-1',
        title: 'Clúster Docker conectado',
        desc: 'Docker Engine y telemetría de host gopsutil operando correctamente.',
        time: 'Hoy',
        read: isRead1
      },
      {
        id: 'notif-2',
        title: 'Seguridad y Políticas',
        desc: 'Monitoreo activo de límites de memoria (OOM killer habilitado).',
        time: 'Hoy',
        read: isRead2
      }
    ];
  }

  notifications = signal(this.loadInitialNotifs());
  unreadCount = computed(() => this.notifications().filter(n => !n.read).length);

  roleLabel = () => {
    const role = this.authService.currentUser()?.role;
    if (role === 'admin') return 'Administrador';
    if (role === 'teacher') return 'Docente';
    return 'Estudiante';
  };

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
      this.markAllAsRead();
    }
  }

  markAllAsRead(): void {
    this.notifications.update(list => list.map(n => ({ ...n, read: true })));
    try {
      localStorage.setItem('solv_notif_1_read', 'true');
      localStorage.setItem('solv_notif_2_read', 'true');
    } catch (_) {}
  }

  closeAllMenus(): void {
    this.notifMenuOpen.set(false);
    this.profileMenuOpen.set(false);
  }

  clearNotifs(): void {
    this.notifications.set([]);
    try {
      localStorage.setItem('solv_notifs_cleared', 'true');
      localStorage.setItem('solv_notif_1_read', 'true');
      localStorage.setItem('solv_notif_2_read', 'true');
    } catch (_) {}
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
