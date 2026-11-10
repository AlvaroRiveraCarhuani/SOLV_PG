import { ComponentFixture, TestBed } from '@angular/core/testing';
import { AiProgressIndicatorComponent } from './ai-progress-indicator.component';

describe('AiProgressIndicatorComponent', () => {
  let fixture: ComponentFixture<AiProgressIndicatorComponent>;
  let component: AiProgressIndicatorComponent;

  function mount(label = 'Diseñando arquitectura...'): void {
    fixture = TestBed.createComponent(AiProgressIndicatorComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('label', label);
    fixture.detectChanges();
  }

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [AiProgressIndicatorComponent]
    }).compileComponents();
  });

  it('muestra la fase en curso y los segundos transcurridos', () => {
    mount();
    fixture.componentRef.setInput('elapsedSeconds', '3.0');
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    expect(el.textContent).toContain('Diseñando arquitectura...');
    expect(el.querySelector('.progress-elapsed')?.textContent).toContain('3.0 s');
  });

  it('expone una barra indeterminada accesible con el nombre del flujo', () => {
    mount();
    fixture.componentRef.setInput('ariaLabel', 'Generando propuesta de arquitectura');
    fixture.componentRef.setInput('elapsedSeconds', '1.4');
    fixture.detectChanges();

    const bar = fixture.nativeElement.querySelector('.ai-progress') as HTMLElement;
    expect(bar.getAttribute('role')).toBe('progressbar');
    expect(bar.getAttribute('aria-label')).toBe('Generando propuesta de arquitectura');
    expect(bar.getAttribute('aria-valuetext')).toBe('Esperando respuesta, 1.4 segundos');
    // Sin porcentaje: el tiempo del modelo no es predecible.
    expect(bar.getAttribute('aria-valuenow')).toBeNull();
    expect(fixture.nativeElement.querySelector('.ai-progress-bar')).toBeTruthy();
  });

  it('usa la fase como nombre accesible cuando no se define ariaLabel', () => {
    mount('Reparando entorno...');

    const bar = fixture.nativeElement.querySelector('.ai-progress') as HTMLElement;
    expect(bar.getAttribute('aria-label')).toBe('Reparando entorno...');
  });

  it('emite cancelled al pulsar Cancelar sin cerrar la operación por su cuenta', () => {
    mount();
    const spy = vi.fn();
    component.cancelled.subscribe(spy);

    const button = fixture.nativeElement.querySelector('.btn-cancel-gen') as HTMLButtonElement;
    expect(button.textContent).toContain('Cancelar');
    button.click();

    expect(spy).toHaveBeenCalledTimes(1);
  });

  it('marca el host como variante plana para los flujos que ya traen contenedor', () => {
    mount();
    const host: HTMLElement = fixture.nativeElement;
    expect(host.classList.contains('is-plain')).toBe(false);

    fixture.componentRef.setInput('variant', 'plain');
    fixture.detectChanges();

    expect(host.classList.contains('is-plain')).toBe(true);
  });
});
