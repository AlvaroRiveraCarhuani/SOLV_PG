import { describe, it, expect, beforeEach } from 'vitest';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { SolvComboboxComponent, ComboboxOption } from './combobox.component';

describe('SolvComboboxComponent', () => {
  let component: SolvComboboxComponent;
  let fixture: ComponentFixture<SolvComboboxComponent>;

  const mockOptions: ComboboxOption[] = [
    { id: '1', label: 'python:3.12-slim', value: 'python:3.12-slim', group: 'Oficial' },
    { id: '2', label: 'python:3.11-slim', value: 'python:3.11-slim', group: 'Oficial' },
    { id: '3', label: 'node:20-alpine', value: 'node:20-alpine', group: 'Oficial' },
    { id: '4', label: 'golang:1.22-bookworm', value: 'golang:1.22-bookworm', group: 'Oficial' },
    { id: '5', label: 'gcc:13.2-bookworm', value: 'gcc:13.2-bookworm', group: 'Oficial' },
    { id: '6', label: 'eclipse-temurin:21-alpine', value: 'eclipse-temurin:21-alpine', group: 'Oficial' },
    { id: '7', label: 'postgres:16-alpine', value: 'postgres:16-alpine', group: 'Oficial' },
    { id: '8', label: 'redis:7-alpine', value: 'redis:7-alpine', group: 'Oficial' },
    { id: '9', label: 'rust:1.80-slim', value: 'rust:1.80-slim', group: 'Oficial' },
    { id: '10', label: 'local/custom:latest', value: 'local/custom:latest', group: 'Local' }
  ];

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [SolvComboboxComponent]
    }).compileComponents();

    fixture = TestBed.createComponent(SolvComboboxComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('options', mockOptions);
    fixture.detectChanges();
  });

  it('debe crearse correctamente con input combobox accesible', () => {
    expect(component).toBeTruthy();
    const input = fixture.nativeElement.querySelector('input[role="combobox"]');
    expect(input).toBeTruthy();
    expect(input.getAttribute('aria-expanded')).toBe('false');
  });

  it('debe acotar las opciones visibles al máximo configurado (por defecto 8)', () => {
    component.isOpen.set(true);
    fixture.detectChanges();

    expect(component.filteredOptions().length).toBe(8);
    const renderedOptions = fixture.nativeElement.querySelectorAll('.option-item');
    expect(renderedOptions.length).toBe(8);
  });

  it('debe mostrar el conteo total en la cabecera del listbox', () => {
    fixture.componentRef.setInput('totalAvailableCount', 10);
    component.isOpen.set(true);
    fixture.detectChanges();

    const header = fixture.nativeElement.querySelector('.header-count');
    expect(header.textContent).toContain('10 disponibles; escriba para filtrar');
  });

  it('debe filtrar en tiempo real al escribir', () => {
    component.isOpen.set(true);
    fixture.componentRef.setInput('value', 'python');
    fixture.detectChanges();

    expect(component.filteredOptions().length).toBe(2);
    expect(component.filteredOptions()[0].label).toBe('python:3.12-slim');
  });

  it('debe navegar opciones con ArrowDown y seleccionar con Enter', () => {
    component.isOpen.set(true);
    fixture.detectChanges();

    let selected: ComboboxOption | null = null;
    component.optionSelected.subscribe((opt) => (selected = opt));

    const input = fixture.nativeElement.querySelector('input');
    input.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true }));
    fixture.detectChanges();
    expect(component.activeFlatIndex()).toBe(0);

    input.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true }));
    fixture.detectChanges();
    expect(component.activeFlatIndex()).toBe(1);

    input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }));
    fixture.detectChanges();

    expect(selected).toBeTruthy();
    expect(selected!.label).toBe('python:3.11-slim');
    expect(component.isOpen()).toBe(false);
  });

  it('debe cerrar el listbox con la tecla Escape', () => {
    component.isOpen.set(true);
    fixture.detectChanges();

    const input = fixture.nativeElement.querySelector('input');
    input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));
    fixture.detectChanges();

    expect(component.isOpen()).toBe(false);
  });

  it('debe cerrar el listbox al hacer clic fuera del componente', () => {
    component.isOpen.set(true);
    fixture.detectChanges();
    expect(component.isOpen()).toBe(true);

    // Clic fuera del elemento nativo del combobox
    document.body.dispatchEvent(new MouseEvent('click', { bubbles: true }));
    fixture.detectChanges();

    expect(component.isOpen()).toBe(false);
  });

  it('debe mostrar el botón para cargar más opciones y expandir la cota progresiva', () => {
    component.isOpen.set(true);
    fixture.detectChanges();

    expect(component.filteredOptions().length).toBe(8);
    expect(component.hasMoreOptions()).toBe(true);

    const moreBtn = fixture.nativeElement.querySelector('.btn-combobox-more');
    expect(moreBtn).toBeTruthy();
    expect(moreBtn.textContent).toContain('Mostrar más (2 restantes)');

    moreBtn.click();
    fixture.detectChanges();

    expect(component.filteredOptions().length).toBe(10);
    expect(component.hasMoreOptions()).toBe(false);
  });

  it('debe filtrar por chip de grupo cuando hay múltiples grupos disponibles', () => {
    component.isOpen.set(true);
    fixture.detectChanges();

    const chips = fixture.nativeElement.querySelectorAll('.group-chip');
    expect(chips.length).toBe(3); // Todas, Oficial, Local

    // Clic en el chip "Local"
    chips[2].click();
    fixture.detectChanges();

    expect(component.selectedGroupFilter()).toBe('Local');
    expect(component.filteredOptions().length).toBe(1);
    expect(component.filteredOptions()[0].label).toBe('local/custom:latest');
  });
});
