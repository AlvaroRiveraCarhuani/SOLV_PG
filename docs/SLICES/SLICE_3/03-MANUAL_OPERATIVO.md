# Manual Operativo - SLICE 3: Ejecución de Pruebas del Juez Virtual

## 1. Ejecución de Pruebas de Integración (Multilenguaje y Multimotor)
```bash
cd backend
# Ejecutar todas las pruebas de integración del Juez Virtual
go test -v ./tests/integration/judge_io_test.go ./tests/integration/judge_db_test.go ./tests/integration/ast_security_test.go
```
