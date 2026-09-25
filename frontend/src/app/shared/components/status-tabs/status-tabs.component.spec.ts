import { describe, it, expect, beforeEach } from 'vitest';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { Component } from '@angular/core';
import { StatusTabsComponent, StatusTabItem } from './status-tabs.component';

@Component({
  standalone: true,
  imports: [StatusTabsComponent],
  template: `
    <status-tabs
      [tabs]="tabs"
      [activeTab]="activeTab"
      (tabChange)="onTabChange($event)"
    />
  `
})
class HostComponent {
  tabs: StatusTabItem[] = [
    { id: 'all', label: 'Todos', count: 12 },
    { id: 'active', label: 'Activos', count: 8, badgeVariant: 'active' },
    { id: 'warning', label: 'En Riesgo', count: 3, badgeVariant: 'warning' },
    { id: 'empty', label: 'Sin Conteo' }
  ];
  activeTab = 'all';
  selectedTabId?: string;

  onTabChange(tabId: string): void {
    this.selectedTabId = tabId;
    this.activeTab = tabId;
  }
}

describe('StatusTabsComponent Unit Tests', () => {
  let fixture: ComponentFixture<HostComponent>;
  let host: HostComponent;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [HostComponent]
    }).compileComponents();

    fixture = TestBed.createComponent(HostComponent);
    host = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('debe crearse y renderizar todos los botones de pestañas', () => {
    const buttons = fixture.nativeElement.querySelectorAll('.tab-btn');
    expect(buttons.length).toBe(4);
    expect(buttons[0].textContent).toContain('Todos');
    expect(buttons[1].textContent).toContain('Activos');
  });

  it('debe marcar como activa la pestaña correspondiente', () => {
    const buttons = fixture.nativeElement.querySelectorAll('.tab-btn');
    expect(buttons[0].classList.contains('active')).toBe(true);
    expect(buttons[0].getAttribute('aria-selected')).toBe('true');
    expect(buttons[1].classList.contains('active')).toBe(false);
  });

  it('debe emitir tabChange al hacer clic en una pestaña distinta', () => {
    const buttons = fixture.nativeElement.querySelectorAll('.tab-btn');
    buttons[1].click();
    fixture.detectChanges();

    expect(host.selectedTabId).toBe('active');
    expect(buttons[1].classList.contains('active')).toBe(true);
  });

  it('no debe emitir tabChange si se hace clic en la pestaña actualmente activa', () => {
    host.selectedTabId = undefined;
    const buttons = fixture.nativeElement.querySelectorAll('.tab-btn');
    buttons[0].click(); // ya está activa 'all'
    fixture.detectChanges();

    expect(host.selectedTabId).toBeUndefined();
  });

  it('debe renderizar badges numéricos con variantes de estilo', () => {
    const badges = fixture.nativeElement.querySelectorAll('.tab-badge');
    expect(badges.length).toBe(3); // La cuarta pestaña no tiene count
    expect(badges[0].textContent.trim()).toBe('12');
    expect(badges[1].getAttribute('data-variant')).toBe('active');
    expect(badges[2].getAttribute('data-variant')).toBe('warning');
  });
});
