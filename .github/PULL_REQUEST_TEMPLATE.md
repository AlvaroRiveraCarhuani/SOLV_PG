## Descripción del Cambio
<!-- Explicá brevemente el Qué, Por qué y Cómo de este cambio usando lenguaje de dominio. -->

## Tipo de Cambio
- [ ] `feat`: Nueva funcionalidad
- [ ] `fix`: Corrección de error / bug
- [ ] `docs`: Cambios en la documentación
- [ ] `style`: Formateo o lint sin cambios en lógica
- [ ] `refactor`: Refactorización de código sin cambio funcional
- [ ] `test`: Adición o corrección de pruebas
- [ ] `ci`: Cambios en la integración continua o automatización

## Checklist de Calidad
- [ ] **Protocolo de Órdenes Atómicas:** Los cambios cumplen con las decisiones cerradas de la especificación.
- [ ] **Tests agregados/actualizados:** Se agregaron pruebas unitarias o de integración que respaldan el cambio.
- [ ] **Migración Reversible:** Si incluye cambios de base de datos, existe script de migración `Up` y `Down` probados.
- [ ] **Gobernanza de Código:** 
  - [ ] 0 colores `#hex` crudos o fallbacks en SCSS de componentes.
  - [ ] Fechas renderizadas con el pipe `dateText`.
  - [ ] Datos técnicos marcados con `[machineData]` o clase `font-mono`.
  - [ ] Sin prefijos `solv-` en selectors o clases Angular.
- [ ] **Build & Pipeline en Verde:** `npm run lint:styles`, `go test ./...` y `ng build` compilan localmente sin errores.
- [ ] **ADR (si aplica):** Se registró un Architecture Decision Record si el cambio altera decisiones arquitectónicas estructurales.
