import { TourDefinition } from '@shared/help/help.types';

/**
 * Recorrido de Categorías Académicas.
 *
 * No lleva tutorial: el flujo es corto y la única restricción real (no borrar
 * una categoría con entornos asociados) se explica en la entrada contextual y
 * en el último paso del recorrido.
 */
export function buildCategoriesTour(): TourDefinition {
  return {
    id: 'academic-categories-tour-v1',
    title: 'Recorrido por Categorías Académicas',
    steps: [
      {
        selector: '[data-tour="categories-summary"]',
        title: 'Cuántas hay registradas',
        description:
          'La categoría agrupa plantillas en la biblioteca y ordena los filtros del catálogo. Una categoría inactiva deja de ofrecerse sin borrar las plantillas que la usan.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="categories-create"]',
        title: 'Crear una categoría',
        description:
          'El alta pide nombre y descripción. La categoría nueva aparece al final de la lista y desde ahí se ordena; el nombre es el que ven docentes y estudiantes en los filtros.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="categories-list"]',
        title: 'El orden se edita desde la lista',
        description:
          'Las flechas o el arrastre cambian el orden en que se muestran las categorías. Ese orden se usa en la biblioteca de entornos y en los selectores del constructor.',
        side: 'top'
      },
      {
        selector: '[data-tour="categories-list"]',
        title: 'Una categoría con entornos no se borra',
        description:
          'La pantalla impide eliminarla mientras tenga plantillas o entornos asociados. Primero se reasignan esos elementos a otra categoría; después el borrado queda disponible.',
        side: 'top'
      }
    ]
  };
}
