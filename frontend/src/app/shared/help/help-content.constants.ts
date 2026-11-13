import { ContextHelpEntry } from './help.types';

/**
 * Ayuda contextual prioritaria (auditoría UX del rol administrador).
 *
 * Sólo explica lo que la pantalla no dice por sí sola; los textos describen el
 * comportamiento real de cada vista.
 */
export const CONTEXT_HELP_ENTRIES: Record<string, ContextHelpEntry> = {
  'dashboard-health': {
    key: 'dashboard-health',
    title: 'Cómo leer los indicadores de salud',
    intro:
      'El indicador superior resume el estado del servidor y del motor Docker. Los motivos se listan debajo del indicador y también aparecen al pasar el cursor.',
    items: [
      {
        label: 'Operativo y Estable',
        text: 'No hay motivos activos: los contenedores responden y la memoria del host está bajo el umbral.'
      },
      {
        label: 'Rendimiento Degradado',
        text: 'Hay motivos que no impiden operar, como contenedores huérfanos, laboratorios detenidos por memoria o estados sin verificar.'
      },
      {
        label: 'Servicio Crítico',
        text: 'El motor Docker no responde o la memoria del host supera el umbral de seguridad. Conviene revisar las incidencias antes de abrir una clase.'
      },
      {
        label: 'Hibernar Todo',
        text: 'Suspende los laboratorios activos y pide confirmación con el alcance real. Las sesiones abiertas se cierran y los entornos quedan en reposo.'
      }
    ],
    manualFragment: '5-monitoreo-salud-y-auditoria'
  },

  'students-penalties': {
    key: 'students-penalties',
    title: 'Faltas por memoria, bloqueos y período',
    intro:
      'Cada vez que un entorno se detiene por exceder su cuota de memoria se registra una falta para el estudiante.',
    items: [
      {
        label: '1 y 2 faltas',
        text: 'El estudiante aparece «En riesgo» y sigue trabajando; la tercera falta activa el bloqueo.'
      },
      {
        label: '3 faltas o más',
        text: 'Queda bloqueado por memoria y no puede abrir laboratorios hasta que un administrador lo desbloquee.'
      },
      {
        label: 'Desbloqueo',
        text: '«Desbloquear memoria» reinicia el contador del estudiante y la acción queda registrada en auditoría.'
      },
      {
        label: 'Filtro de período',
        text: '«Cursando en período» muestra sólo las matrículas del período vigente. Para ver el historial completo, elija «Todos los períodos».'
      }
    ],
    manualFragment: '4-3-directorio-de-estudiantes-y-desbloqueo-de-memoria'
  },

  'teachers-invitations': {
    key: 'teachers-invitations',
    title: 'Invitaciones docentes y su vigencia',
    intro:
      'Registrar un docente envía una invitación por correo para que cree su cuenta institucional.',
    items: [
      {
        label: 'Vigencia',
        text: 'Cada invitación vence a las 72 horas. Pasado ese plazo el enlace ya no permite completar el registro.'
      },
      {
        label: 'Reenviar',
        text: 'Genera una invitación nueva desde ese momento, con otras 72 horas de vigencia para el mismo correo.'
      },
      {
        label: 'Revocar',
        text: 'Anula el enlace vigente; el docente no podrá crear la cuenta con esa invitación.'
      },
      {
        label: 'Pendientes (72h)',
        text: 'Cuenta sólo invitaciones vigentes que todavía no fueron aceptadas.'
      }
    ],
    manualFragment: '4-4-invitaciones-a-docentes'
  },

  'emergency-console': {
    key: 'emergency-console',
    title: 'Alcance de las operaciones de emergencia',
    intro:
      'Cada acción exige la frase de confirmación y un motivo de al menos 10 caracteres. La ejecución y el motivo quedan en el registro de auditoría.',
    items: [
      {
        label: 'Terminar todos los entornos',
        text: 'Destructiva: detiene y destruye los contenedores activos del tenant. El trabajo no guardado se pierde.'
      },
      {
        label: 'Hibernar todos los entornos',
        text: 'Operativa: pausa los contenedores activos preservando su estado; se pueden reanudar después.'
      },
      {
        label: 'Limpiar contenedores huérfanos',
        text: 'Operativa: elimina contenedores que ya no pertenecen a ningún entorno registrado.'
      },
      {
        label: 'Purgar capas huérfanas',
        text: 'Operativa: libera disco eliminando capas sin uso; no toca volúmenes nombrados.'
      },
      {
        label: 'Reiniciar pools',
        text: 'Operativa: reinicia el pool de conexiones de PostgreSQL y los contadores de corte. Útil cuando hay bloqueos.'
      },
      {
        label: 'Aviso de memoria',
        text: 'Si la RAM del host supera el 90 %, el panel lo advierte: hibernar antes de una acción destructiva reduce el riesgo.'
      }
    ],
    manualFragment: '10-mantenimiento-y-consola-de-emergencias'
  },

  'ai-config': {
    key: 'ai-config',
    title: 'Estado de los proveedores de IA',
    intro:
      'La grilla muestra el estado de cada proveedor y el interruptor superior habilita o deshabilita la IA en toda la institución.',
    items: [
      {
        label: 'Habilitado / Deshabilitado',
        text: 'El interruptor decide si el copiloto del constructor y las tareas asistidas están disponibles.'
      },
      {
        label: 'En uso',
        text: 'Es el proveedor que recibe las solicitudes. Sólo uno puede estar en uso a la vez.'
      },
      {
        label: 'Clave guardada',
        text: 'Hay credencial configurada pero el proveedor no está activo. Al seleccionarlo pasa a estar en uso.'
      },
      {
        label: 'Sin configurar',
        text: 'No tiene clave ni modelo. Guarde una clave para poder seleccionarlo.'
      },
      {
        label: 'Clave',
        text: 'Se almacena en el servidor y no se muestra completa. Para cambiarla, escriba una nueva y guarde.'
      }
    ],
    manualFragment: '9-configuracion-de-inteligencia-artificial'
  },

  'studio-template-builder': {
    key: 'studio-template-builder',
    title: 'Cómo se construye una plantilla oficial',
    intro:
      'Una plantilla oficial define el entorno que el docente elige al preparar un ejercicio: modalidad, tecnologías y memoria. Se registra una vez y se reutiliza en todas las materias.',
    items: [
      {
        label: 'Editor en el navegador',
        text: 'La modalidad persistente entrega al estudiante un editor completo con terminal, carpetas de trabajo y base de datos lista para conectar.'
      },
      {
        label: 'Corrección automática',
        text: 'La modalidad de evaluación ejecuta el código contra los casos de prueba en pocos segundos, sin interfaz gráfica.'
      },
      {
        label: 'Tecnologías del catálogo',
        text: 'Sólo se ofrecen lenguajes, interfaces web, complementos y bases de datos homologados. El Modo Avanzado muestra el Dockerfile y los scripts que se generarán.'
      },
      {
        label: 'Memoria por alumno',
        text: 'Determina cuántos estudiantes caben a la vez. El panel indica el mínimo que exigen las tecnologías y el máximo que permite el servidor.'
      },
      {
        label: 'Revisión y publicación',
        text: 'El resumen confirma nombre, imagen, tecnologías y memoria. Comprobar funcionamiento valida que los programas respondan; publicar deja la plantilla disponible para los docentes.'
      }
    ],
    manualFragment: '7-constructor-de-plantillas-de-entornos'
  },

  'templates-catalog': {
    key: 'templates-catalog',
    title: 'Qué muestra el catálogo de plantillas',
    intro:
      'Las plantillas oficiales son los entornos que el docente puede elegir al preparar un ejercicio. Esta página agrupa las publicadas y las rechazadas, y añade la pestaña de aprobación sólo cuando queda algo pendiente.',
    items: [
      {
        label: 'En aprobación',
        text: 'Aparece sólo si hay copias o plantillas heredadas esperando auditoría; ahí se aprueban o se rechazan con un motivo.'
      },
      {
        label: 'Catálogo',
        text: 'Plantillas publicadas y disponibles para asignar a una materia.'
      },
      {
        label: 'Rechazadas',
        text: 'Plantillas descartadas con su motivo; conviene revisarlas antes de preparar una nueva.'
      },
      {
        label: 'Preparar un entorno',
        text: 'Abre el constructor, donde se define la modalidad, las tecnologías y la memoria de la plantilla.'
      }
    ],
    manualFragment: '3-2-ciclo-de-vida-y-gobernanza-de-plantillas'
  },

  'config-overview': {
    key: 'config-overview',
    title: 'Qué se configura aquí',
    intro:
      'La configuración institucional reúne los períodos académicos, la identidad visual, el servidor y la inteligencia artificial. Cada pestaña cambia sólo lo que le corresponde.',
    items: [
      {
        label: 'Períodos académicos',
        text: 'Alta y cierre de semestres. El período vigente es el que se usa por defecto en las demás pantallas.'
      },
      {
        label: 'Identidad (White-Label)',
        text: 'Nombre, logo y colores de la institución, aplicados a toda la plataforma.'
      },
      {
        label: 'Servidor y respaldos',
        text: 'Parámetros del nodo, políticas de memoria y respaldos disponibles.'
      },
      {
        label: 'Inteligencia Artificial',
        text: 'Proveedor en uso, claves y modelos. La ayuda detallada aparece en esa pestaña.'
      },
      {
        label: 'Administradores',
        text: 'Personas con acceso completo al área administrativa.'
      }
    ],
    manualFragment: null
  },

  'config-periods': {
    key: 'config-periods',
    title: 'Períodos académicos y su ciclo',
    intro:
      'El período es el marco temporal de las materias. Solo uno puede estar vigente, y de él dependen las inscripciones y el filtro por período del resto de las pantallas.',
    items: [
      {
        label: 'Vigente',
        text: 'Es el único período que admite nuevas inscripciones. Al activar otro, el anterior deja de recibir matrículas.'
      },
      {
        label: 'Planificado',
        text: 'Tiene fechas futuras y todavía no gobierna nada. Se activa cuando corresponde desde su fila.'
      },
      {
        label: 'Archivado',
        text: 'Cierra el período de forma irreversible y deja sus materias en solo lectura. Para confirmarlo hay que escribir el código del período.'
      },
      {
        label: 'Materias asociadas',
        text: 'Un período con materias no se puede eliminar; primero hay que resolver esas materias o archivarlo.'
      }
    ],
    manualFragment: '4-1-periodos-academicos-semestres'
  },

  'config-branding': {
    key: 'config-branding',
    title: 'Qué alcanza la identidad white-label',
    intro:
      'La marca institucional se aplica a toda la plataforma. El panel derecho muestra el resultado antes de guardar y verifica el contraste del texto sobre el color primario.',
    items: [
      {
        label: 'Nombre y logo',
        text: 'Aparecen en la barra superior, en el inicio de sesión y en los correos. El logo admite archivo de hasta 2 MB o una URL https.'
      },
      {
        label: 'Color primario',
        text: 'En formato #RRGGBB. De él se derivan hover, foco y badges; si el contraste no cumple WCAG AA, el guardado no se habilita.'
      },
      {
        label: 'Tipografía',
        text: 'La fuente de interfaz afecta textos y controles; la monoespaciada se reserva para código, hashes y números. La opción avanzada acepta una hoja de Google Fonts propia.'
      },
      {
        label: 'Propagación',
        text: 'Los cambios tardan hasta 5 minutos en llegar a todos los usuarios por la caché institucional. «Cancelar Cambios» vuelve a lo guardado.'
      }
    ],
    manualFragment: null
  },

  'config-server': {
    key: 'config-server',
    title: 'Qué se cambia en el servidor',
    intro:
      'Esta pestaña gobierna la operación del nodo, no la capacidad de cada laboratorio: la memoria por alumno la define la plantilla de cada entorno.',
    items: [
      {
        label: 'Políticas de autogestión',
        text: 'El techo de memoria del host y los minutos de inactividad deciden cuándo se suspende un laboratorio para proteger el servidor.'
      },
      {
        label: 'Modo mantenimiento',
        text: 'Bloquea a estudiantes y docentes con una pantalla de mantenimiento; los administradores conservan el acceso. Exige la palabra MANTENIMIENTO y un motivo.'
      },
      {
        label: 'Respaldos',
        text: '«Crear Respaldo Ahora» lanza una copia completa. La verificación de integridad es un paso aparte y confirma que el archivo es válido.'
      },
      {
        label: 'Frecuencia y retención',
        text: 'La frecuencia define cada cuántas horas se respalda solo; la retención, cuántos días se conservan. Bajar la retención expira los respaldos más viejos en la próxima rotación.'
      }
    ],
    manualFragment: null
  },

  'config-administrators': {
    key: 'config-administrators',
    title: 'Alcance del rol administrador',
    intro:
      'Un administrador tiene acceso irrestricto a infraestructura, plantillas, períodos y auditoría. Cada alta y cada revocación quedan registradas.',
    items: [
      {
        label: 'Pre-aprovisionamiento',
        text: 'Al registrar el correo institucional, la persona entra con Google SSO de inmediato, sin enlace de confirmación.'
      },
      {
        label: 'Revisar el correo',
        text: 'El alta da privilegios completos: un correo equivocado habilita a la persona equivocada. Conviene confirmarlo antes de guardar.'
      },
      {
        label: 'Revocar',
        text: 'Corta el acceso administrativo y pide confirmación. No se puede revocar la propia cuenta ni dejar la institución sin administradores.'
      },
      {
        label: 'Auditoría',
        text: 'Altas y revocaciones quedan en el registro con su autor y fecha, junto al resto de las acciones sensibles.'
      }
    ],
    manualFragment: null
  },

  'auditoria-overview': {
    key: 'auditoria-overview',
    title: 'Qué registra la auditoría',
    intro:
      'Cada acción administrativa queda guardada con su autor, la fecha y el motivo. La pestaña Emergencias concentra las operaciones del servidor.',
    items: [
      {
        label: 'Búsqueda y filtros',
        text: 'Filtre por acción, severidad o texto para encontrar un movimiento concreto.'
      },
      {
        label: 'Acciones con motivo',
        text: 'Las operaciones sensibles exigen un motivo escrito, que aparece junto al registro.'
      },
      {
        label: 'Emergencias',
        text: 'Pestaña aparte con las cinco acciones del servidor, con doble confirmación.'
      }
    ],
    manualFragment: '5-2-registro-de-auditoria'
  },

  'courses-directory': {
    key: 'courses-directory',
    title: 'Cómo se organizan cursos y períodos',
    intro:
      'Cada materia pertenece a un período académico y tiene un docente titular. Desde aquí se revisan las materias, se asignan docentes y se controla el estado del curso.',
    items: [
      {
        label: 'Períodos académicos',
        text: 'Agrupan las materias de un semestre. Sólo el período vigente admite nuevas inscripciones.'
      },
      {
        label: 'Cursos activos',
        text: 'Tienen docente titular y estudiantes matriculados; los laboratorios se abren desde sus materias.'
      },
      {
        label: 'Cursos sin docente',
        text: 'Todavía no tienen titular asignado. Conviene asignarlo antes de abrir el curso a los estudiantes.'
      },
      {
        label: 'Reasignar docente',
        text: 'Cambia el titular de la materia sin perder el contenido ni los laboratorios de los estudiantes.'
      },
      {
        label: 'Archivados',
        text: 'Materias de períodos cerrados: se consultan, pero no abren nuevas sesiones.'
      }
    ],
    manualFragment: '4-2-cursos-y-asignacion-de-docentes'
  },

  'environment-requests': {
    key: 'environment-requests',
    title: 'Qué son las solicitudes de entornos',
    intro:
      'Un docente pide un entorno que no está en el catálogo. Cada ficha muestra las piezas reconocidas, las que faltan y el uso que tendrá en clase.',
    items: [
      {
        label: 'En revisión',
        text: 'Solicitudes que esperan respuesta, con el uso declarado y las tecnologías pedidas.'
      },
      {
        label: 'Respondidas',
        text: 'Solicitudes ya aprobadas o rechazadas, con la fecha de la respuesta.'
      },
      {
        label: 'Aprobar',
        text: 'Registra el entorno pedido para que el docente pueda usarlo; la respuesta queda guardada en la ficha.'
      },
      {
        label: 'Rechazar',
        text: 'Descarta la solicitud indicando el motivo, para que el docente sepa qué ajustar y vuelva a pedirla.'
      },
      {
        label: 'Piezas faltantes',
        text: 'Si una tecnología no figura en el catálogo, conviene registrarla en Modelos y Categorías antes de construir el entorno.'
      }
    ],
    manualFragment: '3-2-ciclo-de-vida-y-gobernanza-de-plantillas'
  },

  'tech-catalog': {
    key: 'tech-catalog',
    title: 'Catálogo de Tecnologías y Componentes',
    intro:
      'Administra los lenguajes, frameworks, librerías y bases de datos que componen los entornos de laboratorio en el constructor.',
    items: [
      {
        label: 'Lenguajes y Runtimes',
        text: 'Python, Node, Java, Go, C/C++ y Rust con sus versiones, imágenes base y evaluador listo para prácticas.'
      },
      {
        label: 'Complementos y Frameworks',
        text: 'Librerías de aplicación (FastAPI, Django, Angular, etc.) asociadas a su ecosistema de instalación (pip, npm, apt).'
      },
      {
        label: 'Bases de Datos',
        text: 'Motores integrados o dedicados (PostgreSQL, SQLite, Redis, MySQL) preconfigurados para las prácticas.'
      },
      {
        label: 'Mapa del Sistema',
        text: 'Vista gráfica que mapea las relaciones y compatibilidades activas entre runtimes y complementos.'
      }
    ],
    manualFragment: '8-biblioteca-de-modelos-y-categorias'
  },

  'academic-categories': {
    key: 'academic-categories',
    title: 'Categorías Académicas',
    intro:
      'Organiza las áreas de conocimiento y disciplinas que clasifican los cursos y los entornos de laboratorio.',
    items: [
      {
        label: 'Áreas de estudio',
        text: 'Ciencia de Datos, Desarrollo Web, Sistemas y Backend. Permiten filtrar y organizar la oferta del campus.'
      },
      {
        label: 'Orden institucional',
        text: 'Permite reordenar visualmente las categorías para presentarlas por orden de prioridad académica.'
      },
      {
        label: 'Protección de contenido',
        text: 'No se puede eliminar una categoría que tenga entornos, modelos o materias asociadas.'
      }
    ],
    manualFragment: '8-biblioteca-de-modelos-y-categorias'
  },

  'model-library': {
    key: 'model-library',
    title: 'Qué se administra en esta biblioteca',
    intro:
      'Reúne los modelos oficiales que el asistente ofrece a los docentes, el catálogo de tecnologías y las categorías académicas.',
    items: [
      {
        label: 'Modelos Oficiales',
        text: 'Nacen de una plantilla aprobada. Editar sus datos o desactivarlos no borra el historial de cursos anteriores.'
      },
      {
        label: 'Catálogo de Tecnologías',
        text: 'Lenguajes, interfaces web, complementos y bases de datos. Activar o desactivar decide si se ofrecen en el constructor.'
      },
      {
        label: 'Tecnologías base',
        text: 'No se eliminan: sostienen las plantillas y modelos que ya existen.'
      },
      {
        label: 'Categorías Académicas',
        text: 'Agrupan modelos y plantillas en el catálogo. No se puede eliminar una categoría que tenga contenido asociado.'
      },
      {
        label: 'Mapa del Sistema',
        text: 'Vista de sólo lectura con las relaciones entre tecnologías y complementos compatibles.'
      }
    ],
    manualFragment: '8-biblioteca-de-modelos-y-categorias'
  },

  'notifications': {
    key: 'notifications',
    title: 'Cómo funcionan los avisos',
    intro:
      'Los avisos reúnen lo que ocurre en la institución: solicitudes, incidencias del servidor y cambios relevantes. El número de la campana muestra los que todavía no leyó.',
    items: [
      {
        label: 'Sin leer',
        text: 'La pestaña cuenta los avisos pendientes; al abrir uno, deja de contarse.'
      },
      {
        label: 'Severidad',
        text: 'Cada aviso indica si es informativo, una advertencia o un error que conviene revisar. El filtro superior los separa.'
      },
      {
        label: 'Marcar todas como leídas',
        text: 'Deja el contador en cero sin borrar el historial ni los enlaces a cada pantalla.'
      },
      {
        label: 'Enlace al origen',
        text: 'Cuando el aviso lo incluye, abre directamente la pantalla relacionada con el hecho.'
      }
    ],
    manualFragment: null
  }
};

export function getContextHelpEntry(key: string): ContextHelpEntry | null {
  return CONTEXT_HELP_ENTRIES[key] ?? null;
}
