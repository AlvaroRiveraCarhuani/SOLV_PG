package domain

import (
	"context"
	"testing"
)

func actorCtx(role, userID string) context.Context {
	ctx := context.WithValue(context.Background(), TenantIDKey, "tenant-1")
	if role != "" {
		ctx = context.WithValue(ctx, UserRoleKey, role)
	}
	if userID != "" {
		ctx = context.WithValue(ctx, UserIDKey, userID)
	}
	return ctx
}

func TestTeacherWriteScope(t *testing.T) {
	cases := []struct {
		name      string
		ctx       context.Context
		wantID    string
		wantWrite bool
	}{
		{"docente con identidad escribe acotado a sus materias", actorCtx("teacher", "teacher-1"), "teacher-1", true},
		{"administrador no se acota", actorCtx("admin", "admin-1"), "", true},
		{"sin rol (llamador interno) no se acota", actorCtx("", ""), "", true},
		{"docente sin identidad no escribe", actorCtx("teacher", ""), "", false},
		{"estudiante no escribe", actorCtx("student", "student-1"), "", false},
		{"rol desconocido no escribe", actorCtx("auditor", "user-9"), "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			teacherID, canWrite := TeacherWriteScope(tc.ctx)
			if teacherID != tc.wantID {
				t.Fatalf("teacher id = %q, esperado %q", teacherID, tc.wantID)
			}
			if canWrite != tc.wantWrite {
				t.Fatalf("can write = %v, esperado %v", canWrite, tc.wantWrite)
			}
		})
	}
}
