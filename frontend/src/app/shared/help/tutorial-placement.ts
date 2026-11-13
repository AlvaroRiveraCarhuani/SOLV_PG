/**
 * Posición del panel del tutorial respecto del elemento que el paso señala.
 *
 * El panel flota sobre la pantalla y, si se queda fijo en una esquina, puede
 * tapar justo el control que el paso pide usar. Esta función elige la primera
 * esquina que no se cruza con el objetivo y, si ninguna queda libre, la de menor
 * solape.
 */

export type TutorialPanelPlacement = 'top-left' | 'top-right' | 'bottom-left' | 'bottom-right';

export interface PlacementRect {
  left: number;
  top: number;
  right: number;
  bottom: number;
}

export interface PlacementSize {
  width: number;
  height: number;
}

const GAP = 24;
/** Margen extra alrededor del objetivo para que el panel no lo roce. */
const CLEARANCE = 8;
const PREFERENCE: TutorialPanelPlacement[] = [
  'top-right',
  'bottom-right',
  'top-left',
  'bottom-left'
];

function panelRect(
  placement: TutorialPanelPlacement,
  panel: PlacementSize,
  viewport: PlacementSize
): PlacementRect {
  const left = placement.endsWith('left') ? GAP : viewport.width - GAP - panel.width;
  const top = placement.startsWith('top') ? GAP : viewport.height - GAP - panel.height;
  return { left, top, right: left + panel.width, bottom: top + panel.height };
}

function inflate(rect: PlacementRect, amount: number): PlacementRect {
  return {
    left: rect.left - amount,
    top: rect.top - amount,
    right: rect.right + amount,
    bottom: rect.bottom + amount
  };
}

function overlapArea(a: PlacementRect, b: PlacementRect): number {
  const width = Math.min(a.right, b.right) - Math.max(a.left, b.left);
  const height = Math.min(a.bottom, b.bottom) - Math.max(a.top, b.top);
  if (width <= 0 || height <= 0) return 0;
  return width * height;
}

export function pickTutorialPanelPlacement(
  target: PlacementRect | null,
  panel: PlacementSize,
  viewport: PlacementSize
): TutorialPanelPlacement {
  if (!target) return 'bottom-right';

  const blocked = inflate(target, CLEARANCE);
  let best: { placement: TutorialPanelPlacement; area: number } | null = null;

  for (const placement of PREFERENCE) {
    const area = overlapArea(panelRect(placement, panel, viewport), blocked);
    if (area === 0) return placement;
    if (!best || area < best.area) best = { placement, area };
  }

  return best?.placement ?? 'bottom-right';
}
