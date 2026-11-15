import { TourDefinition, TutorialDefinition } from '@shared/help/help.types';

/**
 * Guías de la cola de Solicitudes de entornos.
 *
 * El tutorial no ejecuta la decisión: acerca la ficha, la plantilla y la
 * respuesta hasta los botones, y la aprobación queda en manos del
 * administrador, igual que el archivado de un período.
 */

export interface EnvironmentRequestsTutorialContext {
  /** Hay una solicitud abierta en el panel de revisión. */
  reviewOpen: () => boolean;
  /** Se eligió la plantilla que resuelve la solicitud. */
  hasTemplate: () => boolean;
  /** Respuesta escrita para el docente. */
  note: () => string;
}

const MANUAL_FRAGMENT = '3-2-ciclo-de-vida-y-gobernanza-de-plantillas';

/**
 * Recorrido de la cola: qué llega, cómo se lee una ficha y dónde se responde.
 */
export function buildEnvironmentRequestsTour(): TourDefinition {
  return {
    id: 'requests-tour-v1',
    title: 'Recorrido por Solicitudes de entornos',
    steps: [
      {
        selector: '[data-tour="requests-header"]',
        title: 'Lo que el catálogo no alcanza',
        description:
          'Cada ficha llega desde el docente con las piezas que reconoció el sistema, las que no existen en el catálogo y el uso que tendrá en clase. No es un formulario en blanco.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="requests-filters"]',
        title: 'En revisión y respondidas',
        description:
          'La cola se divide en dos: lo que todavía espera una respuesta y lo que ya se aprobó o rechazó. Cada botón muestra cuántas solicitudes contiene.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="requests-card"]',
        title: 'La ficha del pedido',
        description:
          'El encabezado dice quién pidió, cuándo y para qué modalidad. La modalidad pedida decide qué plantillas puede aceptar la solicitud.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="requests-pieces"]',
        title: 'Lo que el catálogo ya cubre',
        description:
          'Estas piezas existen en el catálogo de tecnologías. Si el pedido no trae piezas, el docente describió el entorno solo con texto y la decisión depende de esa explicación.',
        side: 'right'
      },
      {
        selector: '[data-tour="requests-missing"]',
        title: 'Lo que falta registrar',
        description:
          'Una pieza marcada como faltante todavía no está en el catálogo. Conviene registrarla y construir el entorno antes de aprobar, o responder que el pedido no es viable por ahora.',
        side: 'right'
      },
      {
        selector: '[data-tour="requests-uses"]',
        title: 'Uso declarado y materias',
        description:
          'El uso dice si el entorno se necesita para practicar en clase, para entregar y calificar o para ambas cosas. Las materias que lo esperan muestran el alcance real del pedido.',
        side: 'left'
      },
      {
        selector: '[data-tour="requests-open"]',
        title: 'Responder abre la revisión',
        description:
          'La respuesta se escribe dentro de la misma ficha, sin salir de la cola. Aprobar exige enlazar la plantilla que resuelve el pedido.',
        side: 'top'
      }
    ]
  };
}

/**
 * Tutorial de la cola: abre una ficha, valida la plantilla y la respuesta, y
 * termina explicando las tres salidas sin ejecutar ninguna.
 */
export function buildEnvironmentRequestsTutorial(
  context: EnvironmentRequestsTutorialContext
): TutorialDefinition {
  return {
    id: 'requests-review-queue',
    title: 'Responder una solicitud de entorno',
    manualFragment: MANUAL_FRAGMENT,
    steps: [
      {
        id: 'queue',
        title: 'Qué está esperando respuesta',
        body:
          '«En revisión» reúne los pedidos que todavía no tienen respuesta; «Respondidas» conserva los aprobados y los rechazados. Una solicitud respondida no vuelve a la cola.',
        selector: '[data-tour="requests-filters"]',
        side: 'bottom'
      },
      {
        id: 'open',
        title: 'Abra una solicitud en revisión',
        body:
          '«Responder» despliega el panel de revisión dentro de la ficha, con la plantilla y la respuesta. Cambiar de filtro no descarta lo escrito: primero cierre el panel con «Cancelar».',
        selector: '[data-tour="requests-open"]',
        side: 'top',
        check: {
          isComplete: () => context.reviewOpen(),
          pendingHint: 'Pulse «Responder» en una solicitud en revisión para continuar.'
        }
      },
      {
        id: 'template',
        title: 'Elija la plantilla que la resuelve',
        body:
          'Solo aparecen plantillas aprobadas que cubren la modalidad pedida. Si la lista está vacía, no hay nada que aprobar todavía: cree el entorno en el Constructor y vuelva a esta ficha.',
        selector: '[data-tour="requests-template"]',
        side: 'left',
        check: {
          isComplete: () => context.hasTemplate(),
          pendingHint: 'Elija una plantilla compatible con la modalidad pedida.'
        }
      },
      {
        id: 'note',
        title: 'La respuesta que recibe el docente',
        body:
          'Aprobar puede guardarse sin nota: el docente ve la plantilla enlazada. «Pedir ajuste» y «No es viable» exigen al menos una frase que explique qué falta o por qué el pedido no puede resolverse.',
        selector: '[data-tour="requests-note"]',
        side: 'left'
      },
      {
        id: 'decision',
        title: 'Las tres salidas',
        body:
          'Aprobar registra el entorno y mueve la solicitud a «Respondidas». Pedir ajuste la mantiene en revisión para que el docente complete el pedido. No es viable la cierra con el motivo. Aquí termina la guía: la decisión queda en sus manos.',
        selector: '[data-tour="requests-actions"]',
        side: 'top'
      }
    ]
  };
}
