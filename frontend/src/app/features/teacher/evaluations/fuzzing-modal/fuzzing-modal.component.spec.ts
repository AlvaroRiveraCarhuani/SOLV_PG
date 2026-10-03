import { ComponentFixture, TestBed } from '@angular/core/testing';
import { of, throwError } from 'rxjs';
import { FuzzingModalComponent } from './fuzzing-modal.component';
import { TeacherFuzzingService } from '../../services/teacher-fuzzing.service';
import { FuzzGenerationReport, ApplyFuzzCasesResponse } from '../../models/teacher.models';

describe('FuzzingModalComponent', () => {
  let component: FuzzingModalComponent;
  let fixture: ComponentFixture<FuzzingModalComponent>;
  let mockFuzzingService: any;

  const mockReport: FuzzGenerationReport = {
    total_generated: 3,
    categories: {
      numeric_bounds: 1,
      unicode_special: 1,
      empty_structures: 1
    },
    cases: [
      {
        description: 'Límite numérico INT_MAX',
        category: 'numeric_bounds',
        risk_level: 'critical',
        input: '2147483647',
        expected: '2147483647',
        is_public: false,
        weight: 10,
        selected: true
      },
      {
        description: 'Cadena con caracteres UTF-8 nulos y emojis',
        category: 'unicode_special',
        risk_level: 'warning',
        input: '"\\u0000🚀"',
        expected: '""',
        is_public: true,
        weight: 5,
        selected: true
      },
      {
        description: 'Arreglo vacío',
        category: 'empty_structures',
        risk_level: 'boundary',
        input: '[]',
        expected: '0',
        is_public: true,
        weight: 5,
        selected: false
      }
    ]
  };

  const mockApplyResponse: ApplyFuzzCasesResponse = {
    added_count: 2,
    exercise_id: 'ex-101'
  };

  beforeEach(async () => {
    mockFuzzingService = {
      isGenerating: vi.fn().mockReturnValue(false),
      isApplying: vi.fn().mockReturnValue(false),
      currentReport: vi.fn().mockReturnValue(mockReport),
      generateFuzzCases: vi.fn().mockReturnValue(of(mockReport)),
      applyFuzzCases: vi.fn().mockReturnValue(of(mockApplyResponse))
    };

    await TestBed.configureTestingModule({
      imports: [FuzzingModalComponent],
      providers: [
        { provide: TeacherFuzzingService, useValue: mockFuzzingService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(FuzzingModalComponent);
    component = fixture.componentInstance;
    component.exerciseId = 'ex-101';
    component.exerciseTitle = 'Inversión de Matrices';
    component.language = 'python';
    fixture.detectChanges();
  });

  it('should initialize and fetch generated fuzz cases', () => {
    expect(component).toBeTruthy();
    expect(mockFuzzingService.generateFuzzCases).toHaveBeenCalledWith(expect.objectContaining({
      exercise_id: 'ex-101',
      target_language: 'python',
      count: 20
    }));
    expect(component.generatedCases().length).toBe(3);
    expect(component.selectedCasesCount()).toBe(2);
  });

  it('toggles parameter types in filter', () => {
    expect(component.selectedTypes()).toContain('integer');
    component.toggleType('integer');
    expect(component.selectedTypes()).not.toContain('integer');
    component.toggleType('integer');
    expect(component.selectedTypes()).toContain('integer');
  });

  it('toggles fuzzing categories in filter', () => {
    expect(component.selectedCategories()).toContain('numeric_bounds');
    component.toggleCategory('numeric_bounds');
    expect(component.selectedCategories()).not.toContain('numeric_bounds');
    component.toggleCategory('numeric_bounds');
    expect(component.selectedCategories()).toContain('numeric_bounds');
  });

  it('toggles selection of single case and select-all', () => {
    expect(component.allSelected()).toBe(false);
    
    // Toggle the 3rd case to true -> all should be selected
    component.toggleCaseSelection(2);
    expect(component.allSelected()).toBe(true);
    expect(component.selectedCasesCount()).toBe(3);

    // Toggle select all -> all become unselected
    component.toggleSelectAll();
    expect(component.selectedCasesCount()).toBe(0);
    expect(component.allSelected()).toBe(false);

    // Toggle select all again -> all become selected
    component.toggleSelectAll();
    expect(component.selectedCasesCount()).toBe(3);
  });

  it('toggles case visibility between public and hidden', () => {
    expect(component.generatedCases()[0].is_public).toBe(false);
    component.toggleVisibility(0);
    expect(component.generatedCases()[0].is_public).toBe(true);
    component.toggleVisibility(0);
    expect(component.generatedCases()[0].is_public).toBe(false);
  });

  it('applies selected cases and emits applied event', () => {
    const appliedSpy = vi.spyOn(component.applied, 'emit');
    const closeSpy = vi.spyOn(component.close, 'emit');

    component.applySelectedCases();

    expect(mockFuzzingService.applyFuzzCases).toHaveBeenCalledWith('ex-101', expect.arrayContaining([
      expect.objectContaining({ description: 'Límite numérico INT_MAX' }),
      expect.objectContaining({ description: 'Cadena con caracteres UTF-8 nulos y emojis' })
    ]));
    expect(appliedSpy).toHaveBeenCalledWith(2);
    expect(closeSpy).toHaveBeenCalled();
  });

  it('handles error gracefully when fuzz generation fails', () => {
    mockFuzzingService.generateFuzzCases.mockReturnValue(throwError(() => new Error('API Error')));
    component.generateCases();
    expect(component.feedbackMessage()).toBe('Error al sintetizar casos de prueba.');
  });

  it('handles error gracefully when applying fuzz cases fails', () => {
    mockFuzzingService.applyFuzzCases.mockReturnValue(throwError(() => ({
      error: { message: 'Exercise is locked' }
    })));
    component.applySelectedCases();
    expect(component.feedbackMessage()).toBe('Exercise is locked');
  });

  it('closes on escape key event', () => {
    const closeSpy = vi.spyOn(component.close, 'emit');
    component.onEscape();
    expect(closeSpy).toHaveBeenCalled();
  });
});
