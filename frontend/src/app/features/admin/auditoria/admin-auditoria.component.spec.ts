import '@angular/compiler';
import '@angular/localize/init';
import { describe, it, expect, beforeEach } from 'vitest';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { AdminAuditoriaComponent } from './admin-auditoria.component';

describe('AdminAuditoriaComponent', () => {
  let fixture: ComponentFixture<AdminAuditoriaComponent>;
  let component: AdminAuditoriaComponent;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [AdminAuditoriaComponent]
    }).compileComponents();

    fixture = TestBed.createComponent(AdminAuditoriaComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('debe crearse con la pestaña de registro activa por defecto', () => {
    expect(component.activeTab()).toBe('registro');
    const tabs = fixture.nativeElement.querySelectorAll('.auditoria-tab');
    expect(tabs.length).toBe(2);
    expect(tabs[0].classList.contains('active')).toBe(true);
  });

  it('debe cambiar de pestaña al hacer click', () => {
    const tabs = fixture.nativeElement.querySelectorAll('.auditoria-tab') as NodeListOf<HTMLButtonElement>;
    tabs[1].click();
    fixture.detectChanges();

    expect(component.activeTab()).toBe('emergencias');
    expect(tabs[1].classList.contains('active')).toBe(true);
    expect(tabs[0].classList.contains('active')).toBe(false);
  });

  it('debe renderizar solo el panel de la pestaña activa', () => {
    component.selectTab('emergencias');
    fixture.detectChanges();

    const panel = fixture.nativeElement.querySelector('.auditoria-tab-panel');
    expect(panel.querySelector('admin-auditoria-emergencias')).not.toBeNull();
    expect(panel.querySelector('admin-auditoria-registro')).toBeNull();
  });

  it('debe mostrar el nombre institucional del tenant en el subtítulo', () => {
    const subtitle = fixture.nativeElement.querySelector('.view-subtitle');
    expect(subtitle.textContent).toContain('la institución');
  });
});
