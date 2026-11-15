import { TourDefinition, TutorialDefinition } from '@shared/help/help.types';

/**
 * Guías de la pestaña Emergencias.
 *
 * El tutorial acompaña la preparación de una operación hasta el botón de
 * confirmación y termina ahí: no ejecuta ninguna acción. Tipear la frase y el
 * motivo es seguro porque sin la confirmación final no ocurre nada.
 */

export interface EmergenciasTutorialContext {
  /** El modal de confirmación está abierto. */
  confirmationOpen: () => boolean;
  /** Frase escrita en el formulario. */
  phrase: () => string;
  /** Frase exacta que exige la acción elegida. */
  expectedPhrase: () => string;
  /** Motivo escrito en el formulario. */
  reason: () => string;
}

const MANUAL_FRAGMENT = '10-mantenimiento-y-consola-de-emergencias';

/**
 * Recorrido del panel: alcance, estado del host, catálogo de acciones e
 * historial.
 */
export function buildEmergenciasTour(): TourDefinition {
  return {
    id: 'emergency-console-tour-v1',
    title: 'Recorrido por Emergencias',
    steps: [
      {
        selector: '[data-tour="emergency-header"]',
        title: 'Qué se ejecuta desde aquí',
        description:
          'Son operaciones sobre el host, no sobre una materia. Cada ejecución queda en el Registro de Auditoría con el actor, la hora y el motivo escrito.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="emergency-system"]',
        title: 'El estado antes de decidir',
        description:
          'La franja muestra la memoria usada y los contenedores activos. Si supera el 90 %, aparece un aviso que recomienda hibernar antes de una acción destructiva.',
        side: 'left'
      },
      {
        selector: '[data-tour="emergency-actions"]',
        title: 'Cinco acciones, dos impactos',
        description:
          'Cada tarjeta declara si es destructiva u operativa. Detener y destruir contenedores pierde el trabajo no guardado; hibernar, limpiar huérfanos, purgar capas y reiniciar pools conservan lo necesario para seguir.',
        side: 'top'
      },
      {
        selector: '[data-tour="emergency-action-card"]',
        title: 'La confirmación no es un aviso',
        description:
          'Ejecutar abre un formulario que pide la frase exacta de esa acción y un motivo de al menos 10 caracteres. Sin las dos cosas el botón de confirmación sigue deshabilitado.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="emergency-history"]',
        title: 'Qué se ejecutó antes',
        description:
          'El historial lista las últimas emergencias con fecha, acción y responsable. Sirve para saber si el problema ya se intentó resolver y quién lo hizo.',
        side: 'top'
      }
    ]
  };
}

/**
 * Tutorial de la consola: prepara una operación real hasta la confirmación y
 * no la ejecuta.
 */
export function buildEmergenciasTutorial(context: EmergenciasTutorialContext): TutorialDefinition {
  return {
    id: 'emergency-console-prepare',
    title: 'Preparar una operación de emergencia',
    manualFragment: MANUAL_FRAGMENT,
    steps: [
      {
        id: 'scope',
        title: 'Qué queda registrado',
        body:
          'Toda ejecución guarda el actor, la hora, la acción y el motivo en el Registro de Auditoría. Esta guía no ejecuta nada: solo prepara el formulario y termina antes de confirmar.',
        selector: '[data-tour="emergency-header"]',
        side: 'bottom'
      },
      {
        id: 'impact',
        title: 'Elija el impacto correcto',
        body:
          'Antes de abrir una acción, compare su impacto. Si la memoria está arriba del 90 %, hibernar primero reduce el riesgo de una operación destructiva. Hibernar conserva el estado; terminar lo destruye.',
        selector: '[data-tour="emergency-actions"]',
        side: 'top'
      },
      {
        id: 'history',
        title: 'Revise si ya se intentó',
        body:
          'El historial dice qué se ejecutó y quién lo hizo. Si la misma acción ya se corrió hace minutos, conviene leer el resultado antes de repetirla.',
        selector: '[data-tour="emergency-history"]',
        side: 'top'
      },
      {
        id: 'open',
        title: 'Abra la acción que va a revisar',
        body:
          '«Ejecutar acción» abre el formulario de confirmación. Desde aquí la guía acompaña el llenado; el botón final queda en sus manos.',
        selector: '[data-tour="emergency-action-card"]',
        side: 'bottom',
        check: {
          isComplete: () => context.confirmationOpen(),
          pendingHint: 'Pulse «Ejecutar acción» en la operación que necesita revisar.'
        }
      },
      {
        id: 'phrase',
        title: 'La frase se escribe igual',
        body:
          'La frase debe coincidir carácter por carácter con la que muestra el diálogo. No se copia ni se acepta una variante: es la barrera contra un clic accidental.',
        selector: '[data-tour="emergency-phrase"]',
        side: 'left',
        check: {
          isComplete: () =>
            context.confirmationOpen() &&
            context.expectedPhrase() !== '' &&
            context.phrase() === context.expectedPhrase(),
          pendingHint: 'Escriba la frase exacta que muestra el diálogo para continuar.'
        }
      },
      {
        id: 'reason',
        title: 'El motivo queda en auditoría',
        body:
          'El motivo explica por qué se ejecuta y se guarda junto al evento. Diez caracteres es el mínimo: describa la situación, no escriba «ok» ni «urgente».',
        selector: '[data-tour="emergency-reason"]',
        side: 'left',
        check: {
          isComplete: () =>
            context.confirmationOpen() && context.reason().trim().length >= 10,
          pendingHint: 'Escriba un motivo de al menos 10 caracteres para continuar.'
        }
      },
      {
        id: 'confirm',
        title: 'Aquí termina la guía',
        body:
          'Con la frase y el motivo completos, «Confirmar y ejecutar» se habilita. La guía no lo pulsa: cancelar cierra el diálogo sin ejecutar y ejecutar es una decisión suya, con el impacto ya explicado.',
        selector: '[data-tour="emergency-submit"]',
        side: 'top'
      }
    ]
  };
}
