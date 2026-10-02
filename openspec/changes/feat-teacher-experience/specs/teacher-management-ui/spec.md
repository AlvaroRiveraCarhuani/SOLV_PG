# teacher-management-ui Specification

## Purpose

Define the functional and behavioral requirements for the Teacher Web Interface in SOLV, covering exception-based monitoring, lab authoring with test cases, SpeedGrader submission evaluation, and grades export.

## Requirements

### Requirement: Exception-Based Incident Dashboard

The dashboard MUST classify teacher alerts into three severity levels: critical (OOM-Killed), warning (AST Blocked), and standard (Pending Review). It SHALL compute and display the `at_risk` indicator for students with no submissions and no heartbeat within 24 hours of the deadline.

#### Scenario: Display pending incidents grouped by severity
- GIVEN a teacher authenticated into `/teacher/dashboard` with active course enrollments
- WHEN the dashboard loads metrics and course incident streams
- THEN the system MUST render distinct counter cards for critical, warning, and pending review submissions
- AND student cards flagged as `at_risk` SHALL display a visual badge with the remaining deadline countdown

#### Scenario: Zero incidents state
- GIVEN a course where all submissions are graded and no runtime crashes occurred
- WHEN the teacher views the exception panel
- THEN the system MUST display a clear state banner indicating no pending action items

---

### Requirement: Lab and Exercise Authoring Workflow

The system MUST enforce a two-phase lifecycle for exercises (`draft` and `published`). An exercise SHALL NOT transition to `published` unless it contains at least one valid public test case.

#### Scenario: Prevent publishing exercise without public test case
- GIVEN an exercise in `draft` state with zero test cases or only hidden test cases
- WHEN the teacher clicks the "Publish" action
- THEN the UI MUST disable the action or display a validation error message
- AND the exercise state SHALL remain `draft`

#### Scenario: Bulk upload test cases via CSV
- GIVEN a teacher uploading a CSV file containing test cases
- WHEN the file contains malformed rows (e.g. missing input/output or syntax error)
- THEN the system MUST display the exact row numbers containing errors
- AND no test cases SHALL be saved until validation passes

---

### Requirement: SpeedGrader Continuous Evaluation and Unmasking

The system MUST provide a sequential evaluation interface (`/teacher/revision/:submissionId`) with pointers to previous and next submissions. Hidden test cases (`is_hidden = true`) MUST be unmasked for the teacher, and comments MUST be anchorable to specific line numbers.

#### Scenario: Unmask hidden test cases for teacher
- GIVEN an authenticated teacher viewing a submission detail
- WHEN the test case execution results load
- THEN the system MUST render inputs and expected outputs for both public and hidden test cases

#### Scenario: Add inline comment to student code
- GIVEN a teacher viewing student source code
- WHEN the teacher selects line 15 and submits a feedback note
- THEN the comment MUST be persisted with `line_number = 15`
- AND the comment card SHALL be visibly anchored to line 15 in the diff/code viewer

#### Scenario: Sequential navigation across student submissions
- WHEN the teacher presses "Next" or uses the keyboard shortcut
- THEN the system MUST navigate directly to `/teacher/revision/:next_submission_id` without returning to the index

#### Scenario: Display AST security violations
- GIVEN a submission with verdict `AST_BLOCKED` or with static analysis violations
- WHEN the teacher opens the SpeedGrader revision
- THEN the system MUST display the AST Security Violations inspector indicating the violated rule, severity, and description

---

### Requirement: Massive Environment Hibernation Control

The system MUST provide a safety confirmation before triggering mass hibernation of all active student workspaces in a course.

#### Scenario: Mass hibernate environments with confirmation
- GIVEN an active class with running student workspaces
- WHEN the teacher clicks "Pausar Entornos" and confirms in the security modal
- THEN the system MUST trigger hibernation of active containers and show memory release feedback

---

### Requirement: Export Grades Matrix and Reactive Lab KPI Filtering

The system MUST allow exporting the complete grading matrix for a subject in CSV format and dynamically calculate laboratory KPI averages and pass rates upon filter selection.

#### Scenario: Download grades CSV
- GIVEN a teacher viewing a course gradebook
- WHEN the teacher clicks "Export CSV"
- THEN the browser MUST trigger a direct download of `{course_code}_grades_{timestamp}.csv` formatted with UTF-8 BOM

#### Scenario: Reactive laboratory KPI filtering
- GIVEN a teacher viewing `/teacher/evaluaciones`
- WHEN the teacher selects a specific laboratory from the filter dropdown
- THEN the system MUST focus the matrix table and render KPI metric chips for average score, pass rate, and pending submissions
