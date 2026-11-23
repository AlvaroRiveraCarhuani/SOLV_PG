import { TestBed } from '@angular/core/testing';
import { ThemeService } from './theme.service';

describe('ThemeService', () => {
  let service: ThemeService;

  beforeEach(() => {
    localStorage.clear();
    document.documentElement.removeAttribute('data-theme');
    TestBed.configureTestingModule({
      providers: [ThemeService]
    });
    service = TestBed.inject(ThemeService);
  });

  afterEach(() => {
    localStorage.clear();
    document.documentElement.removeAttribute('data-theme');
  });

  it('permite alternar entre tema claro y oscuro', () => {
    // Si inicia en light (o system sin dark match)
    service.setTheme('light');
    expect(service.isDark()).toBe(false);
    expect(document.documentElement.getAttribute('data-theme')).toBeNull();

    service.toggleTheme();
    expect(service.isDark()).toBe(true);
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark');

    service.toggleTheme();
    expect(service.isDark()).toBe(false);
    expect(document.documentElement.getAttribute('data-theme')).toBeNull();
  });

  it('persiste la preferencia en localStorage', () => {
    service.setTheme('dark');
    expect(localStorage.getItem('solv_theme_preference')).toBe('dark');

    service.setTheme('light');
    expect(localStorage.getItem('solv_theme_preference')).toBe('light');
  });

  it('aplica data-theme="dark" y color-scheme dark en el elemento raíz', () => {
    service.setTheme('dark');
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark');
    expect(document.documentElement.style.colorScheme).toBe('dark');

    service.setTheme('light');
    expect(document.documentElement.getAttribute('data-theme')).toBeNull();
    expect(document.documentElement.style.colorScheme).toBe('light');
  });
});
