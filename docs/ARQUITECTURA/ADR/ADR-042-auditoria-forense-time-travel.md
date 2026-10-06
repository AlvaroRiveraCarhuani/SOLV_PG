# ADR-042: Auditoría forense con Time-Travel Replay y Benchmark O(N)

## Estado
Aprobado

## Contexto
El fraude académico en laboratorios de programación suele presentarse mediante el copiado masivo de soluciones completas desde fuentes externas. Las herramientas tradicionales de detección de plagio por AST analizan la entrega final en frío pero no permiten auditar el proceso de construcción tecla a tecla ni la eficiencia del algoritmo.

## Decisión
1. **Captura continua de Keystrokes:** El editor de código del estudiante captura en búfer los eventos de escritura (`insert`, `delete`, `paste`) con sus timestamps relativos (`timestamp_ms`) y posición en el texto, transmitiéndolos en lotes al backend.
2. **Almacenamiento de Auditoría:** La tabla `submission_keystroke_events` persiste los eventos con marca de `paste_source_detected = true` ante inserciones abruptas mayores a 50 caracteres o eventos explícitos de portapapeles.
3. **Reproductor interactivo Time-Travel Replay:** Integrado en SpeedGrader ([`KeystrokeReplayModalComponent`](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/frontend/src/app/features/teacher/grading/speed-grader/keystroke-replay-modal/keystroke-replay-modal.component.ts)), permite reproducir el desarrollo del código con slider temporal, control de velocidad (0.5x–4x) y resaltado en rojo de fragmentos pegados.
4. **Benchmark de Complejidad Asintótica O(N):** El motor de evaluación calcula la complejidad empírica ($O(1)$, $O(N)$, $O(N \log N)$, $O(N^2)$) mediante regresión de mínimos cuadrados sobre muestras de diferente tamaño de entrada $N$, registrando el resultado en `complexity_analysis JSONB`.
5. **Espejo Forense sin Impacto en Score:** El benchmark de complejidad opera únicamente como herramienta de consulta para el docente en SpeedGrader sin alterar el veredicto ni la calificación automática del alumno.

## Consecuencias
- **Positivas:** Trazabilidad forense completa de la autoría del código entregado. Detección inmediata de copias desde LLMs o repositorios.
- **Negativas:** Incremento moderado en el volumen de almacenamiento de eventos por entrega (mitigado por límite de 10.000 eventos por lote).
