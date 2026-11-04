# Vista 3: Práctica con IDE (`/student/exercises/:exerciseId`)

> **Especificación de interfaz y contratos**  
> **Rol:** Estudiante  
> **Gobernanza:** SOLV Design System · ADR-059 (entorno por práctica)

---

## 1. Modelo mental

La práctica es una sola pantalla: el enunciado, el editor y el juez. El entorno **no** es una vista aparte ni un laboratorio de la materia: es el contenedor de esa práctica, con su volumen propio (`solv_workspace_<alumno>_<materia>_<práctica>`), y se abre desde la cabecera.

Al entrar no se crea nada: si el entorno ya existe se reanuda, y si no, el alumno lo pide con `[Abrir entorno]`. El inicio y el aula solo enlazan acá; nunca aprovisionan.

---

## 2. Composición

```text
┌──────────────────────────────────────────────────────────────────────────────┐
│ SIS-211 · Árboles binarios                          [ Abrir entorno ]        │
│ Estructuras de Datos · clase hasta las 10:00                                │
├──────────────────────────────────────────────────────────────────────────────┤
│ El docente emitió un avance · 3 archivos · hace 2 min                       │
│                                          [ Ver archivos ]  [ Descartar ]    │
├──────────────────────────────────────────────────────────────────────────────┤
│ Consigna           │  Editor / IDE embebido                                 │
│ Ejemplos           │                                                        │
│ Restricciones      │                                                        │
├────────────────────┴────────────────────────────────────────────────────────┤
│ [ Ejecutar pruebas ]  [ Enviar solución ]        AC · WA · TLE · RE · AST   │
└──────────────────────────────────────────────────────────────────────────────┘
```

### 2.1. Cabecera
- Materia y título de la práctica.
- Ventana de la sesión cuando la práctica pertenece a un aula abierta.
- `[Abrir entorno]` / `[Reconectar]` según el estado. Con el aula congelada el botón no aparece y el área del editor se reemplaza por el motivo y la hora del cierre (ADR-005).

### 2.2. Franja de avance del docente
- Una franja por aviso pendiente, sobre el editor. Dice que el docente emitió un avance, cuántos archivos trae y que puede tardar hasta un minuto en aparecer (la consulta es cada minuto, ADR-058 §8).
- `[Ver archivos]` abre la confirmación; `[Descartar]` lo descarta para esa sesión.
- El aviso se filtra por `exercise_id`: un avance de otra práctica no se muestra.

### 2.3. Confirmación del avance
- Dos listas: los archivos que se reemplazan —en ámbar, con el aviso de que la versión actual queda respaldada— y los que se agregan.
- `[Aplicar avance]` escribe en el volumen de la práctica. Al terminar informa cuántos archivos se escribieron y dónde quedó la copia anterior.
- Si el entorno está detenido, el modal lo explica y la cabecera ofrece `[Iniciar IDE]`; el aviso sigue pendiente hasta que se aplique.

### 2.4. Editor y juez
- El IDE se embebe sin exponer identificadores de contenedor, puertos ni consumo interno.
- `[Ejecutar pruebas]` corre los casos visibles; `[Enviar solución]` valida casos privados y análisis estático.
- Veredictos: `AC` verde, `WA` rojo, `TLE` ámbar, `RE` naranja, `AST_BLOCKED` rojo intenso con la regla infringida.

---

## 3. Contratos

| Pedido | Uso |
| :--- | :--- |
| `POST /api/v1/environments/start` | Abre o reanuda el entorno de la práctica. Exige `subject_id` y `exercise_id`. |
| `POST /api/v1/environments/{id}/heartbeat` | Mantiene viva la instancia mientras la pestaña está visible. |
| `POST /api/v1/environments/{id}/pause` | Hibernación manual. |
| `POST /api/v1/environments/{id}/submit` | Entrega el proyecto del volumen. |
| `GET /api/v1/class-advances/pending` | Avisos del docente para las prácticas del alumno. |
| `GET /api/v1/class-advances/{id}/files` | Archivos que reemplaza y agrega el avance. |
| `POST /api/v1/class-advances/{id}/apply` | Aplica el avance en el volumen de la práctica. |
| `POST /api/v1/class-advances/{id}/dismiss` | Descarta el aviso. |
| `POST /api/v1/evaluations` | Ejecuta el juez. |

`start` devuelve `environment_id`, `status` y `access_url`. El estado del contenedor es técnico (`running`, `hibernated`, `failed`); el estado académico de la práctica es otro y se muestra por separado.

---

## 4. Estados y errores

- **Entorno detenido:** la cabecera ofrece abrirlo; el editor explica que está en pausa.
- **Entorno fallido u OOM:** mensaje humano y `[Reintentar]`; nunca se muestra el código de salida ni el id del contenedor.
- **Aula congelada:** sin `[Abrir entorno]`, sin enlace al editor y sin aplicar avances nuevos.
- **Aviso de otra práctica:** no se muestra; el filtro es por `exercise_id`.
- **Fallo de red del aviso:** el aviso no se pierde; se reintenta en el siguiente ciclo.

---

## 5. Verificación

- `student-exercise-solver.component.spec.ts`: arranque del entorno, franja de avance, confirmación y examen congelado.
- `class-advance-notice.service.spec.ts`: consulta, archivos, aplicación y descarte.
- `advance-confirm-modal`: listas de reemplazo y agregado.
- Integración `TestClassAdvance*` y `TestStudentAulas*` contra Postgres; E2E de Docker para emisión y congelación.
