import { describe, it, expect, beforeEach } from 'vitest';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { SolvStepImageComponent } from './step-image.component';
import { LocalImageItem, ImageSuggestion } from '../../../../../services/admin-templates.service';

describe('SolvStepImageComponent', () => {
  let component: SolvStepImageComponent;
  let fixture: ComponentFixture<SolvStepImageComponent>;

  const mockLocalImages: LocalImageItem[] = [
    { repo_tag: 'python:3.12-slim-bookworm', size_mb: 150, is_official: true, has_latest_tag: false, created_at: '', usage_count: 5 },
    { repo_tag: 'node:20-alpine', size_mb: 90, is_official: true, has_latest_tag: false, created_at: '', usage_count: 2 }
  ];

  const mockCuratedImages: ImageSuggestion[] = [
    { repoTag: 'golang:1.22-bookworm', isLocal: false, isOfficial: true, description: 'Go SDK' },
    { repoTag: 'gcc:13.2-bookworm', isLocal: false, isOfficial: true, description: 'GCC compiler' }
  ];

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [SolvStepImageComponent]
    }).compileComponents();

    fixture = TestBed.createComponent(SolvStepImageComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('localImages', mockLocalImages);
    fixture.componentRef.setInput('curatedImages', mockCuratedImages);
    fixture.componentRef.setInput('usageMap', { 'golang:1.22-bookworm': 10 });
    fixture.detectChanges();
  });

  it('debe suprimir el listbox y mostrar mensaje de error si se introduce :latest', () => {
    fixture.componentRef.setInput('dockerImage', 'python:latest');
    fixture.detectChanges();

    expect(component.isLatestImage()).toBe(true);
    const fieldMsg = fixture.nativeElement.querySelector('solv-field-message');
    expect(fieldMsg).toBeTruthy();
    expect(fieldMsg.textContent).toContain('Prohibido el tag :latest');
  });

  it('debe mostrar mensaje semántico de éxito cuando la imagen está verificada', () => {
    fixture.componentRef.setInput('dockerImage', 'python:3.12-slim-bookworm');
    fixture.componentRef.setInput('verificationState', 'verified');
    fixture.detectChanges();

    const fieldMsg = fixture.nativeElement.querySelector('solv-field-message');
    expect(fieldMsg).toBeTruthy();
    expect(fieldMsg.textContent).toContain('Imagen verificada en el nodo');
  });

  it('debe derivar chips de herramientas reactivos según la familia de la imagen', () => {
    // Caso Python
    fixture.componentRef.setInput('dockerImage', 'python:3.12-slim-bookworm');
    fixture.detectChanges();
    expect(component.reactiveSuggestedTools()).toEqual(['python3', 'pip', 'pytest']);

    // Caso Node
    fixture.componentRef.setInput('dockerImage', 'node:20-alpine');
    fixture.detectChanges();
    expect(component.reactiveSuggestedTools()).toEqual(['node', 'npm', 'npx']);

    // Fallback Juez
    fixture.componentRef.setInput('dockerImage', 'unknown/custom:1.0');
    fixture.componentRef.setInput('targetEnvironment', 'JUEZ_EFIMERO');
    fixture.detectChanges();
    expect(component.reactiveSuggestedTools()).toEqual(['gcc', 'python3', 'javac']);
  });

  it('debe agregar herramientas a toolsDeclared al hacer clic en un chip sugerido', () => {
    let updatedTools = '';
    component.toolsDeclaredChange.subscribe((t) => (updatedTools = t));

    fixture.componentRef.setInput('dockerImage', 'python:3.12-slim-bookworm');
    fixture.componentRef.setInput('toolsDeclared', 'python3');
    fixture.detectChanges();

    component.addTool('pip');
    expect(updatedTools).toBe('python3, pip');
  });
});
