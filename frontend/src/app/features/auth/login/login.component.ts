import { Component, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { TenantService } from '@core/services/tenant.service';
import { AuthService } from '@core/services/auth.service';

@Component({
  selector: 'login',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './login.component.html',
  styleUrl: './login.component.scss'
})
export class LoginComponent {
  private readonly tenantService = inject(TenantService);
  private readonly authService = inject(AuthService);

  // Signals reactivos expuestos a la vista
  protected readonly tenant = this.tenantService.config;
  protected readonly loading = this.tenantService.loading;
  protected readonly error = this.tenantService.error;
  protected readonly isAuthenticating = this.authService.isLoading;

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

  protected onLogoError(): void {
    this.logoFailed.set(true);
  }

  protected onLoginWithGoogle(): void {
    this.authService.loginWithGoogle();
  }
}
