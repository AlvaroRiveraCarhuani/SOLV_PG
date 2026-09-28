# backup-verify-contract Specification

## Purpose

Honest backup integrity feedback: verify reads the real envelope, trigger fails closed on bad output, sizes render correctly, and copy never promises restorability. Real `pg_dump` restore pipeline is out of scope.

## Requirements

### Requirement: Verify envelope contract

The system MUST expose verify results as `data.is_valid` with `message`, and the client MUST branch on `data.is_valid` using a typed interface. Error display MUST read the envelope `message`.

#### Scenario: Healthy file verifies OK

- GIVEN a backup file whose bytes match the stored checksum
- WHEN the user triggers verify
- THEN the UI shows the integrity-OK state from `data.is_valid=true`

#### Scenario: Backend error surfaces real message

- GIVEN verify fails server-side with envelope `message`
- WHEN the client resolves the error
- THEN it shows the envelope `message`, not a generic fallback

### Requirement: Distinct verify outcomes

The system MUST render three distinct states: integrity-OK, checksum-mismatch, and file-missing, each with its own copy.

#### Scenario: Tampered file shows mismatch copy

- GIVEN a file whose bytes differ from the stored checksum
- WHEN verify completes with `is_valid=false`
- THEN the UI shows mismatch copy distinct from missing-file copy

#### Scenario: Evicted file shows missing copy

- GIVEN a row whose file no longer exists on disk
- WHEN verify completes with `is_valid=false`
- THEN the UI shows missing-file copy, not mismatch copy

### Requirement: Verify outcome persistence

The system MUST persist every verify outcome to the execution row or an audit record; toast-only feedback is forbidden.

#### Scenario: Verify result recorded

- GIVEN verify completes for any execution
- WHEN the outcome is determined
- THEN a row or audit entry stores execution id, `is_valid`, and timestamp

#### Scenario: Reload keeps last outcome

- GIVEN a persisted verify outcome exists
- WHEN the list reloads
- THEN the row still reflects the last outcome without re-verifying

### Requirement: Sub-MB size display

The system MUST render files under 1 MB in KB or B with one decimal, and MUST NOT render any non-empty file as `0 MB`.

#### Scenario: Placeholder-size file renders in KB

- GIVEN a backup file of ~150 bytes
- WHEN the list renders its size
- THEN it shows a non-zero KB/B value, never `0 MB`

### Requirement: Honest trigger confirmation

The system MUST NOT claim verification on trigger; the trigger toast MUST state only that creation finished and integrity is unverified.

#### Scenario: Trigger toast makes no verified promise

- GIVEN a trigger completes with `success`
- WHEN the confirmation renders
- THEN it contains no "verified" claim and points to verify action

### Requirement: Fail-closed trigger

The system MUST check gzip Write and Close errors, reject outputs under 1024 bytes as `failed`, compute the stored checksum by file re-read, and mark `failed` on any violation. Success rows MUST have a complete file meeting the floor.

#### Scenario: Write error marks failed

- GIVEN gzip Write or Close returns an error during trigger
- WHEN the service finalizes the execution
- THEN it marks `failed` and never stores a success checksum

#### Scenario: Below-floor output marks failed

- GIVEN trigger output is under 1024 bytes
- WHEN the service finalizes the execution
- THEN it marks `failed` regardless of I/O errors

#### Scenario: Checksum comes from re-read

- GIVEN trigger output meets the floor with no I/O error
- WHEN the service stores the checksum
- THEN the value is computed by re-reading the finished file
