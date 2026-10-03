# Proposal: Personalización Libre de Dashboards (Bento Grid) y Tematización Híbrida de Cursos

## Contexto y Motivación
Los usuarios de SOLV (Docentes, Administradores y Estudiantes) necesitan flexibilidad para organizar sus paneles de control según su flujo de trabajo particular (estilo Moodle/Canvas):
1. **Bento Grid Libre**: Posibilidad de arrastrar widgets, alternar su tamaño entre 33%, 50%, 66% y 100% de ancho de pantalla, y ocultar o mostrar bloques según conveniencia.
2. **Tematización Híbrida de Cursos**: Identificación visual inmediata de materias mediante una paleta curada accesible (8 temas predeterminados) con opción de personalización libre de color.
3. **Preferencia Personal de Tipografía**: Capacidad del usuario para elegir su fuente UI y fuente mono favorita de nuestro catálogo curado para mayor ergonomía de lectura y accesibilidad.

## Alcance
- Crear el modelo y servicio de gestión de layout de widgets (`dashboard-layout.service.ts` o utilitario en `@core/services/`).
- Implementar Bento Grid de 12 columnas con `@angular/cdk/drag-drop` y selector de ancho en el dashboard docente (`TeacherDashboardComponent`) y del estudiante (`StudentDashboardComponent`).
- Implementar selector híbrido de color de curso (presets accesibles + color libre) en modales de creación/edición de cursos.
- Implementar selector de tipografía accesible en preferencias de usuario/topbar con aplicación dinámica a `--font-sans` y `--font-mono`.
- Garantizar responsividad (colapso automático a 100% en móviles), pruebas unitarias y cumplimiento del sistema de diseño.
