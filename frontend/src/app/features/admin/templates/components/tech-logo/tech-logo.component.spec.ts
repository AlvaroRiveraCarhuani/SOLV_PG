import { ComponentFixture, TestBed } from '@angular/core/testing';
import { TechLogoComponent } from './tech-logo.component';

describe('TechLogoComponent', () => {
  let component: TechLogoComponent;
  let fixture: ComponentFixture<TechLogoComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [TechLogoComponent]
    }).compileComponents();

    fixture = TestBed.createComponent(TechLogoComponent);
    component = fixture.componentInstance;
  });

  it('should resolve known ecosystem keys to static official SVG assets in /tech-icons/<key>.svg', () => {
    const knownIcons = [
      'python',
      'angular',
      'react',
      'vue',
      'fastapi',
      'django',
      'postgres',
      'mysql',
      'redis',
      'bun',
      'flask',
      'laravel',
      'sqlite',
      'express',
      'nestjs',
      'prisma',
      'typeorm',
      'mariadb'
    ];
    for (const icon of knownIcons) {
      component.iconType = icon;
      fixture.detectChanges();
      const imgEl: HTMLImageElement | null = fixture.nativeElement.querySelector('img.tech-logo-img');
      expect(imgEl).toBeTruthy();
      expect(imgEl?.getAttribute('src')).toBe(`/tech-icons/${icon}.svg`);
    }
  });

  it('should render custom SVG URL or data URI and fallback to monogram on image load error', () => {
    component.iconType = 'https://cdn.example.edu/icons/zig.svg';
    fixture.detectChanges();

    const imgEl: HTMLImageElement | null = fixture.nativeElement.querySelector('img.tech-logo-img');
    expect(imgEl).toBeTruthy();
    expect(imgEl?.getAttribute('src')).toBe('https://cdn.example.edu/icons/zig.svg');

    imgEl?.dispatchEvent(new Event('error'));
    fixture.detectChanges();

    const afterErrImg = fixture.nativeElement.querySelector('img.tech-logo-img');
    expect(afterErrImg).toBeNull();

    const monogramEl: HTMLElement | null = fixture.nativeElement.querySelector('.tech-logo-monogram');
    expect(monogramEl).toBeTruthy();
    expect(monogramEl?.textContent?.trim()).toBe('ZI');
  });

  it('should render deterministic 2-letter monogram immediately when an unknown key is passed', () => {
    component.iconType = 'elixir';
    fixture.detectChanges();

    const imgEl = fixture.nativeElement.querySelector('img.tech-logo-img');
    expect(imgEl).toBeNull();

    const monogramEl: HTMLElement | null = fixture.nativeElement.querySelector('.tech-logo-monogram');
    expect(monogramEl).toBeTruthy();
    expect(monogramEl?.textContent?.trim()).toBe('EL');
  });
});
