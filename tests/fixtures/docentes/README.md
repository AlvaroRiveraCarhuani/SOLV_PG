# Datos de Prueba (Fixtures): Importación de Docentes

Este directorio contiene planillas de prueba para la funcionalidad de importación masiva de docentes vía CSV y Excel (XLSX).

## Archivos disponibles
- `ejemplo_docentes.csv`: Archivo en formato de valores separados por coma con cabeceras estándar.
- `ejemplo_docentes.xlsx`: Libro de Microsoft Excel con formato tabular para verificación de parseo binario.

## Columnas soportadas
- `email` o `correo`: Dirección de correo electrónico institucional (obligatorio).
- `nombre` o `first_name`: Nombres de la persona docente (opcional).
- `apellido` o `last_name`: Apellidos de la persona docente (opcional).
- `rol` o `role`: Tipo de rol sugerido (titular, adjunto, ayudante).
