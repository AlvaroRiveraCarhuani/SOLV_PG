import { ComponentFixture, TestBed } from '@angular/core/testing';
import { TemplateDetailModalComponent } from './template-detail-modal.component';
import { PublishedTemplate } from '../teacher-templates.component';

describe('TemplateDetailModalComponent', () => {
  let component: TemplateDetailModalComponent;
  let fixture: ComponentFixture<TemplateDetailModalComponent>;

  const mockTemplate: PublishedTemplate = {
    id: 'tpl-java',
    name: 'Java 21 Enterprise & Spring Boot',
    description: 'Entorno para desarrollo backend empresarial con Spring Boot y JPA.',
    docker_image: 'eclipse-temurin:21-jdk-alpine',
    default_memory_mb: 1024,
    default_cpu_cores: 2,
    category: 'Empresarial & Backend',
    environment_type: 'IDE_PERSISTENTE',
    is_official: true,
    is_active: true,
    primaryIcon: 'java',
    techChips: [
      { name: 'Java', iconKey: 'java' },
      { name: 'Spring Boot', iconKey: 'spring' }
    ]
  };

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [TemplateDetailModalComponent]
    }).compileComponents();

    fixture = TestBed.createComponent(TemplateDetailModalComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('template', mockTemplate);
    fixture.detectChanges();
  });

  it('renders pedagogical tab by default with title and description', () => {
    expect(component.activeTab()).toBe('pedagogical');
    const el: HTMLElement = fixture.nativeElement;
    expect(el.textContent).toContain('Propósito en el Aula');
    expect(el.textContent).toContain('Entorno para desarrollo backend empresarial');
  });

  it('switches between pedagogical and technical tabs', () => {
    component.setTab('technical');
    fixture.detectChanges();
    expect(component.activeTab()).toBe('technical');

    const el: HTMLElement = fixture.nativeElement;
    expect(el.textContent).toContain('Imagen Contenedora:');
    expect(el.textContent).toContain('eclipse-temurin:21-jdk-alpine');

    component.setTab('pedagogical');
    fixture.detectChanges();
    expect(component.activeTab()).toBe('pedagogical');
  });

  it('emits useTemplate when clicking the main action button', () => {
    let emitted: PublishedTemplate | null = null;
    component.useTemplate.subscribe(tpl => {
      emitted = tpl;
    });

    component.onUse();
    expect(emitted).toEqual(mockTemplate);
  });

  it('emits closed when close is called', () => {
    let closed = false;
    component.closed.subscribe(() => {
      closed = true;
    });

    component.closed.emit();
    expect(closed).toBe(true);
  });
});
