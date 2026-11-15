import { TourDefinition, TutorialDefinition } from '@shared/help/help.types';

/**
 * Guías de la pestaña Registro de Auditoría.
 *
 * El tutorial enseña a acotar la búsqueda y a leer la cronología de un actor.
 * El registro es inmutable: no hay acciones de edición ni de borrado que el
 * tutorial pueda ejecutar.
 */

export interface AuditRegistroTutorialContext {
  /** Hay una búsqueda o un filtro de acción aplicado. */
  hasActiveFilters: () => boolean;
  /** La cronología del actor está abierta. */
  drawerOpen: () => boolean;
}

const MANUAL_FRAGMENT = '5-2-registro-de-auditoria';

/**
 * Recorrido del registro: pestañas, filtros, tabla, cronología y paginación.
 */
export function buildAuditRegistroTour(): TourDefinition {
  return {
    id: 'audit-log-tour-v1',
    title: 'Recorrido por el Registro de Auditoría',
    steps: [
      {
        selector: '[data-tour="audit-tabs"]',
        title: 'Dos secciones, dos guías',
        description:
          'Auditoría reúne el registro cronológico y la consola de emergencias. Cada pestaña tiene su propio recorrido y su propio tutorial.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="audit-toolbar"]',
        title: 'Acotar antes de leer',
        description:
          'La búsqueda cubre evento, actor y recurso. El filtro de acción separa creaciones, actualizaciones y eliminaciones. «Limpiar filtros» aparece solo cuando hay algo aplicado.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="audit-table"]',
        title: 'Qué cuenta cada columna',
        description:
          'Cada fila trae la hora, el actor, la acción con su motivo, el recurso y el resultado. El estado distingue un evento exitoso de uno rechazado o fallido.',
        side: 'top'
      },
      {
        selector: '[data-tour="audit-timeline"]',
        title: 'La cronología del actor',
        description:
          'La última columna abre la actividad completa de ese actor en un panel lateral. Sirve para ver qué hizo antes y después del evento que se está investigando.',
        side: 'left'
      },
      {
        selector: '[data-tour="audit-pagination"]',
        title: 'Páginas y total declarado',
        description:
          'La paginación muestra el rango visible sobre el total de eventos. El registro no se edita ni se borra desde aquí: es solo lectura.',
        side: 'top'
      }
    ]
  };
}

/**
 * Tutorial de la pestaña: aplicar un filtro real y abrir una cronología.
 */
export function buildAuditRegistroTutorial(
  context: AuditRegistroTutorialContext
): TutorialDefinition {
  return {
    id: 'audit-log-investigate',
    title: 'Investigar un evento',
    manualFragment: MANUAL_FRAGMENT,
    steps: [
      {
        id: 'tabs',
        title: 'Elija el registro',
        body:
          'Esta guía cubre el Registro de Auditoría. La pestaña Emergencias tiene su propio recorrido y su propio tutorial, porque las operaciones que ejecuta son distintas.',
        selector: '[data-tour="audit-tabs"]',
        side: 'bottom'
      },
      {
        id: 'narrow',
        title: 'Acote por actor, recurso o acción',
        body:
          'Escriba un correo, un recurso o parte del nombre de un evento en la búsqueda. El filtro de acción sirve para ver solo altas, cambios o bajas de un período de trabajo.',
        selector: '[data-tour="audit-toolbar"]',
        side: 'bottom'
      },
      {
        id: 'filter',
        title: 'Aplique el filtro',
        body:
          'La búsqueda se ejecuta sola después de una pausa corta; el filtro de acción se aplica al elegirlo. Cuando hay algo aplicado aparece «Limpiar filtros» para volver a la lista completa.',
        selector: '[data-tour="audit-toolbar"]',
        side: 'bottom',
        check: {
          isComplete: () => context.hasActiveFilters(),
          pendingHint: 'Escriba una búsqueda o elija una acción para continuar.'
        }
      },
      {
        id: 'open',
        title: 'Abra la cronología del actor',
        body:
          'El icono de la última columna abre la actividad de ese actor. Es el paso que convierte un evento aislado en una secuencia: qué hizo antes, qué hizo después y con qué resultado.',
        selector: '[data-tour="audit-open"]',
        side: 'left',
        check: {
          isComplete: () => context.drawerOpen(),
          pendingHint: 'Abra la cronología de una fila para continuar.'
        }
      },
      {
        id: 'read',
        title: 'Qué mirar en la cronología',
        body:
          'Cada entrada trae la acción, el recurso, el resultado, la hora y el motivo cuando existe. Un rechazo repetido o una ráfaga de cambios del mismo actor suele explicar el incidente. El registro no se modifica desde aquí.',
        selector: '[data-tour="audit-drawer"]',
        side: 'left'
      }
    ]
  };
}
