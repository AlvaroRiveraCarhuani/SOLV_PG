  
## 1. Contexto y Origen de la Necesidad  
  
El orquestador SOLV genera dinámicamente múltiples entornos de trabajo efímeros (contenedores) que requieren ser expuestos al usuario final a través del navegador web. Asignar puertos físicos secuenciales (ej. ip:3000, ip:3001) genera una mala experiencia de usuario y expone la topología interna del servidor.  
  
Además, se necesita un enrutador capaz de actualizar sus reglas de tráfico en milisegundos sin requerir reinicios del servicio (como ocurre con Nginx tradicional), y que sea capaz de integrarse de forma transparente en la topología de red de la Universidad Adventista de Bolivia (UAB).  
  
---  
  
## 2. Decisión Arquitectónica  
  
Se implementará Traefik como Edge Router y Proxy Inverso dinámico.  
  
- El descubrimiento de servicios se realizará mediante la lectura en tiempo real del socket de Docker utilizando Labels (etiquetas de metadatos).  
- El acceso externo se gestionará a través de una estrategia de Wildcard DNS (`*.solv.uab.edu.bo`).  
- La seguridad perimetral se delegará a un Middleware de ForwardAuth acoplado a Traefik.  
  
---  
  
## 3. Justificación y Eficiencia (Primeros Principios)  
  
### Descubrimiento Dinámico (Zero-Downtime)  
  
Traefik automatiza la creación y destrucción de rutas HTTP/HTTPS al escuchar los eventos del Demonio de Docker. Esto elimina la necesidad de editar archivos de configuración manuales o reiniciar el proxy, garantizando alta disponibilidad.  
  
### Reducción de Superficie de Ataque y Anti-Enumeración  
  
Los contenedores de infraestructura interna (bases de datos, cachés) no reciben etiquetas de Traefik, quedando aislados en redes virtuales internas (TCP puro).  
  
Para las rutas web expuestas, el Middleware de autenticación valida el JWT del usuario en la "puerta del servidor". Si un usuario intenta acceder a un subdominio ajeno o inexistente, Traefik aplica una táctica de Black Hole, retornando un error genérico **404 Not Found** en lugar de **403 Forbidden**, mitigando ataques de fuerza bruta y enumeración de endpoints.  
  
### Agnosticismo de Infraestructura  
  
Al operar en la capa de contenedores, la solución de red es independiente del sistema operativo base (Ubuntu, RedHat, etc.).  
  
En un entorno de producción (Enterprise), el servidor opera de forma segura detrás del Core Switch de la institución, requiriendo únicamente la apertura de los puertos **80** y **443**, centralizando el tráfico de todas las VLANs del campus.  
  
---  
  
## 4. Consecuencias y Trade-offs (Análisis de Riesgo)  
  
### Dependencia de DNS Externo  
  
En producción, requiere coordinación con el departamento de TI institucional para configurar el registro Wildcard DNS.  
  
### Plan de Contingencia (Entornos de Prueba)  
  
Para demostraciones aisladas o caídas de red institucional, el sistema requiere la implementación de un servidor DNS local (ej. `dnsmasq`) con dominio `.local` y enrutamiento a través de **Tethering** para mantener la validación SSO activa sin depender de la red LAN física.