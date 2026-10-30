# Principios de Experiencia de Usuario (UX) — SOLV

> **Documento Oficial de Leyes de Interfaz, Psicología UX y Fundamentación de Diseño**  
> **Proyecto:** SOLV (Sistema de Orquestación de Laboratorios Virtuales)  
> **Gobernanza:** SDD / Docs-as-Code / SOLV Design System  

---

## 1. Las 5 Leyes de Interfaz en SOLV

### 1. Dualidad de Estados (Académico vs. Técnico)
Coexistencia armónica del estado académico (plazo de entrega, estado de evaluación, calificación) y el estado técnico del contenedor (`Running`, `Hibernated`, `OOM_Killed`) en la misma unidad visual. El estudiante comprende inmediatamente tanto su situación académica como la disponibilidad del entorno técnico.

### 2. Carga Cognitiva Extrínseca Cero
Traducción automática de errores complejos de infraestructura (fallos en Docker, suspensión por límites de memoria RAM OOM, desconexión de WebSocket) a lenguaje humano directo y accionable, acompañado de un botón de resolución inmediata (ej. `[ Reintentar ]` o `[ Ver Diagnóstico ]`).

### 3. Revelación Progresiva (Disclosure Progresivo)
Las métricas avanzadas de consumo de recursos hardware (porcentajes de CPU/RAM detallados) y rastreos de pila de Docker permanecen ocultos por defecto en las vistas principales. Solo se despliegan en paneles secundarios o diálogos modales cuando el usuario requiere depurar un incidente técnico.

### 4. Inmutabilidad en Modo Revisión
El modo de revisión docente congela la posibilidad de edición a nivel de infraestructura en el backend y proxy Ingress (montaje de solo lectura `:ro`), garantizando que la inmutabilidad no dependa únicamente de restricciones visuales en el cliente web (CSS o propiedades de solo lectura en Angular).

### 5. Comunicación de Frescura
Indicadores transparentes de estado transitorio ("Actualizado hace 5s", "Reconectando WebSocket...", "Pausando contenedor por inactividad...") que mantienen al usuario informado durante los cambios de estado en segundo plano sin interrumpir su flujo de trabajo.

---

## 2. Definición del Estándar Visual y Sistema de Iconos

- **Iconografía:** Catálogo estandarizado de **Lucide Icons** (`lucide:home`, `lucide:terminal`, `lucide:book-open`, `lucide:award`, `lucide:history`, `lucide:settings`, `lucide:calendar`, `lucide:clock`).
- **Tipografía Dual:**
  - **`--font-sans`:** Texto de interfaz de usuario, títulos, botones y bloques descriptivos. La familia se resuelve por token y por tenant; nunca se escribe un nombre de familia en el SCSS de una vista.
  - **`--font-mono`:** Reservada estrictamente para datos de máquina (URLs opacas, identificadores UUID, métricas en ms/MB, códigos de veredicto cuando se piden explícitamente).
- **Esquema de Color:** Tema claro por defecto (`#F6F7F9` fondo, `#FFFFFF` superficies). Bordes sólidos de `1px` (`#E2E5E9`) en lugar de sombras decorativas pesadas (`box-shadow`).

---

## 3. Lenguaje de Resultados: dos Vocabularios, una Fuente

El código del juez (`AC`, `WA`, `TLE`, `MLE`, `RE`, `CE`, `VE`, `AST_BLOCKED`) es un valor técnico interno. Se conserva para comparadores, auditoría y métricas, pero **no se muestra a personas** ni se escribe a mano en templates, filtros u opciones de override.

Lo que ve el docente y el estudiante es el **resultado pedagógico**: una frase que dice qué pasó y, cuando corresponde, qué hacer. La fuente única es `OUTCOME_COPY` en `frontend/src/app/core/models/verdict.model.ts`; de esa tabla derivan la etiqueta del badge, su detalle y las opciones de filtro y de override.

Reglas:

1. Agregar o cambiar un resultado se hace en `OUTCOME_COPY`, nunca en un componente.
2. La etiqueta es de una línea y nombra la consecuencia, no el mecanismo: "No compila", "Se queda sin memoria", "Tarda demasiado".
3. El detalle explica por qué ocurre sin jerga de juez y sin stacktrace crudo.
4. La sigla solo aparece si el usuario la pide de forma explícita (`showAcronym`); en el uso normal, nunca.
5. Un valor desconocido cae en `Sin evaluar`, no en un espacio vacío ni en el código crudo.

El color sigue la semántica del sistema: `Cumple` verde, `No cumple` rojo, `Tarda demasiado` y `Se queda sin memoria` ámbar, `Se cae al ejecutar` naranja, `Usó algo no permitido` rojo intenso, `Sin calificar` neutral.
