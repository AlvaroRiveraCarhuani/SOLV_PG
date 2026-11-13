import { HelpGuideAction } from '@shared/help/help.types';

/**
 * Guía de recorrido de una pestaña de Configuración Institucional.
 *
 * El centro de ayuda lee solo las guías de la pantalla superior de la pila, así
 * que cada pestaña registra su propio recorrido junto a su tutorial. Los pasos
 * de cada recorrido viven en el archivo de definiciones de esa pestaña y solo
 * señalan sus bloques.
 */
export function configTabTourGuide(run: () => void): HelpGuideAction {
  return {
    label: 'Recorrido visual',
    hint: 'Señala los bloques de esta pestaña sin cambiar datos.',
    run
  };
}
