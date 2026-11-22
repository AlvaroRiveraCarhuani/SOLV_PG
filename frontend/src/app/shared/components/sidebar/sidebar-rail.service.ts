import { Injectable, computed, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { NavigationEnd, Router } from '@angular/router';
import { filter } from 'rxjs/operators';

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
 * Estado único del rail del shell. Cada layout de rol consume la misma señal:
 * el ancho inicial depende de la ruta y el usuario puede expandirlo o
 * contraerlo; al cambiar de ruta se reaplica el valor por defecto.
 */
@Injectable({ providedIn: 'root' })
export class SidebarRailService {
  private readonly router = inject(Router);
  private readonly collapsedState = signal(defaultsToRail(this.router.url));

  readonly isCollapsed = this.collapsedState.asReadonly();
  readonly isRail = computed(() => this.collapsedState());

  constructor() {
    this.router.events
      .pipe(
        filter((event): event is NavigationEnd => event instanceof NavigationEnd),
        takeUntilDestroyed()
      )
      .subscribe((event) => {
        this.collapsedState.set(defaultsToRail(event.urlAfterRedirects));
      });
  }

  toggle(): void {
    this.collapsedState.update((collapsed) => !collapsed);
  }

  setCollapsed(collapsed: boolean): void {
    this.collapsedState.set(collapsed);
  }
}
