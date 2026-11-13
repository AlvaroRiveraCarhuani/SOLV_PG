import { Component, ChangeDetectionStrategy, OnInit, OnDestroy, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { DateTextPipe } from '@shared/pipes/date-text.pipe';
import { ModalShellComponent } from '@shared/components/modal-shell/modal-shell.component';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import { ContextHelpService } from '@shared/help/context-help.service';
import { InteractiveTutorialService } from '@shared/help/interactive-tutorial.service';
import { ProductTourService } from '@shared/help/product-tour.service';
import { configTabTourGuide } from '../config-help.definitions';
import { buildAdministradoresTour, buildAdministradoresTutorial } from './administradores-help.definitions';
import { AuthService } from '@core/services/auth.service';
import {
  AdminConfigAdministradoresService,
  Administrator,
  CreateAdministratorPayload
} from '../admin-config-administradores.service';
import {
  LucideShield,
  LucideShieldCheck,
  LucideUserPlus,
  LucideTrash2,
  LucideAlertCircle,
  LucideCheckCircle2,
  LucideAlertTriangle
} from '@lucide/angular';

const ADMINISTRADORES_HELP_KEY = 'config-administrators';

@Component({
  selector: 'admin-config-administradores',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    DateTextPipe,
    ModalShellComponent,
    FormFieldComponent,
    LucideShield,
    LucideShieldCheck,
    LucideUserPlus,
    LucideTrash2,
    LucideAlertCircle,
    LucideCheckCircle2,
    LucideAlertTriangle
  ],
  templateUrl: './admin-config-administradores.component.html',
  styleUrl: './admin-config-administradores.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class AdminConfigAdministradoresComponent implements OnInit, OnDestroy {
  private readonly adminService = inject(AdminConfigAdministradoresService);
  private readonly authService = inject(AuthService);
  private readonly help = inject(ContextHelpService);
  private readonly tutorial = inject(InteractiveTutorialService);
  private readonly productTour = inject(ProductTourService);

  /** Se registró un administrador en esta visita; lo lee el tutorial. */
  readonly hasCreatedAdmin = signal(false);

  readonly administrators = this.adminService.administrators;
  readonly isLoading = this.adminService.isLoading;
  readonly isSubmitting = this.adminService.isSubmitting;
  readonly serviceError = this.adminService.error;

  readonly isCreateModalOpen = signal<boolean>(false);
  readonly adminToRevoke = signal<Administrator | null>(null);

  readonly formFirstName = signal<string>('');
  readonly formLastName = signal<string>('');
  readonly formEmail = signal<string>('');
  readonly formError = signal<string | null>(null);

  readonly toast = signal<{ type: 'success' | 'error'; message: string } | null>(null);

  readonly activeAdminsCount = computed<number>(
    () => this.administrators().filter((a) => a.status === 'active').length
  );

  ngOnInit(): void {
    this.help.registerScreen(ADMINISTRADORES_HELP_KEY);
    this.help.registerGuides(ADMINISTRADORES_HELP_KEY, {
      tour: configTabTourGuide(() => void this.productTour.start(buildAdministradoresTour())),
      tutorial: {
        label: 'Tutorial guiado',
        hint: 'Registra un operador y comprueba el correo antes de confirmar.',
        run: () => this.startAdministradoresTutorial()
      }
    });
    this.adminService.loadAdministrators().subscribe({
      error: () => {
        // Error capturado en el signal del servicio
      }
    });
  }

  ngOnDestroy(): void {
    this.help.unregisterGuides(ADMINISTRADORES_HELP_KEY);
    this.help.unregisterScreen(ADMINISTRADORES_HELP_KEY);
  }

  /** Tutorial del equipo: alta completa, sin ejecutar la revocación. */
  startAdministradoresTutorial(): void {
    this.tutorial.start(
      buildAdministradoresTutorial({
        isCreateModalOpen: () => this.isCreateModalOpen(),
        email: () => this.formEmail(),
        formError: () => this.formError(),
        hasCreated: () => this.hasCreatedAdmin()
      })
    );
  }

  isSelf(admin: Administrator): boolean {
    const current = this.authService.currentUser();
    if (!current) {
      return false;
    }
    return admin.id === current.id || admin.email.toLowerCase() === current.email.toLowerCase();
  }

  canRevoke(admin: Administrator): boolean {
    if (this.isSelf(admin)) {
      return false;
    }
    return this.activeAdminsCount() > 1;
  }

  getRevokeDisabledTooltip(admin: Administrator): string {
    if (this.isSelf(admin)) {
      return 'No podés revocar tus propios privilegios de administrador';
    }
    if (this.activeAdminsCount() <= 1) {
      return 'Debe existir al menos un administrador activo en la institución';
    }
    return 'Revocar privilegios de administrador';
  }

  openCreateModal(): void {
    this.formFirstName.set('');
    this.formLastName.set('');
    this.formEmail.set('');
    this.formError.set(null);
    this.isCreateModalOpen.set(true);
  }

  closeCreateModal(): void {
    if (!this.isSubmitting()) {
      this.isCreateModalOpen.set(false);
      this.formError.set(null);
    }
  }

  submitCreate(): void {
    const first = this.formFirstName().trim();
    const last = this.formLastName().trim();
    const email = this.formEmail().trim().toLowerCase();

    if (!first || !last) {
      this.formError.set('Nombre y apellido son obligatorios.');
      return;
    }

    if (!email || !email.includes('@') || !email.includes('.')) {
      this.formError.set('Ingresa un correo institucional válido.');
      return;
    }

    const payload: CreateAdministratorPayload = {
      first_name: first,
      last_name: last,
      email
    };

    this.adminService.createAdministrator(payload).subscribe({
      next: (created) => {
        this.isCreateModalOpen.set(false);
        this.hasCreatedAdmin.set(true);
        this.showToast('success', `Administrador ${created.first_name} ${created.last_name} registrado exitosamente.`);
      },
      error: (err) => {
        const msg = err?.error?.message || err?.error?.error || 'No se pudo registrar el administrador.';
        this.formError.set(msg);
      }
    });
  }

  confirmRevoke(admin: Administrator): void {
    if (this.canRevoke(admin)) {
      this.adminToRevoke.set(admin);
    }
  }

  cancelRevoke(): void {
    if (!this.isSubmitting()) {
      this.adminToRevoke.set(null);
    }
  }

  executeRevoke(): void {
    const target = this.adminToRevoke();
    if (!target) {
      return;
    }

    this.adminService.revokeAdministrator(target.id).subscribe({
      next: () => {
        this.adminToRevoke.set(null);
        this.showToast('success', `Privilegios de ${target.first_name} ${target.last_name} revocados exitosamente.`);
      },
      error: (err) => {
        const msg = err?.error?.message || err?.error?.error || 'No se pudo revocar al administrador.';
        this.showToast('error', msg);
        this.adminToRevoke.set(null);
      }
    });
  }

  private showToast(type: 'success' | 'error', message: string): void {
    this.toast.set({ type, message });
    setTimeout(() => {
      this.toast.set(null);
    }, 4000);
  }
}
