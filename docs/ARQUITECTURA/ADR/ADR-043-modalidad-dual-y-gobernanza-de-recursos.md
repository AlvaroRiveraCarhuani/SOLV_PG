# ADR-043: Dual Modality and Resource Governance via Template Catalog

- **Status**: Approved
- **Date**: 2026-10-09
- **Author**: Platform Architecture Team

## Context

SOLV operates as a dual-runtime orchestration platform supporting both ephemeral judge evaluation (`JUEZ_EFIMERO`) and persistent OpenVSCode IDE environments (`IDE_PERSISTENTE`). 

Prior to this decision:
1. The `exercises` table did not explicitly differentiate the runtime environment modality (`environment_type`).
2. There was no mandatory foreign key linking exercises to approved templates in `lab_templates`.
3. The `memory_limit_mb` field was exposed in teacher API endpoints, allowing course instructors to specify arbitrary RAM allocations, which violated the administrative resource governance requirement.

## Decision

We establish strict template-driven resource governance and dual modality classification:

1. **Database Schema & Backfill Migration (`00020_dual_modality_and_template_governance.sql`)**:
   - Add `environment_type VARCHAR(50) NOT NULL DEFAULT 'JUEZ_EFIMERO'` with check `CHECK (environment_type IN ('JUEZ_EFIMERO', 'IDE_PERSISTENTE'))`.
   - Add `template_id UUID NOT NULL REFERENCES lab_templates(id) ON DELETE RESTRICT`.
   - Seed system runner default templates (`target_environment='JUEZ_EFIMERO'`, `status='approved'`, `description='SYSTEM_SEED_RUNNER'`) for each supported programming language (python, java, cpp, go, javascript, csharp), dynamically computing `base_ram_mb` using the mode (`MODE()`) of existing exercise RAM configurations (defaulting to 128 MB).
   - Backfill all existing exercises to `JUEZ_EFIMERO` linked to their respective language seed template.

2. **Domain & Application Layer Enforcement (`EvaluationService`)**:
   - On Exercise Create, Update, and Publish: `EvaluationService` resolves the approved template (explicitly via `template_id` or implicitly via `language` for judge environments).
   - Validates that `template.status == 'approved'` and `template.target_environment == exercise.environment_type`.
   - Overwrites `exercise.MemoryLimitMB` with `template.BaseRamMB` prior to persistence.
   - Attaches read-only `template` summary (`TemplateSummary`) to API responses.

3. **HTTP Delivery API Contract**:
   - If a request payload to create or update an exercise contains `memory_limit_mb`, the server rejects it with HTTP 422 (`memory_governed_by_template`).

## Status & Consequences

- **Positive**:
  - Full administrative governance over server RAM allocation per template.
  - Zero changes required in Docker execution engine or runner invocation.
  - Clean separation of concern: Admins define resource envelopes; Teachers compose academic content.
- **Negative**:
  - Exercises require approved runner templates present in `lab_templates` for all supported languages.
