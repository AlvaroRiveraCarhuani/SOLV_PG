import { ComponentFixture, TestBed } from '@angular/core/testing';
import { of, throwError } from 'rxjs';
import { vi, describe, beforeEach, it, expect } from 'vitest';
import { KeystrokeReplayModalComponent } from './keystroke-replay-modal.component';
import { TeacherGradingService } from '../../../services/teacher-grading.service';
import { SubmissionKeystrokeReport } from '../../../models/teacher.models';

describe('KeystrokeReplayModalComponent', () => {
  let component: KeystrokeReplayModalComponent;
  let fixture: ComponentFixture<KeystrokeReplayModalComponent>;
  let mockGradingService: any;

  const mockReport: SubmissionKeystrokeReport = {
    submission_id: 'sub-123',
    total_time_ms: 12000,
    paste_count: 1,
    paste_percentage: 45.0,
    total_chars_typed: 55,
    total_chars_pasted: 45,
    events: [
      {
        id: 'ev-1',
        submission_id: 'sub-123',
        timestamp_ms: 0,
        event_type: 'insert',
        position: 0,
        content: 'def solve():\n',
        paste_source_detected: false,
        created_at: '2026-10-06T12:00:00Z'
      },
      {
        id: 'ev-2',
        submission_id: 'sub-123',
        timestamp_ms: 5000,
        event_type: 'paste',
        position: 13,
        content: '    return sum(range(10))\n',
        paste_source_detected: true,
        created_at: '2026-10-06T12:00:05Z'
      }
    ]
  };

  beforeEach(async () => {
    mockGradingService = {
      getKeystrokeEvents: vi.fn().mockReturnValue(of(mockReport))
    };

    await TestBed.configureTestingModule({
      imports: [KeystrokeReplayModalComponent],
      providers: [
        { provide: TeacherGradingService, useValue: mockGradingService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(KeystrokeReplayModalComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('submissionId', 'sub-123');
    fixture.detectChanges();
  });

  it('should create and load keystroke report', () => {
    expect(component).toBeTruthy();
    expect(mockGradingService.getKeystrokeEvents).toHaveBeenCalledWith('sub-123');
    expect(component.report()).toEqual(mockReport);
    expect(component.totalDurationMS()).toBe(12000);
  });

  it('should toggle playback and step controls', () => {
    expect(component.isPlaying()).toBe(false);
    component.togglePlay();
    expect(component.isPlaying()).toBe(true);
    component.togglePlay();
    expect(component.isPlaying()).toBe(false);

    component.stepForward();
    expect(component.currentTimeMS()).toBe(5000);

    component.stepBackward();
    expect(component.currentTimeMS()).toBe(0);
  });

  it('should adjust speed and handle errors', () => {
    component.setSpeed(2);
    expect(component.playbackSpeed()).toBe(2);

    mockGradingService.getKeystrokeEvents.mockReturnValue(throwError(() => new Error('API error')));
    component.loadKeystrokeEvents();
    expect(component.error()).toBeTruthy();
  });
});
