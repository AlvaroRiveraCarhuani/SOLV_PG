import { describe, it, expect, beforeEach, vi } from 'vitest';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { signal } from '@angular/core';
import { of } from 'rxjs';
import { AdminConfigAdministradoresComponent } from './admin-config-administradores.component';
import {
  AdminConfigAdministradoresService,
  Administrator
} from '../admin-config-administradores.service';
import { AuthService } from '@core/services/auth.service';

describe('AdminConfigAdministradoresComponent', () => {
  let component: AdminConfigAdministradoresComponent;
  let fixture: ComponentFixture<AdminConfigAdministradoresComponent>;

  const mockCurrentUser = {
    id: 'admin-1',
    first_name: 'Admin',
    last_name: 'Principal',
    email: 'admin1@uab.edu.bo',
    role: 'admin' as const
  };

  const mockAdminsList: Administrator[] = [
    {
      id: 'admin-1',
      email: 'admin1@uab.edu.bo',
      first_name: 'Admin',
      last_name: 'Principal',
      role: 'admin',
      status: 'active',
      origin: 'manual',
      created_at: '2026-01-01T00:00:00Z',
      last_login_at: '2026-10-27T10:00:00Z'
    },
    {
      id: 'admin-2',
      email: 'admin2@uab.edu.bo',
      first_name: 'Carlos',
      last_name: 'Gomez',
      role: 'admin',
      status: 'active',
      origin: 'manual',
      created_at: '2026-02-01T00:00:00Z'
    }
  ];

  let adminServiceMock: {
    administrators: ReturnType<typeof signal<Administrator[]>>;
    isLoading: ReturnType<typeof signal<boolean>>;
    isSubmitting: ReturnType<typeof signal<boolean>>;
    error: ReturnType<typeof signal<string | null>>;
    loadAdministrators: ReturnType<typeof vi.fn>;
    createAdministrator: ReturnType<typeof vi.fn>;
    revokeAdministrator: ReturnType<typeof vi.fn>;
  };

  let authServiceMock: {
    currentUser: ReturnType<typeof signal<typeof mockCurrentUser | null>>;
  };

  beforeEach(async () => {
    adminServiceMock = {
      administrators: signal<Administrator[]>(mockAdminsList),
      isLoading: signal<boolean>(false),
      isSubmitting: signal<boolean>(false),
      error: signal<string | null>(null),
      loadAdministrators: vi.fn().mockReturnValue(of(mockAdminsList)),
      createAdministrator: vi.fn(),
      revokeAdministrator: vi.fn()
    };

    authServiceMock = {
      currentUser: signal(mockCurrentUser)
    };

    await TestBed.configureTestingModule({
      imports: [AdminConfigAdministradoresComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: AdminConfigAdministradoresService, useValue: adminServiceMock },
        { provide: AuthService, useValue: authServiceMock }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(AdminConfigAdministradoresComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('debe crear el componente y cargar la lista de administradores', () => {
    expect(component).toBeTruthy();
    expect(adminServiceMock.loadAdministrators).toHaveBeenCalled();
    expect(component.activeAdminsCount()).toBe(2);
  });

  it('identifica correctamente al usuario autenticado (isSelf)', () => {
    expect(component.isSelf(mockAdminsList[0])).toBe(true);
    expect(component.isSelf(mockAdminsList[1])).toBe(false);
  });

  it('impide revocar al usuario actual (invariante de no-auto-revocación)', () => {
    expect(component.canRevoke(mockAdminsList[0])).toBe(false);
    expect(component.getRevokeDisabledTooltip(mockAdminsList[0])).toContain('propios privilegios');
  });

  it('permite revocar a otro administrador cuando hay más de uno activo', () => {
    expect(component.canRevoke(mockAdminsList[1])).toBe(true);
    expect(component.getRevokeDisabledTooltip(mockAdminsList[1])).toBe('Revocar privilegios de administrador');
  });

  it('impide revocar a cualquier administrador si solo queda 1 activo (invariante de no-orfandad)', () => {
    adminServiceMock.administrators.set([mockAdminsList[1]]);
    authServiceMock.currentUser.set({
      id: 'other-user',
      first_name: 'Other',
      last_name: 'User',
      email: 'other@uab.edu.bo',
      role: 'admin'
    });

    expect(component.activeAdminsCount()).toBe(1);
    expect(component.canRevoke(mockAdminsList[1])).toBe(false);
    expect(component.getRevokeDisabledTooltip(mockAdminsList[1])).toContain('al menos un administrador activo');
  });

  it('valida campos del formulario al registrar administrador', () => {
    component.openCreateModal();
    expect(component.isCreateModalOpen()).toBe(true);

    // Campos vacíos
    component.submitCreate();
    expect(component.formError()).toBe('Nombre y apellido son obligatorios.');

    // Correo inválido
    component.formFirstName.set('Juan');
    component.formLastName.set('Perez');
    component.formEmail.set('no-email');
    component.submitCreate();
    expect(component.formError()).toBe('Ingresa un correo institucional válido.');

    expect(adminServiceMock.createAdministrator).not.toHaveBeenCalled();
  });

  it('envía datos válidos y cierra el modal al crear administrador', () => {
    component.openCreateModal();
    component.formFirstName.set('Elena');
    component.formLastName.set('Suarez');
    component.formEmail.set('elena.suarez@uab.edu.bo');

    const createdAdmin: Administrator = {
      id: 'admin-new',
      first_name: 'Elena',
      last_name: 'Suarez',
      email: 'elena.suarez@uab.edu.bo',
      role: 'admin',
      status: 'active',
      origin: 'manual',
      created_at: '2026-10-27T16:00:00Z'
    };

    adminServiceMock.createAdministrator.mockReturnValue(of(createdAdmin));

    component.submitCreate();

    expect(adminServiceMock.createAdministrator).toHaveBeenCalledWith({
      first_name: 'Elena',
      last_name: 'Suarez',
      email: 'elena.suarez@uab.edu.bo'
    });
    expect(component.isCreateModalOpen()).toBe(false);
    expect(component.toast()?.type).toBe('success');
  });

  it('abre modal de confirmación y ejecuta revocación', () => {
    component.confirmRevoke(mockAdminsList[1]);
    expect(component.adminToRevoke()?.id).toBe('admin-2');

    adminServiceMock.revokeAdministrator.mockReturnValue(of(void 0));
    component.executeRevoke();

    expect(adminServiceMock.revokeAdministrator).toHaveBeenCalledWith('admin-2');
    expect(component.adminToRevoke()).toBeNull();
    expect(component.toast()?.type).toBe('success');
  });
});
