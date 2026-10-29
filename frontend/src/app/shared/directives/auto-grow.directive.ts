import {
  AfterViewInit,
  Directive,
  ElementRef,
  HostListener,
  NgZone,
  OnDestroy,
  inject,
  input
} from '@angular/core';

@Directive({
  selector: 'textarea[autoGrow]',
  standalone: true
})
export class AutoGrowDirective implements AfterViewInit, OnDestroy {
  private readonly elementRef = inject<ElementRef<HTMLTextAreaElement>>(ElementRef);
  private readonly ngZone = inject(NgZone);
  private resizeObserver?: ResizeObserver;

  readonly minRows = input<number>(3);
  readonly maxRows = input<number>(5);

  ngAfterViewInit(): void {
    const textarea = this.elementRef.nativeElement;
    if (!textarea) {
      return;
    }

    this.ngZone.runOutsideAngular(() => {
      this.adjust();

      // En modales o vistas animadas, el cálculo de dimensiones finales se estabiliza tras el frame inicial
      if (typeof requestAnimationFrame !== 'undefined') {
        requestAnimationFrame(() => this.adjust());
      }

      if (typeof ResizeObserver !== 'undefined') {
        let lastWidth = 0;
        this.resizeObserver = new ResizeObserver((entries) => {
          for (const entry of entries) {
            const width = Math.round(entry.contentRect.width);
            if (width !== lastWidth && width > 0) {
              lastWidth = width;
              this.adjust();
            }
          }
        });
        this.resizeObserver.observe(textarea);
      }
    });
  }

  ngOnDestroy(): void {
    this.resizeObserver?.disconnect();
  }

  @HostListener('input')
  onInput(): void {
    this.adjust();
  }

  @HostListener('focus')
  onFocus(): void {
    this.adjust();
  }

  adjust(): void {
    const textarea = this.elementRef.nativeElement;
    if (!textarea || typeof window === 'undefined') {
      return;
    }

    const computed = window.getComputedStyle(textarea);
    const paddingTop = parseFloat(computed.paddingTop) || 0;
    const paddingBottom = parseFloat(computed.paddingBottom) || 0;
    const borderTop = parseFloat(computed.borderTopWidth) || 0;
    const borderBottom = parseFloat(computed.borderBottomWidth) || 0;
    const verticalExtras = paddingTop + paddingBottom + borderTop + borderBottom;

    let lineHeight = parseFloat(computed.lineHeight);
    if (isNaN(lineHeight) || lineHeight <= 0) {
      const fontSize = parseFloat(computed.fontSize) || 14;
      lineHeight = fontSize * 1.5;
    }

    const minHeight = this.minRows() * lineHeight + verticalExtras;
    const maxHeight = this.maxRows() * lineHeight + verticalExtras;

    // Resetea momentáneamente la altura inline para medir el scrollHeight real del texto
    textarea.style.height = 'auto';

    const scrollHeightWithBorders = textarea.scrollHeight + borderTop + borderBottom;
    const targetHeight = Math.min(Math.max(scrollHeightWithBorders, minHeight), maxHeight);

    textarea.style.height = `${Math.ceil(targetHeight)}px`;
    textarea.style.minHeight = `${Math.ceil(minHeight)}px`;
    textarea.style.maxHeight = `${Math.ceil(maxHeight)}px`;
    textarea.style.overflowY = scrollHeightWithBorders > maxHeight + 1 ? 'auto' : 'hidden';
  }
}
