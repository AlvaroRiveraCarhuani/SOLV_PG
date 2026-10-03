import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { of } from 'rxjs';
import { SpeedGraderComponent } from './speed-grader.component';
import { TeacherGradingService } from '../../services/teacher-grading.service';
import { HotkeysService } from '@core/services/hotkeys.service';
import { TeacherSubmissionReviewDTO, SubmissionTimeline } from '../../models/teacher.models';

describe('SpeedGraderComponent', () => {
  let component: SpeedGraderComponent;
  let fixture: ComponentFixture<SpeedGraderComponent>;

  const mockReview: TeacherSubmissionReviewDTO = {
    id: 'rev-1',
    exercise_id: 'ex-1',
    exercise_title: 'Algoritmo de Dijkstra',
    subject_id: 'sub-1',
    subject_name: 'Grafos y Algoritmos',
    student_id: 'std-1',
    student_name: 'Santiago Morales',
    student_email: 'santiago@uab.edu.bo',
    code: '#include <iostream>\nint main() { return 0; }',
    verdict: 'AC',
    score: 100,
    manual_override: false,
    execution_time_ms: 45,
    memory_used_mb: 18,
    test_cases: [
      { input: '5 6', expected_output: '12', is_hidden: false, passed: true }
    ],
    comments: [],
    submitted_at: new Date().toISOString()
  };

  const mockTimeline: SubmissionTimeline = {
    submission_id: 'rev-1',
    student_id: 'std-1',
    student_name: 'Santiago Morales',
    total_duration_seconds: 150,
    total_keystrokes: 280,
    paste_events_count: 0,
    paste_percentage: 0,
    suspicious_paste_flag: false,
    keyframes: [
      {
        offset_ms: 1000,
        action: 'insert',
        content: '#include <iostream>',
        cursor_line: 1,
        is_paste: false,
        char_count: 19
      }
    ]
  };

  beforeEach(async () => {
    const mockGradingService = {
      currentReview: () => mockReview,
      comments: () => [],
      isLoading: () => false,
      isRunningEphemeral: () => false,
      ephemeralResult: () => null,
      getSubmissionReview: vi.fn().mockReturnValue(of(mockReview)),
      getSubmissionTimeline: vi.fn().mockReturnValue(of(mockTimeline)),
      addComment: vi.fn().mockReturnValue(of({})),
      overrideSubmission: vi.fn().mockReturnValue(of(void 0))
    };

    await TestBed.configureTestingModule({
      imports: [SpeedGraderComponent],
      providers: [
        provideRouter([]),
        HotkeysService,
        { provide: TeacherGradingService, useValue: mockGradingService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(SpeedGraderComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should initialize and display review code and metrics', () => {
    expect(component).toBeTruthy();
    expect(component.review()).toEqual(mockReview);
    expect(component.codeLines().length).toBeGreaterThan(0);
  });

  it('toggles time-travel replay view and loads timeline', () => {
    expect(component.showReplayPlayer()).toBe(false);
    component.toggleReplay();
    expect(component.showReplayPlayer()).toBe(true);
  });

  it('handles line selection for comments', () => {
    component.selectLine(2);
    expect(component.selectedLineNumber()).toBe(2);

    component.selectLine(2);
    expect(component.selectedLineNumber()).toBeNull();
  });

  it('opens and closes override modal', () => {
    component.openOverrideModal();
    expect(component.showOverrideModal()).toBe(true);

    component.closeOverrideModal();
    expect(component.showOverrideModal()).toBe(false);
  });
});
