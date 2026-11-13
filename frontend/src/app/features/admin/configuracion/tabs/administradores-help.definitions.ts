import { TourDefinition, TutorialDefinition } from '@shared/help/help.types';

/**
 * Tutorial de la pestaña Administradores.
 *
 * El alta es la única acción que la guía acompaña de punta a punta: deja al
 * operador registrado con Google SSO. Revocar solo se explica, porque corta el
 * acceso de una persona y queda en auditoría.
 */

export interface AdministradoresTutorialContext {
  /** El modal de alta está abierto. */
  isCreateModalOpen: () => boolean;
  /** Correo institucional escrito en el formulario. */
  email: () => string;
  /** Error de validación del formulario, si existe. */
  formError: () => string | null;
  /** Se registró un administrador en esta sesión. */
  hasCreated: () => boolean;
}

export function buildAdministradoresTutorial(
  context: AdministradoresTutorialContext
): TutorialDefinition {
  return {
    id: 'config-admin-team',
    title: 'Sumar un administrador',
    manualFragment: null,
    steps: [
      {
        id: 'team',
        title: 'Quiénes tienen privilegios hoy',
        body:
          'La tabla muestra a todo el equipo con rol administrador, desde cuándo tiene el acceso y cuándo entró por última vez. La fila marcada como «Tú» es la cuenta con la que se está operando.',
        selector: '[data-tour="admins-list"]',
        side: 'top'
      },
      {
        id: 'create',
        title: 'Registrar un operador',
        body:
          'El alta pre-aprovisiona la cuenta: la persona entra con Google SSO usando su correo institucional, sin enlace de confirmación. Revisar dos veces el correo evita dar acceso a la persona equivocada.',
        selector: '[data-tour="admins-create"]',
        side: 'bottom',
        check: {
          isComplete: () => context.isCreateModalOpen(),
          pendingHint: 'Abra «Agregar Administrador» para continuar.'
        }
      },
      {
        id: 'email',
        title: 'Correo institucional del operador',
        body:
          'El correo debe coincidir con la cuenta de Google Workspace de la institución. Nombre y apellido son obligatorios y se muestran en la tabla para identificar a la persona.',
        selector: '#admin-email',
        side: 'left',
        check: {
          isComplete: () =>
            context.email().includes('@') &&
            context.email().includes('.') &&
            context.formError() === null,
          pendingHint: 'Escriba un correo institucional válido para continuar.'
        }
      },
      {
        id: 'submit',
        title: 'Confirmar el alta',
        body:
          'Al confirmar, la cuenta queda habilitada de inmediato y el alta se registra en la auditoría. El modal se cierra y la persona aparece en la tabla como activa.',
        selector: '[data-tour="admins-submit"]',
        side: 'top',
        check: {
          isComplete: () => context.hasCreated(),
          pendingHint: 'Confirme el alta para continuar.'
        }
      },
      {
        id: 'revoke',
        title: 'Revocar privilegios',
        body:
          'Revocar corta el acceso administrativo de esa persona y pide confirmación. No se puede revocar la propia cuenta ni dejar la institución sin administradores. Aquí termina la guía: la acción queda en sus manos.',
        selector: '[data-tour="admins-row-actions"]',
        side: 'left'
      }
    ]
  };
}

/**
 * Recorrido de la pestaña Administradores.
 *
 * Presenta la lista, el alta y la revocación como un mismo control de acceso.
 * El tutorial de la pestaña acompaña el alta completa con validación del correo.
 */
export function buildAdministradoresTour(): TourDefinition {
  return {
    id: 'config-admins-tour-v1',
    title: 'Recorrido por Administradores',
    steps: [
      {
        selector: '[data-tour="config-tab-administradores"]',
        title: 'Esta guía cubre una sola pestaña',
        description:
          'El recorrido se limita a Administradores. Es la pestaña de mayor privilegio de Configuración y cada cambio queda registrado en auditoría.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="config-description"]',
        title: 'Quiénes pueden administrar',
        description:
          'Un administrador tiene acceso irrestricto a infraestructura, plantillas, períodos y auditoría. No existe un nivel intermedio de permisos.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="admins-list"]',
        title: 'Quiénes tienen privilegios hoy',
        description:
          'La tabla muestra correo, estado y última actividad. Las cuentas inactivas conservan su registro, pero no pueden entrar al panel.',
        side: 'top'
      },
      {
        selector: '[data-tour="admins-create"]',
        title: 'El alta da privilegios completos',
        description:
          'El correo institucional se registra tal cual, sin enlace de confirmación: un correo equivocado habilita a la persona equivocada.',
        side: 'bottom'
      },
      {
        selector: '[data-tour="admins-row-actions"]',
        title: 'Revocar es la única salida',
        description:
          'Revocar corta el acceso administrativo y pide confirmación. No se puede revocar la propia cuenta ni dejar la institución sin administradores.',
        side: 'left'
      }
    ]
  };
}
