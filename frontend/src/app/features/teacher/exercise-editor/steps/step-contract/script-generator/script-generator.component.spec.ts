import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ScriptGeneratorComponent } from './script-generator.component';
import { TeacherCourseService } from '../../../../services/teacher-course.service';
import { of } from 'rxjs';
import { ScriptGenerationResponse } from '../../../../models/teacher.models';

describe('ScriptGeneratorComponent', () => {
  let component: ScriptGeneratorComponent;
  let fixture: ComponentFixture<ScriptGeneratorComponent>;
  let mockTeacherCourseService: jasmine.SpyObj<TeacherCourseService>;

  const mockResponse: ScriptGenerationResponse = {
    mode: 'dry_run',
    execution_time_ms: 120,
    cases_count: 1,
    can_import: true,
    total_valid: 1,
    total_invalid: 0,
    cases: [
      {
        index: 0,
        input: '5\n',
        expected_output: '15\n',
        valid: true
      }
    ]
  };

  beforeEach(async () => {
    mockTeacherCourseService = jasmine.createSpyObj('TeacherCourseService', ['generateCasesFromScript']);
    mockTeacherCourseService.generateCasesFromScript.and.returnValue(of(mockResponse));

    await TestBed.configureTestingModule({
      imports: [ScriptGeneratorComponent],
      providers: [
        { provide: TeacherCourseService, useValue: mockTeacherCourseService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(ScriptGeneratorComponent);
    component = fixture.componentInstance;
    component.exerciseId = 'ex-123';
    fixture.detectChanges();
  });

  it('should create component', () => {
    expect(component).toBeTruthy();
  });

  it('should toggle expand state', () => {
    expect(component.isExpanded()).toBeFalse();
    component.toggleExpand();
    expect(component.isExpanded()).toBeTrue();
  });

  it('should execute script in dry run mode', () => {
    component.executeScript();

    expect(mockTeacherCourseService.generateCasesFromScript).toHaveBeenCalledWith('ex-123', component.scriptCode(), true);
    expect(component.previewResult()).toEqual(mockResponse);
  });

  it('should confirm and emit generated cases', () => {
    component.previewResult.set(mockResponse);

    const actualResponse: ScriptGenerationResponse = {
      mode: 'import',
      can_import: true,
      imported_count: 1
    };
    mockTeacherCourseService.generateCasesFromScript.and.returnValue(of(actualResponse));

    spyOn(component.casesGenerated, 'emit');

    component.confirmAndAdd();

    expect(mockTeacherCourseService.generateCasesFromScript).toHaveBeenCalledWith('ex-123', component.scriptCode(), false);
    expect(component.casesGenerated.emit).toHaveBeenCalled();
  });
});
