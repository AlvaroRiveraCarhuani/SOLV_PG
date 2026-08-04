# Verificación de Pruebas - SLICE 2

## 1. Pruebas de Persistencia de Volúmenes Nombrados
1. Instanciar contenedor con montaje en `/workspace`.
2. Crear archivo `test_script.py` en `/workspace`.
3. Detener y eliminar el contenedor.
4. Volver a instanciar el contenedor con el mismo `student_id` y `subject_id`.
* **Resultado:** El archivo `test_script.py` permanece intacto en `/workspace`.
* Veredicto: **PASS**.
