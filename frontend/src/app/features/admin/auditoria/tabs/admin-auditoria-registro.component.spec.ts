import '@angular/compiler';
import '@angular/localize/init';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { of, throwError } from 'rxjs';
import { signal } from '@angular/core';
import { AdminAuditoriaRegistroComponent } from './admin-auditoria-registro.component';
import { AdminAuditoriaService } from '../admin-auditoria.service';
import { AuditLog } from '@core/models/audit-log.model';

function makeLog(overrides: Partial<AuditLog>): AuditLog {
  return {
    id: overrides['id'] ?? crypto.randomUUID(),
    tenant_id: '00000000-0000-0000-0000-000000000001',
    actor_id: overrides['actor_id'] ?? '11111111-1111-1111-1111-111111111111',
    actor_email: overrides['actor_email'] ?? 'docente@uab.edu.bo',
    action: overrides['action'] ?? 'POST /api/v1/subjects',
    resource_type: overrides['resource_type'] ?? 'subjects',
    resource_id: null,
    status_code: overrides['status_code'] ?? 201,
    metadata: overrides['metadata'] ?? null,
    ip_address: '',
    user_agent: '',
    created_at: overrides['created_at'] ?? new Date().toISOString()
  };
}

describe('AdminAuditoriaRegistroComponent', () => {
  let fixture: ComponentFixture<AdminAuditoriaRegistroComponent>;
  let component: AdminAuditoriaRegistroComponent;
  const listSpy = vi.fn();
  const timelineSpy = vi.fn();

  beforeEach(async () => {
    listSpy.mockReset();
    timelineSpy.mockReset();

    const serviceStub = {
      isExecuting: signal(false),
      listAuditLogs: listSpy,
      getActorTimeline: timelineSpy
    };

    await TestBed.configureTestingModule({
      imports: [AdminAuditoriaRegistroComponent],
      providers: [{ provide: AdminAuditoriaService, useValue: serviceStub }]
    }).compileComponents();

    fixture = TestBed.createComponent(AdminAuditoriaRegistroComponent);
    component = fixture.componentInstance;
  });

  it('carga y enriquece la tabla con el enriquecimiento semántico del wireframe', () => {
    listSpy.mockReturnValue(of({
      tenant_id: 't', limit: 20, offset: 0,
      data: [
        makeLog({ action: 'POST /api/v1/subjects', status_code: 201 }),
        makeLog({ action: 'PUT /api/v1/admin/branding', status_code: 200 }),
        makeLog({ action: 'DELETE /api/v1/workspaces/abc', status_code: 403 })
      ]
    }));
    fixture.detectChanges();

    const rows = fixture.nativeElement.querySelectorAll('.audit-table tbody tr');
    expect(rows.length).toBe(3);
    const kinds = [...fixture.nativeElement.querySelectorAll('.action-kind')].map((el: HTMLElement) => el.textContent?.trim());
    expect(kinds).toEqual(['Creación', 'Actualización', 'Eliminación']);

    const pills = [...fixture.nativeElement.querySelectorAll('.cell-status .status-pill')].map((el: HTMLElement) => el.className);
    expect(pills[0]).toContain('status-success');
    expect(pills[1]).toContain('status-info');
    expect(pills[2]).toContain('status-error');
  });

  it('muestra el email institucional del actor resuelto por el backend', () => {
    listSpy.mockReturnValue(of({
      tenant_id: 't', limit: 20, offset: 0,
      data: [makeLog({ actor_email: 'mhamilton@uab.edu.bo' })]
    }));
    fixture.detectChanges();

    const actorCell = fixture.nativeElement.querySelector('.cell-actor');
    expect(actorCell.textContent).toContain('mhamilton@uab.edu.bo');
  });

  it('cambiar el filtro de acción relanza la consulta con el verbo seleccionado', () => {
    listSpy.mockReturnValue(of({ tenant_id: 't', limit: 20, offset: 0, data: [] }));
    fixture.detectChanges();
    expect(listSpy).toHaveBeenCalledTimes(1);

    component.onActionFilterChange('DELETE');

    expect(listSpy).toHaveBeenLastCalledWith(1, 20, { action: 'DELETE' });
  });

  it('la paginación pide la siguiente página al servicio', () => {
    listSpy.mockReturnValue(of({ tenant_id: 't', limit: 20, offset: 0, data: [] }));
    fixture.detectChanges();

    component.onPageChange(3);

    expect(listSpy).toHaveBeenLastCalledWith(3, 20, {});
  });

  it('el drawer carga la cronología del actor seleccionado', () => {
    const actorId = '22222222-2222-2222-2222-222222222222';
    listSpy.mockReturnValue(of({
      tenant_id: 't', limit: 20, offset: 0,
      data: [makeLog({ actor_id: actorId, actor_email: 'alovelace@uab.edu.bo' })]
    }));
    timelineSpy.mockReturnValue(of({
      tenant_id: 't', actor_id: actorId, count: 2,
      data: [
        makeLog({ actor_id: actorId, action: 'POST /api/v1/subjects', created_at: new Date().toISOString() }),
        makeLog({ actor_id: actorId, action: 'PUT /api/v1/labs/4', created_at: new Date(Date.now() - 86_400_000).toISOString() })
      ]
    }));
    fixture.detectChanges();

    fixture.nativeElement.querySelector('.open-drawer-btn').click();
    fixture.detectChanges();

    expect(timelineSpy).toHaveBeenCalledWith(actorId, 200);
    expect(component.drawerOpen()).toBe(true);
    expect(fixture.nativeElement.querySelectorAll('.timeline-item').length).toBe(2);
    expect(fixture.nativeElement.querySelector('.drawer-title p').textContent).toContain('alovelace@uab.edu.bo');
  });

  it('muestra estado vacío cuando no hay eventos y maneja errores del backend', () => {
    listSpy.mockReturnValueOnce(of({ tenant_id: 't', limit: 20, offset: 0, data: [] }));
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('.empty-state')).not.toBeNull();

    listSpy.mockReturnValueOnce(throwError(() => new Error('down')));
    component.load();
    expect(component.rows().length).toBe(0);
  });
});
