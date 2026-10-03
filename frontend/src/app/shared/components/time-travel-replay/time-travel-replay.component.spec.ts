import { ComponentFixture, TestBed } from '@angular/core/testing';
import { TimeTravelReplayComponent } from './time-travel-replay.component';
import { SubmissionTimeline } from '../../../features/teacher/models/teacher.models';

describe('TimeTravelReplayComponent', () => {
  let component: TimeTravelReplayComponent;
  let fixture: ComponentFixture<TimeTravelReplayComponent>;

  const mockTimeline: SubmissionTimeline = {
    submission_id: 'sub-1',
    student_id: 'std-1',
    student_name: 'Mateo Quispe',
    total_duration_seconds: 180,
    total_keystrokes: 350,
    paste_events_count: 2,
    paste_percentage: 45,
    suspicious_paste_flag: true,
    keyframes: [
      {
        offset_ms: 1000,
        action: 'insert',
        content: '#include <iostream>',
        cursor_line: 1,
        is_paste: false,
        char_count: 19,
        ast_node_count: 2,
        cyclomatic_complexity: 1
      },
      {
        offset_ms: 2500,
        action: 'paste',
        content: '#include <iostream>\n\nint main() { return 0; }',
        cursor_line: 3,
        is_paste: true,
        char_count: 45,
        ast_node_count: 5,
        cyclomatic_complexity: 1
      },
      {
        offset_ms: 4000,
        action: 'checkpoint',
        content: '#include <iostream>\n\nint main() {\n    std::cout << "OK";\n    return 0;\n}',
        cursor_line: 4,
        is_paste: false,
        char_count: 75,
        ast_node_count: 8,
        cyclomatic_complexity: 1
      }
    ]
  };

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [TimeTravelReplayComponent]
    }).compileComponents();

    fixture = TestBed.createComponent(TimeTravelReplayComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('timeline', mockTimeline);
    fixture.componentRef.setInput('initialBoilerplate', '// Starter code');
    fixture.detectChanges();
  });

  it('should initialize at the latest keyframe and calculate metrics', () => {
    expect(component).toBeTruthy();
    expect(component.currentKeyframeIndex()).toBe(2);
    expect(component.maxKeyframeIndex()).toBe(2);
    expect(component.currentCodeContent()).toContain('std::cout << "OK"');
  });

  it('allows stepping forward and backward', () => {
    component.stepBackward();
    expect(component.currentKeyframeIndex()).toBe(1);
    expect(component.currentKeyframe()?.is_paste).toBe(true);

    component.stepForward();
    expect(component.currentKeyframeIndex()).toBe(2);
  });

  it('handles play, pause, speed and reset', () => {
    component.resetPlayback();
    expect(component.currentKeyframeIndex()).toBe(0);

    component.setSpeed(2);
    expect(component.speed()).toBe(2);

    component.togglePlayPause();
    expect(component.isPlaying()).toBe(true);

    component.togglePlayPause();
    expect(component.isPlaying()).toBe(false);
  });

  it('computes diff lines when diff mode is toggled', () => {
    component.toggleDiff();
    expect(component.showDiff()).toBe(true);
    expect(component.diffLines().length).toBeGreaterThan(0);
  });
});
