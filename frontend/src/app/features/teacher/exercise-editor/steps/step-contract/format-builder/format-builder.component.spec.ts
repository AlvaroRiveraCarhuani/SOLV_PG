import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { FormatBuilderComponent } from './format-builder.component';
import { ExerciseEditorStore } from '../../../exercise-editor.store';
import { FORMAT_PRESETS } from './format-builder.models';

describe('FormatBuilderComponent', () => {
  let component: FormatBuilderComponent;
  let fixture: ComponentFixture<FormatBuilderComponent>;
  let store: ExerciseEditorStore;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [FormatBuilderComponent],
      providers: [
        ExerciseEditorStore,
        provideHttpClient(),
        provideHttpClientTesting()
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(FormatBuilderComponent);
    component = fixture.componentInstance;
    store = TestBed.inject(ExerciseEditorStore);
    fixture.detectChanges();
  });

  it('debe inicializarse y cargar el preset inicial', () => {
    expect(component).toBeTruthy();
    expect(component.lines().length).toBeGreaterThan(0);
  });

  it('debe aplicar un preset seleccionado y sincronizar con el store', () => {
    const matrixPreset = FORMAT_PRESETS.find(p => p.id === 'matrix_nm')!;
    component.applyPreset(matrixPreset);

    expect(component.activePresetId()).toBe('matrix_nm');
    expect(component.lines().length).toBe(3);
    expect(store.contract()?.input.lines.length).toBe(3);
  });

  it('debe agregar y eliminar líneas dinámicamente', () => {
    const initialCount = component.lines().length;
    component.addLine();
    expect(component.lines().length).toBe(initialCount + 1);

    component.removeLine(0);
    expect(component.lines().length).toBe(initialCount);
  });

  it('debe sincronizar la edición de JSON con la vista visual', () => {
    const customJson = JSON.stringify({
      version: 1,
      input: {
        lines: [
          { id: 'x', type: 'int', min: 5, max: 50 },
          { id: 'y', type: 'string', max_len: 200 }
        ]
      }
    });

    component.onJsonCodeChange(customJson);
    expect(component.jsonError()).toBeNull();
    expect(component.lines().length).toBe(2);
    expect(component.lines()[0].id).toBe('x');
    expect(component.lines()[1].id).toBe('y');
  });

  it('debe capturar errores de sintaxis JSON sin romper el formulario visual', () => {
    component.onJsonCodeChange('{ version: 1, invalid_json ');
    expect(component.jsonError()).not.toBeNull();
  });
});
