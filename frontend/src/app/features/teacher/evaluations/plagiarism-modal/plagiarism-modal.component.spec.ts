import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { of } from 'rxjs';
import { PlagiarismModalComponent } from './plagiarism-modal.component';
import { TeacherCourseService } from '../../services/teacher-course.service';
import { TeacherGradingService } from '../../services/teacher-grading.service';
import { PlagiarismReport, SubmissionTimeline } from '../../models/teacher.models';

describe('PlagiarismModalComponent', () => {
  let component: PlagiarismModalComponent;
  let fixture: ComponentFixture<PlagiarismModalComponent>;

  const mockReport: PlagiarismReport = {
    subject_id: 'sub-1',
    subject_name: 'Estructuras de Datos',
    analyzed_at: new Date().toISOString(),
    total_submissions: 25,
    suspect_pairs_count: 1,
    matches: [
      {
        submission_id_a: 'sub-a',
        student_id_a: 'std-a',
        student_name_a: 'Ana García',
        submission_id_b: 'sub-b',
        student_id_b: 'std-b',
        student_name_b: 'Carlos Gómez',
        exercise_id: 'ex-1',
        exercise_title: 'Árboles Binarios AVL',
        similarity: 88,
        risk_level: 'critical',
        matching_tokens: 120,
        total_tokens_a: 140,
        total_tokens_b: 145,
        common_structures: ['AVL::rotateLeft', 'AVL::rebalance']
      }
    ]
  };

  const mockTimeline: SubmissionTimeline = {
    submission_id: 'sub-a',
    student_id: 'std-a',
    student_name: 'Ana García',
    total_duration_seconds: 120,
    total_keystrokes: 250,
    paste_events_count: 1,
    paste_percentage: 50,
    suspicious_paste_flag: true,
    keyframes: [
      {
        offset_ms: 1000,
        action: 'paste',
        content: 'class AVL {}',
        cursor_line: 1,
        is_paste: true,
        char_count: 12
      }
    ]
  };

  beforeEach(async () => {
    const mockCourseService = {
      analyzePlagiarism: vi.fn().mockReturnValue(of(mockReport))
    };

    const mockGradingService = {
      getSubmissionTimeline: vi.fn().mockReturnValue(of(mockTimeline))
    };

    await TestBed.configureTestingModule({
      imports: [PlagiarismModalComponent],
      providers: [
        provideRouter([]),
        { provide: TeacherCourseService, useValue: mockCourseService },
        { provide: TeacherGradingService, useValue: mockGradingService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(PlagiarismModalComponent);
    component = fixture.componentInstance;
    component.subjectId = 'sub-1';
    fixture.detectChanges();
  });

  it('should initialize and load plagiarism report matches', () => {
    expect(component).toBeTruthy();
    expect(component.report()).toEqual(mockReport);
    expect(component.getCriticalCount()).toBe(1);
    expect(component.getWarningCount()).toBe(0);
    expect(component.selectedMatch()).toBeDefined();
  });

  it('opens forensic replay modal when inspectSubmissionReplay is invoked', () => {
    component.inspectSubmissionReplay('sub-a', 'Ana García');
    expect(component.showForensicReplay()).toBe(true);
    expect(component.activeReplayStudentName()).toBe('Ana García');
    expect(component.activeReplayTimeline()?.submission_id).toBe('sub-a');
  });
});
