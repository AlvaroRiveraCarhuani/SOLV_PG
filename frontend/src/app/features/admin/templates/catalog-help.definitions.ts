import { TourDefinition, TutorialDefinition } from '@shared/help/help.types';

/**
 * Tutorial del catálogo de plantillas: recorre la bandeja de revisión y valida
 * que el administrador esté realmente en la pestaña que toca.
 *
 * Las plantillas se revisan en un modal que queda por encima del panel, así que
 * la guía se queda en la página y explica la ficha sin exigir abrirla.
 *
 * Sin aprobaciones pendientes la bandeja no existe, así que el tutorial cambia
 * de guion: explica dónde se revisa y qué se puede hacer en el catálogo activo,
 * en vez de pedir acciones que esa pestaña no ofrece.
 */

export interface CatalogTutorialContext {
  /** Pestaña visible del catálogo. */
  activeTab: () => 'pending' | 'catalog' | 'rejected';
  /** Plantillas que esperan aprobación; la pestaña se oculta cuando son 0. */
  pendingCount: () => number;
}

const MANUAL_FRAGMENT = '3-2-ciclo-de-vida-y-gobernanza-de-plantillas';

/**
 * Recorrido del catálogo: resumen, acciones de encabezado, pestañas, filtros y
 * fichas. El tutorial de revisión es otra guía de esta misma pantalla.
 */
export function buildCatalogTour(): TourDefinition {
  return {
    id: 'templates-catalog-tour-v1',
    title: 'Recorrido por el Catálogo de Entornos',
    steps: [
      {
        selector: '[data-tour="templates-summary"]',
        title: 'Qué cuenta el resumen',
        description:
          'Entornos registrados, activos, pausados y la memoria promedio por alumno. Un entorno pausado deja de ofrecerse sin borrar el historial de las materias que lo usaron.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="templates-actions"]',
        title: 'Preparar y crear',
        description:
          '«Preparar entornos» verifica e instala en el servidor las imágenes de los entornos activos para que la clase no espere descargas. «Nuevo Entorno» abre el constructor.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="templates-tabs"]',
        title: 'Pestañas de estado y vista',
        description:
          'Cada pestaña cuenta lo que contiene: aprobación pendiente, catálogo activo, pausados y rechazados. El alternador cambia entre tarjetas y tabla sin perder los filtros.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="templates-filters"]',
        title: 'Buscar por propósito y tecnología',
        description:
          'La búsqueda cubre nombre, tecnología, docente y motivo. Los filtros de propósito y de tecnología acotan la lista; los contadores de cada chip dicen cuánto seleccionan.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="templates-list"]',
        title: 'Qué muestra cada ficha',
        description:
          'Cada ficha trae propósito, tecnologías, memoria por alumno, imagen base y quién la creó. Las acciones dependen de la pestaña: en aprobación, «Revisar y Aprobar» y «Rechazar» con motivo; en el catálogo y en rechazadas, activar o pausar y editar.',
        side: 'top'
      }
    ]
  };
}

export function buildCatalogTutorial(context: CatalogTutorialContext): TutorialDefinition {
  if (context.pendingCount() === 0) {
    return {
      id: 'catalog-work-without-pending',
      title: 'Trabajar el catálogo sin pendientes',
      manualFragment: MANUAL_FRAGMENT,
      steps: [
        {
          id: 'purpose',
          title: 'Qué hay para revisar hoy',
          body:
            'La pestaña «En aprobación» existe sólo mientras una copia o una plantilla heredada espera auditoría. Con la bandeja vacía, el catálogo activo es la vista de trabajo y no hay nada que aprobar ni rechazar.',
          selector: '[data-tour="templates-tabs"]',
          side: 'bottom'
        },
        {
          id: 'card',
          title: 'Qué se hace en cada ficha',
          body:
            'La ficha muestra propósito, tecnologías, memoria por alumno e imagen base. Desde su menú se activa o pausa el entorno, se ajusta la cuota y la descripción, y se duplica una plantilla para partir de ella. Pausar pide un motivo que queda registrado.',
          selector: '[data-tour="templates-list"]',
          side: 'top'
        },
        {
          id: 'catalog',
          title: 'Cuando vuelva a haber pendientes',
          body:
            'Si entra una plantilla a auditoría, la pestaña reaparecerá con su contador y el tutorial de revisión volverá a guiarle por «Revisar y Aprobar» y «Rechazar». Mientras tanto, este catálogo es lo que ven los docentes al preparar una materia.',
          selector: '[data-tour="templates-tabs"]',
          side: 'bottom'
        }
      ]
    };
  }

  return {
    id: 'catalog-review-first-template',
    title: 'Revisar y aprobar una plantilla',
    manualFragment: MANUAL_FRAGMENT,
    steps: [
      {
        id: 'purpose',
        title: 'Cómo funciona esta bandeja',
        body:
          'Al publicar desde el constructor, la plantilla entra activa de inmediato. La pestaña «En aprobación» aparece sólo cuando queda algo pendiente de auditoría, por ejemplo una copia duplicada o una plantilla heredada.',
        selector: '[data-tour="templates-tabs"]',
        side: 'bottom'
      },
      {
        id: 'pending',
        title: 'Plantillas en aprobación',
        body:
          'La pestaña sólo se muestra cuando hay algo que aprobar. El contador indica cuántas esperan tu decisión; si está en cero, no hay nada pendiente hoy.',
        selector: '[data-tour="templates-tabs"]',
        side: 'bottom',
        check: {
          // Si se aprueba lo último pendiente durante la guía, la pestaña
          // desaparece: el paso se da por cumplido en vez de bloquear el avance.
          isComplete: () => context.activeTab() === 'pending' || context.pendingCount() === 0,
          pendingHint: 'Abra la pestaña «En aprobación» para continuar.'
        }
      },
      {
        id: 'card',
        title: 'Qué mirar en cada tarjeta',
        body:
          'Quién la creó, tecnologías pedidas, memoria por alumno e imagen. La justificación pedagógica explica para qué se usará en clase y conviene leerla antes de decidir.',
        selector: '[data-tour="templates-list"]',
        side: 'top'
      },
      {
        id: 'review',
        title: 'Revisar y fijar cuotas',
        body:
          '«Revisar y Aprobar» abre la ficha, donde se confirma la tecnología y se ajusta la memoria por alumno antes de aprobar. «Rechazar» pide un motivo de al menos 10 caracteres, que el docente verá para corregir.',
        selector: '[data-tour="templates-list"]',
        side: 'top'
      },
      {
        id: 'catalog',
        title: 'Ver la plantilla publicada',
        body:
          'Al aprobar, la plantilla pasa a «Catálogo Activo» y los docentes pueden asignarla a sus materias. Si algo falla después, puede suspenderse sin borrar el historial de los cursos anteriores.',
        selector: '[data-tour="templates-tabs"]',
        side: 'bottom',
        check: {
          isComplete: () => context.activeTab() === 'catalog',
          pendingHint: 'Abra la pestaña «Catálogo Activo» para terminar.'
        }
      }
    ]
  };
}
