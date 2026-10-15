import { Component, inject, signal, computed, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { TenantService } from '@core/services/tenant.service';
import { AuthService } from '@core/services/auth.service';

@Component({
  selector: 'login',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './login.component.html',
  styleUrl: './login.component.scss'
})
export class LoginComponent implements OnInit {
  private readonly tenantService = inject(TenantService);
  private readonly authService = inject(AuthService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);

  // Signals reactivos expuestos a la vista
  protected readonly tenant = this.tenantService.config;
  protected readonly loading = this.tenantService.loading;
  protected readonly error = this.tenantService.error;
  protected readonly isAuthenticating = this.authService.isLoading;
  protected readonly inviteToken = signal<string | null>(null);

  // Signal para gestionar fallback si el logo falla al cargar
  protected readonly logoFailed = signal<boolean>(false);

  // Iniciales institucionales para el escudo de respaldo
  protected readonly institutionInitials = computed(() => {
    const name = this.tenant()?.institution_name;
    if (!name) return 'SO';
    return name
      .split(' ')
      .filter((word) => word.length > 2 && !['de', 'del', 'la', 'los'].includes(word.toLowerCase()))
      .slice(0, 3)
      .map((w) => w[0].toUpperCase())
      .join('');
  });

  ngOnInit(): void {
    const inviteFromQuery = this.route.snapshot.queryParams['invite'];
    if (inviteFromQuery) {
      if (typeof window !== 'undefined') {
        sessionStorage.setItem('solv_invite_token', inviteFromQuery);
      }
      this.inviteToken.set(inviteFromQuery);
    } else if (typeof window !== 'undefined') {
      const stored = sessionStorage.getItem('solv_invite_token');
      if (stored) {
        this.inviteToken.set(stored);
      }
    }

    // Si el usuario ya cuenta con sesión activa en el navegador
    this.authService.resolveCurrentUser().subscribe((user) => {
      if (user) {
        this.redirectAfterLogin(user.role);
      }
    });
  }

  private redirectAfterLogin(role: string): void {
    if (role === 'teacher') {
      this.router.navigate(['/teacher']);
    } else if (role === 'admin') {
      this.router.navigate(['/admin/dashboard']);
    } else {
      this.router.navigate(['/student']);
    }
  }

  protected onLogoError(): void {
    this.logoFailed.set(true);
  }

  protected onLoginWithGoogle(): void {
    this.authService.loginWithGoogle();
  }
}
