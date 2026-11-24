# Anexo D — Capturas del Manual de Usuario

## Alcance

- Fecha de captura: 2026-11-24.
- Servidor: Asus, con el stack `solv_frontend`, `solv_backend`, `solv_traefik`, `solv_db` y `solv_seaweedfs` en ejecución.
- Origen: `https://solv.dedyn.io`.
- Navegador: Chromium sin interfaz mediante Playwright.
- Viewport: `1440x900` con `deviceScaleFactor: 2`. Los PNG resultantes miden `2880x1800` px.
- Tema: claro, sin animaciones ni transiciones durante la captura.
- Sesiones: cuentas de demostración existentes en la base. Estudiante `andrea.gutierrez@uab.edu.bo`; docente `docente.dev@uab.edu.bo`; administrador `admin@uab.edu.bo`. No se crearon usuarios, materias ni entregas.

Los cinco slots cubren el recorrido del manual: acceso, estudiante, práctica, docencia y administración.

## Figuras

| Archivo | Slot | Vista y ruta | Origen | Resolución |
| --- | --- | --- | --- | --- |
| `figura-d-01.png` | D-01 — Acceso institucional | Inicio de sesión, `/login` | Capturada hoy con Playwright | `2880x1800` px |
| `figura-d-02.png` | D-02 — Panel del estudiante | Inicio del estudiante, `/student`, con menú lateral expandido | Capturada hoy con Playwright | `2880x1800` px |
| `figura-d-03.png` | D-03 — Resolución de práctica | Práctica «Validador de Expresiones Balanceadas», `/student/exercises/9e000000-0000-4000-8000-000000000001` | Capturada hoy con Playwright | `2880x1800` px |
| `figura-d-04.png` | D-04 — Panel del docente | Inicio del docente, `/teacher/dashboard`, con menú lateral expandido | Capturada hoy con Playwright | `2880x1800` px |
| `figura-d-05.png` | D-05 — Administración de estudiantes | Gestión de estudiantes, `/admin/estudiantes` | Capturada hoy con Playwright | `2880x1800` px |

## Verificación visual

- Las cinco imágenes se abrieron y revisaron una por una después de la captura.
- No hay pantallas en blanco, indicadores de carga congelados, modales de inicio de sesión inesperados ni avisos de error visibles.
- D-01: la configuración del tenant referencia `/assets/uab-logo.png`, que no está publicado. La interfaz usa el escudo con las iniciales «UAB»; no aparece un aviso de error. La consulta de sesión recibe `401` porque no hay sesión, que es el estado esperado de la pantalla de acceso.
- D-02: menú lateral expandido para que las etiquetas de navegación sean legibles en impresión.
- D-03: enunciado, restricciones y código Python visibles. La práctica no declara casos de ejemplo, por eso el panel «Casos de ejemplo» indica `0`; no es un error de la captura. El código de partida se muestra completo.
- D-04: menú lateral expandido. La vista muestra tres materias, doce entregas por calificar y siete cierres próximos.
- D-05: tabla poblada con 210 estudiantes, filtros, paginación y el estado del entorno de una cuenta en ejecución.

## Auditoría de las capturas anteriores

Las nueve imágenes de `docs/evidencias/multilenguaje/` pertenecen al commit `1b8847c` del 2026-11-20, entre las 22:12 y las 22:44. El rediseño de interfaz de ADR-093 se registró en el commit `31c3b83` del 2026-11-22 y, después de las capturas, se acumularon 31 commits sobre `frontend/src`. Por eso ninguna coincide al 100 % con la interfaz actual.

| Archivo | Resolución | Diagnóstico |
| --- | --- | --- |
| `estudiante-cpp.png` | `590x657` | Recorte de la interfaz anterior a ADR-093; no es una vista completa del manual. |
| `estudiante-cpp-aceptado.png` | `590x657` | Recorte de la interfaz anterior a ADR-093; no es una vista completa del manual. |
| `estudiante-cpp-cumple.png` | `1280x1100` | Shell, tipografía y escala de color anteriores a ADR-093. Conserva el distintivo del navegador de captura. |
| `estudiante-java-seleccionado.png` | `1280x1100` | Shell anterior a ADR-093; tamaño menor al requerido para impresión. |
| `estudiante-java-cumple.png` | `1280x1100` | Shell anterior a ADR-093; tamaño menor al requerido para impresión. |
| `docente-editor.png` | `1280x1100` | Editor con la barra superior y los controles anteriores a ADR-093. Conserva el distintivo del navegador de captura. |
| `docente-lenguajes.png` | `1280x1100` | Shell anterior a ADR-093; tamaño menor al requerido para impresión. |
| `docente-java-tabs.png` | `1280x1100` | Shell anterior a ADR-093; tamaño menor al requerido para impresión. |
| `admin-catalogo-lenguajes.png` | `1280x1100` | Catálogo con la escala y la barra laterales anteriores a ADR-093. Conserva el distintivo del navegador de captura. |

Conclusión: las nueve imágenes están desactualizadas y ninguna cubre los cinco slots del manual. No se reutilizó ninguna.

## Limpieza

- No se iniciaron ni detuvieron contenedores.
- No se escribieron usuarios, materias, prácticas ni entregas.
- Los scripts y tokens temporales usados para la captura se eliminaron de `/tmp` al terminar.
