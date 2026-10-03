import { TestBed } from '@angular/core/testing';
import { CourseColorService, CURATED_COURSE_PALETTE } from './course-color.service';

describe('CourseColorService', () => {
  let service: CourseColorService;

  beforeEach(() => {
    localStorage.clear();
    TestBed.configureTestingModule({
      providers: [CourseColorService]
    });
    service = TestBed.inject(CourseColorService);
  });

  afterEach(() => {
    localStorage.clear();
  });

  it('should be created and return curated presets', () => {
    expect(service).toBeTruthy();
    expect(service.getPresets().length).toBe(CURATED_COURSE_PALETTE.length);
  });

  it('should return a deterministic curated color when no custom color is set', () => {
    const color1 = service.getCourseColor('course-101', 'CS101');
    const color2 = service.getCourseColor('course-101', 'CS101');
    expect(color1).toBe(color2);
    expect(CURATED_COURSE_PALETTE.some(p => p.hex === color1)).toBe(true);
  });

  it('should set and persist custom color', () => {
    service.setCourseColor('course-101', '#FF5500');
    const retrieved = service.getCourseColor('course-101');
    expect(retrieved).toBe('#FF5500');

    // Reset should revert to default
    service.resetCourseColor('course-101');
    expect(service.getCourseColor('course-101')).not.toBe('#FF5500');
  });

  it('should calculate safe theme styles with RGBA backgrounds and borders', () => {
    const style = service.getCourseThemeStyle('#6366F1');
    expect(style.accent).toBe('#6366F1');
    expect(style.accentBg).toContain('rgba(99, 102, 241, 0.12)');
    expect(style.accentBorder).toContain('rgba(99, 102, 241, 0.35)');
  });
});
