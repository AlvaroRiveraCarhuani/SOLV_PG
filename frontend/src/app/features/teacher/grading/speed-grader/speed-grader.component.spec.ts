import { ComponentFixture, TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { provideRouter } from '@angular/router';
import { of } from 'rxjs';
import { vi, describe, beforeEach, it, expect } from 'vitest';
import { SpeedGraderComponent } from './speed-grader.component';
import { TeacherGradingService } from '../../services/teacher-grading.service';
import { HotkeysService } from '@core/services/hotkeys.service';
import {
  TeacherSubmissionReviewDTO,
  SubmissionComment,
  SubmissionTimeline
} from '../../models/teacher.models';

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
    environment_type: 'JUEZ_EFIMERO',
    language: 'cpp',
    boilerplate: '',
    code: '#include <iostream>\nint main() { return 0; }',
    verdict: 'AC',
    score: 100,
    manual_override: false,
    execution_time_ms: 45,
    memory_used_mb: 18,
    expected_complexity: 'O(N)',
    complexity_analysis: {
      time_complexity: 'O(N)',
      space_complexity: 'O(1)',
      measurements: [
        { input_size: 10, time_ms: 0.1, memory_kb: 64 },
        { input_size: 100, time_ms: 1.0, memory_kb: 64 },
        { input_size: 1000, time_ms: 10.0, memory_kb: 64 }
      ],
      confidence: 0.98
    },
    test_cases: [
      { input: '5 6', expected_output: '12', is_hidden: false, passed: true }
    ],
    files_available: false,
    metrics_available: true,
    files: [],
    base_files: [],
    base_files_available: false,
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

  const reviewSignal = signal<TeacherSubmissionReviewDTO>(mockReview);
  const commentSignal = signal<SubmissionComment[]>([]);

  beforeEach(async () => {
    reviewSignal.set(mockReview);
    commentSignal.set([]);

    const mockGradingService = {
      currentReview: reviewSignal,
      comments: commentSignal,
      isLoading: () => false,
      isRunningEphemeral: () => false,
      ephemeralResult: () => null,
      benchmarkReport: () => null,
      isRunningBenchmark: () => false,
      getSubmissionReview: vi.fn().mockReturnValue(of(mockReview)),
      getSubmissionTimeline: vi.fn().mockReturnValue(of(mockTimeline)),
      getKeystrokeEvents: vi.fn().mockReturnValue(of({
        submission_id: 'rev-1',
        total_time_ms: 5000,
        paste_count: 0,
        paste_percentage: 0,
        total_chars_typed: 40,
        total_chars_pasted: 0,
        events: []
      })),
      addComment: vi.fn().mockReturnValue(of({})),
      runEphemeral: vi.fn(),
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
    component.submissionId.set('rev-1');
    fixture.detectChanges();
  });

  it('should initialize and display review code and metrics', () => {
    expect(component).toBeTruthy();
    expect(component.review()).toEqual(mockReview);
    expect(component.codeLines().length).toBeGreaterThan(0);
  });

  it('resolves judge mode from the exercise environment', () => {
    expect(component.reviewMode()).toBe('judge');
    expect(component.isJudge()).toBe(true);
  });

  it('hides judge-only actions when the submission belongs to an IDE project', () => {
    const gradingService = TestBed.inject(TeacherGradingService);
    reviewSignal.set({ ...mockReview, environment_type: 'IDE_PERSISTENTE', test_cases: [] });
    fixture.detectChanges();

    expect(component.reviewMode()).toBe('project');

    component.toggleDiffView();
    expect(component.showDiffView()).toBe(false);

    component.runEphemeralSandbox();
    expect(gradingService.runEphemeral).not.toHaveBeenCalled();
  });

  it('selects the first IDE file and resets the line when switching files', () => {
    const files = [
      {
        path: 'src/main.py',
        language: 'python',
        content: 'print("main")\n',
        start_line: 2,
        line_count: 1
      },
      {
        path: 'src/helper.py',
        language: 'python',
        content: 'VALUE = 1\n',
        start_line: 4,
        line_count: 1
      }
    ];

    reviewSignal.set({
      ...mockReview,
      environment_type: 'IDE_PERSISTENTE',
      files_available: true,
      files,
      test_cases: []
    });
    commentSignal.set([
      {
        id: 'comment-1',
        submission_id: 'rev-1',
        author_id: 'teacher-1',
        author_name: 'Docente',
        line_number: 2,
        comment: 'Comentario en main.py',
        created_at: new Date().toISOString()
      }
    ]);
    fixture.detectChanges();

    expect(component.showFileTree()).toBe(true);
    expect(component.activeFile()?.path).toBe('src/main.py');
    expect(component.activeFileComments().length).toBe(1);
    expect(component.generalComments()).toEqual([]);

    component.selectLine(2);
    component.selectFile('src/helper.py');

    expect(component.activeFile()?.path).toBe('src/helper.py');
    expect(component.selectedLineNumber()).toBeNull();
    expect(component.activeFileComments()).toEqual([]);
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

  it('keeps the end line only when the selection is a real block', () => {
    component.selectLineRange({ start: 2, end: 4 });
    expect(component.selectedLineNumber()).toBe(2);
    expect(component.selectedLineEndNumber()).toBe(4);
    expect(component.currentCommentRange()).toEqual({ line: 2, endLine: 4 });

    component.selectLine(2);
    expect(component.selectedLineNumber()).toBeNull();
    expect(component.selectedLineEndNumber()).toBeNull();
  });

  it('submits a multiline comment with end_line_number', () => {
    const gradingService = TestBed.inject(TeacherGradingService);
    component.selectLineRange({ start: 2, end: 4 });
    component.newCommentText.set('El bloque no valida el body');

    component.submitComment(component.currentCommentRange()!);

    expect(gradingService.addComment).toHaveBeenCalledWith('rev-1', {
      line_number: 2,
      end_line_number: 4,
      comment: 'El bloque no valida el body'
    });
  });

  it('enables the native diff in IDE only when the active file has a base', () => {
    reviewSignal.set({
      ...mockReview,
      environment_type: 'IDE_PERSISTENTE',
      files_available: true,
      files: [
        { path: 'src/main.py', language: 'python', content: 'print("nuevo")\n', start_line: 2, line_count: 1 }
      ],
      base_files: [
        { path: 'src/main.py', language: 'python', content: 'print("base")\n', start_line: 1, line_count: 1 }
      ],
      base_files_available: true,
      test_cases: []
    });
    fixture.detectChanges();

    expect(component.canShowDiff()).toBe(true);
    expect(component.diffOriginal()).toContain('base');
    expect(component.diffModified()).toContain('nuevo');

    component.toggleDiffView();
    expect(component.showDiffView()).toBe(true);
  });

  it('opens and closes override modal', () => {
    component.openOverrideModal();
    expect(component.showOverrideModal()).toBe(true);

    component.closeOverrideModal();
    expect(component.showOverrideModal()).toBe(false);
  });

  it('submits manual override with optional justification defaulting to fallback reason', () => {
    const gradingService = TestBed.inject(TeacherGradingService);
    component.openOverrideModal();
    component.overrideReason.set('');
    component.overrideScore.set(95);
    component.overrideVerdict.set('AC');

    component.submitOverride();

    expect(gradingService.overrideSubmission).toHaveBeenCalledWith('rev-1', {
      verdict: 'AC',
      override_reason: 'Ajuste manual docente',
      score: 95
    });
    expect(component.showOverrideModal()).toBe(false);
  });

  it('returns correct CSS classes for complexity badges', () => {
    expect(component.getComplexityClass('O(1)')).toBe('badge-success');
    expect(component.getComplexityClass('O(N)')).toBe('badge-success');
    expect(component.getComplexityClass('O(N log N)')).toBe('badge-warning');
    expect(component.getComplexityClass('O(N²)')).toBe('badge-danger');
    expect(component.getComplexityClass('unknown')).toBe('badge-secondary');
  });

  it('opens and closes keystroke replay modal', () => {
    expect(component.showKeystrokeReplay()).toBe(false);
    component.openKeystrokeReplay();
    expect(component.showKeystrokeReplay()).toBe(true);
    component.closeKeystrokeReplay();
    expect(component.showKeystrokeReplay()).toBe(false);
  });
});

