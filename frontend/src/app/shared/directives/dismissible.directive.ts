import { Directive, ElementRef, EventEmitter, HostListener, Input, Output, inject } from '@angular/core';

@Directive({
  selector: '[dismissible]',
  standalone: true
})
export class DismissibleDirective {
  private elementRef = inject(ElementRef);

  @Input() dismissibleEnabled = true;
  @Input() dismissOnEscape = true;
  @Input() dismissOnClickOutside = true;
  @Input() dismissExclude?: HTMLElement | HTMLElement[] | null = null;

  @Output() readonly dismiss = new EventEmitter<void>();

  @HostListener('document:keydown.escape')
  onEscape(): void {
    if (!this.dismissibleEnabled || !this.dismissOnEscape) {
      return;
    }
    this.dismiss.emit();
  }

  @HostListener('document:click', ['$event'])
  onClickOutside(event: MouseEvent): void {
    if (!this.dismissibleEnabled || !this.dismissOnClickOutside) {
      return;
    }

    const target = event.target as Node | null;
    if (!target) {
      return;
    }

    // Don't dismiss if the click happened inside the dismissible element itself
    if (this.elementRef.nativeElement.contains(target)) {
      return;
    }

    // Don't dismiss if the click happened inside an excluded element (e.g. trigger button)
    const exclude = this.dismissExclude;
    if (exclude) {
      if (Array.isArray(exclude)) {
        if (exclude.some(el => el && el.contains(target))) {
          return;
        }
      } else if (exclude.contains(target)) {
        return;
      }
    }

    this.dismiss.emit();
  }
}
