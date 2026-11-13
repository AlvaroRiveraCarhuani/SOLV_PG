import { describe, it, expect } from 'vitest';
import { pickTutorialPanelPlacement } from './tutorial-placement';

const panel = { width: 360, height: 300 };
const viewport = { width: 1280, height: 800 };

describe('Posición del panel del tutorial', () => {
  it('usa la esquina superior derecha cuando no hay objetivo', () => {
    expect(pickTutorialPanelPlacement(null, panel, viewport)).toBe('bottom-right');
  });

  it('se aparta de un objetivo en la esquina inferior derecha', () => {
    const target = { left: 900, top: 620, right: 1180, bottom: 700 };

    expect(pickTutorialPanelPlacement(target, panel, viewport)).toBe('top-right');
  });

  it('se aparta de un objetivo en la esquina superior derecha', () => {
    const target = { left: 940, top: 40, right: 1160, bottom: 110 };

    expect(pickTutorialPanelPlacement(target, panel, viewport)).toBe('bottom-right');
  });

  it('elige la esquina opuesta cuando el objetivo ocupa el centro', () => {
    const target = { left: 500, top: 320, right: 780, bottom: 470 };

    expect(pickTutorialPanelPlacement(target, panel, viewport)).toBe('top-right');
  });

  it('con el objetivo cubriendo toda la pantalla elige la de menor solape', () => {
    const target = { left: 0, top: 0, right: viewport.width, bottom: viewport.height };

    expect(pickTutorialPanelPlacement(target, panel, viewport)).toBe('top-right');
  });
});
