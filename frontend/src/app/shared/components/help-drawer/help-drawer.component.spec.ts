import { describe, it, expect, beforeEach } from 'vitest';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { RouterTestingModule } from '@angular/router/testing';
import { SolvHelpDrawerComponent } from './help-drawer.component';

describe('SolvHelpDrawerComponent', () => {
  let component: SolvHelpDrawerComponent;
  let fixture: ComponentFixture<SolvHelpDrawerComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [SolvHelpDrawerComponent, RouterTestingModule]
    }).compileComponents();

    fixture = TestBed.createComponent(SolvHelpDrawerComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('no debe renderizar nada si isOpen es false', () => {
    expect(fixture.nativeElement.querySelector('.help-drawer')).toBeNull();
  });

  it('debe renderizar el drawer cuando isOpen es true con su título y badge de paso', () => {
    fixture.componentRef.setInput('isOpen', true);
    fixture.componentRef.setInput('title', 'Propósito del Entorno');
    fixture.componentRef.setInput('stepNumber', 1);
    fixture.detectChanges();

    const drawer = fixture.nativeElement.querySelector('.help-drawer');
    expect(drawer).toBeTruthy();
    expect(drawer.querySelector('.drawer-title').textContent).toContain('Propósito del Entorno');
    expect(drawer.querySelector('.drawer-step-badge').textContent).toContain('Paso 1');
  });

  it('debe emitir closed al pulsar el botón de cerrar', () => {
    fixture.componentRef.setInput('isOpen', true);
    fixture.detectChanges();

    let closed = false;
    component.closed.subscribe(() => (closed = true));

    const btn = fixture.nativeElement.querySelector('.btn-close-drawer');
    btn.click();
    expect(closed).toBe(true);
  });

  it('debe emitir closed al presionar la tecla Escape', () => {
    fixture.componentRef.setInput('isOpen', true);
    fixture.detectChanges();

    let closed = false;
    component.closed.subscribe(() => (closed = true));

    const escEvent = new KeyboardEvent('keydown', { key: 'Escape', bubbles: true });
    document.dispatchEvent(escEvent);

    expect(closed).toBe(true);
  });
});
