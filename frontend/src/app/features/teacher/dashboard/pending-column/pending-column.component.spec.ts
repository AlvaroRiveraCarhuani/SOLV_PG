import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { PendingColumnComponent } from './pending-column.component';
import {
  TeacherOverviewDue,
  TeacherOverviewQueue,
  TeacherOverviewSession
} from '../../models/teacher.models';

const SESSION: TeacherOverviewSession = {
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
};

const DUE: TeacherOverviewDue = {
  subject_id: 's2',
  subject_code: 'SIS-420',
  subject_name: 'Desarrollo Web Fullstack',
  exercise_id: 'e1',
  exercise_title: 'Práctica 3',
  urgency: 'today',
  pending_review: 3,
  due_at: '2026-10-31T23:59:00Z'
};

const EMPTY_QUEUE: TeacherOverviewQueue = { pending: 0, practices: 0, oldest_submitted_at: null };

describe('PendingColumnComponent', () => {
  let fixture: ComponentFixture<PendingColumnComponent>;

  function render(
    session: TeacherOverviewSession | null,
    queue: TeacherOverviewQueue,
    dueSoon: TeacherOverviewDue[] = [],
    dueSoonHidden = 0
  ): void {
    fixture.componentRef.setInput('session', session);
    fixture.componentRef.setInput('queue', queue);
    fixture.componentRef.setInput('dueSoon', dueSoon);
    fixture.componentRef.setInput('dueSoonHidden', dueSoonHidden);
    fixture.detectChanges();
  }

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [PendingColumnComponent],
      providers: [provideRouter([])]
    }).compileComponents();

    fixture = TestBed.createComponent(PendingColumnComponent);
  });

  afterEach(() => fixture.destroy());

  it('sin aula abierta no aparece el bloque Aula ahora', () => {
    render(null, EMPTY_QUEUE, [DUE]);

    expect(fixture.nativeElement.textContent).not.toContain('Aula ahora');
    expect(fixture.nativeElement.textContent).toContain('Vencimientos');
  });

  it('con aula muestra la presencia y la hora de cierre', () => {
    render(SESSION, EMPTY_QUEUE);

    expect(fixture.nativeElement.textContent).toContain('Aula ahora');
    expect(fixture.nativeElement.textContent).toContain('Ir al aula');
  });

  it('declara los vencimientos que quedaron fuera del límite', () => {
    render(null, EMPTY_QUEUE, [DUE], 4);

    expect(fixture.nativeElement.querySelector('.block-more').textContent).toContain('4');
  });

  it('muestra las prácticas vencidas con su urgencia y entregas por calificar', () => {
    const overdue: TeacherOverviewDue = { ...DUE, urgency: 'overdue', due_at: '2026-10-20T23:59:00Z' };
    render(null, EMPTY_QUEUE, [overdue]);

    expect(fixture.nativeElement.textContent).toContain('Vencimientos');
    expect(fixture.nativeElement.textContent).toContain('vencida');
    expect(fixture.nativeElement.textContent).toContain('por calificar');
  });

  it('cada práctica de vencimientos es un enlace directo al detalle de entregas del curso', () => {
    render(null, EMPTY_QUEUE, [DUE]);

    const link = fixture.nativeElement.querySelector('a.due-item');
    expect(link).toBeTruthy();
    expect(link.getAttribute('href')).toContain('/teacher/cursos/s2');
  });

  it('muestra minipaginador y pagina cuando hay más de 4 vencimientos', () => {
    const items: TeacherOverviewDue[] = Array.from({ length: 6 }, (_, i) => ({
      ...DUE,
      exercise_id: `ex-${i}`,
      exercise_title: `Práctica ${i + 1}`
    }));

    render(null, EMPTY_QUEUE, items);

    const pagination = fixture.nativeElement.querySelector('.due-pagination');
    expect(pagination).toBeTruthy();
    expect(fixture.nativeElement.textContent).toContain('1/2');

    // Inicialmente muestra 4 ítems
    let displayedItems = fixture.nativeElement.querySelectorAll('a.due-item');
    expect(displayedItems.length).toBe(4);
    expect(fixture.nativeElement.textContent).toContain('Práctica 1');
    expect(fixture.nativeElement.textContent).not.toContain('Práctica 5');

    // Avanzamos de página
    const nextBtn = fixture.nativeElement.querySelector('button[aria-label="Página siguiente"]');
    nextBtn.click();
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('2/2');
    displayedItems = fixture.nativeElement.querySelectorAll('a.due-item');
    expect(displayedItems.length).toBe(2);
    expect(fixture.nativeElement.textContent).toContain('Práctica 5');
    expect(fixture.nativeElement.textContent).toContain('Práctica 6');
  });

  it('sin cola no aparece el bloque Por calificar', () => {
    render(null, EMPTY_QUEUE, [DUE]);

    expect(fixture.nativeElement.textContent).not.toContain('Por calificar');
  });

  it('con cola resume entregas y prácticas con su acción', () => {
    render(null, { pending: 21, practices: 4, oldest_submitted_at: '2026-10-28T10:00:00Z' });

    expect(fixture.nativeElement.textContent).toContain('21');
    expect(fixture.nativeElement.querySelector('.block-action').textContent).toContain('Calificar');
  });
});
