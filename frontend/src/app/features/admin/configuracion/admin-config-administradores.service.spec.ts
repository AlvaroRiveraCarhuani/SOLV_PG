import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { AdminConfigAdministradoresService, Administrator } from './admin-config-administradores.service';

describe('AdminConfigAdministradoresService', () => {
  let service: AdminConfigAdministradoresService;
  let httpTesting: HttpTestingController;

  const mockAdmins: Administrator[] = [
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
      first_name: 'Admin',
      last_name: 'Secundario',
      role: 'admin',
      status: 'active',
      origin: 'manual',
      created_at: '2026-02-01T00:00:00Z'
    }
  ];

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting()]
    });
    service = TestBed.inject(AdminConfigAdministradoresService);
    httpTesting = TestBed.inject(HttpTestingController);
  });

  afterEach(() => httpTesting.verify());

  it('loadAdministrators consulta el endpoint GET y actualiza el signal', () => {
    service.loadAdministrators().subscribe((admins) => {
      expect(admins.length).toBe(2);
    });

    const req = httpTesting.expectOne('/api/v1/admin/administrators');
    expect(req.request.method).toBe('GET');
    req.flush({ data: mockAdmins, message: 'OK' });

    expect(service.administrators().length).toBe(2);
    expect(service.isLoading()).toBe(false);
    expect(service.error()).toBeNull();
  });

  it('createAdministrator envía payload POST y añade el nuevo admin al signal', () => {
    const payload = {
      first_name: 'Nuevo',
      last_name: 'Operador',
      email: 'nuevo@uab.edu.bo'
    };

    const createdAdmin: Administrator = {
      id: 'admin-3',
      ...payload,
      role: 'admin',
      status: 'active',
      origin: 'manual',
      created_at: '2026-10-27T12:00:00Z'
    };

    service.administrators.set(mockAdmins);

    service.createAdministrator(payload).subscribe((res) => {
      expect(res.id).toBe('admin-3');
    });

    const req = httpTesting.expectOne('/api/v1/admin/administrators');
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual(payload);
    req.flush({ data: createdAdmin, message: 'Created' });

    expect(service.administrators().length).toBe(3);
    expect(service.isSubmitting()).toBe(false);
  });

  it('revokeAdministrator envía POST revoke y remueve al admin del signal', () => {
    service.administrators.set(mockAdmins);

    service.revokeAdministrator('admin-2').subscribe();

    const req = httpTesting.expectOne('/api/v1/admin/administrators/admin-2/revoke');
    expect(req.request.method).toBe('POST');
    req.flush({ data: { id: 'admin-2', status: 'revoked' }, message: 'Revoked' });

    expect(service.administrators().length).toBe(1);
    expect(service.administrators()[0].id).toBe('admin-1');
    expect(service.isSubmitting()).toBe(false);
  });

  it('maneja errores de API y propaga el mensaje al signal de error', () => {
    service.loadAdministrators().subscribe({
      next: () => expect.unreachable(),
      error: (err) => {
        expect(err.status).toBe(500);
      }
    });

    const req = httpTesting.expectOne('/api/v1/admin/administrators');
    req.flush({ message: 'Error interno de base de datos' }, { status: 500, statusText: 'Server Error' });

    expect(service.error()).toBe('Error interno de base de datos');
    expect(service.isLoading()).toBe(false);
  });
});
