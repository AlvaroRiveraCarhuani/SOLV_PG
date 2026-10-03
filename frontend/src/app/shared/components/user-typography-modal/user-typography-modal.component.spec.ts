import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { UserTypographyModalComponent } from './user-typography-modal.component';
import { UserTypographyService } from '@core/services/user-typography.service';
import { TenantService } from '@core/services/tenant.service';

describe('UserTypographyModalComponent', () => {
  let component: UserTypographyModalComponent;
  let fixture: ComponentFixture<UserTypographyModalComponent>;
  let userTypoService: UserTypographyService;

  beforeEach(async () => {
    localStorage.clear();

    await TestBed.configureTestingModule({
      imports: [UserTypographyModalComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        TenantService,
        UserTypographyService
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(UserTypographyModalComponent);
    component = fixture.componentInstance;
    userTypoService = TestBed.inject(UserTypographyService);
    fixture.detectChanges();
  });

  afterEach(() => {
    localStorage.clear();
  });

  it('should create and load curated sans and mono options', () => {
    expect(component).toBeTruthy();
    expect(component.sansOptions().length).toBeGreaterThan(1);
    expect(component.monoOptions().length).toBeGreaterThan(1);
  });

  it('should select sans option and update service', () => {
    component.onSansSelected({ id: 'open-sans', label: 'Open Sans', value: 'open-sans' });
    expect(userTypoService.sansPreference()).toBe('open-sans');
    expect(component.selectedSansSlug()).toBe('open-sans');
    expect(component.selectedSansLabel()).toBe('Open Sans');
  });

  it('should select mono option and update service', () => {
    component.onMonoSelected({ id: 'fira-code', label: 'Fira Code', value: 'fira-code' });
    expect(userTypoService.monoPreference()).toBe('fira-code');
    expect(component.selectedMonoSlug()).toBe('fira-code');
    expect(component.selectedMonoLabel()).toBe('Fira Code');
  });

  it('should reset typography to default institutional fonts', () => {
    userTypoService.setSansPreference('lato');
    expect(component.selectedSansSlug()).toBe('lato');

    component.resetToDefault();
    expect(component.selectedSansSlug()).toBe('');
    expect(component.selectedSansLabel()).toBe('Predeterminada Institucional');
  });
});
