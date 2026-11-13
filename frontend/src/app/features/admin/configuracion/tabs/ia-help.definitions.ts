import { TourDefinition, TutorialDefinition } from '@shared/help/help.types';

/**
 * Tutorial de la pestaña Inteligencia Artificial.
 *
 * El recorrido llega hasta dejar el copiloto en uso: interruptor, proveedor,
 * credenciales, prueba de conexión y guardado. La gobernanza experta solo se
 * explica, no se modifica desde la guía.
 */

export interface IaTutorialContext {
  /** La IA está habilitada para la institución. */
  enabled: () => boolean;
  /** Identificador del proveedor seleccionado; vacío si no hay ninguno. */
  provider: () => string;
  /** Modelo de inferencia escrito o elegido del catálogo. */
  model: () => string;
  /** La última prueba de conexión respondió bien. */
  testPassed: () => boolean;
  /** La configuración se guardó en esta sesión. */
  hasSaved: () => boolean;
}

const MANUAL_FRAGMENT = '9-configuracion-de-inteligencia-artificial';

export function buildIaTutorial(context: IaTutorialContext): TutorialDefinition {
  return {
    id: 'config-ai-copilot',
    title: 'Dejar el copiloto en uso',
    manualFragment: MANUAL_FRAGMENT,
    steps: [
      {
        id: 'enable',
        title: 'Habilitar la IA institucional',
        body:
          'El interruptor decide si el copiloto del constructor y las tareas asistidas están disponibles para administradores y docentes. Nunca se usa para estudiantes durante una evaluación.',
        selector: '#ai-enabled-toggle',
        side: 'bottom',
        check: {
          isComplete: () => context.enabled(),
          pendingHint: 'Active el interruptor de IA para continuar.'
        }
      },
      {
        id: 'provider',
        title: 'Elegir el proveedor en uso',
        body:
          'Solo un proveedor recibe las solicitudes a la vez. «Agregar proveedor» permite registrar un endpoint propio; las tarjetas muestran si tienen clave guardada o si están sin configurar.',
        selector: '[data-tour="ai-providers"]',
        side: 'top',
        check: {
          isComplete: () => context.provider().trim().length > 0,
          pendingHint: 'Seleccione un proveedor para continuar.'
        }
      },
      {
        id: 'credentials',
        title: 'Clave, endpoint y modelo',
        body:
          'La clave se guarda en el servidor y nunca se muestra completa; para cambiarla se escribe una nueva. El modelo se elige del catálogo del proveedor o se escribe a mano.',
        selector: '[data-tour="ai-credentials"]',
        side: 'top',
        check: {
          isComplete: () => context.model().trim().length > 0,
          pendingHint: 'Defina el modelo de inferencia para continuar.'
        }
      },
      {
        id: 'test',
        title: 'Probar la conexión',
        body:
          'La prueba envía una solicitud real y devuelve latencia o el error del proveedor. Es la forma de saber que la clave y el endpoint funcionan antes de guardar.',
        selector: '[data-tour="ai-test"]',
        side: 'top',
        check: {
          isComplete: () => context.testPassed(),
          pendingHint: 'Ejecute «Probar Conexión» y espere una respuesta correcta.'
        }
      },
      {
        id: 'governance',
        title: 'Gobernanza del motor',
        body:
          'El motor calibra la cuota horaria y el tiempo máximo de espera según el perfil del modelo elegido. Los valores expertos solo se abren cuando hace falta ajustarlos a mano.',
        selector: '[data-tour="ai-governance"]',
        side: 'top'
      },
      {
        id: 'save',
        title: 'Guardar la configuración',
        body:
          'Guardar aplica el proveedor, la clave y la gobernanza en una sola operación. Si el formulario tiene errores, el botón queda bloqueado y el aviso indica qué corregir.',
        selector: '[data-tour="ai-save"]',
        side: 'top',
        check: {
          isComplete: () => context.hasSaved(),
          pendingHint: 'Guarde la configuración de IA para terminar la guía.'
        }
      }
    ]
  };
}

/**
 * Recorrido de la pestaña Inteligencia Artificial.
 *
 * Explica qué decide cada bloque de la configuración institucional. El tutorial
 * de la pestaña acompaña el orden interruptor, proveedor, credenciales, prueba
 * y guardado.
 */
export function buildIaTour(): TourDefinition {
  return {
    id: 'config-ai-tour-v1',
    title: 'Recorrido por Inteligencia Artificial',
    steps: [
      {
        selector: '[data-tour="config-tab-ia"]',
        title: 'Esta guía cubre una sola pestaña',
        description:
          'El recorrido se limita a Inteligencia Artificial. El copiloto se configura para toda la institución y consume la cuota de la clave cargada aquí.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="config-description"]',
        title: 'Una sola configuración para toda la institución',
        description:
          'Docentes y administradores usan este mismo copiloto. Cambiar de proveedor afecta a todas las plantillas y ejercicios que lo invoquen.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="ai-providers"]',
        title: 'Proveedor del copiloto',
        description:
          'El proveedor define qué modelos están disponibles y cómo se mide el consumo. El que aparece marcado es el que está en uso hoy.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="ai-credentials"]',
        title: 'Clave, endpoint y modelo',
        description:
          'La clave se guarda en el servidor y no se vuelve a mostrar completa. El endpoint solo se cambia cuando el proveedor lo exige.',
        side: 'right'
      },
      {
        selector: '[data-tour="ai-governance"]',
        title: 'Gobernanza del motor',
        description:
          'La cuota horaria y el tiempo máximo de espera se derivan del perfil del modelo. Los valores expertos solo se abren cuando hace falta ajustarlos a mano.',
        side: 'top'
      },
      {
        selector: '[data-tour="ai-save"]',
        title: 'Un guardado para toda la configuración',
        description:
          'Guardar aplica proveedor, clave y gobernanza en una sola operación. La prueba de conexión es opcional, pero confirma que la clave responde antes de guardar.',
        side: 'top'
      }
    ]
  };
}
