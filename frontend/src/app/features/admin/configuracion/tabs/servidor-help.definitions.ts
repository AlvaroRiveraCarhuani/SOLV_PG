import { TourDefinition, TutorialDefinition } from '@shared/help/help.types';

/**
 * Tutorial de la pestaña Servidor y Respaldos.
 *
 * Solo comprueba acciones que el administrador ya completó: guardar políticas,
 * abrir la ventana de mantenimiento, crear un respaldo y guardar la estrategia.
 * La guía no activa el mantenimiento ni toca la rotación por su cuenta.
 */

export interface ServidorTutorialContext {
  /** Las políticas de autogestión se guardaron en esta sesión. */
  hasSavedPolicies: () => boolean;
  /** El modal de mantenimiento está abierto. */
  maintenanceModalOpen: () => boolean;
  /** Se creó un respaldo manual en esta sesión. */
  hasTriggeredBackup: () => boolean;
  /** La estrategia de respaldos se guardó en esta sesión. */
  hasSavedBackupStrategy: () => boolean;
}

export function buildServidorTutorial(context: ServidorTutorialContext): TutorialDefinition {
  return {
    id: 'config-server-operation',
    title: 'Políticas, mantenimiento y respaldos',
    manualFragment: null,
    steps: [
      {
        id: 'policies',
        title: 'Políticas de autogestión',
        body:
          'El techo de seguridad de la memoria del host y los minutos de inactividad deciden cuándo se suspende un laboratorio. La memoria por alumno no se define aquí: la fija la plantilla de cada entorno.',
        selector: '[data-tour="server-policies"]',
        side: 'top',
        check: {
          isComplete: () => context.hasSavedPolicies(),
          pendingHint: 'Ajuste un valor y guarde las políticas para continuar.'
        }
      },
      {
        id: 'maintenance',
        title: 'Modo mantenimiento global',
        body:
          'Bloquea a estudiantes y docentes con una pantalla de mantenimiento mientras los administradores conservan el acceso. Para activarlo hay que escribir la palabra MANTENIMIENTO y un motivo de al menos 10 caracteres.',
        selector: '[data-tour="server-maintenance"]',
        side: 'top',
        check: {
          isComplete: () => context.maintenanceModalOpen(),
          pendingHint: 'Abra la ventana de mantenimiento para leer la confirmación sin activarla.'
        }
      },
      {
        id: 'backups',
        title: 'Crear un respaldo ahora',
        body:
          '«Crear Respaldo Ahora» lanza una copia completa y la agrega al historial. Queda pendiente de verificación de integridad: usar «Verificar» confirma que el archivo es válido.',
        selector: '[data-tour="server-backups"]',
        side: 'top',
        check: {
          isComplete: () => context.hasTriggeredBackup(),
          pendingHint: 'Cree un respaldo para continuar.'
        }
      },
      {
        id: 'strategy',
        title: 'Frecuencia y retención',
        body:
          'La frecuencia define cada cuántas horas se respalda solo; la retención, cuántos días se conservan antes de borrarse. Si la retención baja, el aviso indica cuántos respaldos expirarían en la próxima rotación.',
        selector: '[data-tour="server-backup-strategy"]',
        side: 'top',
        check: {
          isComplete: () => context.hasSavedBackupStrategy(),
          pendingHint: 'Ajuste la estrategia y guárdela para terminar la guía.'
        }
      }
    ]
  };
}

/**
 * Recorrido de la pestaña Servidor y Respaldos.
 *
 * Ordena los bloques por riesgo: lo que protege al host, lo que afecta a toda
 * la institución y lo que conserva copias. El tutorial de la pestaña recorre
 * las mismas zonas con comprobaciones reales.
 */
export function buildServidorTour(): TourDefinition {
  return {
    id: 'config-server-tour-v1',
    title: 'Recorrido por Servidor y Respaldos',
    steps: [
      {
        selector: '[data-tour="config-tab-servidor"]',
        title: 'Esta guía cubre una sola pestaña',
        description:
          'El recorrido se limita a Servidor y Respaldos. La memoria de cada laboratorio se define en la plantilla de su entorno, no aquí.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="config-description"]',
        title: 'Operación del nodo, no de cada laboratorio',
        description:
          'Esta pestaña gobierna cómo se comporta el servidor bajo carga y cómo se conserva su estado; el dimensionamiento fino pertenece a las plantillas.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="server-policies"]',
        title: 'Políticas que protegen el host',
        description:
          'El techo de memoria y los minutos de inactividad deciden cuándo se suspende un laboratorio. Se aplican a todos los entornos a la vez.',
        side: 'right'
      },
      {
        selector: '[data-tour="server-maintenance"]',
        title: 'La zona de riesgo',
        description:
          'El mantenimiento bloquea a docentes y estudiantes con una pantalla propia, mientras los administradores conservan el acceso. Exige la palabra MANTENIMIENTO y un motivo.',
        side: 'right'
      },
      {
        selector: '[data-tour="server-backups"]',
        title: 'Respaldos, frecuencia y retención',
        description:
          '«Crear Respaldo Ahora» lanza una copia y la verificación de integridad es un paso aparte. La frecuencia y la retención del bloque inferior definen la rotación automática.',
        side: 'top'
      }
    ]
  };
}
