import { describe, it, expect, beforeEach } from 'vitest';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { Component } from '@angular/core';
import { ViewSwitcherComponent, ViewMode } from './view-switcher.component';

@Component({
  standalone: true,
  imports: [ViewSwitcherComponent],
  template: `
    <view-switcher [(mode)]="currentMode" />
  `
})
class HostComponent {
  currentMode: ViewMode = 'table';
}

describe('ViewSwitcherComponent', () => {
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

  it('debe crearse y renderizar ambos botones de vista', () => {
    const buttons = fixture.nativeElement.querySelectorAll('.switcher-btn');
    expect(buttons.length).toBe(2);
    expect(buttons[0].textContent).toContain('Tarjetas');
    expect(buttons[1].textContent).toContain('Tabla');
  });

  it('debe reflejar la clase activa según el modo actual', () => {
    const buttons = fixture.nativeElement.querySelectorAll('.switcher-btn');
    expect(buttons[1].classList.contains('active')).toBe(true);
    expect(buttons[0].classList.contains('active')).toBe(false);
  });

  it('debe cambiar de modo al hacer click en el botón de tarjetas', () => {
    const buttons = fixture.nativeElement.querySelectorAll('.switcher-btn') as NodeListOf<HTMLButtonElement>;
    buttons[0].click();
    fixture.detectChanges();

    expect(host.currentMode).toBe('cards');
    expect(buttons[0].classList.contains('active')).toBe(true);
    expect(buttons[1].classList.contains('active')).toBe(false);
  });
});
