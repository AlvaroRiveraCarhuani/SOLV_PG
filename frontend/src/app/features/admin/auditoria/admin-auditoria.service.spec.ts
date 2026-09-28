import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { AdminAuditoriaService } from './admin-auditoria.service';

describe('AdminAuditoriaService', () => {
  let service: AdminAuditoriaService;
  let httpTesting: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting()]
    });
    service = TestBed.inject(AdminAuditoriaService);
    httpTesting = TestBed.inject(HttpTestingController);
  });

  afterEach(() => httpTesting.verify());

  it('envía búsqueda y verbo como filtros del endpoint paginado', () => {
    service.listAuditLogs(2, 20, { search: 'resource-id-42', action: 'DELETE' }).subscribe();

    const request = httpTesting.expectOne((req) => req.url === '/api/v1/admin/audit-logs');
    expect(request.request.method).toBe('GET');
    expect(request.request.params.get('search')).toBe('resource-id-42');
    expect(request.request.params.get('action')).toBe('DELETE');
    expect(request.request.params.get('limit')).toBe('20');
    expect(request.request.params.get('offset')).toBe('20');
    request.flush({ tenant_id: 't', limit: 20, offset: 20, total: 0, data: [] });
  });
});