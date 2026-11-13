import { TourDefinition, TutorialDefinition } from '@shared/help/help.types';

/**
 * Tutorial de la pestaña Períodos Académicos.
 *
 * Cada comprobación lee el estado real del formulario y de la lista: el
 * tutorial no avanza por pulsar «Siguiente», sino cuando el período existe.
 */

export interface PeriodosTutorialContext {
  /** El modal de alta o edición está abierto. */
  formOpen: () => boolean;
  /** Nombre del período en el formulario. */
  name: () => string;
  /** Código corto del período en el formulario. */
  code: () => string;
  /** Nombre, código y fechas coherentes. */
  isFormValid: () => boolean;
  /** Se creó o actualizó un período en esta sesión. */
  hasCreated: () => boolean;
  /** Períodos marcados como vigentes. */
  activeCount: () => number;
}

const MANUAL_FRAGMENT = '4-1-periodos-academicos-semestres';

export function buildPeriodosTutorial(context: PeriodosTutorialContext): TutorialDefinition {
  return {
    id: 'config-period-first-cycle',
    title: 'Crear y activar un período',
    manualFragment: MANUAL_FRAGMENT,
    steps: [
      {
        id: 'summary',
        title: 'Qué resumen los indicadores',
        body:
          'Los cuatro indicadores separan lo que está registrado, lo que está vigente, lo que empieza más adelante y lo que ya quedó archivado. Solo puede haber un período vigente a la vez.',
        selector: '[data-tour="periods-kpis"]',
        side: 'bottom'
      },
      {
        id: 'create',
        title: 'Abra el alta del período',
        body:
          '«Nuevo Período» abre el formulario de alta. Desde la misma tabla se edita un período existente con el lápiz, sin crear uno nuevo.',
        selector: '[data-tour="periods-create"]',
        side: 'bottom',
        check: {
          isComplete: () => context.formOpen(),
          pendingHint: 'Abra «Nuevo Período» para continuar.'
        }
      },
      {
        id: 'identity',
        title: 'Nombre y código institucional',
        body:
          'El nombre es el que verán docentes y estudiantes; el código corto es el que se escribe para confirmar operaciones delicadas. Ambos son obligatorios y deben ser reconocibles.',
        selector: '[data-tour="periods-identity"]',
        side: 'right',
        check: {
          isComplete: () => context.name().trim().length > 0 && context.code().trim().length > 0,
          pendingHint: 'Complete el nombre y el código del período para continuar.'
        }
      },
      {
        id: 'dates',
        title: 'Vigencia del ciclo lectivo',
        body:
          'La fecha de fin tiene que ser posterior a la de inicio. La casilla inferior deja el período activo de inmediato; si no se marca, queda planificado para cuando llegue su fecha.',
        selector: '[data-tour="periods-dates"]',
        side: 'left',
        check: {
          isComplete: () => context.isFormValid(),
          pendingHint: 'Complete ambas fechas con un rango válido para continuar.'
        }
      },
      {
        id: 'submit',
        title: 'Registre el período',
        body:
          'El botón de guardado permanece deshabilitado hasta que el formulario es válido. Al registrar, el período aparece en la tabla con su estado real.',
        selector: '[data-tour="periods-submit"]',
        side: 'top',
        check: {
          isComplete: () => context.hasCreated(),
          pendingHint: 'Guarde el período para continuar.'
        }
      },
      {
        id: 'lifecycle',
        title: 'Activar, archivar y eliminar',
        body:
          'Un período planificado se activa desde su fila. Archivar es irreversible y exige escribir el código del período; eliminar solo se permite mientras no tenga materias asociadas. Aquí termina la guía: estas acciones quedan en sus manos.',
        selector: '[data-tour="periods-row-actions"]',
        side: 'left'
      }
    ]
  };
}

/**
 * Recorrido de la pestaña Períodos Académicos.
 *
 * Es el mapa de la pestaña, no el flujo de alta: el tutorial de esta misma
 * pestaña enseña a crear un período. Aquí solo se explica qué hay y qué
 * permite cada bloque.
 */
export function buildPeriodosTour(): TourDefinition {
  return {
    id: 'config-periods-tour-v1',
    title: 'Recorrido por Períodos Académicos',
    steps: [
      {
        selector: '[data-tour="config-tab-periodos"]',
        title: 'Esta guía cubre una sola pestaña',
        description:
          'El recorrido se limita a Períodos Académicos. Las otras cuatro pestañas de Configuración tienen el suyo, con sus propios bloques y sus propios datos.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="config-description"]',
        title: 'El alcance de la pestaña abierta',
        description:
          'La línea de contexto confirma qué gobierna esta pestaña. Cambiar de pestaña descarta lo que no se haya guardado en la anterior.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="periods-kpis"]',
        title: 'Los cuatro estados del ciclo',
        description:
          'Registrados, vigentes, próximos y archivados. Solo un período puede estar vigente, y es el que usan las demás pantallas para inscribir materias.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="periods-create"]',
        title: 'Altas y ediciones en el formulario',
        description:
          'El alta y la edición se guardan dentro del formulario; la tabla no tiene un guardado propio. El mismo control abre el formulario vacío o con los datos de la fila.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="periods-row-actions"]',
        title: 'Cada fila muestra solo lo posible',
        description:
          'Las acciones dependen del estado: activar un período próximo, archivar uno vigente o editarlo mientras no esté archivado. El archivado es irreversible.',
        side: 'left'
      }
    ]
  };
}
