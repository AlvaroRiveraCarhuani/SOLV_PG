import { TestBed } from '@angular/core/testing';
import { NavigationEnd, Router } from '@angular/router';
import { Subject } from 'rxjs';
import { beforeEach, describe, expect, it } from 'vitest';
import { SidebarRailService } from './sidebar-rail.service';

const STORAGE_KEY = 'solv.sidebar.collapsed';

describe('SidebarRailService', () => {
  let events: Subject<unknown>;
  let router: { url: string; events: ReturnType<Subject<unknown>['asObservable']> };

  beforeEach(() => {
    localStorage.clear();
    events = new Subject<unknown>();
    router = { url: '/student', events: events.asObservable() };
    TestBed.resetTestingModule();
    TestBed.configureTestingModule({
      providers: [SidebarRailService, { provide: Router, useValue: router }]
    });
  });

  it('arranca en rail en una ruta que maximiza el área de trabajo', () => {
    const service = TestBed.inject(SidebarRailService);
    expect(service.isCollapsed()).toBe(true);
  });

  it('mantiene la elección del usuario al navegar entre rutas', () => {
    const service = TestBed.inject(SidebarRailService);

    service.setCollapsed(false);
    events.next(new NavigationEnd(1, '/student/history', '/student/history'));

    expect(service.isCollapsed()).toBe(false);
    expect(service.isRail()).toBe(false);
  });

  it('persiste la preferencia y la recupera al reconstruir el servicio', () => {
    const first = TestBed.inject(SidebarRailService);
    first.setCollapsed(false);
    expect(localStorage.getItem(STORAGE_KEY)).toBe('false');

    TestBed.resetTestingModule();
    TestBed.configureTestingModule({
      providers: [SidebarRailService, { provide: Router, useValue: router }]
    });
    const second = TestBed.inject(SidebarRailService);

    expect(second.isCollapsed()).toBe(false);
  });

  it('alterna el estado y lo guarda', () => {
    const service = TestBed.inject(SidebarRailService);

    service.toggle();
    expect(service.isCollapsed()).toBe(false);
    expect(localStorage.getItem(STORAGE_KEY)).toBe('false');

    service.toggle();
    expect(service.isCollapsed()).toBe(true);
    expect(localStorage.getItem(STORAGE_KEY)).toBe('true');
  });
});
