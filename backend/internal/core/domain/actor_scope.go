package domain

import "context"

// TeacherWriteScope devuelve el docente al que hay que acotar una escritura
// sobre una materia, y si el actor puede escribir en ella.
//
//   - Rol de docente: devuelve su id. Solo escribe en las materias que dicta.
//   - Rol de administrador: devuelve "" (sin restricción dentro de la institución).
//   - Sin rol en el contexto: devuelve "" (llamadores internos, como los workers
//     y los tests, que ya resuelven su propio alcance).
//   - Cualquier otro rol, o un docente sin identidad: ok=false. No escribe nada.
func TeacherWriteScope(ctx context.Context) (teacherID string, ok bool) {
	switch GetUserRole(ctx) {
	case "admin", "":
		return "", true
	case "teacher":
		id := GetUserID(ctx)
		if id == "" {
			return "", false
		}
		return id, true
	default:
		return "", false
	}
}

// SubjectOwnershipReader responde si un docente dicta una materia. Existe para
// que los servicios que escriben sobre una materia puedan acotarse al docente
// sin depender del almacenamiento concreto.
type SubjectOwnershipReader interface {
	TeacherOwnsSubject(ctx context.Context, tenantID, subjectID, teacherID string) (bool, error)
}
