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

  function render(subject: Partial<TeacherOverviewSubject>, openSessionSubjectId: string | null = null): void {
    fixture.componentRef.setInput('subject', { ...SUBJECT, ...subject });
    fixture.componentRef.setInput('openSessionSubjectId', openSessionSubjectId);
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

  it('sin cola ni aula no lleva acciones y dice que está al día', () => {
    render({});

    expect(fixture.nativeElement.querySelectorAll('.action').length).toBe(0);
    expect(fixture.nativeElement.querySelector('.figure-clear')).not.toBeNull();
  });

  function actionLabels(): Array<string | undefined> {
    const actions = fixture.nativeElement.querySelectorAll('.action') as NodeListOf<HTMLElement>;
    return Array.from(actions, action => action.textContent?.trim());
  }

  it('con cola ofrece calificar y abrir el aula de esa materia', () => {
    render({ pending_review: 15 });

    expect(actionLabels()).toEqual(['Calificar', 'Abrir aula']);
  });

  it('con el aula abierta en esa materia cambia la acción a ir al aula', () => {
    render({ pending_review: 15 }, 's1');

    expect(actionLabels()).toEqual(['Calificar', 'Ir al aula']);
  });

  it('omite la cifra de faltantes cuando el servidor no la calcula', () => {
    render({ without_submission: null });

    expect(fixture.nativeElement.textContent).not.toContain('sin entregar');
  });

  it('no afirma estar al día cuando hay alumnos sin entregar', () => {
    render({ pending_review: 0, without_submission: 2 });

    expect(fixture.nativeElement.querySelector('.figure-clear')).toBeNull();
    expect(fixture.nativeElement.textContent).toContain('sin entregar');
  });

  it('nombra la práctica que vence junto a su fecha', () => {
    render({ next_due_at: '2026-10-31T23:59:00Z', next_due_title: 'Práctica 3' });

    expect(fixture.nativeElement.querySelector('.card-due')).not.toBeNull();
    expect(fixture.nativeElement.textContent).toContain('Práctica 3');
  });
});
