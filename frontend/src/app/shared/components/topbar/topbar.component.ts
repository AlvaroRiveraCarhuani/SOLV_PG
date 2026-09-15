import { Component, inject, signal, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { TenantService } from '@core/services/tenant.service';
import { AuthService } from '@core/services/auth.service';
import { LucidePanelLeft, LucideBell, LucideLogOut, LucideUser } from '@lucide/angular';

@Component({
  selector: 'solv-topbar',
  standalone: true,
  imports: [CommonModule, LucidePanelLeft, LucideBell, LucideLogOut, LucideUser],
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
          <button class="btn-icon" (click)="toggleNotifMenu()" title="Notificaciones" aria-label="Notificaciones">
            <svg lucideBell class="icon"></svg>
            @if (unreadCount() > 0) {
              <span class="notif-badge">{{ unreadCount() }}</span>
            }
          </button>

          @if (notifMenuOpen()) {
            <div class="notif-dropdown">
              <div class="notif-header">
                <span class="notif-heading">Avisos del Sistema</span>
                @if (unreadCount() > 0) {
                  <button class="btn-clear-notifs" (click)="clearNotifs()">Marcar leídas</button>
                }
              </div>
              <div class="notif-list">
                @if (unreadCount() > 0) {
                  <div class="notif-item">
                    <span class="notif-title">Clúster Docker conectado</span>
                    <span class="notif-desc">Docker Engine y telemetría de host gopsutil operando correctamente.</span>
                  </div>
                  <div class="notif-item">
                    <span class="notif-title">Seguridad y Políticas</span>
                    <span class="notif-desc">Monitoreo activo de límites de memoria (OOM killer habilitado).</span>
                  </div>
                } @else {
                  <div class="notif-empty">No hay avisos pendientes en el clúster.</div>
                }
              </div>
            </div>
          }
        </div>

        <!-- Perfil del Usuario -->
        <div class="profile-wrapper">
          <div class="profile-trigger" (click)="toggleProfileMenu()">
            <div class="profile-avatar">
              @if (authService.currentUser()?.avatar_url) {
                <img [src]="authService.currentUser()?.avatar_url" alt="Avatar" />
              } @else {
                <svg lucideUser class="avatar-fallback"></svg>
              }
            </div>
            <div class="profile-details">
              <span class="profile-name">
                {{ authService.currentUser()?.first_name || 'Usuario' }}
              </span>
              <span class="profile-role">{{ roleLabel() }}</span>
            </div>
          </div>

          @if (profileMenuOpen()) {
            <div class="profile-dropdown">
              <div class="dropdown-header">
                <span class="user-fullname">
                  {{ authService.currentUser()?.first_name }} {{ authService.currentUser()?.last_name }}
                </span>
                <span class="user-email">{{ authService.currentUser()?.email }}</span>
              </div>
              <div class="dropdown-divider"></div>
              <button class="dropdown-item logout" (click)="logout()">
                <svg lucideLogOut class="dropdown-icon"></svg>
                <span>Cerrar Sesión</span>
              </button>
            </div>
          }
        </div>
      </div>
    </header>
  `,
  styles: [`
    .topbar {
      height: 60px;
      background-color: var(--bg-surface, #FFFFFF);
      border-bottom: 1px solid var(--border-subtle, #E2E8F0);
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 0 var(--space-4, 16px);
      position: sticky;
      top: 0;
      z-index: 50;
      user-select: none;
    }

    .topbar-left, .topbar-right {
      display: flex;
      align-items: center;
      gap: var(--space-3, 12px);
    }

    .btn-icon {
      background: transparent;
      border: none;
      width: 36px;
      height: 36px;
      border-radius: var(--radius-md, 6px);
      display: flex;
      align-items: center;
      justify-content: center;
      cursor: pointer;
      color: var(--text-secondary, #64748B);
      transition: all var(--transition-fast, 150ms ease);
      position: relative;

      &:hover {
        background-color: var(--bg-canvas, #F6F7F9);
        color: var(--text-primary, #0F172A);
      }

      .icon {
        width: 20px;
        height: 20px;
      }
    }

    .brand {
      display: flex;
      align-items: center;
      gap: var(--space-2-5, 10px);
      margin-left: var(--space-1, 4px);
    }

    .brand-logo {
      height: 32px;
      max-width: 120px;
      object-fit: contain;
    }

    .brand-shield {
      width: 32px;
      height: 32px;
      background-color: var(--tenant-primary, #2563EB);
      color: var(--tenant-primary-text, #FFFFFF);
      border-radius: var(--radius-md, 6px);
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: var(--font-size-xs, 12px);
      font-weight: 700;
      letter-spacing: 0.5px;
    }

    .brand-info {
      display: flex;
      flex-direction: column;
    }

    .brand-name {
      font-size: var(--font-size-sm, 14px);
      font-weight: 600;
      color: var(--text-primary, #0F172A);
      line-height: 1.2;
    }

    .brand-system {
      font-size: 11px;
      color: var(--text-muted, #94A3B8);
      font-weight: 400;
    }

    .notif-wrapper {
      position: relative;
    }

    .notif-badge {
      position: absolute;
      top: 4px;
      right: 4px;
      background-color: var(--verdict-wa, #DC2626);
      color: #FFFFFF;
      font-size: 10px;
      font-weight: 700;
      min-width: 16px;
      height: 16px;
      border-radius: var(--radius-full, 9999px);
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 0 4px;
      border: 2px solid var(--bg-surface, #FFFFFF);
    }

    .notif-dropdown {
      position: absolute;
      top: calc(100% + 8px);
      right: 0;
      width: 280px;
      background-color: var(--bg-surface, #FFFFFF);
      border: 1px solid var(--border-subtle, #E2E8F0);
      border-radius: var(--radius-lg, 8px);
      box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05);
      z-index: 100;
      padding: var(--space-3, 12px);
      display: flex;
      flex-direction: column;
      gap: var(--space-2, 8px);
      animation: dropdownFadeIn 150ms ease;
    }

    .notif-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      border-bottom: 1px solid var(--border-subtle, #E2E8F0);
      padding-bottom: var(--space-2, 8px);

      .notif-heading {
        font-size: var(--font-size-xs, 12px);
        font-weight: 700;
        color: var(--text-primary, #0F172A);
      }

      .btn-clear-notifs {
        background: transparent;
        border: none;
        color: var(--tenant-primary, #2563EB);
        font-size: 11px;
        cursor: pointer;
        padding: 0;

        &:hover {
          text-decoration: underline;
        }
      }
    }

    .notif-list {
      display: flex;
      flex-direction: column;
      gap: var(--space-2, 8px);
      max-height: 240px;
      overflow-y: auto;
    }

    .notif-item {
      display: flex;
      flex-direction: column;
      gap: 2px;
      padding: var(--space-2, 8px);
      background-color: var(--bg-canvas, #F8FAFC);
      border-radius: var(--radius-md, 6px);
      border-left: 3px solid var(--tenant-primary, #2563EB);

      .notif-title {
        font-size: 11px;
        font-weight: 600;
        color: var(--text-primary, #0F172A);
      }

      .notif-desc {
        font-size: 11px;
        color: var(--text-secondary, #64748B);
        line-height: 1.3;
      }
    }

    .notif-empty {
      font-size: 12px;
      color: var(--text-muted, #94A3B8);
      text-align: center;
      padding: var(--space-4, 16px) 0;
    }

    .profile-wrapper {
      position: relative;
    }

    .profile-trigger {
      display: flex;
      align-items: center;
      gap: var(--space-2, 8px);
      padding: var(--space-1, 4px) var(--space-2, 8px);
      border-radius: var(--radius-md, 6px);
      cursor: pointer;
      transition: background-color var(--transition-fast, 150ms ease);

      &:hover {
        background-color: var(--bg-canvas, #F6F7F9);
      }
    }

    .profile-avatar {
      width: 32px;
      height: 32px;
      border-radius: 50%;
      overflow: hidden;
      background-color: var(--border-subtle, #E2E8F0);
      display: flex;
      align-items: center;
      justify-content: center;

      img {
        width: 100%;
        height: 100%;
        object-fit: cover;
      }

      .avatar-fallback {
        width: 18px;
        height: 18px;
        color: var(--text-secondary, #64748B);
      }
    }

    .profile-details {
      display: flex;
      flex-direction: column;
      text-align: left;

      @media (max-width: 640px) {
        display: none;
      }
    }

    .profile-name {
      font-size: var(--font-size-xs, 12px);
      font-weight: 600;
      color: var(--text-primary, #0F172A);
      line-height: 1.2;
    }

    .profile-role {
      font-size: 10px;
      color: var(--text-muted, #94A3B8);
      font-weight: 500;
    }

    .profile-dropdown {
      position: absolute;
      top: calc(100% + 8px);
      right: 0;
      width: 220px;
      background-color: var(--bg-surface, #FFFFFF);
      border: 1px solid var(--border-subtle, #E2E8F0);
      border-radius: var(--radius-lg, 8px);
      box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.05), 0 2px 4px -1px rgba(0, 0, 0, 0.03);
      padding: var(--space-2, 8px);
      z-index: 100;
      animation: dropdownFadeIn 150ms ease;
    }

    @keyframes dropdownFadeIn {
      from {
        opacity: 0;
        transform: translateY(-4px);
      }
      to {
        opacity: 1;
        transform: translateY(0);
      }
    }

    .dropdown-header {
      padding: var(--space-2, 8px);
      display: flex;
      flex-direction: column;
      gap: 2px;
    }

    .user-fullname {
      font-size: var(--font-size-xs, 12px);
      font-weight: 600;
      color: var(--text-primary, #0F172A);
    }

    .user-email {
      font-size: 11px;
      color: var(--text-muted, #94A3B8);
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .dropdown-divider {
      height: 1px;
      background-color: var(--border-subtle, #E2E8F0);
      margin: var(--space-1, 4px) 0;
    }

    .dropdown-item {
      width: 100%;
      padding: var(--space-2, 8px);
      display: flex;
      align-items: center;
      gap: var(--space-2, 8px);
      background: transparent;
      border: none;
      border-radius: var(--radius-md, 6px);
      font-size: var(--font-size-xs, 12px);
      color: var(--text-secondary, #64748B);
      cursor: pointer;
      transition: all var(--transition-fast, 150ms ease);

      &:hover {
        background-color: var(--bg-canvas, #F6F7F9);
        color: var(--text-primary, #0F172A);
      }

      &.logout:hover {
        background-color: var(--state-failed-bg, #FEF2F2);
        color: var(--state-failed-text, #B91C1C);
      }

      .dropdown-icon {
        width: 16px;
        height: 16px;
      }
    }
  `]
})
export class TopbarComponent {
  tenantService = inject(TenantService);
  authService = inject(AuthService);
  private router = inject(Router);

  toggleSidebar = output<void>();

  logoFailed = signal<boolean>(false);
  profileMenuOpen = signal<boolean>(false);
  notifMenuOpen = signal<boolean>(false);
  unreadCount = signal<number>(2);

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
    this.notifMenuOpen.update(v => !v);
    if (this.notifMenuOpen()) {
      this.profileMenuOpen.set(false);
    }
  }

  clearNotifs(): void {
    this.unreadCount.set(0);
  }

  logout(): void {
    this.authService.logout();
    this.router.navigate(['/login']);
  }
}
