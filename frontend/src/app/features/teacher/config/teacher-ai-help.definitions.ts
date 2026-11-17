import { TourDefinition, TutorialDefinition } from '@shared/help/help.types';

/**
 * Contexto reactivo para evaluar el estado real de la pantalla en cada paso del tutorial.
 * La progresión la decide Angular mediante lecturas a este contrato.
 */
export interface TeacherAiTutorialContext {
  /** El interruptor de clave personal BYOK está activo */
  enabled: () => boolean;
  /** Identificador del proveedor elegido; vacío si no hay ninguno */
  provider: () => string;
  /** Indica si hay una clave escrita en el campo o ya persistida previamente */
  hasKey: () => boolean;
  /** Modelo de inferencia seleccionado del catálogo o ingresado a mano */
  model: () => string;
  /** La última prueba de conexión respondió satisfactoriamente */
  testPassed: () => boolean;
  /** La configuración se guardó en la sesión actual */
  hasSaved: () => boolean;
}

/**
 * Construye la definición del tutorial interactivo para configurar la clave personal de IA.
 * Acompaña el orden: activar interruptor -> proveedor -> credenciales y modelo -> prueba -> guardado.
 */
export function buildTeacherAiTutorial(context: TeacherAiTutorialContext): TutorialDefinition {
  return {
    id: 'teacher-ai-byok-tutorial',
    title: 'Configurar mi clave personal de IA',
    manualFragment: null,
    steps: [
      {
        id: 'enable-byok',
        title: 'Activar la clave personal',
        body:
          'Al activar el interruptor de clave personal, el copiloto usará su propia cuenta y cuota en lugar del motor compartido de la institución.',
        selector: '[data-tour="teacher-ai-byok-toggle"]',
        side: 'bottom',
        check: {
          isComplete: () => context.enabled(),
          pendingHint: 'Active el interruptor de clave personal para continuar.'
        }
      },
      {
        id: 'select-provider',
        title: 'Elegir el proveedor',
        body:
          'Seleccione el proveedor donde tiene su cuenta (OpenAI, Anthropic, Google Gemini o agregue un proveedor compatible).',
        selector: '[data-tour="teacher-ai-providers"]',
        side: 'top',
        check: {
          isComplete: () => context.provider().trim().length > 0,
          pendingHint: 'Seleccione un proveedor para continuar.'
        }
      },
      {
        id: 'credentials-and-model',
        title: 'Ingresar API Key y modelo',
        body:
          'Pegue su clave personal y elija un modelo del catálogo o ingrese su identificador manualmente. Su clave se guarda cifrada en el servidor.',
        selector: '[data-tour="teacher-ai-credentials"]',
        side: 'top',
        check: {
          isComplete: () => context.hasKey() && context.model().trim().length > 0,
          pendingHint: 'Ingrese su clave API y seleccione un modelo para continuar.'
        }
      },
      {
        id: 'test-connection',
        title: 'Probar la conexión',
        body:
          'La prueba envía una solicitud real al proveedor para validar que las credenciales y el modelo respondan antes de persistir.',
        selector: '[data-tour="teacher-ai-test"]',
        side: 'top',
        check: {
          isComplete: () => context.testPassed(),
          pendingHint: 'Ejecute «Probar Conexión» y espere una respuesta correcta.'
        }
      },
      {
        id: 'save-configuration',
        title: 'Guardar la configuración',
        body:
          'Guardar activa su clave para todas las operaciones del copiloto en la redacción de ejercicios.',
        selector: '[data-tour="teacher-ai-save"]',
        side: 'top',
        check: {
          isComplete: () => context.hasSaved(),
          pendingHint: 'Haga clic en «Guardar configuración» para finalizar la guía.'
        }
      }
    ]
  };
}

/**
 * Construye el recorrido visual por la pantalla de Inteligencia Artificial del docente.
 * Señala los bloques de estado, el motor institucional y la configuración personal.
 */
export function buildTeacherAiTour(): TourDefinition {
  return {
    id: 'teacher-ai-tour-v1',
    title: 'Recorrido por Mi Inteligencia Artificial',
    steps: [
      {
        selector: '[data-tour="teacher-ai-usage"]',
        title: 'Estado del copiloto',
        description:
          'Indica claramente qué motor está atendiendo sus solicitudes: el compartido de la institución, su clave personal o si la IA está inactiva.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="teacher-ai-institutional"]',
        title: 'Motor institucional',
        description:
          'Muestra el proveedor y modelo que la institución ofrece por defecto. Si activa su clave personal, este motor queda en pausa para su cuenta.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="teacher-ai-byok-toggle"]',
        title: 'Interruptor de clave personal',
        description:
          'Permite alternar entre su clave propia y el motor institucional en cualquier momento sin perder la configuración.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="teacher-ai-providers"]',
        title: 'Proveedores disponibles',
        description:
          'Seleccione un proveedor predeterminado o sume endpoints compatibles con la API de OpenAI (como Groq, Mistral o Together AI).',
        side: 'top'
      },
      {
        selector: '[data-tour="teacher-ai-credentials"]',
        title: 'Credencial y modelo',
        description:
          'Ingrese su API Key personal y explore el catálogo de modelos disponibles o ingrese un identificador específico.',
        side: 'top'
      },
      {
        selector: '[data-tour="teacher-ai-test"]',
        title: 'Prueba de conexión',
        description:
          'Verifica la latencia y la validez de su credencial con una consulta real antes de guardar.',
        side: 'top'
      },
      {
        selector: '[data-tour="teacher-ai-save"]',
        title: 'Guardar cambios',
        description:
          'Aplica la configuración inmediatamente para su usuario en todas las herramientas del panel docente.',
        side: 'top'
      }
    ]
  };
}
