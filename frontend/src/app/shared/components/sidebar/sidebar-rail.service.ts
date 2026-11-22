import { Injectable, computed, inject, signal } from '@angular/core';
import { Router } from '@angular/router';

/**
 * Vistas que maximizan el área de trabajo: el sidebar arranca en modo rail
 * (52px) y se puede expandir a demanda. La lista cubre las rutas vigentes de
 * la aplicación y los alias de la especificación que todavía no se renombraron.
 */
const RAIL_BY_DEFAULT_ROUTES: readonly RegExp[] = [
  /^\/student\/?$/,
  /^\/student\/materias\/?$/,
  /^\/student\/courses\/?$/,
  /^\/student\/courses\/[^/]+\/?$/,
  /^\/student\/evaluations\/?$/,
  /^\/student\/evaluaciones\/?$/,
  /^\/student\/exercises\/[^/]+\/?$/,
  /^\/teacher\/?$/,
  /^\/teacher\/dashboard\/?$/,
  /^\/teacher\/cursos\/[^/]+\/?$/,
  /^\/teacher\/cursos\/[^/]+\/practicas\/[^/]+\/?$/,
  /^\/teacher\/cursos\/[^/]+\/ejercicios\/nuevo\/?$/,
  /^\/teacher\/cursos\/[^/]+\/ejercicios\/[^/]+\/editar\/?$/,
  /^\/teacher\/aula\/?$/,
  /^\/teacher\/por-calificar\/?$/,
  /^\/teacher\/revision\/[^/]+\/?$/,
  /^\/teacher\/submissions\/[^/]+\/review\/?$/
];

const STORAGE_KEY = 'solv.sidebar.collapsed';

/** Normaliza la URL para comparar solo el path, sin query ni fragmento. */
export function railPath(url: string): string {
  return url.split(/[?#]/)[0];
}

/** Indica si la ruta debe iniciar con el sidebar en modo rail. */
export function defaultsToRail(url: string): boolean {
  const path = railPath(url);
  return RAIL_BY_DEFAULT_ROUTES.some((pattern) => pattern.test(path));
}

/**
 * Estado único del rail del shell. La ruta define el valor inicial, pero una
 * vez que el usuario elige expandir o contraer, esa preferencia manda: navegar
 * entre vistas no vuelve a aplicar el default de la ruta. La elección se
 * persiste para conservarla entre recargas.
 */
@Injectable({ providedIn: 'root' })
export class SidebarRailService {
  private readonly router = inject(Router);
  private readonly collapsedState = signal(this.readInitialState());

  readonly isCollapsed = this.collapsedState.asReadonly();
  readonly isRail = computed(() => this.collapsedState());

  toggle(): void {
    this.setCollapsed(!this.collapsedState());
  }

  setCollapsed(collapsed: boolean): void {
    this.collapsedState.set(collapsed);
    this.persist(collapsed);
  }

  private readInitialState(): boolean {
    const stored = this.readStoredPreference();
    return stored ?? defaultsToRail(this.router.url);
  }

  private readStoredPreference(): boolean | null {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (raw === 'true') return true;
      if (raw === 'false') return false;
    } catch {
      // Navegación privada o almacenamiento bloqueado: se usa el default de ruta.
    }
    return null;
  }

  private persist(collapsed: boolean): void {
    try {
      localStorage.setItem(STORAGE_KEY, String(collapsed));
    } catch {
      // Sin almacenamiento disponible el estado sigue viviendo en memoria.
    }
  }
}
