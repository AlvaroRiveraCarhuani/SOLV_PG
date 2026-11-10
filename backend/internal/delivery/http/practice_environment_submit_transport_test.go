package httpdelivery

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReadSubmitPracticeEnvironmentRequestMultipart(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("exercise_id", "11111111-1111-4111-8111-111111111111")
	part, err := writer.CreateFormFile("files", "src/main.py")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	_, _ = part.Write([]byte("print('hola')\n"))
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/environments/id/submit", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	parsed, err := readSubmitPracticeEnvironmentRequest(rec, req)
	if err != nil {
		t.Fatalf("multipart válido rechazado: %v", err)
	}
	if parsed.ExerciseID != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("exercise_id = %q", parsed.ExerciseID)
	}
	// multipart.FileHeader expone el nombre base del archivo; las rutas
	// anidadas viajan en el snapshot del volumen, no en el nombre del part.
	if !strings.Contains(parsed.Files["main.py"], "hola") {
		t.Fatalf("archivo multipart no leído: %+v", parsed.Files)
	}
}

func TestCleanSubmittedFilePathRejectsTraversal(t *testing.T) {
	for _, candidate := range []string{"", "/etc/passwd", "../secreto.py", "src/../../secreto.py"} {
		if _, err := cleanSubmittedFilePath(candidate); !errors.Is(err, errInvalidSubmitFile) {
			t.Fatalf("ruta %q debía rechazarse, devolvió %v", candidate, err)
		}
	}
	if cleaned, err := cleanSubmittedFilePath("src/main.py"); err != nil || cleaned != "src/main.py" {
		t.Fatalf("ruta válida: cleaned=%q err=%v", cleaned, err)
	}
}

func TestReadSubmitPracticeEnvironmentRequestKeepsSmallJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/environments/id/submit",
		strings.NewReader(`{"exercise_id":"11111111-1111-4111-8111-111111111111","files":{"main.py":"print(1)\n"}}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	parsed, err := readSubmitPracticeEnvironmentRequest(rec, req)
	if err != nil {
		t.Fatalf("JSON pequeño rechazado: %v", err)
	}
	if parsed.Files["main.py"] != "print(1)\n" {
		t.Fatalf("JSON no decodificado: %+v", parsed.Files)
	}
}

func TestReadSubmitPracticeEnvironmentRequestRejectsLargeJSON(t *testing.T) {
	payload := `{"exercise_id":"11111111-1111-4111-8111-111111111111","files":{"main.py":"` +
		strings.Repeat("x", 512*1024) + `"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/environments/id/submit", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	_, err := readSubmitPracticeEnvironmentRequest(rec, req)
	if !errors.Is(err, errSubmitPayloadTooLarge) {
		t.Fatalf("JSON grande devolvió %v", err)
	}
}
