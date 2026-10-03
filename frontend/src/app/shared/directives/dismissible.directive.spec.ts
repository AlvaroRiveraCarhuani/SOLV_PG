import { Component, signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { DismissibleDirective } from './dismissible.directive';

@Component({
  standalone: true,
  imports: [DismissibleDirective],
  template: `
    <button #triggerBtn id="trigger">Toggle</button>
    <div 
      id="popover"
      dismissible 
      [dismissExclude]="triggerBtn"
      [dismissibleEnabled]="enabled()"
      [dismissOnEscape]="escapeEnabled()"
      [dismissOnClickOutside]="clickOutsideEnabled()"
      (dismiss)="onDismiss()"
    >
      <button id="inside-btn">Inside</button>
    </div>
    <div id="outside-area">Outside</div>
  `
})
class TestHostComponent {
  enabled = signal(true);
  escapeEnabled = signal(true);
  clickOutsideEnabled = signal(true);
  dismissCount = 0;

  onDismiss(): void {
    this.dismissCount++;
  }
}

describe('DismissibleDirective', () => {
  let fixture: ComponentFixture<TestHostComponent>;
  let host: TestHostComponent;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [TestHostComponent]
    }).compileComponents();

    fixture = TestBed.createComponent(TestHostComponent);
    host = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should emit dismiss when clicking outside the element', () => {
    const outsideEl = fixture.nativeElement.querySelector('#outside-area') as HTMLElement;
    outsideEl.dispatchEvent(new MouseEvent('click', { bubbles: true }));

    expect(host.dismissCount).toBe(1);
  });

  it('should not emit dismiss when clicking inside the element', () => {
    const insideEl = fixture.nativeElement.querySelector('#inside-btn') as HTMLElement;
    insideEl.dispatchEvent(new MouseEvent('click', { bubbles: true }));

    expect(host.dismissCount).toBe(0);
  });

  it('should not emit dismiss when clicking on an excluded element', () => {
    const triggerEl = fixture.nativeElement.querySelector('#trigger') as HTMLElement;
    triggerEl.dispatchEvent(new MouseEvent('click', { bubbles: true }));

    expect(host.dismissCount).toBe(0);
  });

  it('should emit dismiss when Escape key is pressed', () => {
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));

    expect(host.dismissCount).toBe(1);
  });

  it('should not emit dismiss when disabled', () => {
    host.enabled.set(false);
    fixture.detectChanges();

    const outsideEl = fixture.nativeElement.querySelector('#outside-area') as HTMLElement;
    outsideEl.dispatchEvent(new MouseEvent('click', { bubbles: true }));
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));

    expect(host.dismissCount).toBe(0);
  });

  it('should respect dismissOnEscape and dismissOnClickOutside flags', () => {
    host.escapeEnabled.set(false);
    fixture.detectChanges();

    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
    expect(host.dismissCount).toBe(0);

    host.escapeEnabled.set(true);
    host.clickOutsideEnabled.set(false);
    fixture.detectChanges();

    const outsideEl = fixture.nativeElement.querySelector('#outside-area') as HTMLElement;
    outsideEl.dispatchEvent(new MouseEvent('click', { bubbles: true }));
    expect(host.dismissCount).toBe(0);

    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
    expect(host.dismissCount).toBe(1);
  });
});
