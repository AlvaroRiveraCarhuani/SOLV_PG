import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { GenerateCasesModalComponent } from './generate-cases-modal.component';

describe('GenerateCasesModalComponent', () => {
  let component: GenerateCasesModalComponent;
  let fixture: ComponentFixture<GenerateCasesModalComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [GenerateCasesModalComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting()
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(GenerateCasesModalComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('debe inicializarse correctamente', () => {
    expect(component).toBeTruthy();
    expect(component.count()).toBe(10);
  });

  it('debe validar y limitar el rango de casos entre 1 y 100', () => {
    component.onCountChange(150);
    expect(component.count()).toBe(100);

    component.onCountChange(-5);
    expect(component.count()).toBe(1);
  });

  it('debe emitir cancelación al cerrar', () => {
    spyOn(component.cancel, 'emit');
    component.onClose();
    expect(component.cancel.emit).toHaveBeenCalled();
  });
});
