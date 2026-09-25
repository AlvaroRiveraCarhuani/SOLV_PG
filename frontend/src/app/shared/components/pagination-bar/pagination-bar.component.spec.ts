import { describe, it, expect, beforeEach } from 'vitest';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { Component } from '@angular/core';
import { signal } from '@angular/core';
import { PaginationBarComponent } from './pagination-bar.component';

@Component({
  standalone: true,
  imports: [PaginationBarComponent],
  template: `
    <pagination-bar
      [currentPage]="currentPage()"
      [totalPages]="totalPages()"
      [totalItems]="totalItems()"
      [from]="from()"
      [to]="to()"
      [itemLabel]="itemLabel()"
      [showPageNumbers]="showPageNumbers()"
      (pageChange)="onPageChange($event)"
    />
  `
})
class HostComponent {
  readonly currentPage = signal(1);
  readonly totalPages = signal(5);
  readonly totalItems = signal<number | undefined>(50);
  readonly from = signal<number | undefined>(1);
  readonly to = signal<number | undefined>(10);
  readonly itemLabel = signal('estudiantes');
  readonly showPageNumbers = signal(true);
  selectedPage?: number;

  onPageChange(page: number): void {
    this.selectedPage = page;
    this.currentPage.set(page);
  }
}

describe('PaginationBarComponent Unit Tests', () => {
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

  it('debe crearse y renderizar el rango from-to y el total de elementos', () => {
    const info = fixture.nativeElement.querySelector('.pagination-info');
    expect(info.textContent).toContain('1–10');
    expect(info.textContent).toContain('50');
    expect(info.textContent).toContain('estudiantes');
  });

  it('debe deshabilitar el botón anterior en la primera página', () => {
    const prevBtn = fixture.nativeElement.querySelectorAll('.btn-page')[0];
    expect(prevBtn.disabled).toBe(true);
  });

  it('debe emitir pageChange al hacer clic en Siguiente', () => {
    const nextBtn = fixture.nativeElement.querySelectorAll('.btn-page')[1];
    expect(nextBtn.disabled).toBe(false);

    nextBtn.click();
    fixture.detectChanges();

    expect(host.selectedPage).toBe(2);
    expect(host.currentPage()).toBe(2);
  });

  it('debe deshabilitar el botón siguiente en la última página', () => {
    host.currentPage.set(5);
    fixture.detectChanges();

    const nextBtn = fixture.nativeElement.querySelectorAll('.btn-page')[1];
    expect(nextBtn.disabled).toBe(true);
  });

  it('debe emitir pageChange al hacer clic en un número de página', () => {
    const pageButtons = fixture.nativeElement.querySelectorAll('.btn-page-number');
    expect(pageButtons.length).toBe(5);

    pageButtons[2].click(); // Page 3
    fixture.detectChanges();

    expect(host.selectedPage).toBe(3);
  });

  it('no debe renderizar controles si totalPages <= 1', () => {
    host.totalPages.set(1);
    fixture.detectChanges();

    const controls = fixture.nativeElement.querySelector('.pagination-controls');
    expect(controls).toBeNull();
  });
});
