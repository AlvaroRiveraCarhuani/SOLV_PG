import { Component, signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { SparklineComponent } from './sparkline.component';

@Component({
  standalone: true,
  imports: [SparklineComponent],
  template: `
    <sparkline [data]="samples()" [color]="color()" [filled]="isFilled()" />
  `
})
class SparklineHostComponent {
  samples = signal<number[]>([10, 20, 15, 30, 45, 40]);
  color = signal<string>('var(--tenant-primary)');
  isFilled = signal<boolean>(true);
}

describe('SparklineComponent', () => {
  let fixture: ComponentFixture<SparklineHostComponent>;
  let host: SparklineHostComponent;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [SparklineHostComponent]
    }).compileComponents();

    fixture = TestBed.createComponent(SparklineHostComponent);
    host = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('renders SVG path when data has multiple samples', () => {
    const strokePath = fixture.nativeElement.querySelector('.sparkline-stroke') as SVGPathElement;
    expect(strokePath).not.toBeNull();
    expect(strokePath.getAttribute('d')).toContain('M');

    const fillPath = fixture.nativeElement.querySelector('.sparkline-fill') as SVGPathElement;
    expect(fillPath).not.toBeNull();
  });

  it('renders baseline when data is empty or has only 1 point', () => {
    host.samples.set([]);
    fixture.detectChanges();

    const baseline = fixture.nativeElement.querySelector('.sparkline-baseline');
    expect(baseline).not.toBeNull();
  });

  it('renders endpoint dot when multi-point data is present', () => {
    const dot = fixture.nativeElement.querySelector('.sparkline-dot') as SVGCircleElement;
    expect(dot).not.toBeNull();
    expect(Number(dot.getAttribute('cx'))).toBeGreaterThan(0);
  });
});
