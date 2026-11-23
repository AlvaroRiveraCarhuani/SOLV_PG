import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting, HttpTestingController } from '@angular/common/http/testing';
import { provideRouter } from '@angular/router';
import { AdminManualComponent } from './admin-manual.component';

const mockMarkdown = `# Manual de Administración del Sistema — SOLV

Guía operativa para administradores.

## 1. Visión General

### 1.1 Propósito
Contenido de propósito.

## 2. Entornos de Laboratorio

Contenido de entornos.
`;

interface ScrollHarness {
  attachScrollTracking(): void;
  updateScrollState(): void;
}

interface RectStub {
  top: number;
}

function rect(top: number, height: number): DOMRect {
  return {
    top,
    height,
    bottom: top + height,
    left: 0,
    right: 0,
    width: 0,
    x: 0,
    y: top,
    toJSON: () => ({})
  } as DOMRect;
}

describe('AdminManualComponent — scrollspy y progreso', () => {
  let fixture: ComponentFixture<AdminManualComponent>;
  let httpMock: HttpTestingController;
  let wrapper: HTMLElement;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [AdminManualComponent],
      providers: [provideHttpClient(), provideHttpClientTesting(), provideRouter([])]
    }).compileComponents();

    fixture = TestBed.createComponent(AdminManualComponent);
    httpMock = TestBed.inject(HttpTestingController);
    fixture.detectChanges();

    httpMock.expectOne('/api/v1/admin/manual').flush(mockMarkdown);
    fixture.detectChanges();
  });

  afterEach(() => {
    wrapper?.remove();
    httpMock.verify();
  });

  function mountScrollableWrapper(): HTMLElement {
    wrapper = document.createElement('div');
    wrapper.style.overflowY = 'auto';
    document.body.appendChild(wrapper);
    wrapper.appendChild(fixture.nativeElement as HTMLElement);

    Object.defineProperty(wrapper, 'scrollHeight', { configurable: true, value: 2000 });
    Object.defineProperty(wrapper, 'clientHeight', { configurable: true, value: 500 });
    Object.defineProperty(wrapper, 'scrollTop', {
      configurable: true,
      writable: true,
      value: 0
    });
    wrapper.getBoundingClientRect = () => rect(0, 500);
    return wrapper;
  }

  function stubHeadingPositions(tops: number[]): HTMLElement[] {
    const host = fixture.nativeElement as HTMLElement;
    const headings = Array.from(
      host.querySelectorAll<HTMLElement>('.manual-content-card h2, .manual-content-card h3')
    );
    headings.forEach((heading, index) => {
      const top = tops[index] ?? 5000;
      heading.getBoundingClientRect = () => rect(top, 32);
    });
    return headings;
  }

  it('usa el ancestro scrolleable real y actualiza progreso y botón de volver arriba', () => {
    const container = mountScrollableWrapper();
    container.scrollTop = 500;

    (fixture.componentInstance as unknown as ScrollHarness).attachScrollTracking();
    fixture.detectChanges();

    expect(fixture.componentInstance.readingProgress()).toBe(33);
    expect(fixture.componentInstance.showBackToTop()).toBe(true);
  });

  it('actualiza la sección activa al desplazarse el contenedor', () => {
    const container = mountScrollableWrapper();
    const headings = stubHeadingPositions([10, 900, 1200]);
    const harness = fixture.componentInstance as unknown as ScrollHarness;

    harness.attachScrollTracking();
    fixture.detectChanges();
    expect(fixture.componentInstance.activeSectionId()).toBe(headings[0].id);

    // La segunda sección cruza el umbral superior del contenedor.
    stubHeadingPositions([-400, 120, 1200]);
    container.scrollTop = 900;
    harness.updateScrollState();
    fixture.detectChanges();

    expect(fixture.componentInstance.activeSectionId()).toBe(headings[1].id);
    expect(fixture.componentInstance.readingProgress()).toBe(60);
  });
});
