import { ComponentFixture, TestBed } from '@angular/core/testing';
import { CourseColorPickerComponent } from './course-color-picker.component';
import { CourseColorService } from '@core/services/course-color.service';

describe('CourseColorPickerComponent', () => {
  let component: CourseColorPickerComponent;
  let fixture: ComponentFixture<CourseColorPickerComponent>;
  let service: CourseColorService;

  beforeEach(async () => {
    localStorage.clear();
    await TestBed.configureTestingModule({
      imports: [CourseColorPickerComponent],
      providers: [CourseColorService]
    }).compileComponents();

    fixture = TestBed.createComponent(CourseColorPickerComponent);
    component = fixture.componentInstance;
    service = TestBed.inject(CourseColorService);
    fixture.detectChanges();
  });

  afterEach(() => {
    localStorage.clear();
  });

  it('should create and display curated presets', () => {
    expect(component).toBeTruthy();
    expect(component.presets.length).toBeGreaterThan(0);
  });

  it('should select preset and emit colorChange', () => {
    let emittedColor = '';
    component.colorChange.subscribe(c => (emittedColor = c));

    const preset = component.presets[1];
    component.selectPreset(preset.hex);

    expect(component.currentColor()).toBe(preset.hex.toUpperCase());
    expect(emittedColor).toBe(preset.hex.toUpperCase());
  });

  it('should handle custom color input', () => {
    let emittedColor = '';
    component.colorChange.subscribe(c => (emittedColor = c));

    const event = { target: { value: '#ff00aa' } } as unknown as Event;
    component.onCustomColorInput(event);

    expect(component.currentColor()).toBe('#FF00AA');
    expect(component.isCustom()).toBe(true);
    expect(emittedColor).toBe('#FF00AA');
  });

  it('should reset to default color', () => {
    component.courseId = 'course-xyz';
    service.setCourseColor('course-xyz', '#123456');
    component.currentColor.set('#123456');

    component.resetToDefault();
    expect(component.currentColor()).not.toBe('#123456');
  });
});
