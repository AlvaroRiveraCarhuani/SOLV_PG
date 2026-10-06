import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { CasesTableComponent } from './cases-table.component';
import { ExerciseEditorStore } from '../../../exercise-editor.store';

describe('CasesTableComponent', () => {
  let component: CasesTableComponent;
  let fixture: ComponentFixture<CasesTableComponent>;
  let store: ExerciseEditorStore;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [CasesTableComponent],
      providers: [
        ExerciseEditorStore,
        provideHttpClient(),
        provideHttpClientTesting()
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(CasesTableComponent);
    component = fixture.componentInstance;
    store = TestBed.inject(ExerciseEditorStore);
    fixture.detectChanges();
  });

  it('debe inicializarse correctamente', () => {
    expect(component).toBeTruthy();
  });

  it('debe permitir agregar un nuevo caso', () => {
    const initialCount = store.cases().length;
    component.addNewCase();
    expect(store.cases().length).toBe(initialCount + 1);
  });

  it('debe duplicar un caso existente', () => {
    store.setCases([
      { input: '5', expected_output: '10', visibility: 'example', weight: 1.0 }
    ]);
    component.duplicateCase(0);
    expect(store.cases().length).toBe(2);
    expect(store.cases()[1].input).toBe('5');
    expect(store.cases()[1].expected_output).toBe('10');
  });

  it('debe eliminar un caso y actualizar el store', () => {
    store.setCases([
      { input: '1', expected_output: '2', visibility: 'public', weight: 1.0 },
      { input: '3', expected_output: '4', visibility: 'hidden', weight: 2.0 }
    ]);
    component.removeCase(0);
    expect(store.cases().length).toBe(1);
    expect(store.cases()[0].input).toBe('3');
  });

  it('debe reordenar casos hacia abajo y hacia arriba', () => {
    store.setCases([
      { input: 'A', expected_output: '1', visibility: 'public', weight: 1.0 },
      { input: 'B', expected_output: '2', visibility: 'public', weight: 1.0 }
    ]);
    component.moveDown(0);
    expect(store.cases()[0].input).toBe('B');
    expect(store.cases()[1].input).toBe('A');

    component.moveUp(1);
    expect(store.cases()[0].input).toBe('A');
    expect(store.cases()[1].input).toBe('B');
  });

  it('debe calcular el porcentaje de peso relativo correctamente', () => {
    store.setCases([
      { input: '1', expected_output: '1', visibility: 'public', weight: 1.0 },
      { input: '2', expected_output: '2', visibility: 'hidden', weight: 3.0 }
    ]);
    expect(component.getWeightPercentage(1.0)).toBe('25.0%');
    expect(component.getWeightPercentage(3.0)).toBe('75.0%');
  });

  it('debe equiparar pesos de todos los casos a 1.0', () => {
    store.setCases([
      { input: '1', expected_output: '1', visibility: 'public', weight: 0.5 },
      { input: '2', expected_output: '2', visibility: 'hidden', weight: 2.5 }
    ]);
    component.normalizeWeights();
    expect(store.cases()[0].weight).toBe(1.0);
    expect(store.cases()[1].weight).toBe(1.0);
  });

  it('debe abrir y guardar en el modal multilínea', () => {
    store.setCases([
      { input: '1 2 3', expected_output: '6', visibility: 'public', weight: 1.0 }
    ]);
    component.openModal(0, 'input');
    expect(component.modalState().isOpen).toBe(true);
    expect(component.modalState().value).toBe('1 2 3');

    component.onModalSave('1 2 3 4 5');
    expect(component.modalState().isOpen).toBe(false);
    expect(store.cases()[0].input).toBe('1 2 3 4 5');
  });

  it('debe abrir y cerrar el modal de generación de casos', () => {
    expect(component.isGenerateModalOpen()).toBe(false);
    component.openGenerateModal();
    expect(component.isGenerateModalOpen()).toBe(true);
    component.closeGenerateModal();
    expect(component.isGenerateModalOpen()).toBe(false);
  });

  it('debe agregar casos generados con visibilidad hidden y peso 1.0', () => {
    store.setCases([]);
    store.setReferenceSolution('print("test")');
    component.onCasesGenerated([
      { input: '10' },
      { input: '20' }
    ]);

    expect(store.cases().length).toBe(2);
    expect(store.cases()[0].input).toBe('10');
    expect(store.cases()[0].visibility).toBe('hidden');
    expect(store.cases()[0].weight).toBe(1.0);
    expect(component.showAutoCalculatePrompt()).toBe(true);
    expect(component.lastGeneratedCount()).toBe(2);
  });

  it('debe descartar la sugerencia de auto-cálculo', () => {
    component.showAutoCalculatePrompt.set(true);
    component.dismissAutoCalculatePrompt();
    expect(component.showAutoCalculatePrompt()).toBe(false);
  });
});
