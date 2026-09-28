# Backup Config Validation Specification

## Purpose

Fail-closed validation for backup frequency and retention settings, frontend and API.

## Requirements

### Requirement: Frequency Range

The system MUST accept only integers 1-168 (inclusive) for backup frequency hours.

#### Scenario: Valid frequency accepted

- GIVEN a frequency of 6
- WHEN the admin saves backup settings
- THEN the settings persist and a green success toast appears

#### Scenario: Out-of-range or non-integer rejected

- GIVEN a frequency of 0, 169, empty, NaN, or decimal
- WHEN the value is entered or submitted
- THEN the frontend shows an inline error and the API returns 422 `backup_frequency_invalid`

### Requirement: Retention Range

The system MUST accept only integers 1-365 (inclusive) for retention days.

#### Scenario: Boundary values accepted

- GIVEN retention of 1 or 365
- WHEN the admin saves backup settings
- THEN the settings persist

#### Scenario: Out-of-range rejected

- GIVEN retention of 0, 366, empty, NaN, or decimal
- WHEN the value is entered or submitted
- THEN the frontend shows an inline error and the API returns 422 `backup_retention_invalid`

### Requirement: Inline Blocking and Save Gate

The system MUST show the validation error immediately on typing and MUST keep Save disabled while any value is invalid, and MUST NOT show a success toast for invalid input.

#### Scenario: Invalid input blocks save

- GIVEN an invalid frequency or retention value
- WHEN the admin types it
- THEN an inline error appears at once and Save is disabled

### Requirement: Retention Reduction Warning

The system MUST show a pre-save warning with the purge count of backups that would expire when the new retention is lower than the current one. The purge behavior itself MUST NOT change.

#### Scenario: Lowering retention warns

- GIVEN current retention 30 with 5 backups older than 7 days
- WHEN the admin enters 7
- THEN a warning states 5 backups would expire before saving
