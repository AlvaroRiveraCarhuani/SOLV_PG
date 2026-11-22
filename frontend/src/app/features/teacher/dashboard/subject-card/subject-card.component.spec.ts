import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { SubjectCardComponent } from './subject-card.component';
import { TeacherOverviewSubject } from '../../models/teacher.models';

const SUBJECT: TeacherOverviewSubject = {
  id: 's1',
  code: 'SIS-211',
  name: 'Estructuras de Datos',
  enrolled: 28,
  pending_review: 0,
  without_submission: null,
  next_due_at: null
};

describe('SubjectCardComponent', () => {
  let fixture: ComponentFixture<SubjectCardComponent>;

  function render(subject: Partial<TeacherOverviewSubject>): void {
    fixture.componentRef.setInput('subject', { ...SUBJECT, ...subject });
    fixture.detectChanges();
  }

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [SubjectCardComponent],
      providers: [provideRouter([])]
    }).compileComponents();

    fixture = TestBed.createComponent(SubjectCardComponent);
  });

  afterEach(() => fixture.destroy());

  it('sin cola no ofrece calificar y deja abrir el curso', () => {
    render({});

    expect(fixture.nativeElement.querySelector('.card-pending')).toBeNull();
    expect(fixture.nativeElement.textContent).not.toContain('Calificar');
    expect(fixture.nativeElement.querySelector('.card-open')?.textContent).toContain('Abrir curso');
  });

  it('con cola muestra pendientes y envía a la cola filtrada por materia', () => {
    render({ pending_review: 15 });

    const action = fixture.nativeElement.querySelector('a[href*="/teacher/por-calificar"]') as HTMLAnchorElement;
    expect(fixture.nativeElement.querySelector('.card-pending')?.textContent).toContain('15');
    expect(action.textContent).toContain('Calificar');
    expect(action.getAttribute('href')).toContain('subject_id=s1');
  });

  it('omite la cifra de faltantes cuando el servidor no la calcula', () => {
    render({ without_submission: null });

    expect(fixture.nativeElement.textContent).not.toContain('sin entregar');
    expect(fixture.nativeElement.querySelector('.delivery')).toBeNull();
  });

  it('dibuja el avance de entregas cuando hay faltantes', () => {
    render({ without_submission: 2 });

    expect(fixture.nativeElement.textContent).toContain('2');
    expect(fixture.nativeElement.textContent).toContain('sin entregar');
    expect(fixture.nativeElement.querySelector('.delivery-value')?.textContent).toContain('93');
  });

  it('nombra la práctica que vence junto a su fecha', () => {
    render({ next_due_at: '2026-10-31T23:59:00Z', next_due_title: 'Práctica 3' });

    expect(fixture.nativeElement.querySelector('.card-due')).not.toBeNull();
    expect(fixture.nativeElement.textContent).toContain('Práctica 3');
  });
});
