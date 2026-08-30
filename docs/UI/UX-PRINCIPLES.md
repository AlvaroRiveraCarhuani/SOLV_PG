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
  - **Inter:** Texto de interfaz de usuario, títulos, botones y bloques descriptivos.
  - **JetBrains Mono:** Reservada estrictamente para datos de máquina (URLs opacas, identificadores UUID, métricas en ms/MB y veredictos).
- **Esquema de Color:** Tema claro por defecto (`#F6F7F9` fondo, `#FFFFFF` superficies). Bordes sólidos de `1px` (`#E2E5E9`) en lugar de sombras decorativas pesadas (`box-shadow`).
