import { Component, Input, computed, signal } from '@angular/core';

export type TechIconKey =
  | 'python'
  | 'rust'
  | 'node'
  | 'typescript'
  | 'go'
  | 'java'
  | 'c'
  | 'cpp'
  | 'php'
  | 'dotnet'
  | 'angular'
  | 'react'
  | 'vue'
  | 'nextjs'
  | 'vite'
  | 'fastapi'
  | 'django'
  | 'flask'
  | 'spring'
  | 'laravel'
  | 'streamlit'
  | 'bun'
  | 'deno'
  | 'postgres'
  | 'mysql'
  | 'mariadb'
  | 'mongodb'
  | 'sqlite'
  | 'redis'
  | 'git'
  | 'db'
  | 'docker'
  | string;

export interface CuratedIconMeta {
  key: string;
  label: string;
}

export const CURATED_TECH_ICONS: CuratedIconMeta[] = [
  { key: 'python', label: 'Python' },
  { key: 'node', label: 'Node.js' },
  { key: 'typescript', label: 'TypeScript' },
  { key: 'java', label: 'Java' },
  { key: 'go', label: 'Go' },
  { key: 'c', label: 'C' },
  { key: 'cpp', label: 'C++' },
  { key: 'rust', label: 'Rust' },
  { key: 'php', label: 'PHP' },
  { key: 'dotnet', label: '.NET / C#' },
  { key: 'angular', label: 'Angular' },
  { key: 'react', label: 'React' },
  { key: 'vue', label: 'Vue' },
  { key: 'vite', label: 'Vite' },
  { key: 'fastapi', label: 'FastAPI' },
  { key: 'django', label: 'Django' },
  { key: 'flask', label: 'Flask' },
  { key: 'spring', label: 'Spring / Maven' },
  { key: 'laravel', label: 'Laravel' },
  { key: 'streamlit', label: 'Streamlit' },
  { key: 'bun', label: 'Bun' },
  { key: 'deno', label: 'Deno' },
  { key: 'postgres', label: 'PostgreSQL' },
  { key: 'mysql', label: 'MySQL' },
  { key: 'sqlite', label: 'SQLite' },
  { key: 'mongodb', label: 'MongoDB' },
  { key: 'redis', label: 'Redis' },
  { key: 'docker', label: 'Estándar' }
];

export const KNOWN_TECH_ICON_KEYS = new Set<string>([
  ...CURATED_TECH_ICONS.map(icon => icon.key),
  'git',
  'db',
  'nextjs',
  'mariadb'
]);

function isDirectAssetSource(value: string): boolean {
  return (
    value.startsWith('/') ||
    value.startsWith('https://') ||
    value.startsWith('http://') ||
    value.startsWith('data:image/')
  );
}

@Component({
  selector: 'tech-logo',
  imports: [],
  template: `
    <div
      class="tech-logo-container"
      [class]="'logo-' + normalizedKey()"
      [class.is-monogram]="!resolvedIconSrc()"
      [style.width.px]="size"
      [style.height.px]="size"
    >
      @if (resolvedIconSrc(); as src) {
        <img
          class="tech-logo-img"
          [src]="src"
          [alt]="normalizedKey()"
          [width]="size - 8"
          [height]="size - 8"
          loading="lazy"
          (error)="onImgError()"
        />
      } @else {
        <span class="tech-logo-monogram" [attr.aria-label]="monogram()">{{ monogram() }}</span>
      }
    </div>
  `,
  styleUrl: './tech-logo.component.scss'
})
export class TechLogoComponent {
  private readonly iconTypeSignal = signal<TechIconKey>('docker');
  private readonly customIconUrlSignal = signal<string>('');
  readonly imgLoadFailed = signal<boolean>(false);

  @Input({ required: true })
  set iconType(value: TechIconKey) {
    this.iconTypeSignal.set(value || 'docker');
    this.imgLoadFailed.set(false);
  }
  get iconType(): TechIconKey {
    return this.iconTypeSignal();
  }

  @Input()
  set customIconUrl(value: string | undefined) {
    this.customIconUrlSignal.set((value || '').trim());
    this.imgLoadFailed.set(false);
  }
  get customIconUrl(): string {
    return this.customIconUrlSignal();
  }

  @Input() size = 32;

  readonly normalizedKey = computed<string>(() => {
    const raw = (this.iconTypeSignal() || 'docker').trim();
    if (isDirectAssetSource(raw)) {
      if (raw.startsWith('data:image/')) {
        return 'custom';
      }
      const lastSegment = raw.split('/').pop() || 'custom';
      const withoutQuery = lastSegment.split('?')[0].split('#')[0];
      return withoutQuery.replace(/\.[a-z0-9]+$/i, '').toLowerCase() || 'custom';
    }
    return raw.toLowerCase();
  });

  readonly resolvedIconSrc = computed<string | null>(() => {
    if (this.imgLoadFailed()) {
      return null;
    }

    const customUrl = this.customIconUrlSignal();
    if (customUrl && isDirectAssetSource(customUrl)) {
      return customUrl;
    }

    const rawIcon = (this.iconTypeSignal() || '').trim();
    if (rawIcon && isDirectAssetSource(rawIcon)) {
      return rawIcon;
    }

    const key = this.normalizedKey();
    if (KNOWN_TECH_ICON_KEYS.has(key)) {
      return `/tech-icons/${key}.svg`;
    }

    return null;
  });

  readonly monogram = computed<string>(() => {
    const clean = this.normalizedKey().replace(/[^a-zA-Z0-9]/g, '').toUpperCase();
    return clean.slice(0, 2) || 'TX';
  });

  onImgError(): void {
    this.imgLoadFailed.set(true);
  }
}
