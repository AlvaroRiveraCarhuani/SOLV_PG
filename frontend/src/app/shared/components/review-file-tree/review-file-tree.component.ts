import { CommonModule } from '@angular/common';
import { Component, ElementRef, computed, inject, input, output, signal } from '@angular/core';
import {
  LucideChevronDown,
  LucideChevronRight,
  LucideFileCode,
  LucideFolder,
  LucideFolderOpen,
  LucidePanelLeftClose
} from '@lucide/angular';
import { MachineDataDirective } from '@shared/directives/machine-data.directive';
import { ProjectFile } from '@core/models/project-file.model';

export interface ReviewTreeNode {
  kind: 'dir' | 'file';
  name: string;
  path: string;
  depth: number;
}

interface DirectoryEntries {
  dirs: Map<string, DirectoryEntries>;
  files: { name: string; path: string }[];
}

function createEntries(): DirectoryEntries {
  return { dirs: new Map(), files: [] };
}

function buildEntries(files: ProjectFile[]): DirectoryEntries {
  const root = createEntries();

  for (const file of files) {
    const segments = file.path.split('/').filter(Boolean);
    if (segments.length === 0) continue;

    let current = root;
    for (const segment of segments.slice(0, -1)) {
      let next = current.dirs.get(segment);
      if (!next) {
        next = createEntries();
        current.dirs.set(segment, next);
      }
      current = next;
    }

    const name = segments[segments.length - 1];
    if (!current.files.some((entry) => entry.name === name)) {
      current.files.push({ name, path: file.path });
    }
  }

  return root;
}

function flattenEntries(
  entries: DirectoryEntries,
  collapsed: ReadonlySet<string>,
  depth: number,
  parentPath: string,
  out: ReviewTreeNode[]
): ReviewTreeNode[] {
  const dirs = [...entries.dirs.entries()].map(([name, child]) => ({
    kind: 'dir' as const,
    name,
    path: parentPath ? `${parentPath}/${name}` : name,
    child
  }));
  const files = entries.files.map((file) => ({
    kind: 'file' as const,
    name: file.name,
    path: file.path
  }));

  const ordered = [...dirs, ...files].sort((a, b) => a.name.localeCompare(b.name));

  for (const entry of ordered) {
    out.push({ kind: entry.kind, name: entry.name, path: entry.path, depth });
    if (entry.kind === 'dir' && !collapsed.has(entry.path)) {
      flattenEntries(entry.child, collapsed, depth + 1, entry.path, out);
    }
  }

  return out;
}

@Component({
  selector: 'review-file-tree',
  standalone: true,
  imports: [
    CommonModule,
    LucideChevronDown,
    LucideChevronRight,
    LucideFileCode,
    LucideFolder,
    LucideFolderOpen,
    LucidePanelLeftClose,
    MachineDataDirective
  ],
  templateUrl: './review-file-tree.component.html',
  styleUrl: './review-file-tree.component.scss'
})
export class ReviewFileTreeComponent {
  readonly files = input.required<ProjectFile[]>();
  readonly selectedPath = input<string | null>(null);

  readonly fileSelected = output<string>();
  readonly collapse = output<void>();

  private readonly host = inject<ElementRef<HTMLElement>>(ElementRef);
  private readonly collapsedDirs = signal<ReadonlySet<string>>(new Set());
  readonly focusedPath = signal<string | null>(null);

  readonly visibleNodes = computed(() =>
    flattenEntries(buildEntries(this.files()), this.collapsedDirs(), 0, '', [])
  );

  readonly tabbablePath = computed(
    () => this.focusedPath() ?? this.selectedPath() ?? this.visibleNodes()[0]?.path ?? null
  );

  isExpanded(node: ReviewTreeNode): boolean {
    return !this.collapsedDirs().has(node.path);
  }

  onNodeClick(node: ReviewTreeNode): void {
    this.focusedPath.set(node.path);
    if (node.kind === 'dir') {
      this.toggleDir(node.path);
      return;
    }
    this.fileSelected.emit(node.path);
  }

  onKeydown(event: KeyboardEvent): void {
    const handled = ['ArrowDown', 'ArrowUp', 'Home', 'End', 'ArrowRight', 'ArrowLeft', 'Enter', ' '];
    if (!handled.includes(event.key)) return;

    const nodes = this.visibleNodes();
    if (nodes.length === 0) return;

    event.preventDefault();

    const currentPath = this.tabbablePath();
    const currentIndex = Math.max(
      0,
      nodes.findIndex((node) => node.path === currentPath)
    );
    const node = nodes[currentIndex];

    switch (event.key) {
      case 'ArrowDown':
        this.focusAt(nodes, Math.min(currentIndex + 1, nodes.length - 1));
        return;
      case 'ArrowUp':
        this.focusAt(nodes, Math.max(currentIndex - 1, 0));
        return;
      case 'Home':
        this.focusAt(nodes, 0);
        return;
      case 'End':
        this.focusAt(nodes, nodes.length - 1);
        return;
      case 'ArrowRight':
        if (node.kind !== 'dir') return;
        if (!this.isExpanded(node)) {
          this.toggleDir(node.path);
          return;
        }
        if (nodes[currentIndex + 1]?.depth === node.depth + 1) {
          this.focusAt(nodes, currentIndex + 1);
        }
        return;
      case 'ArrowLeft':
        if (node.kind === 'dir' && this.isExpanded(node)) {
          this.toggleDir(node.path);
          return;
        }
        for (let index = currentIndex - 1; index >= 0; index -= 1) {
          if (nodes[index].depth < node.depth) {
            this.focusAt(nodes, index);
            return;
          }
        }
        return;
      case 'Enter':
      case ' ':
        this.onNodeClick(node);
        return;
    }
  }

  private toggleDir(path: string): void {
    this.collapsedDirs.update((current) => {
      const next = new Set(current);
      if (next.has(path)) {
        next.delete(path);
      } else {
        next.add(path);
      }
      return next;
    });
    this.focusedPath.set(path);
  }

  private focusAt(nodes: ReviewTreeNode[], index: number): void {
    const target = nodes[index];
    if (!target) return;
    this.focusedPath.set(target.path);

    setTimeout(() => {
      const items = this.host.nativeElement.querySelectorAll<HTMLElement>('[role="treeitem"]');
      items[index]?.focus();
    });
  }
}
