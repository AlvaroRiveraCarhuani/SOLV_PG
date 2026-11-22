import { ComponentFixture, TestBed } from '@angular/core/testing';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { FilterPopoverComponent } from './filter-popover.component';

function triggerRect(left: number, width = 100): DOMRect {
  return {
    x: left,
    y: 0,
    left,
    right: left + width,
    top: 0,
    bottom: 32,
    width,
    height: 32,
    toJSON: () => ({})
  } as DOMRect;
}

describe('FilterPopoverComponent', () => {
  let fixture: ComponentFixture<FilterPopoverComponent>;
  let component: FilterPopoverComponent;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [FilterPopoverComponent]
    }).compileComponents();

    fixture = TestBed.createComponent(FilterPopoverComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  function setViewport(width: number): void {
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: width });
  }

  function setTriggerRect(left: number, width = 100): void {
    const trigger = fixture.nativeElement.querySelector('.filter-trigger') as HTMLElement;
    vi.spyOn(trigger, 'getBoundingClientRect').mockReturnValue(triggerRect(left, width));
  }

  function openPanel(): HTMLElement {
    component.toggle();
    fixture.detectChanges();
    return fixture.nativeElement.querySelector('.filter-panel') as HTMLElement;
  }

  it('abre alineado a la izquierda cuando hay espacio suficiente', () => {
    setViewport(1366);
    setTriggerRect(300);

    const panel = openPanel();

    expect(panel.style.left).toBe('0px');
    expect(panel.style.width).toBe('440px');
  });

  it('se alinea a la derecha cuando el botón está cerca del borde derecho', () => {
    setViewport(1366);
    setTriggerRect(1000);

    const panel = openPanel();

    expect(panel.style.left).toBe('-340px');
    expect(panel.style.width).toBe('440px');
  });

  it('se desplaza para no salir por la izquierda con un panel más ancho que el espacio', () => {
    setViewport(1366);
    setTriggerRect(120);
    fixture.componentRef.setInput('width', '1400px');

    const panel = openPanel();

    expect(panel.style.width).toBe('1342px');
    expect(panel.style.left).toBe('-108px');
  });

  it('cierra el panel al segundo clic', () => {
    setViewport(1366);
    setTriggerRect(300);

    openPanel();
    component.toggle();
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.filter-panel')).toBeNull();
  });
});
