import { describe, it, expect, beforeEach } from 'vitest';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { FieldMessageComponent } from './field-message.component';

describe('FieldMessageComponent', () => {
  let component: FieldMessageComponent;
  let fixture: ComponentFixture<FieldMessageComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [FieldMessageComponent]
    }).compileComponents();

    fixture = TestBed.createComponent(FieldMessageComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('debe crearse correctamente con variante info por defecto', () => {
    expect(component).toBeTruthy();
    expect(component.variant()).toBe('info');
    const el = fixture.nativeElement.querySelector('.field-message');
    expect(el.classList.contains('variant-info')).toBe(true);
  });

  it('debe aplicar la clase de variante error y aria-live assertive', () => {
    fixture.componentRef.setInput('variant', 'error');
    fixture.detectChanges();
    const el = fixture.nativeElement.querySelector('.field-message');
    expect(el.classList.contains('variant-error')).toBe(true);
    expect(el.getAttribute('aria-live')).toBe('assertive');
  });

  it('debe mostrar botón de acción y emitir actionClicked al pulsar', () => {
    let clicked = false;
    component.actionClicked.subscribe(() => (clicked = true));

    fixture.componentRef.setInput('actionLabel', 'Reintentar');
    fixture.detectChanges();

    const btn = fixture.nativeElement.querySelector('.message-action-btn');
    expect(btn).toBeTruthy();
    expect(btn.textContent).toContain('Reintentar');

    btn.click();
    expect(clicked).toBe(true);
  });
});
