import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { UserTypographyService } from './user-typography.service';
import { TenantService } from './tenant.service';

describe('UserTypographyService', () => {
  let service: UserTypographyService;
  let tenantService: TenantService;

  beforeEach(() => {
    localStorage.clear();
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        TenantService,
        UserTypographyService
      ]
    });

    service = TestBed.inject(UserTypographyService);
    tenantService = TestBed.inject(TenantService);
  });

  afterEach(() => {
    localStorage.clear();
  });

  it('should be created and list curated sans and mono options', () => {
    expect(service).toBeTruthy();
    expect(service.sansOptions.length).toBeGreaterThan(0);
    expect(service.monoOptions.length).toBeGreaterThan(0);
  });

  it('should update and persist sans font preference', () => {
    service.setSansPreference('roboto');
    expect(service.sansPreference()).toBe('roboto');

    const rootFontSans = document.documentElement.style.getPropertyValue('--font-sans');
    expect(rootFontSans).toContain('Roboto');
  });

  it('should update and persist mono font preference', () => {
    service.setMonoPreference('fira-code');
    expect(service.monoPreference()).toBe('fira-code');

    const rootFontMono = document.documentElement.style.getPropertyValue('--font-mono');
    expect(rootFontMono).toContain('Fira Code');
  });

  it('should reset preferences to institutional default', () => {
    service.setSansPreference('lato');
    service.setMonoPreference('ibm-plex-mono');
    expect(service.sansPreference()).toBe('lato');

    service.resetToInstitutional();
    expect(service.sansPreference()).toBeNull();
    expect(service.monoPreference()).toBeNull();
  });
});
