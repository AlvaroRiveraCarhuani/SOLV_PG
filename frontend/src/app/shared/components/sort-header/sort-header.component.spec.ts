import '@angular/compiler';
import { describe, it, expect } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { SortHeaderComponent } from './sort-header.component';
import { SortDirection } from './sort-header.types';

describe('SortHeaderComponent', () => {
  it('debe crearse correctamente', () => {
    const fixture = TestBed.createComponent(SortHeaderComponent);
    fixture.componentRef.setInput('column', 'code');
    fixture.detectChanges();
    expect(fixture.componentInstance).toBeTruthy();
  });

  it('cambia cíclicamente de dirección: none -> asc -> desc -> none', () => {
    const fixture = TestBed.createComponent(SortHeaderComponent);
    fixture.componentRef.setInput('column', 'name');
    fixture.componentRef.setInput('activeColumn', null);
    fixture.componentRef.setInput('direction', 'none');
    fixture.detectChanges();

    const emitted: { column: string; direction: SortDirection }[] = [];
    fixture.componentInstance.sortChange.subscribe(e => emitted.push(e));

    // Primer clic: de 'none' pasa a 'asc'
    fixture.componentInstance.onToggle();
    expect(emitted[0]).toEqual({ column: 'name', direction: 'asc' });

    // Cuando pasa a estar activa en 'asc'
    fixture.componentRef.setInput('activeColumn', 'name');
    fixture.componentRef.setInput('direction', 'asc');
    fixture.detectChanges();

    // Segundo clic: de 'asc' pasa a 'desc'
    fixture.componentInstance.onToggle();
    expect(emitted[1]).toEqual({ column: 'name', direction: 'desc' });

    // Cuando pasa a estar activa en 'desc'
    fixture.componentRef.setInput('activeColumn', 'name');
    fixture.componentRef.setInput('direction', 'desc');
    fixture.detectChanges();

    // Tercer clic: de 'desc' pasa a 'none'
    fixture.componentInstance.onToggle();
    expect(emitted[2]).toEqual({ column: 'name', direction: 'none' });
  });

  it('calcula ariaSortValue correctamente', () => {
    const fixture = TestBed.createComponent(SortHeaderComponent);
    fixture.componentRef.setInput('column', 'name');
    fixture.componentRef.setInput('activeColumn', 'name');
    fixture.componentRef.setInput('direction', 'asc');
    fixture.detectChanges();

    expect(fixture.componentInstance.ariaSortValue()).toBe('ascending');

    fixture.componentRef.setInput('direction', 'desc');
    fixture.detectChanges();
    expect(fixture.componentInstance.ariaSortValue()).toBe('descending');

    fixture.componentRef.setInput('direction', 'none');
    fixture.detectChanges();
    expect(fixture.componentInstance.ariaSortValue()).toBe('none');
  });
});
