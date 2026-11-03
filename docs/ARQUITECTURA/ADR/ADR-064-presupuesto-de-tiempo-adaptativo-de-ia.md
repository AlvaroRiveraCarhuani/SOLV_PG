# ADR-064: Presupuesto de tiempo adaptativo por motor de IA

**Estado:** Implementado
**Área:** Inteligencia Artificial Híbrida

## Contexto

El presupuesto de tiempo de cada llamada al motor de IA se resolvía con una tabla fija por perfil de motor (`fast_cloud`, `queued_or_reasoning`, `local_gpu`) y por acción, con el `timeout_seconds` del administrador como piso. La tabla asume que todos los modelos de un perfil responden igual, y no es cierto: dos modelos gratuitos de OpenRouter pueden diferir en decenas de segundos, y la misma combinación varía según la hora y la carga del proveedor.

El resultado práctico era un plazo que servía para algunos modelos y quedaba corto para otros. Cuando se agotaba, el docente recibía un aviso de tiempo y perdía la interpretación de su práctica; si se subía el tope a mano para compensar, las llamadas rápidas esperaban de más antes de fallar.

## Decisión

El presupuesto se calcula con la latencia que el servicio mide del motor real, no solo con el perfil.

1. **Historial por motor y acción.** Cada llamada completada deja una muestra en `tenants.config["ai_settings"]["latency_profile"]`, identificada por proveedor, modelo y acción. El valor guardado es un promedio móvil con peso 0,4 para la última medición y 0,6 para el historial, acotado a 20 muestras.
2. **Presupuesto con margen.** El plazo es `3 × latencia observada`, con el presupuesto del perfil como piso y `MaxAITimeoutSeconds` (240 s) como techo. Sin historial se usa el presupuesto del perfil, de modo que la primera llamada de un motor nuevo nunca queda sin plazo.
3. **Plazo agotado como señal.** Si la llamada anterior agotó el plazo, la siguiente recibe `1,4 × lo esperado` en vez de repetir el mismo límite, y esa marca se limpia con la primera respuesta exitosa. La llamada que no contestó no entra al promedio: no dice cuánto tarda el motor, solo que el límite quedó corto, y contarla inflaría el plazo de las siguientes hasta convertir un fallo en minutos de espera.
4. **El tope del administrador manda.** `timeout_seconds` sigue siendo un piso configurable en Modo Experto: el cálculo automático nunca lo recorta.
5. **Escritura moderada.** Las dos primeras mediciones y los plazos agotados se guardan de inmediato, porque son los que cambian el presupuesto; el resto se persiste como mucho cada dos minutos. La escritura mezcla solo el campo de latencias, así no pisa un cambio del administrador hecho mientras la llamada estaba en curso, y sobrevive al corte del navegador.
6. **Techo de espera frente a la persona.** Las acciones que el docente espera en pantalla (`suggest-template`, `suggest-component`) nunca pasan de 90 s, aunque el perfil o el administrador permitan más: pasado ese punto la pantalla se percibe colgada y conviene que decida con la lista a mano. Las tareas de fondo conservan el tope de 240 s.
7. **El cliente también corta.** La consulta del formulario lleva su propio plazo de 100 s y muestra los segundos transcurridos con opción de cancelar. Así una red o un proxy colgado no deja el botón girando para siempre, y quien espera sabe cuánto lleva y puede detenerse.

## Consecuencias

- Un modelo lento deja de fallar por un plazo pensado para otro modelo, y uno rápido no espera de más.
- El comportamiento se corrige solo: si el proveedor se degrada, el historial sube el plazo; si mejora, el promedio baja y el piso del perfil evita plazos ridículamente cortos.
- El historial es dato operativo del tenant, no preferencia del administrador: se conserva al guardar la configuración y se descarta si queda corrupto.
- Queda una llamada por motor y acción que puede seguir agotando el plazo la primera vez que aparece un modelo nuevo; el perfil del motor es el respaldo para ese caso.

## Alternativas descartadas

- **Subir el plazo global a 240 s:** cambia un fallo por una espera larga en todos los modelos, incluso los rápidos.
- **Confiar solo en la prueba de conexión:** mide un JSON mínimo y no representa el tiempo de una generación completa.
- **Cola asíncrona con sondeo:** resuelve la espera percibida, pero multiplica la infraestructura; el presupuesto medido ataca la causa del fallo con el modelo actual.
- **Streaming de la respuesta:** incompatible con el JSON estricto que consumen el copiloto y el formulario.
