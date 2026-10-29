import { Component } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { AutoGrowDirective } from './auto-grow.directive';

@Component({
  standalone: true,
  imports: [AutoGrowDirective],
  template: `
    <textarea
      autoGrow
      [minRows]="minRows"
      [maxRows]="maxRows"
      style="padding: 8px 12px; font-size: 14px; line-height: 21px; box-sizing: border-box; border: 1px solid black;"
    ></textarea>
  `
})
class TestHostComponent {
  minRows = 3;
  maxRows = 5;
}

describe('AutoGrowDirective', () => {
  let fixture: ComponentFixture<TestHostComponent>;
  let textarea: HTMLTextAreaElement;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [TestHostComponent]
    }).compileComponents();

    fixture = TestBed.createComponent(TestHostComponent);
    fixture.detectChanges();
    textarea = fixture.nativeElement.querySelector('textarea');
  });

  it('debe crearse e inicializar la altura mínima para 3 líneas', () => {
    expect(textarea).toBeTruthy();
    // 3 líneas * 21px + 16px padding + 2px borde = 81px
    const minHeight = parseFloat(textarea.style.minHeight);
    expect(minHeight).toBeGreaterThanOrEqual(80);
    expect(minHeight).toBeLessThanOrEqual(84);
  });

  it('debe limitar la altura máxima para 5 líneas', () => {
    // 5 líneas * 21px + 16px padding + 2px borde = 123px
    const maxHeight = parseFloat(textarea.style.maxHeight);
    expect(maxHeight).toBeGreaterThanOrEqual(122);
    expect(maxHeight).toBeLessThanOrEqual(126);
  });

  it('debe mantener overflow hidden cuando el contenido está dentro del límite', () => {
    textarea.value = 'Línea 1\nLínea 2\nLínea 3';
    textarea.dispatchEvent(new Event('input'));
    fixture.detectChanges();

    expect(textarea.style.overflowY).toBe('hidden');
  });

  it('debe activar overflow auto cuando el contenido supera las líneas máximas', () => {
    textarea.value = 'Línea 1\nLínea 2\nLínea 3\nLínea 4\nLínea 5\nLínea 6\nLínea 7\nLínea 8';
    // Simulamos un scrollHeight mayor a maxHeight
    Object.defineProperty(textarea, 'scrollHeight', {
      configurable: true,
      value: 200
    });

    textarea.dispatchEvent(new Event('input'));
    fixture.detectChanges();

    expect(textarea.style.overflowY).toBe('auto');
    expect(parseFloat(textarea.style.height)).toBeLessThanOrEqual(parseFloat(textarea.style.maxHeight));
  });
});
