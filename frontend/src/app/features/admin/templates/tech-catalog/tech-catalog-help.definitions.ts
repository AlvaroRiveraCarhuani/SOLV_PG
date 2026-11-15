import { TourDefinition, TutorialDefinition } from '@shared/help/help.types';

/**
 * Guías del Catálogo de Tecnologías.
 *
 * El tutorial recorre el caso que no se adivina: qué hacer con una tecnología
 * que ya usan otras plantillas. No elimina ni desactiva nada. El catálogo
 * oficial no ofrece eliminar, así que la guía enseña la salida real
 * —desactivar— en lugar de exigir un botón que puede no existir.
 */

export interface TechCatalogTutorialContext {
  /** Hay una búsqueda escrita en el filtro. */
  hasSearchQuery: () => boolean;
}

const MANUAL_FRAGMENT = '8-2-catalogo-de-tecnologias';

/**
 * Recorrido del catálogo: resumen, pestañas, filtros, tarjetas y acciones.
 */
export function buildTechCatalogTour(): TourDefinition {
  return {
    id: 'tech-catalog-tour-v1',
    title: 'Recorrido por el Catálogo de Tecnologías',
    steps: [
      {
        selector: '[data-tour="tech-summary"]',
        title: 'Qué compone los entornos',
        description:
          'El catálogo reúne lenguajes, interfaces web, complementos y bases de datos. Los contadores dicen cuántas tecnologías hay y cuántas están activas para el constructor.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="tech-tabs"]',
        title: 'El catálogo y su mapa',
        description:
          '«Tecnologías y Componentes» es la gestión. «Mapa del Sistema» es una vista de solo lectura que relaciona lenguajes, capas y complementos compatibles.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="tech-filters"]',
        title: 'Filtrar antes de tocar',
        description:
          'La búsqueda cubre nombre, slug, herramientas y paquete. Los filtros de perfil y estado acotan por tipo de tecnología y por disponibilidad en el constructor.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="tech-grid"]',
        title: 'Qué declara cada tarjeta',
        description:
          'Perfil, memoria estimada, ecosistema, paquetes, herramientas y uso. Una tarjeta con «En uso» la referencian plantillas existentes; «Huérfano» marca un complemento que ningún lenguaje activo admite.',
        side: 'top'
      },
      {
        selector: '[data-tour="tech-card-actions"]',
        title: 'Activar, editar y dar de baja',
        description:
          'El interruptor decide si la tecnología se ofrece en el constructor sin borrar historial. Editar corrige sus datos. Eliminar solo se permite cuando ninguna plantilla la usa; si está en uso, la salida es desactivarla.',
        side: 'left'
      }
    ]
  };
}

/**
 * Tutorial de baja segura: busca la tecnología, lee su uso y explica por qué
 * el catálogo oficial se retira desactivando. No abre diálogos ni cambia el
 * estado de ninguna tecnología.
 */
export function buildTechCatalogTutorial(
  context: TechCatalogTutorialContext
): TutorialDefinition {
  return {
    id: 'tech-catalog-safe-removal',
    title: 'Retirar una tecnología en uso',
    manualFragment: MANUAL_FRAGMENT,
    steps: [
      {
        id: 'search',
        title: 'Encuentre la tecnología',
        body:
          'Busque por nombre o slug. La tarjeta indica cuántos entornos la referencian antes de que se intente cualquier baja.',
        selector: '[data-tour="tech-filters"]',
        side: 'bottom',
        check: {
          isComplete: () => context.hasSearchQuery(),
          pendingHint: 'Escriba el nombre o el slug de una tecnología para continuar.'
        }
      },
      {
        id: 'usage',
        title: 'Lea el uso antes de decidir',
        body:
          'La etiqueta «En uso: N» dice cuántas plantillas o modelos referencian la tecnología. Ese número decide la baja: si es mayor que cero, retirarla del constructor es desactivarla.',
        selector: '[data-tour="tech-usage"]',
        side: 'bottom'
      },
      {
        id: 'origin',
        title: 'Qué permite el catálogo oficial',
        body:
          'Las tecnologías del catálogo oficial no muestran el icono de eliminar: son la base del sistema y se conservan. Solo una tecnología creada por el administrador puede eliminarse, y únicamente cuando ninguna plantilla la usa.',
        selector: '[data-tour="tech-card-actions"]',
        side: 'top'
      },
      {
        id: 'exit',
        title: 'La salida segura: desactivar',
        body:
          'El interruptor retira la tecnología del constructor sin alterar las plantillas que ya la usan. Esta guía no la desactiva: la decisión y el clic quedan en sus manos.',
        selector: '[data-tour="tech-toggle"]',
        side: 'left'
      }
    ]
  };
}
