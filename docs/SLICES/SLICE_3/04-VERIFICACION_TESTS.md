# Verificación de Pruebas - SLICE 3

## 1. Matriz de Cobertura de Lenguajes (Algorítmico + AST)
* C, C++, C#, Java, JavaScript, Python:
  * `AC` (Accepted): Salida correcta dentro del tiempo límite.
  * `WA` (Wrong Answer): Salida diferente a la esperada.
  * `TLE` (Time Limit Exceeded): Ejecución superó el límite de tiempo.
  * `AST_VIOLATION`: Intento de importación peligrosa bloqueado en fase estática.

## 2. Matriz de Cobertura de Motores de Base de Datos
* PostgreSQL, MySQL, MongoDB:
  * Dry-Run autogenera `expected_json`.
  * Evaluación de mutaciones valida el estado de la base de datos con veredicto estricto.
* Veredicto General: **PASS** (100% de la suite de integración superada).
