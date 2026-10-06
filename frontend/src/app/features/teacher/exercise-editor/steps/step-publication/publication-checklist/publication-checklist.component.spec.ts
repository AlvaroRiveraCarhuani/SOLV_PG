import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { PublicationChecklistComponent } from './publication-checklist.component';
import { ExerciseEditorStore } from '../../../exercise-editor.store';

describe('PublicationChecklistComponent', () => {
  let component: PublicationChecklistComponent;
  let fixture: ComponentFixture<PublicationChecklistComponent>;
  let store: ExerciseEditorStore;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [PublicationChecklistComponent],
      providers: [
        ExerciseEditorStore,
        provideHttpClient(),
        provideHttpClientTesting()
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(PublicationChecklistComponent);
    component = fixture.componentInstance;
    store = TestBed.inject(ExerciseEditorStore);
    fixture.detectChanges();
  });

  it('debe inicializarse correctamente', () => {
    expect(component).toBeTruthy();
  });

  it('debe mostrar el estado de bloqueantes si existen', () => {
    store.checklistReport.set({
      blockers: ['Debes incluir al menos un caso de prueba'],
      warnings: [],
      info: [],
      can_publish: false
    });
    fixture.detectChanges();

    expect(component.hasBlockers()).toBe(true);
    expect(component.blockers().length).toBe(1);
  });

  it('debe permitir aceptar advertencias pedagógicas', () => {
    store.checklistReport.set({
      blockers: [],
      warnings: ['0 casos ocultos'],
      info: [],
      can_publish: true
    });
    fixture.detectChanges();

    expect(store.acceptedWarnings()).toBe(false);
    component.onAcceptWarnings(true);
    expect(store.acceptedWarnings()).toBe(true);
  });
});
