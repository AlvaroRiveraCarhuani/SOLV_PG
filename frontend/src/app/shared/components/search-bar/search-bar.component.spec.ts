import { describe, it, expect, beforeEach } from 'vitest';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { SearchBarComponent } from './search-bar.component';
import { Component, signal } from '@angular/core';

@Component({
  standalone: true,
  imports: [SearchBarComponent],
  template: `
    <search-bar
      [query]="searchText()"
      [placeholder]="placeholder()"
      [compact]="isCompact()"
      (queryChange)="onQueryChange($event)"
      (cleared)="onCleared()"
    ></search-bar>
  `
})
class HostComponent {
  searchText = signal<string>('');
  placeholder = signal<string>('Buscar...');
  isCompact = signal<boolean>(true);
  lastQuery = '';
  clearedCount = 0;

  onQueryChange(q: string): void {
    this.lastQuery = q;
    this.searchText.set(q);
  }

  onCleared(): void {
    this.clearedCount++;
  }
}

describe('SearchBarComponent Unit Tests', () => {
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

  it('debe crearse correctamente', () => {
    const searchBar = fixture.nativeElement.querySelector('search-bar');
    expect(searchBar).toBeTruthy();
  });

  it('debe aplicar la clase compact si compact es true', () => {
    const wrapper = fixture.nativeElement.querySelector('.search-bar-wrapper');
    expect(wrapper.classList.contains('compact')).toBe(true);

    host.isCompact.set(false);
    fixture.detectChanges();
    expect(wrapper.classList.contains('compact')).toBe(false);
  });

  it('debe emitir queryChange al tipear en el input', () => {
    const input = fixture.nativeElement.querySelector('input');
    input.value = 'docente';
    input.dispatchEvent(new Event('input'));
    fixture.detectChanges();

    expect(host.lastQuery).toBe('docente');
  });

  it('el botón de limpiar no debe mostrarse cuando el input está vacío', () => {
    const clearBtn = fixture.nativeElement.querySelector('.btn-clear');
    expect(clearBtn).toBeNull();
  });

  it('el botón de limpiar debe mostrarse cuando hay texto y limpiar al hacer clic', () => {
    host.searchText.set('sistemas');
    fixture.detectChanges();

    const clearBtn = fixture.nativeElement.querySelector('.btn-clear');
    expect(clearBtn).toBeTruthy();

    clearBtn.click();
    fixture.detectChanges();

    expect(host.lastQuery).toBe('');
    expect(host.clearedCount).toBe(1);
    expect(fixture.nativeElement.querySelector('input').value).toBe('');
  });

  it('debe limpiar la búsqueda al presionar la tecla Escape', () => {
    host.searchText.set('redes');
    fixture.detectChanges();

    const input = fixture.nativeElement.querySelector('input');
    input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
    fixture.detectChanges();

    expect(host.lastQuery).toBe('');
    expect(host.clearedCount).toBe(1);
  });
});
