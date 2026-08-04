### El Plan de Acción (Nuestros siguientes pasos lógicos)

Para terminar el flujo principal de asignación y encendido de laboratorios, propongo que sigamos exactamente este orden de construcción:

**Paso 1: El Contrato de Base de Datos (Dominio)** Crear las estructuras (`structs`) en Go que representan a `LabInstance` y la interfaz del repositorio (`LabInstanceRepository`). Esto define los métodos que usaremos (ej. `CreateInstance`, `FindByUserAndTemplate`, `UpdateStatus`).

**Paso 2: El Adaptador de Base de Datos (Infraestructura)** Implementar esos métodos usando `sqlx` para hablar con PostgreSQL. Aquí escribiremos los `INSERT` y los `SELECT` que escribirán en la tabla que migramos hace unos momentos.

**Paso 3: El Orquestador Lógico (Capa de Aplicación - `LabService`)** Aquí programaremos la función principal `StartLab(userID, templateID)`. Esta función será la "magia" que:

1. Va al Repositorio (Paso 2) a ver si el alumno ya tiene ese laboratorio.
    
2. Calcula/Lee el límite de RAM que definimos que tendrá ese template.
    
3. Llama al adaptador de Docker (`client.go`) para levantarlo.
    
4. Si Docker dice "OK", guarda el `container_id` en PostgreSQL con estado `active`.
    

**Paso 4: El Control de Recursos (Worker Pool)** Una vez que el flujo principal de asignar estudiantes a contenedores funcione, programaremos el _Worker Pool_ con Goroutines que hará el _Dry-Run_ de los templates nuevos para establecer ese límite de RAM base del que tanto hablamos.