import '@angular/compiler';
import '@angular/localize/init';
import { describe, it, expect, beforeEach } from 'vitest';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { AdminConfiguracionComponent } from './admin-configuracion.component';

describe('AdminConfiguracionComponent', () => {
  let fixture: ComponentFixture<AdminConfiguracionComponent>;
  let component: AdminConfiguracionComponent;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [AdminConfiguracionComponent]
    }).compileComponents();

    fixture = TestBed.createComponent(AdminConfiguracionComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('debe crearse con la pestaña de períodos activa por defecto', () => {
    expect(component.activeTab()).toBe('periodos');
    const tabs = fixture.nativeElement.querySelectorAll('.config-tab');
    expect(tabs.length).toBe(3);
    expect(tabs[0].classList.contains('active')).toBe(true);
  });

  it('debe cambiar de pestaña al hacer click', () => {
    const tabs = fixture.nativeElement.querySelectorAll('.config-tab') as NodeListOf<HTMLButtonElement>;
    tabs[2].click();
    fixture.detectChanges();

    expect(component.activeTab()).toBe('servidor');
    expect(tabs[2].classList.contains('active')).toBe(true);
    expect(tabs[0].classList.contains('active')).toBe(false);
  });

  it('debe renderizar solo el panel de la pestaña activa', () => {
    component.selectTab('identidad');
    fixture.detectChanges();

    const panel = fixture.nativeElement.querySelector('.config-tab-panel');
    expect(panel.querySelector('admin-config-identidad')).not.toBeNull();
    expect(panel.querySelector('admin-config-periodos')).toBeNull();
  });

  it('debe mostrar el nombre institucional del tenant en el subtítulo', () => {
    const subtitle = fixture.nativeElement.querySelector('.view-subtitle');
    expect(subtitle.textContent).toContain('la institución');
  });
});
