import '@angular/compiler';
import '@angular/localize/init';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { of, throwError } from 'rxjs';
import { signal } from '@angular/core';
import { AdminAuditoriaEmergenciasComponent } from './admin-auditoria-emergencias.component';
import { AdminAuditoriaService, EMERGENCY_ACTIONS, EmergencyActionResult } from '../admin-auditoria.service';
import { AdminMetricsService } from '@features/admin/services/admin-metrics.service';

const EMPTY_LIST = { tenant_id: 't', limit: 10, offset: 0, total: 0, data: [] };

describe('AdminAuditoriaEmergenciasComponent', () => {
  let fixture: ComponentFixture<AdminAuditoriaEmergenciasComponent>;
  let component: AdminAuditoriaEmergenciasComponent;
  const executeSpy = vi.fn();
  const listSpy = vi.fn();
  const isExecuting = signal(false);
  const health = signal<any>(null);

  function result(affected: number, message: string): EmergencyActionResult {
    return {
      action: 'docker_prune',
      affected_count: affected,
      executed_by: 'admin',
      message
    };
  }

  beforeEach(async () => {
    executeSpy.mockReset();
    listSpy.mockReset();
    listSpy.mockReturnValue(of(EMPTY_LIST));
    isExecuting.set(false);
    health.set(null);

    await TestBed.configureTestingModule({
      imports: [AdminAuditoriaEmergenciasComponent],
      providers: [
        { provide: AdminAuditoriaService, useValue: { isExecuting, executeEmergencyAction: executeSpy, listAuditLogs: listSpy } },
        { provide: AdminMetricsService, useValue: { health } }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(AdminAuditoriaEmergenciasComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('renderiza las 5 acciones del catálogo ADR-032 con su clasificación de impacto', () => {
    const cards = fixture.nativeElement.querySelectorAll('.action-card');
    expect(cards.length).toBe(5);
    const destructive = fixture.nativeElement.querySelectorAll('.action-card.destructive');
    expect(destructive.length).toBe(1);
    expect(destructive[0].textContent).toContain('Terminar todos los workspaces');
  });

  it('el modal exige la frase exacta y un motivo de al menos 10 caracteres', () => {
    const target = component.actions[3]; // docker_prune
    component.openConfirmation(target);
    fixture.detectChanges();

    expect(component.confirmDisabled()).toBe(true);

    component.confirmPhrase.set(target.phrase);
    component.confirmReason.set('muy corto');
    expect(component.confirmDisabled()).toBe(true);

    component.confirmPhrase.set(target.phrase.toLowerCase());
    component.confirmReason.set('Motivo suficientemente largo');
    expect(component.confirmDisabled()).toBe(true);

    component.confirmPhrase.set(target.phrase);
    expect(component.confirmDisabled()).toBe(false);
  });

  it('ejecutar confirma la acción, cierra el modal y recarga el historial', () => {
    executeSpy.mockReturnValue(of(result(3, 'Poda completada: 3 huérfanos eliminados')));
    const callsBefore = listSpy.mock.calls.length;

    const target = component.actions[3];
    component.openConfirmation(target);
    component.confirmPhrase.set(target.phrase);
    component.confirmReason.set('Saturacion de disco del host');
    fixture.detectChanges();

    component.executeConfirmed();

    expect(executeSpy).toHaveBeenCalledWith('docker_prune', 'Saturacion de disco del host');
    expect(component.confirmTarget()).toBeNull();
    expect(component.toast()?.type).toBe('success');
    expect(listSpy.mock.calls.length).toBeGreaterThan(callsBefore);
  });

  it('una frase incorrecta muestra el error en el modal sin cerrarlo', () => {
    executeSpy.mockReturnValue(
      throwError(() => ({ status: 422, error: { error: 'invalid_confirmation_phrase' } }))
    );

    const target = component.actions[0];
    component.openConfirmation(target);
    component.confirmPhrase.set(target.phrase);
    component.confirmReason.set('Frase correcta pero backend la rechaza');
    fixture.detectChanges();

    component.executeConfirmed();

    expect(component.confirmTarget()).not.toBeNull();
    expect(component.confirmError()).toContain('no coincide');
  });

  it('muestra banner crítico cuando la RAM del host supera el 90%', () => {
    health.set({ metrics: { ram_percent: 94, containers_active: 12 } });
    fixture.detectChanges();

    expect(component.ramCritical()).toBe(true);
    expect(fixture.nativeElement.querySelector('.critical-banner')).not.toBeNull();
  });

  it('renderiza el historial con el evento de auditoría más reciente primero', () => {
    listSpy.mockImplementation((_page: number, _limit: number, filters?: { action?: string }) => {
      if (filters?.action === 'EMERGENCY_PRUNE_DOCKER') {
        return of({
          tenant_id: 't', limit: 10, offset: 0,
          data: [
            {
              id: 'log-1', tenant_id: 't', actor_id: 'a', actor_email: 'admin@uab.edu.bo',
              action: 'EMERGENCY_PRUNE_DOCKER', resource_type: 'emergency_action', resource_id: null,
              status_code: 200, metadata: null, ip_address: '', user_agent: '',
              created_at: '2026-09-27T12:00:00Z'
            }
          ]
        });
      }
      return of(EMPTY_LIST);
    });

    component.ngOnInit();
    fixture.detectChanges();

    const items = fixture.nativeElement.querySelectorAll('.history-item');
    expect(items.length).toBe(1);
    expect(items[0].textContent).toContain('admin@uab.edu.bo');
    expect(items[0].textContent).toContain('27/09/2026');
  });
});
