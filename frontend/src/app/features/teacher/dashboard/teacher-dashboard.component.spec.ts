import { ComponentFixture, TestBed } from '@angular/core/testing';
import { By } from '@angular/platform-browser';
import { provideRouter } from '@angular/router';
import { signal } from '@angular/core';
import { of } from 'rxjs';

import { TeacherDashboardComponent } from './teacher-dashboard.component';
import { SubjectCardComponent } from './subject-card/subject-card.component';
import { TeacherDashboardService } from '../services/teacher-dashboard.service';
import { TeacherOverview } from '../models/teacher.models';

const PERIOD = { id: 'period-1', name: 'Gestión 2026', code: '2026-II', is_active: true };

function overviewWith(partial: Partial<TeacherOverview>): TeacherOverview {
  return {
    period: PERIOD,
    session: null,
    queue: { pending: 0, practices: 0, oldest_submitted_at: null },
    due_soon: [],
    due_soon_total: 0,
    due_soon_hidden: 0,
    subjects: [],
    signals: [],
    ...partial
  };
}

describe('TeacherDashboardComponent', () => {
  let fixture: ComponentFixture<TeacherDashboardComponent>;
  const overview = signal<TeacherOverview | null>(null);
  const isLoading = signal(false);

  beforeEach(async () => {
    overview.set(null);
    isLoading.set(false);

    const mockService = {
      overview,
      isLoading,
      periods: signal([PERIOD]),
      getAcademicPeriods: () => of([PERIOD]),
      loadOverview: () => of(overview())
    };

    await TestBed.configureTestingModule({
      imports: [TeacherDashboardComponent],
      providers: [
        provideRouter([]),
        { provide: TeacherDashboardService, useValue: mockService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(TeacherDashboardComponent);
  });

  afterEach(() => fixture.destroy());

  it('pinta una tarjeta por materia del resumen', () => {
    overview.set(
      overviewWith({
        subjects: [
          {
            id: 's1',
            code: 'SIS-211',
            name: 'Estructuras de Datos',
            enrolled: 28,
            pending_review: 6,
            without_submission: 4,
            next_due_at: '2026-10-31T23:59:00Z',
            next_due_title: 'Práctica 3'
          }
        ]
      })
    );
    fixture.detectChanges();

    const cards = fixture.debugElement.queryAll(By.directive(SubjectCardComponent));
    expect(cards.length).toBe(1);
    expect(fixture.nativeElement.textContent).toContain('Estructuras de Datos');
  });

  it('sin pendientes no dibuja la columna y deja la línea de estado', () => {
    overview.set(
      overviewWith({
        subjects: [
          {
            id: 's1',
            code: 'SIS-211',
            name: 'Estructuras de Datos',
            enrolled: 28,
            pending_review: 0,
            without_submission: null,
            next_due_at: null
          }
        ]
      })
    );
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('pending-column')).toBeNull();
    expect(fixture.nativeElement.querySelector('.hoy-quiet')).not.toBeNull();
  });

  it('con aula abierta muestra la franja de sesión y la columna', () => {
    overview.set(
      overviewWith({
        session: {
          id: 'sess-1',
          subject_id: 's1',
          subject_code: 'SIS-211',
          subject_name: 'Estructuras de Datos',
          exercise_title: 'Práctica 3',
          opens_at: '2026-10-30T10:00:00Z',
          ends_at: '2026-10-30T11:30:00Z',
          closes_at: null,
          enrolled: 28,
          present: 20
        },
        subjects: [
          {
            id: 's1',
            code: 'SIS-211',
            name: 'Estructuras de Datos',
            enrolled: 28,
            pending_review: 0,
            without_submission: null,
            next_due_at: null
          }
        ]
      })
    );
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.hoy-session')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('pending-column')).not.toBeNull();
  });

  it('un período sin materias muestra su estado vacío', () => {
    overview.set(overviewWith({ subjects: [] }));
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('Sin materias en este período');
  });
});
