# Informe de Validación Experimental y Capacidad del Nodo Piloto Asus

**Fecha de ejecución:** 2026-11-24T11:58:36Z  
**Commit evaluado:** `f75321d` (árbol de trabajo sobre `26e109c`)  
**Modalidad de ejecución:** Core IEEE (campaña empírica cerrada en nodo físico)  
**Ubicación de salidas en crudo:** `docs/CAPACITY/pilot-raw/`  

***

## 1. Ficha Técnica del Nodo Piloto

La totalidad de los datos de este informe proviene de ejecuciones directas realizadas sobre el hardware físico del nodo piloto de SOLV.

| Parámetro | Especificación Real del Host |
| :--- | :--- |
| **Modelo del equipo** | Asus K501UX (plataforma portátil de pruebas) |
| **Procesador (CPU)** | Intel Core i7-6700HQ @ 2.60 GHz (4 núcleos físicos, 8 hilos, Skylake) |
| **Frecuencia base / turbo** | 2.60 GHz base, hasta 3.50 GHz turbo |
| **Memoria RAM física** | 12 GB DDR4 (11.0 GiB detectados por el kernel, 11283 MiB) |
| **Memoria disponible en reposo** | 8.1 GiB (MemAvailable) |
| **Almacenamiento** | SSD/HDD con volumen LVM de 914 GB (`/dev/mapper/ubuntu--vg-ubuntu--lv`) |
| **Uso de disco durante corrida** | 87 GB usados, 789 GB disponibles (10 % de uso, 90 % libre) |
| **Sistema Operativo** | Ubuntu 24.04.1 LTS (Noble Numbat) |
| **Kernel Linux** | 6.8.0-149-generic (x86_64) |
| **Controlador cgroups** | cgroups v2 unificado (`cpuset cpu io memory hugetlb pids rdma misc`) |
| **Motor Docker** | Docker Engine v29.3.1 (API 1.52) |
| **Compilador Go** | go1.27.3 linux/amd64 |
| **Analizador Semgrep** | Semgrep v1.186.0 (instalado aisladamente en `~/.local/bin/semgrep` vía pipx para evaluación de 3 capas; se documentan tanto el modo de 2 capas en memoria como el modo completo de 3 capas) |
| **Sensores de potencia** | `/sys/class/power_supply/{AC0,BAT0}` detectados. Registro energético restringido a root. Sin medidor de enchufe calibrado; se declara como estimación técnica conforme a D1-B |

***

## 2. Declaración de Configuración y Trampas Técnicas

Conforme a las directivas de `BENCHMARK_PLAN.md`, se aplicaron los siguientes ajustes de laboratorio para evitar distorsiones:

1. **Perfil de cupo por IP (ADR-089):** El limitador por defecto aplica 600 req/min con ráfaga de 120. Para la prueba de carga k6 sobre bucle local (`127.0.0.1`), se exportó `RATE_LIMIT_IP_BURST=10000` y `RATE_LIMIT_IP_PER_MINUTE=120000` en `.env` reiniciando el servicio para medir el cuello de botella real de la aplicación y no el limitador de tráfico.
2. **Límite de envíos por alumno:** Se respetó la cota de 10 entregas por minuto y por estudiante distribuyendo la carga entre los 200 alumnos generados por la siembra (`load-tests/seed_load_test_data.sql`), incorporando pausas entre 6 y 8 segundos en cada ciclo de usuario virtual.
3. **Recibo de evaluación obligatorio (ADR-066):** Se corrigió el flujo del script de carga para obtener primero el recibo criptográfico en `POST /api/v1/evaluations` y presentarlo luego en `POST /api/v1/submissions`, evitando respuestas 422 de recibo inválido.
4. **Camino de autenticación ForwardAuth:** Las peticiones ingresaron a través de Traefik por el puerto HTTPS 443 con la cookie de sesión `solv_session` firmada mediante JWT sobre el secreto real de la plataforma, evaluando el flujo auténtico sin atajos por el puerto interno 3000.
5. **Aislamiento de red de examen:** Se verificó la red interna `solv-exam-net` configurada con `Internal=true` y driver bridge, comprobando la ausencia de salida a redes públicas.
6. **Protección del almacenamiento:** El volumen de disco se monitoreó antes y después de cada bloque; se mantuvieron 789 GB libres, muy por encima del umbral de rechazo crítico del 5 %. Todos los contenedores de prueba fueron eliminados al terminar.

***

## 3. Tabla Maestra de Métricas (Formato IEEE)

Todas las métricas corresponden a mediciones reales obtenidas en este nodo. Los archivos crudos asociados se encuentran disponibles en `docs/CAPACITY/pilot-raw/`.

| Escenario | Métrica evaluada | N | Media (μ) | Desv. Est. (σ) | p50 | p95 | p99 | Mín / Máx | Archivo crudo |
| :--- | :--- | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :--- |
| **E1** | Latencia lectura ejercicios (`GET /api/v1/exercises`) | 298 | 18.70 ms | 16.48 ms | 19.84 ms | 24.90 ms | 35.12 ms | 5.71 / 208.59 ms | `e1_k6_summary.json` |
| **E1** | Peticiones de evaluación (`POST /api/v1/evaluations`) | 204 | 52.41 s | 14.82 s | 60.00 s | 60.00 s | 60.00 s | 1.84 / 60.00 s | `e1_k6_summary.json` |
| **E3** | Verificación ForwardAuth (`GET /api/v1/auth/verify`) | 200 | 2.07 ms | 0.21 ms | 2.03 ms | 2.43 ms | 2.72 ms | 1.64 / 2.73 ms | `e3_forwardauth.json` |
| **E3** | Cierre de sesión y revocación (`POST /api/v1/auth/logout`) | 1 | 35.37 ms | 0.00 ms | 35.37 ms | 35.37 ms | 35.37 ms | 35.37 / 35.37 ms | `e3_forwardauth.json` |
| **E4 (2 capas)** | Pre-chequeo en memoria código prohibido (`AST_BLOCKED`) | 200 | 0.0273 ms | 0.0442 ms | 0.0169 ms | 0.0621 ms | 0.2590 ms | 0.0105 / 0.4548 ms | `e4_precheck.json` |
| **E4 (2 capas)** | Pre-chequeo en memoria código limpio (`PASS`) | 200 | 0.1000 ms | 0.0859 ms | 0.0802 ms | 0.3096 ms | 0.5090 ms | 0.0450 / 0.5263 ms | `e4_precheck.json` |
| **E4 (3 capas)** | Pre-chequeo completo con Semgrep CLI (`AST_BLOCKED`) | 35 | 2294.91 ms | 69.69 ms | 2274.94 ms | 2450.84 ms | 2497.37 ms | 2222.14 / 2497.37 ms | `e4_precheck_3layers.json` |
| **E4 (3 capas)** | Pre-chequeo completo con Semgrep CLI (`PASS`) | 35 | 2271.06 ms | 76.88 ms | 2240.55 ms | 2469.39 ms | 2488.81 ms | 2202.61 / 2488.81 ms | `e4_precheck_3layers.json` |
| **E6** | Provisión en frío de IDE persistente | 1 | 15842.88 ms | 0.00 ms | 15842.88 ms | 15842.88 ms | 15842.88 ms | 15842.88 / 15842.88 ms | `e6_provision_resume.json` |
| **E6** | Hibernación de entorno (`POST /environments/{id}/pause`) | 30 | 186.91 ms | 70.47 ms | 154.05 ms | 285.05 ms | 294.75 ms | 24.75 / 294.75 ms | `e6_provision_resume.json` |
| **E6** | Reanudación tras hibernación (`POST /environments/start`) | 30 | 8939.50 ms | 2232.26 ms | 8101.51 ms | 14753.37 ms | 14753.54 ms | 6357.73 / 14753.54 ms | `e6_provision_resume.json` |
| **E9** | Tiempo de contención y muerte OOM | 1 | 480.00 ms | 0.00 ms | 480.00 ms | 480.00 ms | 480.00 ms | 480.00 / 480.00 ms | `e9_oom_containment.txt` |
| **E10** | Aislamiento de red de examen (`solv-exam-net`) | 1 | 4012.00 ms | 0.00 ms | 4012.00 ms | 4012.00 ms | 4012.00 ms | 4012.00 / 4012.00 ms | `e10_exam_net_isolation.txt` |
| **E11** | Clasificación de complejidad temporal | 4 | 0.12 ms | 0.04 ms | 0.11 ms | 0.18 ms | 0.19 ms | 0.08 / 0.19 ms | `e11_complexity_service.txt` |
| **E12** | Comparación sintáctica de plagio (k-gramas) | 6 | 0.15 ms | 0.05 ms | 0.14 ms | 0.22 ms | 0.24 ms | 0.09 / 0.24 ms | `e12_plagiarism_engine.txt` |
| **E13** | Validación de archivo y empaquetado SeaweedFS | 6 | 0.10 ms | 0.03 ms | 0.09 ms | 0.14 ms | 0.15 ms | 0.07 / 0.15 ms | `e13_seaweedfs_sha256.txt` |

***

## 4. Capacidad Concurrente Real en los 12 GB de RAM

El nodo cuenta con 11283 MiB físicos. El controlador de admisión de SOLV (`practice_environment_capacity.go`) reserva el 15 % para el sistema operativo (1693 MiB), fijando el techo de asignación de laboratorios en el 85 % (9590 MiB).

Los motores satélites gestionados compartidos (PostgreSQL de aula con 384 MiB, Redis con 96 MiB y MongoDB con 256 MiB) ocupan una sola reserva conjunta de 736 MiB en el nodo, independientemente de la cantidad de alumnos conectados. Esto deja un presupuesto neto repartible de 8854 MiB para contenedores de estudiantes.

### Matriz Paso a Paso bajo Préstamo Dinámico (ADR-082)

En este modelo, el sistema concede memoria adicional a los primeros estudiantes mientras haya margen (`loanQuotaMB`), doblando la cuota hasta 4096 MiB. A medida que ingresan más usuarios, la cuota se reduce progresivamente.

| Entorno | Cuota Solicitada | Cuota Concedida | Memoria Comprometida | Presupuesto Restante | Estado de Admisión | Memoria Estimada en Host |
| :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| 1 | 512 MiB | 4096 MiB | 4096 MiB | 4758 MiB | ADMITIDO | 6451 MiB |
| 2 | 512 MiB | 2048 MiB | 6144 MiB | 2710 MiB | ADMITIDO | 4403 MiB |
| 3 | 512 MiB | 1024 MiB | 7168 MiB | 1686 MiB | ADMITIDO | 3379 MiB |
| 4 | 512 MiB | 512 MiB | 7680 MiB | 1174 MiB | ADMITIDO | 2867 MiB |
| 5 | 512 MiB | 512 MiB | 8192 MiB | 662 MiB | ADMITIDO | 2355 MiB |
| 6 | 512 MiB | 512 MiB | 8704 MiB | 150 MiB | ADMITIDO | 1843 MiB |
| 7 | 512 MiB | 0 MiB | 8704 MiB | 150 MiB | **RECHAZADO (`ErrHostMemoryExhausted`)** | 1843 MiB |

### Comparativa de Modelos de Capacidad

| Modelo Evaluado | Cuota Base | Techo Máximo | Satélites Activos | Concurrencia Máxima | Paso de Rechazo |
| :--- | :---: | :---: | :---: | :---: | :---: |
| **Préstamo Dinámico Estándar (ADR-082)** | 512 MiB | 4096 MiB | 736 MiB | **6 entornos** | Paso 7 |
| **Préstamo Dinámico Proyecto (Go / Node)** | 1024 MiB | 4096 MiB | 736 MiB | **4 entornos** | Paso 5 |
| **Cuota Fija Estricta (con satélites)** | 512 MiB | 512 MiB | 736 MiB | **17 entornos** | Paso 18 |
| **Cuota Fija Estricta (sin satélites)** | 512 MiB | 512 MiB | 0 MiB | **18 entornos** | Paso 19 |
| **Cuota Fija Modo Evaluación (código ligero)** | 256 MiB | 256 MiB | 0 MiB | **37 estudiantes** | Paso 38 |

**Hallazgo técnico respecto a la tesis:**  
La cifra de "15 a 24 entornos IDE y 35 a 40 estudiantes en 12 GB" documentada en los borradores de la tesis deriva de una estimación estática basada en cuotas rígidas de 256 MiB a 512 MiB. Cuando entra en funcionamiento el controlador de admisión dinámica de ADR-082, los primeros alumnos reciben hasta 4096 MiB, lo que reduce la capacidad simultánea a 6 laboratorios. Para alcanzar 17 entornos completos en el nodo Asus, la configuración del tenant debe fijar cuotas estáticas sin préstamo dinámico.

***

## 5. Resultados de Seguridad y Aislamiento

### Contención OOM en cgroups v2 (Escenario E9)

Se ejecutó un contenedor con límite estricto de memoria de 50 MiB ejecutando una asignación forzada de 100 MiB mediante Python.

**Evidencia literal obtenida:**
```text
=== E9: OOM CONTAINMENT TEST ===
Host memory before: Mem: 11852 total, 3735 used, 2390 free, 8116 available
Launching container limited to 50MB allocating 100MB...
Container ExitCode: 137
Container OOMKilled: true
Container Status: exited
Host memory after:  Mem: 11852 total, 3694 used, 2430 free, 8157 available
Neighbor health: solv_backend is healthy
```
El kernel Linux y cgroups v2 terminaron inmediatamente el contenedor infractor con código 137 (`SIGKILL`) marcando `OOMKilled: true`. La memoria disponible en el host se mantuvo estable (8116 MiB antes frente a 8157 MiB después) y los contenedores vecinos conservaron su estado saludable sin reinicios ni degradación.

### Aislamiento en la Cadena DOCKER-USER (Escenario E10)

Se verificaron las reglas activas del cortafuegos mediante `sudo -n iptables -S DOCKER-USER`.

**Evidencia literal obtenida:**
```text
-N DOCKER-USER
-A DOCKER-USER -d 172.21.0.2/32 -i br-3c44c7bed060 -o br-3c44c7bed060 -j ACCEPT
-A DOCKER-USER -s 172.21.0.2/32 -i br-3c44c7bed060 -o br-3c44c7bed060 -j ACCEPT
-A DOCKER-USER -d 172.24.0.1/32 -i br-ed6ae1c39411 -p tcp -m tcp --dport 8080 -j DROP
-A DOCKER-USER -d 172.24.0.1/32 -i br-ed6ae1c39411 -p tcp -m tcp --dport 3000 -j DROP
-A DOCKER-USER -d 172.24.0.1/32 -i br-ed6ae1c39411 -p tcp -m tcp --dport 9090 -j DROP
-A DOCKER-USER -d 172.24.0.1/32 -i br-ed6ae1c39411 -p tcp -m tcp --dport 5432 -j DROP
-A DOCKER-USER -i wlp2s0 -o br-ed6ae1c39411 -m conntrack --ctstate NEW -j DROP
-A DOCKER-USER -d 172.17.0.1/32 -i docker0 -p tcp -m tcp --dport 8080 -j DROP
-A DOCKER-USER -d 172.17.0.1/32 -i docker0 -p tcp -m tcp --dport 3000 -j DROP
-A DOCKER-USER -d 172.17.0.1/32 -i docker0 -p tcp -m tcp --dport 9090 -j DROP
-A DOCKER-USER -d 172.17.0.1/32 -i docker0 -p tcp -m tcp --dport 5432 -j DROP
```
Los puentes virtuales de Docker tienen bloqueado el acceso directo a los puertos del host correspondientes a la API interna (3000), base de datos principal (5432), métricas (9090) y proxy (8080), impidiendo movimientos laterales desde contenedores de alumnos.

### Red de Examen sin Salida a Internet (Escenario E10)

Se ejecutó el test de integración de extremo a extremo `TestExamNetwork_E2E_Isolation` con `EXAM_NETWORK_E2E=1`.

**Evidencia literal obtenida:**
```text
=== RUN   TestExamNetwork_E2E_Isolation
--- PASS: TestExamNetwork_E2E_Isolation (4.00s)
PASS
ok      solv-backend/internal/infrastructure/docker     4.012s
```
La red `solv-exam-net` confirmó su atributo `Internal=true` y driver `bridge`. La sonda lanzada dentro de la red intentó comunicarse con `8.8.8.8` y fue bloqueada por el kernel, validando que durante un examen el entorno carece de salida a internet.

***

## 6. Análisis de Incidentes y Límites Observados

1. **Cuello de botella en evaluaciones sincrónicas bajo 200 usuarios (E1):**  
   Durante la corrida k6 a 200 usuarios virtuales, las peticiones de lectura (`GET /api/v1/exercises`) respondieron con una media de 18.70 ms y p95 de 24.90 ms. No obstante, las evaluaciones de código (`POST /api/v1/evaluations`) instancian contenedores efímeros de Docker de forma sincrónica. Al recibir 200 peticiones concurrentes, el daemon de Docker saturó la tasa de creación de procesos, provocando colas que alcanzaron el límite de contexto HTTP de 60 segundos.  
   *Recomendación de arquitectura:* La evaluación debe transicionar de un modelo síncrono bloqueante en HTTP a una cola de trabajo asincrónica con trabajadores dedicados y entrega de resultados desacoplada.

2. **Latencia de reanudación frente a la afirmación del manuscrito (E6):**  
   El manuscrito contemplaba una reanudación tras hibernación inferior a 500 ms basándose en la primitiva `docker unpause`. La medición real en el endpoint `POST /api/v1/environments/start` arrojó una media de 8939.50 ms (p50 de 8101.51 ms y p95 de 14753.37 ms).  
   *Causa técnica:* La reanudación no se limita a despausar el proceso. El servicio ejecuta `VerifyWorkspaceVolume` mediante un contenedor auxiliar para verificar la integridad del volumen del alumno y realiza la conciliación de bases de datos satélites. Esta comprobación es necesaria para la persistencia, pero eleva el tiempo de respuesta a 8 segundos.

3. **Comportamiento del pre-chequeo de código estático (E4: 2 capas frente a 3 capas con Semgrep CLI):**  
   Se contrastaron los dos perfiles de ejecución del analizador estático:
   - *Modo de 2 capas en memoria (AST estático de Go y reglas regex):* Satisface ampliamente el objetivo de latencia del manuscrito (< 100 ms), registrando medias de 0.027 ms para código prohibido y 0.100 ms para código limpio.
   - *Modo completo de 3 capas (filtro regex, Semgrep CLI 1.186.0 y analizador Go AST):* La ejecución del binario Semgrep como subproceso CLI independiente introduce un tiempo adicional constante de 2.2 a 2.3 segundos (media de 2294.91 ms para código bloqueado y 2271.06 ms para código limpio). Dicha latencia proviene del arranque del intérprete Python, la inicialización del entorno y la carga de reglas en disco.  
   *Recomendación de arquitectura:* Para conservar la inspección profunda de Semgrep sin comprometer la latencia en peticiones HTTP sincrónicas, la integración debe evolucionar de invocaciones por proceso CLI hacia un servicio demonio persistente en memoria o ejecutarse de forma asincrónica en cola de evaluación.

***

## 7. Escenarios Omitidos por Ventana de Tiempo (Core IEEE)

Para respetar la ventana operativa de 45 a 60 minutos en el servidor físico, se ejecutaron los bloques esenciales (E0, E1, E3, E4, E6, E7, E9, E10, E11, E12, E13) y se omitieron los siguientes escenarios de larga duración:

- **E2 (Muestreo prolongado de RSS del orquestador durante 60 minutos):** Omitido por tiempo; el orquestador se mantuvo estable durante toda la campaña sin fugas visibles en `ps`.
- **E5 (Construcción en frío y caliente de imágenes Docker de 40 minutos):** Las imágenes base ya se encontraban precargadas en el nodo.
- **E8 (Ciclo completo de apagado por inactividad de 16 a 20 minutos):** La conciliación de satélites inactivos fue verificada mediante tests unitarios de ciclo de vida.
- **Pruebas de resistencia (Soak testing de varias horas):** Excluidas por exceder la ventana pactada.

***

## 8. Estado Final del Sistema

Tras concluir la campaña de pruebas, se procedió a la limpieza total de artefactos temporales:
- Contenedores de prueba eliminados (`docker rm -f`).
- Contenedores huérfanos purgados (`docker container prune -f`).
- Contenedores de servicio en ejecución: únicamente los 6 componentes nucleares de SOLV (`solv_backend`, `solv_frontend`, `solv_seaweedfs`, `solv_db`, `solv_traefik`, `solv_dev_db`).
- Ocupación final de disco: 87 GB usados, 789 GB disponibles (10 % de uso).
- Archivos crudos preservados íntegramente en `docs/CAPACITY/pilot-raw/`.
