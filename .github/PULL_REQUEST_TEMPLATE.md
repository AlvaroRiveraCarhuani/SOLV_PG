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
- [ ] **Build & Pipeline en Verde:** ejecuté el plan de impacto y las suites seleccionadas; `build` solo compiló el lado que el plan indicó. No ejecuté suites completas por costumbre.
- [ ] **Validación completa:** el informe tiene `validated: true` y `cli.mjs report` devolvió 0. No presenté como completa una ejecución `--partial`, un dry-run ni un plan sintético.
- [ ] **Cobertura faltante:** revisé `uncovered` y `coverageGaps`; si el plan declaró cobertura faltante, la PR la explica o la corrige.
- [ ] **ADR (si aplica):** Se registró un Architecture Decision Record si el cambio altera decisiones arquitectónicas estructurales.
