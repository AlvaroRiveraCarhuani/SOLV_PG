import { ComponentFixture, TestBed } from '@angular/core/testing';
import { beforeEach, describe, expect, it } from 'vitest';
import { ProjectFile } from '@core/models/project-file.model';
import { ReviewFileTreeComponent } from './review-file-tree.component';

const files: ProjectFile[] = [
  {
    path: 'README.md',
    language: 'markdown',
    content: '# Proyecto\n',
    start_line: 2,
    line_count: 1
  },
  {
    path: 'src/app.js',
    language: 'javascript',
    content: 'console.log("hola");\n',
    start_line: 4,
    line_count: 1
  },
  {
    path: 'src/lib/util.js',
    language: 'javascript',
    content: 'export const answer = 42;\n',
    start_line: 6,
    line_count: 1
  }
];

describe('ReviewFileTreeComponent', () => {
  let fixture: ComponentFixture<ReviewFileTreeComponent>;
  let component: ReviewFileTreeComponent;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ReviewFileTreeComponent]
    }).compileComponents();

    fixture = TestBed.createComponent(ReviewFileTreeComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('files', files);
    fixture.componentRef.setInput('selectedPath', 'src/app.js');
    fixture.detectChanges();
  });

  function treeItems(): HTMLElement[] {
    const host = fixture.nativeElement as HTMLElement;
    return Array.from(host.querySelectorAll<HTMLElement>('[role="treeitem"]'));
  }

  function itemByText(text: string): HTMLElement {
    const item = treeItems().find((node) => node.textContent?.includes(text));
    expect(item).toBeTruthy();
    return item as HTMLElement;
  }

  it('renders folders and files sorted by name with ARIA tree roles', () => {
    const labels = treeItems().map((node) => node.textContent?.trim());
    expect(labels).toEqual(['README.md', 'src', 'app.js', 'lib', 'util.js']);

    const selected = itemByText('app.js');
    expect(selected.getAttribute('aria-selected')).toBe('true');
    expect(selected.getAttribute('aria-level')).toBe('2');
    expect(itemByText('src').getAttribute('aria-expanded')).toBe('true');
  });

  it('collapses a folder and hides its children', () => {
    itemByText('src').click();
    fixture.detectChanges();

    expect(treeItems().map((node) => node.textContent?.trim())).toEqual(['README.md', 'src']);
    expect(itemByText('src').getAttribute('aria-expanded')).toBe('false');
  });

  it('emits the path of the selected file', () => {
    const emitted: string[] = [];
    component.fileSelected.subscribe((path) => emitted.push(path));

    itemByText('util.js').click();

    expect(emitted).toEqual(['src/lib/util.js']);
  });
});
