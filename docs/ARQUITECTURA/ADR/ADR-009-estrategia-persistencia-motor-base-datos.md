**Estado:** Aprobado

---

## 1. Contexto y Origen de la Necesidad

El sistema SOLV requiere un motor de base de datos robusto para almacenar entidades altamente relacionales (Docentes, Estudiantes, Materias, Laboratorios y Calificaciones), así como configuraciones dinámicas (plantillas de contenedores, límites de RAM/CPU).

Dado que la plataforma operará en una arquitectura **On-Premise** con alta concurrencia de escritura en momentos críticos (ej. 40 estudiantes enviando una evaluación simultáneamente), se necesita garantizar:

- Integridad de los datos (**ACID**)
    
- Flexibilidad en el modelo de datos
    
- Uso eficiente de los recursos del servidor anfitrión
    
---

## 2. Alternativas a Evaluar

### Alternativa Aceptada

- **Opción 3: PostgreSQL**
    
    Sistema de gestión de bases de datos relacional (RDBMS) de nivel empresarial, desplegado como un contenedor independiente dentro de la red interna de SOLV.
    

---

### Alternativas Descartadas

#### Opción 1: SQLite

- Base de datos basada en archivos locales.
    

**Motivo de descarte:**

- Riesgo crítico de bloqueos de escritura (`Database is locked`) durante picos de concurrencia.
    
- Posible pérdida de entregas o calificaciones en escenarios reales de aula.
    

---

#### Opción 2: MongoDB

- Base de datos orientada a documentos (NoSQL).
    

**Motivo de descarte:**

- Los datos educativos son inherentemente relacionales.
    
- Consultas complejas (reportes académicos) se vuelven ineficientes en un modelo documental puro.
    

---

## 3. Decisión

Se opta por **PostgreSQL** como motor principal de base de datos.

La arquitectura de persistencia se basa en un modelo híbrido optimizado para entornos con recursos limitados:

### Integridad Relacional

- Entidades nucleares (Usuarios, Roles, Materias, Laboratorios) modeladas con:
    
    - tablas tradicionales
        
    - llaves foráneas estrictas
        

Garantiza:

- consistencia de datos
    
- ausencia de registros huérfanos
    
- integridad de calificaciones
    

---

### Estrategia de Nulabilidad y Tabla Única (Single Table Design)

- Consolidación de entidades con atributos variables en tablas maestras únicas.
    
- Uso de **Null Bitmap** de PostgreSQL:
    
    - ~1 bit por campo nulo
        
    - almacenamiento eficiente
        
    - lecturas rápidas
        

Beneficio:

- Reducción de operaciones costosas (**JOINs**)
    
- Menor carga sobre CPU y RAM del servidor
    

---

### Flexibilidad Documental (JSONB)

- Uso de columnas **JSONB** para configuraciones técnicas avanzadas:
    
    - perfiles de recursos
        
    - reglas dinámicas
        
    - restricciones específicas
        

Beneficio:

- Evita migraciones destructivas
    
- Permite evolución rápida del sistema
    
- Soporte para métricas dinámicas (auto-escalado)
    

---

### Despliegue Aislado

- PostgreSQL se ejecutará en un contenedor independiente dentro de la red Docker.
    
- Comunicación exclusiva con el backend en **Go (Gin)**.
    
- Sin exposición directa de puertos al exterior.
    

---

## 4. Consecuencias

### Positivas

- **Tolerancia a la Concurrencia**
    
    Soporta múltiples escrituras simultáneas sin pérdida de datos.
    
- **Protección de Hardware (Anti-JOINs)**
    
    Diseño optimizado reduce carga en CPU y RAM.
    
- **Flexibilidad Arquitectónica**
    
    JSONB permite adaptaciones sin rediseñar el esquema.
    
- **Integración Nativa**
    
    Excelente compatibilidad con el nuevo stack tecnológico:
    
    - Go (Golang)
        
    - Framework Gin
        
    - ORMs y Query Builders modernos del ecosistema Go (sqlx)
        

---

### Negativas

- **Consumo Base de Recursos**
    
    PostgreSQL requiere memoria RAM constante para operar.
    
- **Sobrecarga Operativa**
    
    Necesidad de gestionar:
    
    - backups
        
    - volúmenes persistentes
        
    - migraciones de esquema