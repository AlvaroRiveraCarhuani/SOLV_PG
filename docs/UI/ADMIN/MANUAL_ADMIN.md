# Manual de Administración de Plantillas de Entorno — SOLV

Guía oficial de gobernanza, configuración técnica y operaciones para administradores de la plataforma SOLV.

---

## 1. Comparativa de Entornos: IDE Persistente vs Juez Virtual

SOLV provee dos modalidades diferenciadas de ejecución de laboratorios virtuales según el objetivo pedagógico de la cátedra:

| Característica | Laboratorio Interactivo (IDE Persistente) | Juez Virtual (Sandbox Algorítmico) |
| :--- | :--- | :--- |
| **Propósito Pedagógico** | Sesiones interactivas de desarrollo, experimentación y proyectos integradores. | Evaluación automática de algoritmos, pruebas unitarias y exámenes cronometrados. |
| **Interfaz de Usuario** | Editor web completo OpenVSCode Server con terminal integrada y visor de archivos. | Sin interfaz gráfica ni web. Ejecución automatizada vía API / CLI de evaluación. |
| **Ciclo de Vida** | Sesiones de larga duración. Soporta estados activo, inactivo y suspensión controlada. | Efímero y desechable. Se instancia por envío, ejecuta la prueba y se destruye de inmediato. |
| **Persistencia de Datos** | Volumen de trabajo persistente por estudiante montado en `/home/workspace`. | Sistema de archivos efímero (`tmpfs` / capa descartable sin persistencia posterior). |
| **Servicios Satélite** | Admite bases de datos satélite desacopladas (PostgreSQL, MariaDB, Redis). | Aislado estrictamente: sin servicios satélite ni dependencias de red externa. |
| **Acceso a Red** | Restringido a la red interna institucional del laboratorio; sin salida pública salvo allowlist. | Aislamiento estricto de red (`network_mode: none` o bridge local sin gateway externo). |
| **Parámetros de Ejecución** | Puerto HTTP para OpenVSCode Server, script de inicialización (`setup_script`). | Comando de compilación/ejecución, tiempo límite (`timeout_ms`), entrada estándar (`stdin`). |
| **Límites de Recursos** | Dimensionado para soporte de IDE web y herramientas de compilación/ejecución concurrentes. | Dimensionado compacto y austero para una evaluación unitaria rápida. |

---

## 2. Flujo de Publicación y Gobernanza Institucional

El ciclo de vida de una plantilla garantiza que ningún entorno alcance a los estudiantes sin validación técnica previa:

```
[ BORRADOR ] 
      │  (Guardado en PostgreSQL por usuario y tenant)
      ▼
[ PENDIENTE_AUDITORIA ]
      │  (Prueba de entorno / Smoke test + Auditoría CVE)
      ├──────────────────────────────┐
      ▼                              ▼
[ APROBADA ]                   [ RECHAZADA ]
      │                         (Motivo obligatorio >= 10 caracteres)
      ├─────────────────┐
      ▼                 ▲
[ SUSPENDIDA ] ─────────┘ (Reactivación exige smoke test en verde)
  (Deprecación suave, no altera entornos históricos)
```

### Reglas Clave de Gobernanza:
1. **Separación de Roles (Decisión D4):** El docente propone los requerimientos pedagógicos; la potestad sobre memoria RAM, cuotas vCPU y aprobación definitiva es exclusiva del Administrador.
2. **Inmutabilidad de Entornos en Producción:** Las plantillas aprobadas que han sido utilizadas por cátedras nunca sufren eliminación física (`hard delete`). Si una imagen queda obsoleta, se marca como `SUSPENDIDA`.
3. **Criterio de Uso Institucional:** El conteo de uso de una imagen (`usage_count`) contabiliza plantillas activas y suspendidas que efectivamente operaron en el sistema. Las plantillas con estado `RECHAZADA` quedan expresamente excluidas.

---

## 3. Guía de Imágenes OCI y Buenas Prácticas

### 3.1 Prohibición Estricta del Tag `:latest`
- El uso de `:latest` está terminantemente prohibido en SOLV.
- **Justificación:** Viola el principio de reproducibilidad académica. Una entrega evaluada hoy debe comportarse idénticamente si se vuelve a auditar en tres años.
- **Práctica requerida:** Fijar siempre la versión de distribución y versión de lenguaje (por ejemplo, `python:3.12-slim-bookworm` en lugar de `python:latest`).

### 3.2 Anatomía de una Referencia OCI
Una referencia completa se compone de:
- **Registro:** Servidor donde reside la imagen (ej: `docker.io` para Docker Hub, `ghcr.io` para GitHub Packages). Si se omite, se asume Docker Hub oficial.
- **Repositorio / Namespace:** Organización y nombre de la imagen (ej: `library/python` o `alvaro/solv-worker`).
- **Tag:** Versión explícita e inmutable (ej: `3.12-slim-bookworm`).

### 3.3 Variantes Recomendadas
- **`-slim` (Debian Slim):** Variante recomendada por defecto. Excelente compatibilidad con dependencias C/C++ y tamaño moderado.
- **`-alpine` (Alpine Linux):** Solo recomendada para microservicios muy compactos o donde `musl-libc` no genere incompatibilidades con paquetes binarios (ej. `numpy` o `wheels` de Python).
- **Distroless:** Indicada para ejecutables estáticos compilados en Go o Rust en entornos de juez virtual.

---

## 4. Perfiles de Recursos y Derivación cgroups v2

Para garantizar estabilidad en servidores on-premise compartidos, SOLV calcula de forma automática y determinística el perfil de memoria cgroups v2 a partir de la memoria base (`base_ram_mb`):

$$\text{Memoria Mínima Garantizada } (memory.min) = \frac{\text{base\_ram\_mb}}{2}$$
$$\text{Umbral de Throttling } (memory.high) = \text{base\_ram\_mb} \times 1.5$$
$$\text{Límite Máximo Duro } (memory.max) = \text{base\_ram\_mb} \times 2$$

- **Ejemplo con base de 1024 MB:**
  - `min_mb`: 512 MB garantizados sin reclamación de memoria.
  - `high_mb`: 1536 MB donde el kernel comienza a frenar la asignación de páginas.
  - `max_mb`: 2048 MB límite absoluto que dispara el OOM killer si es superado.

---

## 5. Casos de Uso Típicos por Disciplina

### Algoritmos y Programación Inicial (C / C++ / Python)
- **Modalidad:** Juez Virtual o IDE Persistente Ligero.
- **Imagen recomendada:** `gcc:14-bookworm` o `python:3.12-slim-bookworm`.
- **RAM base:** 512 MB (Juez) / 1024 MB (IDE).

### Desarrollo Web y Frontend (Node.js / TypeScript)
- **Modalidad:** IDE Persistente.
- **Imagen recomendada:** `node:20-bookworm-slim`.
- **RAM base:** 1536 MB a 2048 MB para soportar `npm install` y servidores Vite/Webpack concurrentes.

### Sistemas y Bases de Datos (Fullstack con Satélite)
- **Modalidad:** IDE Persistente con Base de Datos Satélite.
- **Imagen principal:** `golang:1.24-bookworm` o `openjdk:21-slim-bookworm`.
- **Servicio satélite:** `postgres:18-alpine` desacoplado en red interna con volumen propio.
- **RAM base:** 2048 MB para la aplicación + cuota independiente del motor relacional.
