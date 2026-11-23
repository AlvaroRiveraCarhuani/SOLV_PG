package postgres

import (
	"strings"
	"testing"
)

func TestMatchesActionFilter(t *testing.T) {
	tests := []struct {
		name    string
		filter  string
		action  string
		matched bool
	}{
		// CREATE / POST
		{name: "POST verb", filter: "POST", action: "POST /api/v1/templates", matched: true},
		{name: "CREATE category matches _CREATED", filter: "CREATE", action: "TEMPLATE_COMPONENT_CREATED", matched: true},
		{name: "CREATE category matches CREATE_ prefix", filter: "CREATE", action: "CREATE_SUBJECT", matched: true},
		{name: "POST category matches _REGISTERED", filter: "POST", action: "TEMPLATE_REGISTERED", matched: true},
		{name: "CREATE category matches _EMITTED", filter: "CREATE", action: "CLASS_ADVANCE_EMITTED", matched: true},
		{name: "CREATE category matches _DUPLICATED", filter: "CREATE", action: "TEMPLATE_DUPLICATED", matched: true},
		{name: "CREATE category matches _GRANTED", filter: "CREATE", action: "EXAM_NETWORK_EGRESS_GRANTED", matched: true},
		{name: "CREATE does not match CLASS_FROZEN", filter: "CREATE", action: "CLASS_FROZEN", matched: false},
		{name: "CREATE does not match EMERGENCY_TERMINATE_ALL", filter: "CREATE", action: "EMERGENCY_TERMINATE_ALL", matched: false},

		// UPDATE / PUT
		{name: "PUT verb", filter: "PUT", action: "PUT /api/v1/branding", matched: true},
		{name: "PATCH verb", filter: "UPDATE", action: "PATCH /api/v1/config", matched: true},
		{name: "UPDATE matches CLASS_FROZEN", filter: "UPDATE", action: "CLASS_FROZEN", matched: true},
		{name: "PUT matches CLASS_UNFROZEN", filter: "PUT", action: "CLASS_UNFROZEN", matched: true},
		{name: "UPDATE matches _UPDATED", filter: "UPDATE", action: "SUBMISSION_REVIEW_UPDATED", matched: true},
		{name: "UPDATE matches _SET", filter: "UPDATE", action: "CLASS_ADVANCE_GRACE_SET", matched: true},
		{name: "UPDATE matches _CLOSED", filter: "UPDATE", action: "CLASS_ADVANCE_GRACE_CLOSED", matched: true},
		{name: "UPDATE matches _APPLIED", filter: "UPDATE", action: "CLASS_ADVANCE_APPLIED", matched: true},
		{name: "UPDATE matches _DISMISSED", filter: "UPDATE", action: "CLASS_ADVANCE_DISMISSED", matched: true},
		{name: "UPDATE matches _ROLLBACK", filter: "UPDATE", action: "CLASS_ADVANCE_ROLLBACK", matched: true},
		{name: "UPDATE matches _REVIEWED", filter: "UPDATE", action: "TEMPLATE_REVIEWED", matched: true},
		{name: "UPDATE matches _PROMOTED_TO_MODEL", filter: "UPDATE", action: "TEMPLATE_PROMOTED_TO_MODEL", matched: true},
		{name: "UPDATE matches _ENABLED", filter: "UPDATE", action: "MAINTENANCE_ENABLED", matched: true},
		{name: "UPDATE matches _DISABLED", filter: "UPDATE", action: "MAINTENANCE_DISABLED", matched: true},
		{name: "UPDATE matches _CHANGED", filter: "UPDATE", action: "EXERCISE_PURPOSE_CHANGED", matched: true},
		{name: "UPDATE matches _SELECTED", filter: "UPDATE", action: "EXERCISE_GRADE_SELECTED", matched: true},
		{name: "UPDATE matches _RECONCILED_", filter: "UPDATE", action: "PRACTICE_ENVIRONMENT_RECONCILED_EXITED", matched: true},
		{name: "UPDATE matches _REACTIVATED", filter: "UPDATE", action: "TEMPLATE_MODEL_REACTIVATED", matched: true},
		{name: "UPDATE matches _DEACTIVATED", filter: "UPDATE", action: "TEMPLATE_MODEL_DEACTIVATED", matched: true},
		{name: "UPDATE matches IMAGE_VERIFICATION_FORCE_BYPASS", filter: "UPDATE", action: "IMAGE_VERIFICATION_FORCE_BYPASS", matched: true},
		{name: "UPDATE does not match TEMPLATE_COMPONENT_CREATED", filter: "UPDATE", action: "TEMPLATE_COMPONENT_CREATED", matched: false},

		// DELETE
		{name: "DELETE verb", filter: "DELETE", action: "DELETE /api/v1/sessions/1", matched: true},
		{name: "DELETE matches _DELETED", filter: "DELETE", action: "TEMPLATE_CATEGORY_DELETED", matched: true},
		{name: "DELETE matches _REVOKED", filter: "DELETE", action: "ADMINISTRATOR_REVOKED", matched: true},
		{name: "DELETE matches _RELEASED", filter: "DELETE", action: "ORPHAN_VOLUME_RELEASED", matched: true},
		{name: "DELETE matches EMERGENCY_ prefix", filter: "DELETE", action: "EMERGENCY_TERMINATE_ALL", matched: true},
		{name: "DELETE does not match CLASS_FROZEN", filter: "DELETE", action: "CLASS_FROZEN", matched: false},

		// EMERGENCY_
		{name: "EMERGENCY_ prefix matches EMERGENCY_TERMINATE_ALL", filter: "EMERGENCY_", action: "EMERGENCY_TERMINATE_ALL", matched: true},
		{name: "EMERGENCY_ prefix matches EMERGENCY_HIBERNATE_ALL", filter: "EMERGENCY_", action: "EMERGENCY_HIBERNATE_ALL", matched: true},
		{name: "EMERGENCY_ prefix does not match CLASS_FROZEN", filter: "EMERGENCY_", action: "CLASS_FROZEN", matched: false},
		{name: "EMERGENCY_ prefix does not match TEMPLATE_COMPONENT_CREATED", filter: "EMERGENCY_", action: "TEMPLATE_COMPONENT_CREATED", matched: false},

		// Empty filter
		{name: "Empty filter matches any action", filter: "", action: "CLASS_FROZEN", matched: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := MatchesActionFilter(tc.filter, tc.action)
			if got != tc.matched {
				t.Errorf("MatchesActionFilter(%q, %q) = %v; want %v", tc.filter, tc.action, got, tc.matched)
			}
		})
	}
}

func TestAuditLogFilterWhereContainsSemanticRules(t *testing.T) {
	mustContain := []string{
		"a.action LIKE 'POST %'",
		"a.action LIKE 'CREATE_%'",
		"a.action LIKE '%_CREATED'",
		"a.action LIKE '%_FROZEN'",
		"a.action LIKE '%_DELETED'",
		"a.action LIKE 'EMERGENCY_%'",
		"IMAGE_VERIFICATION_FORCE_BYPASS",
	}

	for _, snippet := range mustContain {
		if !strings.Contains(auditLogFilterWhere, snippet) {
			t.Errorf("auditLogFilterWhere is missing expected predicate snippet: %q", snippet)
		}
	}
}
