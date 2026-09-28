# Maintenance Activation Specification

## Purpose

Potent, audited confirmation for maintenance mode with motive visibility and expiry handling.

## Requirements

### Requirement: Type-to-Confirm Phrase

The system MUST enable Confirm only when the admin types exactly `MANTENIMIENTO`.

#### Scenario: Exact phrase enables

- GIVEN motive valid and phrase `MANTENIMIENTO`
- WHEN the admin confirms
- THEN maintenance activates

#### Scenario: Phrase mismatch blocks

- GIVEN any other phrase, casing, or empty input
- WHEN the admin attempts to confirm
- THEN Confirm stays disabled with an inline mismatch error

### Requirement: Motive Length and Visibility

The system MUST require a motive of at least 10 characters and MUST display its full text on screen and in `audit_logs`.

#### Scenario: Short motive rejected

- GIVEN a motive under 10 characters
- WHEN the admin confirms
- THEN an inline error appears and the API returns 422 `maintenance_reason_invalid`

#### Scenario: Motive recorded and visible

- GIVEN a valid motive
- WHEN maintenance activates
- THEN the full text shows in the status block and an audit event stores it verbatim

### Requirement: Optional Vigencia with Warning and Auto-Off

The system MUST allow empty vigencia, MUST warn that empty means indefinite, MUST reject past dates, and MUST transition maintenance to off after expiry.

#### Scenario: Empty vigencia warns

- GIVEN no vigencia entered
- WHEN the modal is open
- THEN a warning states maintenance stays active until manual deactivation

#### Scenario: Expiry deactivates

- GIVEN active maintenance with vigencia in the past
- WHEN any request or admin view evaluates state
- THEN maintenance reads as off and the status block reflects it
