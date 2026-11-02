# Manual del Estudiante — Plataforma Académica SOLV

Bienvenido a **SOLV**, la plataforma de laboratorios virtuales y evaluación de programación. Esta guía rápida te ayudará a navegar por tus cursos, resolver ejercicios y consultar tus sugerencias de refuerzo.

---

## 1. Navegación por el Mapa Curricular

Al ingresar a un curso (`/student/courses/:courseId`), visualizarás la estructura de módulos de la materia:

### Indicadores de Estado de Módulo
- **🔒 Bloqueado:** El módulo requiere que completes previamente los ejercicios de los módulos prerrequisito con el puntaje mínimo de aprobación.
- **▶ En Progreso:** El módulo está abierto y tenés ejercicios pendientes por resolver.
- **✅ Completado:** Todos los ejercicios del módulo fueron aprobados exitosamente.

> [!NOTE]
> **Exención de Exámenes:** Los ejercicios evaluativos marcados como Examen son accesibles siempre, sin importar si el módulo temático al que pertenecen se encuentra bloqueado.

---

## 2. Resolución y Envío de Ejercicios

Al abrir un laboratorio o ejercicio de algoritmia:

1. **Enunciado y Casos de Ejemplo:** Revisá la descripción del problema en el panel izquierdo. Los casos con etiqueta **Ejemplo** muestran entradas y salidas de prueba para guiar tu código.
2. **Editor de Código:** Escribí tu solución utilizando la plantilla base precargada en el lenguaje configurado.
3. **Envío y Evaluación:** Pulsá el botón **Enviar**. El Juez Virtual ejecutará tu programa contra los casos de prueba privados y te devolverá el veredicto en tiempo real:
   - **AC (Aceptado):** Tu solución pasó todos los casos correctamente.
   - **WA (Wrong Answer):** La salida producida no coincide con la esperada.
   - **TLE (Time Limit Exceeded):** Tu programa superó el tiempo límite.
   - **RE (Runtime Error):** Error de ejecución o excepción no capturada.

---

## 3. Panel de Práctica de Refuerzo (Opcional)

En la barra lateral de la vista del curso encontrarás el panel **"💡 Práctica de refuerzo (opcional)"**:

- **¿Cómo funciona?:** Si tenés intentos en temas donde tu tasa de éxito es menor al 60%, el sistema te sugerirá ejercicios de práctica extracurricular etiquetados con esos temas.
- **Sin impacto académico:** Estas recomendaciones son **100% opcionales**. No afectan tu nota, tu avance en el mapa curricular ni el desbloqueo de módulos. Podés realizarlas cuando desees para reforzar tus conocimientos.

---

## 4. SOLV Companion en el editor

En las prácticas con IDE persistente podés instalar **SOLV Companion**, la extensión que muestra los comentarios del docente dentro de OpenVSCode Server.

1. Ejecutá `SOLV: Configurar conexión`.
2. Indicá la URL de la API, el identificador de tu entrega y tu token de sesión.
3. Los comentarios anclados a un archivo aparecen en el margen de la línea y al pasar el mouse.

El token se guarda cifrado en el almacenamiento del editor. La extensión también registra la cadencia de escritura de los archivos del proyecto para la revisión del docente. La guía técnica está en [SOLV_COMPANION.md](UI/SOLV_COMPANION.md).
