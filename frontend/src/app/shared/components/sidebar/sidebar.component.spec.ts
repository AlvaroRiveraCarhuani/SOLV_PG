import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { beforeEach, describe, expect, it } from 'vitest';
import { SidebarComponent } from './sidebar.component';

describe('SidebarComponent', () => {
  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [SidebarComponent],
      providers: [provideRouter([])]
    }).compileComponents();
  });

  it('refleja el estado colapsado en el contenedor', () => {
    const fixture = TestBed.createComponent(SidebarComponent);
    fixture.componentRef.setInput('collapsed', true);
    fixture.detectChanges();

    const aside = fixture.nativeElement.querySelector('.sidebar') as HTMLElement;
    expect(aside.classList.contains('collapsed')).toBe(true);
    expect(fixture.nativeElement.querySelectorAll('.nav-item').length).toBeGreaterThan(0);
  });

  it('expande el contenedor cuando collapsed es falso', () => {
    const fixture = TestBed.createComponent(SidebarComponent);
    fixture.componentRef.setInput('collapsed', false);
    fixture.detectChanges();

    const aside = fixture.nativeElement.querySelector('.sidebar') as HTMLElement;
    expect(aside.classList.contains('collapsed')).toBe(false);
  });
});
