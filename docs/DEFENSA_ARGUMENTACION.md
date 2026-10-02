# Argumentación de Defensa — Punto más difícil de argumentar

Documento de preparación para la defensa. Actividad: elegir el aspecto del proyecto más difícil de argumentar y construir una defensa con contra-argumento.

Tema elegido: **Arquitectura on-premise en nodo único.**

Estructura: cinco párrafos (Elección, Por qué sí, Texto, Por qué no / contra-argumento, Conclusiones).

---

## 1. Elección

Elegimos la arquitectura on-premise en un nodo único como el aspecto más difícil de argumentar del proyecto, por encima de la metodología y del stack tecnológico. La razón es que decisiones como usar Go, Angular o Docker se justifican con criterios técnicos acotados, mientras que concentrar toda la plataforma en un solo servidor físico exige defender una limitación estructural de disponibilidad, escalabilidad y continuidad, frente a la expectativa generalizada de que todo sistema moderno debe ser elástico y redundante. Es, por tanto, la decisión que un tribunal con mayor probabilidad sometería a crítica directa.

## 2. Por qué sí

La elección de un nodo único on-premise se sostiene en tres argumentos. Primero, el contexto de la institución: la universidad opera con hardware propio y políticas restrictivas de presupuesto en divisas, por lo que una solución local aprovecha infraestructura existente y elimina el gasto recurrente en la nube. Segundo, el propósito educativo: los laboratorios prácticos tienen una demanda acotada y previsible de aproximadamente 35 a 40 sesiones dentro de una ventana horaria, no una carga continua e impredecible que justifique elasticidad automática. Tercero, la soberanía de datos y la independencia de conectividad externa, que en un entorno académico con restricciones de red resultan más valiosas que el escalado horizontal. En conjunto, el nodo único no es una carencia, sino una decisión coherente con el problema que se busca resolver.

## 3. Texto

El sistema concentra la orquestación de contenedores efímeros en un servidor local que aprovisiona entornos aislados con cuotas duras de memoria, bloqueo de red entre contenedores y liberación de recursos por hibernación. Sobre ese nodo corre un juez virtual dual que combina análisis estático y evaluación por entrada y salida, de modo que las fallas de un estudiante quedan contenidas en su propio contenedor y no comprometen al servidor ni al resto de la clase. La arquitectura privilegia la densidad controlada sobre la redundancia: en lugar de replicar servicios para tolerar fallas de infraestructura, el diseño evita que la falla de un entorno se propague, sosteniendo la operación completa del aula desde un único punto.

## 4. Por qué no (contra-argumento)

El contra-argumento más fuerte es que el nodo único constituye un punto único de falla: si el servidor cae, se detiene toda la plataforma, sin alta disponibilidad ni escalado horizontal, con un techo físico inamovible y un mantenimiento concentrado en una sola persona. Frente a esto sostenemos que la alternativa cloud resuelve esa disponibilidad, pero a cambio traslada el problema a un costo recurrente en dólares insostenible para el presupuesto institucional y a una dependencia de red externa que agrava la saturación de la conectividad durante una clase de 35 a 40 estudiantes simultáneos. Es decir, el nodo único cambia una falla total improbable por un costo estructural permanente; la limitación se asume de forma explícita y se declara como trabajo futuro, con la incorporación de un segundo nodo o curso piloto como siguiente paso.

## 5. Conclusiones

La arquitectura on-premise en nodo único es una decisión válida para un producto mínimo viable institucional, porque prioriza viabilidad económica, soberanía de datos y contención de fallas por sobre la elasticidad total. Su debilidad real no es de tecnología sino de continuidad, y por eso la postura correcta no es negarla, sino declararla: el sistema contiene la falla de un estudiante para que no afecte al resto del aula, y proyecta su escala como trabajo futuro con instrumentación y un segundo nodo. Defender una limitación conocida y acotada es más sólido que prometer una disponibilidad que el contexto no permite sostener.

---

## 6. Evidencia que respalda la postura

La defensa no se sostiene solo en prosa; se apoya en mecanismos verificables del sistema:

- **Contención individual:** ante una prueba de integración, un entorno que excede la cuota de 256 MB es terminado por el sistema operativo (OOMKilled) sin afectar al nodo ni al resto de los entornos.
- **Aislamiento de red:** la comunicación entre contenedores está bloqueada a nivel de red, con descarte total del tráfico lateral.
- **Liberación de recursos:** la hibernación por inactividad libera memoria y permite reanudar el entorno, lo que sostiene la densidad del aula.
- **Efecto agregado:** la falla de un estudiante queda confinada a su contenedor, lo que respalda la afirmación de contención sin prometer disponibilidad total.

## 7. Preguntas anticipadas (Q&A)

| Pregunta probable | Respuesta corta |
| :--- | :--- |
| ¿Y si se cae el nodo? | Es una falla total declarada como limitación; se mitiga con un segundo nodo como trabajo futuro, no se niega. |
| ¿Por qué no usar la nube? | Traslada el costo a un gasto recurrente en dólares y a dependencia de red; el nodo local es costo único y soberanía. |
| ¿Cuánto cuesta? | CAPEX Bs 4.670–15.750 y OPEX Bs 206–305/mes; el payback contra cloud es lento, por eso el valor se sostiene en horas docentes y soberanía. |
| ¿Por qué un solo equipo y no un clúster? | La demanda es acotada (35–40 sesiones en ventana horaria) y el presupuesto es limitado; un clúster estaría sobredimensionado. |
| ¿Cómo sabés que contiene fallas? | Con pruebas de integración: aislamiento por cuota, bloqueo de red y hibernación verificados. |

## 8. Trade-offs (qué se gana y qué se resigna)

| Se gana | Se resigna |
| :--- | :--- |
| Costo único y licencias en cero | Alta disponibilidad |
| Contención de fallas por entorno | Escalado horizontal automático |
| Soberanía de datos | Redundancia de infraestructura |
| Independencia de red externa | Crecimiento elástico sin intervención |

Límite de validez de la postura: el nodo único es sostenible mientras la demanda se mantenga en la ventana acotada de 35 a 40 sesiones. Superado ese umbral, la decisión deja de ser suficiente y exige un segundo nodo, lo que se declara como trabajo futuro.

---

Nota de uso: en el documento escrito, la limitación del nodo único se declara en la Discusión y en los Límites. En la defensa, este documento funciona como guion de la respuesta a la pregunta difícil.
