# Anexo D — Capturas del Manual de Usuario

## Alcance

- Fecha de captura: 2026-11-24.
- Servidor: Asus, con el stack `solv_frontend`, `solv_backend`, `solv_traefik`, `solv_db` y `solv_seaweedfs` en ejecución.
- Origen: `https://solv.dedyn.io` para el panel; `https://dd8a459e-4abd-440c-9846-db87abd4654e.solv.dedyn.io` para el IDE de la práctica.
- Navegador: Chromium sin interfaz mediante Playwright.
- Viewport: `1440x900` con `deviceScaleFactor: 2`. Los seis PNG miden `2880x1800` px.
- Tema: claro. Sin modales de bienvenida ni indicadores de carga en el encuadre final.
- Sesiones: estudiante `estudiante.dev@uab.edu.bo`; docente `docente.dev@uab.edu.bo`; administrador canónico `00000000-0000-0000-0000-000000000001`. No se crearon usuarios ni materias.

Las seis capturas cubren el recorrido del manual: IDE, resolución de una práctica, revisión docente, edición de la práctica y administración del entorno.

## Figuras

| Archivo | Slot | Vista y ruta | Origen | Resolución |
| --- | --- | --- | --- | --- |
| `figura-d-01.png` | D-01 — IDE de la práctica | Proyecto `cursosDeportivos`, README abierto y panel `SOLV Companion` con «Conexión activa», `https://dd8a459e-4abd-440c-9846-db87abd4654e.solv.dedyn.io/` | Capturada hoy con Playwright | `2880x1800` px |
| `figura-d-02.png` | D-02 — Resolución de práctica | Práctica «Inversión de Cadenas (Multilenguaje)», `/student/exercises/9e000000-0000-4000-8000-000000000099` | Capturada hoy con Playwright | `2880x1800` px |
| `figura-d-03.png` | D-03 — Revisión docente | SpeedGrader de la entrega «Sistema de Reservas de Laboratorio», `/teacher/revision/a1000000-0000-4000-8000-000000000012` | Capturada hoy con Playwright | `2880x1800` px |
| `figura-d-04.png` | D-04 — Edición de la práctica | Práctica «cursosDeportivos — Flask + React», `/teacher/cursos/6879c008-375e-4849-a341-6517463ab81e/practicas/08d80113-cc02-44cb-91fa-08e671453675` | Capturada hoy con Playwright | `2880x1800` px |
| `figura-e-01.png` | E-01 — Administración de entornos | Dashboard de infraestructura, `/admin/dashboard`, con la tabla de contenedores expandida | Capturada hoy con Playwright | `2880x1800` px |
| `figura-e-02.png` | E-02 — Catálogo de plantillas | Editor de la plantilla «Python + React + MySQL — Cursos Deportivos», `/admin/plantillas/fc594f84-8180-42ab-8417-bf265bde0211/editar` | Capturada hoy con Playwright | `2880x1800` px |

## Verificación visual

- Las seis imágenes se abrieron y revisaron una por una después de la captura.
- No hay pantallas en blanco, indicadores de carga congelados, modales de inicio de sesión inesperados ni avisos de error visibles.
- D-01: árbol del proyecto, README y panel Companion. El panel muestra «Conexión activa · SOLV Companion»; la barra de estado muestra `SOLV` sin `Conectando…`.
- D-02: la práctica permite Go, Python y C++. La entrega se resolvió en Python. Se ven las dos restricciones traducidas al español, el caso de ejemplo, el código, el veredicto `Cumple`, `4877 ms` y el identificador abreviado `f67eddeb`. El motor devuelve `memory_used_mb = 0`; la interfaz omite el bloque de memoria cuando el valor es cero. No se inventó una cifra.
- D-03: `main.py` abierto, comentario docente visible y rúbrica con sus cuatro criterios. El comentario está anclado a la línea global 7 de la entrega y se muestra como `L1` dentro de `main.py`.
- D-04: especificación técnica con MySQL, Flask y React; «Bases de los alumnos» expandida con el aviso del script vigente y la base de `Student Dev` en estado `running`.
- E-01: tabla «Contenedores de Laboratorio Activos» expandida con una instancia, `260 / 2048 MB`, tiempo restante y estado `Activo`.
- E-02: pestaña «Base de datos» seleccionada, MySQL v8.4 marcado y `1024 MB` por alumno en el resumen del entorno.

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

Conclusión: las nueve imágenes están desactualizadas y ninguna cubre los seis slots del manual. No se reutilizó ninguna.

La antigua `figura-d-05.png` se retiró porque no corresponde a ninguno de los seis slots actuales.

## Limpieza

- Se registró una entrega de práctica en «Inversión de Cadenas» y se añadió un comentario docente en la entrega de reservas.
- El entorno `dd8a459e-4abd-440c-9846-db87abd4654e` se dejó hibernado al terminar las capturas.
- Los scripts y tokens temporales usados para la captura se eliminaron de `/tmp` al terminar.
