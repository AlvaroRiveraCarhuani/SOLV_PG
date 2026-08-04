# Diseño Técnico - SLICE 3: Juez Virtual Híbrido y Patrón Strategy

## 1. Patrón Strategy para Motores de Base de Datos (SOLID Open/Closed)

```
                       ┌─────────────────────────┐
                       │    DBEngineStrategy     │  (Interfaz de Dominio)
                       └────────────┬────────────┘
                                    │
         ┌──────────────────────────┼──────────────────────────┐
         ▼                          ▼                          ▼
┌──────────────────┐       ┌──────────────────┐       ┌──────────────────┐
│ StrategyPostgres │       │  StrategyMySQL   │       │ StrategyMongoDB  │
└──────────────────┘       └──────────────────┘       └──────────────────┘
```

## 2. Flujo del Juez Algorítmico (I/O + AST Security)

1. **Analizador Estático AST:** Procesa el código entregado por el estudiante. Si detecta librerías o funciones prohibidas (ej. `import os`, `sys`, `fs`, `eval`), rechaza de inmediato con veredicto `AST_VIOLATION`.
2. **Ejecución Efímera:** Levanta un contenedor en red deshabilitada (`network: none`).
3. **Evaluación de Salida:** Se inyectan los casos de prueba por `stdin` y se evalúa la salida `stdout` contra los casos de prueba esperados (`AC` o `WA`).

## 3. Flujo del Juez de Base de Datos y Dry-Run

```
[Docente crea ejercicio] ──> [Dry Run Automático]
                                   │
     ┌─────────────────────────────┴─────────────────────────────┐
     ▼                                                           ▼
 1. Levanta Contenedor Efímero                      3. Ejecuta Solución Referencia
 2. Inyecta init_script                             4. Ejecuta validation_query
                                                                 │
                                                                 ▼
                                                  5. Serializa Estado a JSON
                                                                 │
                                                                 ▼
                                                  6. Guarda expected_json en DB
```
