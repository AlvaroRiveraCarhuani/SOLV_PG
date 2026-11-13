import '@angular/localize/init';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ComponentRef } from '@angular/core';
import { vi } from 'vitest';
import { IdeDbInitScriptComponent, MAX_RECOMMENDED_SQL_BYTES } from './ide-db-init-script.component';

describe('IdeDbInitScriptComponent', () => {
  let component: IdeDbInitScriptComponent;
  let componentRef: ComponentRef<IdeDbInitScriptComponent>;
  let fixture: ComponentFixture<IdeDbInitScriptComponent>;
  let compiled: HTMLElement;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [IdeDbInitScriptComponent]
    }).compileComponents();

    fixture = TestBed.createComponent(IdeDbInitScriptComponent);
    component = fixture.componentInstance;
    componentRef = fixture.componentRef;
    compiled = fixture.nativeElement;
  });

  it('no debe renderizar nada si hasDatabaseService es falso', () => {
    componentRef.setInput('hasDatabaseService', false);
    fixture.detectChanges();

    expect(compiled.querySelector('.card-section')).toBeNull();
  });

  it('debe renderizar la zona de carga cuando hasDatabaseService es true y no hay script', () => {
    componentRef.setInput('hasDatabaseService', true);
    componentRef.setInput('dbInitScript', '');
    fixture.detectChanges();

    expect(compiled.querySelector('.card-section')).toBeTruthy();
    expect(compiled.querySelector('.dropzone-strip')).toBeTruthy();
    expect(compiled.querySelector('.script-meta-bar')).toBeNull();
  });

  it('debe aclarar que la base de datos es la misma durante toda la materia', () => {
    componentRef.setInput('hasDatabaseService', true);
    componentRef.setInput('dbInitScript', '');
    fixture.detectChanges();

    const note = compiled.querySelector('.scope-note');
    expect(note).toBeTruthy();
    expect(note?.textContent).toContain('misma durante toda la materia');
  });

  it('debe declarar que el script es exclusivo de la práctica', () => {
    componentRef.setInput('hasDatabaseService', true);
    componentRef.setInput('dbInitScript', '');
    fixture.detectChanges();

    expect(compiled.querySelector('.scope-badge')?.textContent).toContain('Exclusivo de esta práctica');
  });

  it('debe renderizar la barra de metadatos cuando hay contenido', () => {
    componentRef.setInput('hasDatabaseService', true);
    componentRef.setInput('dbInitScript', 'CREATE TABLE test (id INT);\nINSERT INTO test VALUES (1);');
    fixture.detectChanges();

    const metaBar = compiled.querySelector('.script-meta-bar');
    expect(metaBar).toBeTruthy();
    expect(component.lineCount()).toBe(2);
    expect(component.approxStatementsCount()).toBe(2);
    expect(metaBar?.textContent).toContain('script_personalizado.sql');
  });

  it('debe contar cero líneas y cero sentencias sin contenido', () => {
    componentRef.setInput('hasDatabaseService', true);
    componentRef.setInput('dbInitScript', '   ');
    fixture.detectChanges();

    expect(component.lineCount()).toBe(0);
    expect(component.approxStatementsCount()).toBe(0);
    expect(component.hasContent()).toBe(false);
  });

  it('debe emitir initScriptChange al limpiar el script', () => {
    componentRef.setInput('hasDatabaseService', true);
    componentRef.setInput('dbInitScript', 'SELECT 1;');
    fixture.detectChanges();

    const emitSpy = vi.fn();
    component.initScriptChange.subscribe(emitSpy);

    component.clearScript();

    expect(emitSpy).toHaveBeenCalledWith('');
    expect(component.fileName()).toBeNull();
    expect(component.fileSize()).toBeNull();
  });

  it('debe detectar scripts que superan el tamaño máximo recomendado', () => {
    componentRef.setInput('hasDatabaseService', true);
    const bigScript = 'A'.repeat(MAX_RECOMMENDED_SQL_BYTES + 50);
    componentRef.setInput('dbInitScript', bigScript);
    fixture.detectChanges();

    expect(component.isOversized()).toBe(true);
    expect(compiled.querySelector('.alert-warning')).toBeTruthy();
  });

  it('no debe advertir por tamaño con un script dentro del límite', () => {
    componentRef.setInput('hasDatabaseService', true);
    componentRef.setInput('dbInitScript', 'SELECT 1;');
    fixture.detectChanges();

    expect(component.isOversized()).toBe(false);
  });

  it('debe procesar un archivo .sql válido y emitir su contenido', async () => {
    const file = new File(['CREATE TABLE x (id INT);'], 'migracion.sql', { type: 'text/plain' });
    const emitSpy = vi.fn();
    component.initScriptChange.subscribe(emitSpy);

    component.processFile(file);

    await new Promise(resolve => setTimeout(resolve, 50));

    expect(component.fileName()).toBe('migracion.sql');
    expect(component.fileSize()).toBe(file.size);
    expect(emitSpy).toHaveBeenCalledWith('CREATE TABLE x (id INT);');
  });

  it('debe advertir si se carga un archivo con extensión no habitual', () => {
    const file = new File(['test'], 'datos.csv', { type: 'text/csv' });
    component.processFile(file);

    expect(component.errorMessage()).toContain('Formato de archivo no habitual');
  });

  it('debe descargar el script cuando downloadScript() es ejecutado', () => {
    componentRef.setInput('hasDatabaseService', true);
    componentRef.setInput('dbInitScript', 'SELECT 42;');
    fixture.detectChanges();

    const anchor = document.createElement('a');
    anchor.click = vi.fn();
    const createElementSpy = vi.spyOn(document, 'createElement').mockReturnValue(anchor);
    const createObjectURL = vi.fn(() => 'blob:mock');
    const revokeObjectURL = vi.fn();
    vi.stubGlobal('URL', { ...URL, createObjectURL, revokeObjectURL });

    component.downloadScript();

    expect(createElementSpy).toHaveBeenCalledWith('a');
    expect(anchor.download).toBe('init_script.sql');
    expect(anchor.click).toHaveBeenCalled();

    createElementSpy.mockRestore();
    vi.unstubAllGlobals();
  });

  it('debe desactivar el estado de arrastre al salir de la zona', () => {
    componentRef.setInput('hasDatabaseService', true);
    fixture.detectChanges();

    const event = new Event('dragleave') as DragEvent;
    component.onDragOver(event);
    expect(component.isDragging()).toBe(true);

    component.onDragLeave(event);
    expect(component.isDragging()).toBe(false);
  });
});
