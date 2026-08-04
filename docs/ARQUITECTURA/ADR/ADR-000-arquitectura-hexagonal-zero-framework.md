# ADR-000: Arquitectura Hexagonal y Stack de Servidor Nulo (Zero-Framework Go)

* **Estado:** Aceptado (Parcialmente actualizado por ADR-019)
* **Fecha:** 2026-06-01

## Contexto y Problema
El proyecto **SOLV** requiere un sistema de orquestación de laboratorios virtuales On-Premise capaz de ejecutarse con alto rendimiento en servidores con recursos acotados. La elección de frameworks HTTP pesados o librerías ORM con sobrecarga de abstracción dificulta la mantenibilidad a largo plazo, entorpece las pruebas unitarias y aumenta la huella de memoria en reposo del backend.

## Alternativas Evaluadas
1. **Framework Gin/Fiber + ORM GORM:** Ofrecen rapidez inicial de desarrollo pero introducen acoplamiento fuerte a dependencias externas, magia en reflejos y mayor consumo de memoria.
2. **Arquitectura Hexagonal (Clean) con Go Nulo (`http.ServeMux`) + `sqlx`:** Utiliza la librería estándar de Go para enrutamiento nativo con patrones de ruta claros, desacoplando totalmente la lógica de negocio de la infraestructura mediante interfaces (puertos) y repositorios SQL explícitos.

## Decisión Tomada
Se adopta la **Arquitectura Hexagonal (Ports & Adapters)** combinada con **Go nativo (`net/http.ServeMux`)** y **`sqlx`** para consultas SQL parametrizadas directas.

## Consecuencias
* **Positivas:**
  * Cero sobrecarga de memoria por frameworks de terceros.
  * Facilidad absoluta para cambiar o mockear adaptadores de infraestructura (Docker SDK, PostgreSQL, Gopsutil).
  * Control total sobre las consultas SQL y transacciones de base de datos.
* **Negativas:**
  * Se requiere escribir estructuras y mappers explícitos en la capa de entrega y almacenamiento.
