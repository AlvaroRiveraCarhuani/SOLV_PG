import { TourDefinition, TutorialDefinition } from '@shared/help/help.types';

/**
 * Tutorial de la pestaña Identidad (White-Label).
 *
 * La vista previa es la comprobación real: el color se valida contra el
 * contraste WCAG y el guardado solo se habilita cuando hay cambios válidos.
 */

export interface IdentidadTutorialContext {
  /** Nombre institucional en el formulario. */
  institutionName: () => string;
  /** El color primario es un hex válido. */
  colorValid: () => boolean;
  /** El texto sobre el color primario cumple contraste AA. */
  contrastOk: () => boolean;
  /** La marca se guardó en esta sesión. */
  hasSaved: () => boolean;
}

export function buildIdentidadTutorial(context: IdentidadTutorialContext): TutorialDefinition {
  return {
    id: 'config-branding-identity',
    title: 'Aplicar la identidad institucional',
    manualFragment: null,
    steps: [
      {
        id: 'identity',
        title: 'Nombre y logo de la institución',
        body:
          'El nombre aparece en la barra superior, en los correos y en el inicio de sesión. El logo admite un archivo de hasta 2 MB o una URL https; conviene usar una imagen con fondo transparente.',
        selector: '[data-tour="branding-identity"]',
        side: 'right',
        check: {
          isComplete: () => context.institutionName().trim().length > 0,
          pendingHint: 'Escriba el nombre de la institución para continuar.'
        }
      },
      {
        id: 'color',
        title: 'Color primario de la marca',
        body:
          'El color se escribe en formato #RRGGBB. De él se derivan hover, foco y badges, así que un valor inválido deja toda la interfaz sin acento.',
        selector: '[data-tour="branding-color"]',
        side: 'right',
        check: {
          isComplete: () => context.colorValid(),
          pendingHint: 'Corrija el color primario para continuar.'
        }
      },
      {
        id: 'typography',
        title: 'Tipografía institucional',
        body:
          'La fuente de interfaz afecta textos, botones, tablas y formularios; la monoespaciada se usa en código, hashes y números. «Opción avanzada» permite pegar una hoja de Google Fonts propia.',
        selector: '[data-tour="branding-typography"]',
        side: 'right'
      },
      {
        id: 'preview',
        title: 'Vista previa en vivo',
        body:
          'El panel derecho muestra la marca aplicada y verifica el contraste del texto sobre el color primario. Si el contraste no cumple, la interfaz avisa antes de guardar.',
        selector: '[data-tour="branding-preview"]',
        side: 'left',
        check: {
          isComplete: () => context.contrastOk(),
          pendingHint: 'Ajuste el color hasta que el contraste sea válido para continuar.'
        }
      },
      {
        id: 'save',
        title: 'Guardar la marca',
        body:
          'El botón solo se habilita cuando hay cambios y todo es válido. Guardar tarda hasta 5 minutos en propagarse a todos los usuarios por la caché institucional; «Cancelar Cambios» vuelve a los valores guardados.',
        selector: '[data-tour="branding-save"]',
        side: 'top',
        check: {
          isComplete: () => context.hasSaved(),
          pendingHint: 'Guarde la marca para terminar la guía.'
        }
      }
    ]
  };
}

/**
 * Recorrido de la pestaña Identidad (White-Label).
 *
 * Describe los bloques de la marca y su alcance; el tutorial de la pestaña
 * enseña el orden de edición y las comprobaciones de contraste.
 */
export function buildIdentidadTour(): TourDefinition {
  return {
    id: 'config-branding-tour-v1',
    title: 'Recorrido por Identidad (White-Label)',
    steps: [
      {
        selector: '[data-tour="config-tab-identidad"]',
        title: 'Esta guía cubre una sola pestaña',
        description:
          'El recorrido se limita a Identidad. Las otras pestañas de Configuración tienen el suyo, y cambiar de pestaña descarta los cambios sin guardar.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="config-description"]',
        title: 'La marca alcanza toda la plataforma',
        description:
          'Lo que se define aquí no se queda en el panel administrativo: se aplica al inicio de sesión, a los correos y a las pantallas de docentes y estudiantes.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="branding-identity"]',
        title: 'Nombre y logo institucional',
        description:
          'El nombre aparece en la barra superior, en los correos y en el inicio de sesión. El logo se toma de un archivo o de una URL https, y conviene que tenga fondo transparente.',
        side: 'right'
      },
      {
        selector: '[data-tour="branding-color"]',
        title: 'Color primario y tipografía',
        description:
          'Del color salen hover, foco y badges. La fuente de interfaz afecta textos y controles; la monoespaciada se reserva para código, hashes y números.',
        side: 'right'
      },
      {
        selector: '[data-tour="branding-preview"]',
        title: 'La marca aplicada antes de guardar',
        description:
          'La vista previa usa el color y la tipografía elegidos y verifica el contraste del texto. Si el contraste no cumple, el guardado no se habilita.',
        side: 'left'
      },
      {
        selector: '[data-tour="branding-save"]',
        title: 'Un solo guardado para toda la marca',
        description:
          '«Guardar» aplica nombre, logo, color y tipografía a la vez. La propagación tarda hasta 5 minutos por la caché institucional; «Cancelar Cambios» vuelve a lo guardado.',
        side: 'top'
      }
    ]
  };
}
