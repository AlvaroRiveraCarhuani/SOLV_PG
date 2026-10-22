# Feature: Multi-Role E2E Playwright Automation (Core Lifecycle)
Branch: main | Delivery: ask-on-risk | Runner: npx playwright test --config=playwright.e2e.config.mjs

## Specs
- **S1**: "si, quiero probbar el flujo real, ,del admin crea algo nuevo, el docente crea un entorno, el estudiante se une y tal, ahora mismo entiendo que una sola automatizacoin no da, o un solo enfoque, ya que nuestro sistema tiene ide persistente y juez, y tambien otras cosas, pero el core es el que queremos validarlo ahora mismo"
- **S2**: "una duda, se puede con esta herramienta, si hay algun fallo que se cuerte directamente? pero que loguee el error, asi tambien despues tu tienes en contexto y lo arreglamos"
- **S3**: "dale, esto en donde lo guardaremos, osea la documentacion de este test, deberiamos crear un directorio propio? ya que son test mas \"duros\" en sentido de que toma tanto el backend y frontend, y una cosa mas, cuando se entre a localhost [Image 1] [Image 2] no tendra problemas la herramienta para pasar estos pasos?"

## Tasks
- [x] T1 (S2, S3, inline): Document E2E architecture and Core Flow 01 contract in `docs/E2E/00-ARQUITECTURA_HARNESS_PLAYWRIGHT.md` and `docs/E2E/01-FLUJO_CORE_CONSTRUCTOR_A_ESTUDIANTE.md`
- [x] T2 (S1, S2, S3, inline): Create multi-role JWT auth fixture (`frontend/e2e/fixtures/auth-roles.fixture.mjs`), fail-fast forensic logger (`frontend/e2e/fixtures/forensic-logger.fixture.mjs`), and dedicated Playwright config (`frontend/playwright.e2e.config.mjs` + `package.json` script `test:e2e:flows`)
- [x] T3 (S1, S2, inline): Implement `frontend/e2e/flows/01-core-lifecycle.spec.mjs` covering Admin (Catalog + Template Studio `AMBOS` + Course & Enrollment) -> Teacher (Course detail + Exercise Editor + Test Cases + Publish) -> Student (Course Map + Submission AC) -> Teacher (SpeedGrader verification), and run against live stack

## Log
- L1 (2026-10-21): "en que andamos, ahora si podemos planear la automatizacion de test del flujo que te mencione? con playwright, hablemos" / "si, quiero probbar el flujo real, ,del admin crea algo nuevo, el docente crea un entorno, el estudiante se une y tal..." / "se puede con esta herramienta, si hay algun fallo que se cuerte directamente? pero que loguee el error, asi tambien despues tu tienes en contexto y lo arreglamos" / "dale, vamos, raelizalo papu"
- L2 (2026-10-21): Completed T1 — created `docs/E2E/00-ARQUITECTURA_HARNESS_PLAYWRIGHT.md` and `docs/E2E/01-FLUJO_CORE_CONSTRUCTOR_A_ESTUDIANTE.md`.
- L3 (2026-10-21): Completed T2 & T3 — implemented `auth-roles.fixture.mjs`, `forensic-logger.fixture.mjs`, `playwright.e2e.config.mjs`, and `01-core-lifecycle.spec.mjs`. First run stopped immediately on `course-create-modal` host visibility (`display: inline`), fixed selector to `course-create-modal .modal-dialog`, re-ran `npm run test:e2e:flows`: 1 passed (1.5m) end-to-end against live Angular + Go + PostgreSQL + Docker.
