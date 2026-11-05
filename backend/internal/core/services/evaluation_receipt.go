package services

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"time"

	"solv-backend/internal/core/domain"
)

// El juez corre en el servidor, pero la entrega llega por otro request y el
// navegador decide qué veredicto envía. Este recibo es la prueba de que el
// veredicto lo emitió esta instancia: se firma al responder la evaluación y se
// verifica al registrar la entrega. No se guarda nada: el recibo viaja con el
// resultado y caduca.
const (
	evaluationReceiptVersion = "v1"
	// evaluationReceiptTTL acota la ventana entre evaluar y entregar. Una
	// evaluación vieja se repite; no se acepta un recibo vencido.
	evaluationReceiptTTL = 15 * time.Minute
	// evaluationReceiptClockSkew tolera relojes adelantados del mismo host.
	evaluationReceiptClockSkew = time.Minute
	evaluationReceiptContext   = "solv:evaluation-receipt:v1"
)

// EvaluationReceipt es el contenido firmado del resultado de evaluación.
type EvaluationReceipt struct {
	TenantID        string `json:"tenant_id"`
	ExerciseID      string `json:"exercise_id"`
	StudentID       string `json:"student_id"`
	Verdict         string `json:"verdict"`
	ExecutionTimeMS int    `json:"execution_time_ms"`
	MemoryUsedMB    int    `json:"memory_used_mb"`
	// CodeHash ata el veredicto al código evaluado: el recibo no sirve para
	// registrar otro texto con el mismo resultado.
	CodeHash string `json:"code_hash"`
	// JTI identifica esta emisión concreta. `issued_at` solo tiene precisión de
	// segundos: dos evaluaciones idénticas en el mismo segundo necesitan un
	// identificador propio para no compartir entrega.
	JTI      string `json:"jti"`
	IssuedAt int64  `json:"issued_at"`
}

// evaluationReceiptKey deriva una clave propia del secreto de sesión. No se
// reutiliza la clave del JWT: si una se rota, la otra sigue siendo válida.
func evaluationReceiptKey() []byte {
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if secret == "" {
		// Entorno de desarrollo sin secreto configurado. El recibo sigue
		// caducando y sigue atado a ejercicio y alumno.
		secret = "solv-development-evaluation-receipt"
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(evaluationReceiptContext))
	return mac.Sum(nil)
}

// SignEvaluationReceipt devuelve el token que acompaña al resultado. Cada
// llamada emite una identidad nueva, aunque repita ejercicio, alumno, código,
// veredicto y segundo de emisión.
func SignEvaluationReceipt(receipt EvaluationReceipt) (string, error) {
	if receipt.JTI == "" {
		id, err := newReceiptID()
		if err != nil {
			return "", err
		}
		receipt.JTI = id
	}
	payload, err := json.Marshal(receipt)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, evaluationReceiptKey())
	mac.Write(payload)

	return evaluationReceiptVersion + "." +
		base64.RawURLEncoding.EncodeToString(payload) + "." +
		base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

// ParseEvaluationReceipt valida la firma y la vigencia. Cualquier token
// manipulado, vencido o de otra evaluación se rechaza como ausencia de recibo.
func ParseEvaluationReceipt(token string, now time.Time) (*EvaluationReceipt, error) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 || parts[0] != evaluationReceiptVersion {
		return nil, domain.ErrInvalidEvaluationReceipt
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, domain.ErrInvalidEvaluationReceipt
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, domain.ErrInvalidEvaluationReceipt
	}

	mac := hmac.New(sha256.New, evaluationReceiptKey())
	mac.Write(payload)
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return nil, domain.ErrInvalidEvaluationReceipt
	}

	var receipt EvaluationReceipt
	if err := json.Unmarshal(payload, &receipt); err != nil {
		return nil, domain.ErrInvalidEvaluationReceipt
	}
	if receipt.ExerciseID == "" || receipt.StudentID == "" || receipt.Verdict == "" ||
		receipt.CodeHash == "" || receipt.JTI == "" {
		return nil, domain.ErrInvalidEvaluationReceipt
	}

	issuedAt := time.Unix(receipt.IssuedAt, 0)
	if now.Sub(issuedAt) > evaluationReceiptTTL || issuedAt.Sub(now) > evaluationReceiptClockSkew {
		return nil, domain.ErrInvalidEvaluationReceipt
	}

	return &receipt, nil
}

// newReceiptID genera el identificador de una emisión del recibo.
func newReceiptID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

// HashEvaluatedCode identifica el texto que el juez evaluó.
func HashEvaluatedCode(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}
