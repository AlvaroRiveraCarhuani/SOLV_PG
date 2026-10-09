#!/bin/bash
set -euo pipefail

echo "=== Smoke Test: IDE Persistente ==="

# Variables de entorno opcionales con defaults
TEACHER_TOKEN="${TEACHER_TOKEN:-mock-teacher-token}"
STUDENT_TOKEN="${STUDENT_TOKEN:-mock-student-token}"
TEMPLATE_ID="${TEMPLATE_ID:-tpl-fastapi-pg}"
API_URL="${API_URL:-http://localhost:8080/api/v1}"

# 1. Crear ejercicio IDE Persistente vía API
echo "1. Creando ejercicio IDE Persistente..."
EXERCISE_RESP=$(curl -s -X POST "${API_URL}/exercises" \
  -H "Authorization: Bearer ${TEACHER_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Lab IDE: API REST con FastAPI",
    "description": "Construir una API REST con FastAPI y PostgreSQL",
    "environment_type": "IDE_PERSISTENTE",
    "template_id": "'"${TEMPLATE_ID}"'",
    "due_date": "2026-12-31T23:59:59Z"
  }')

EXERCISE_ID=$(echo "$EXERCISE_RESP" | jq -r '.id // .data.id')
echo "Ejercicio creado: ${EXERCISE_ID}"

# 2. Crear rúbrica
echo "2. Creando rúbrica..."
curl -s -X PUT "${API_URL}/exercises/${EXERCISE_ID}/rubric" \
  -H "Authorization: Bearer ${TEACHER_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "criteria": [
      {
        "name": "Endpoints REST",
        "description": "Implementación correcta de GET, POST, PUT, DELETE",
        "weight": 50,
        "levels": [
          {"name": "Excelente", "score": 100},
          {"name": "Bueno", "score": 75},
          {"name": "Regular", "score": 50},
          {"name": "Deficiente", "score": 25}
        ]
      },
      {
        "name": "Documentación",
        "description": "README claro y ejemplos de uso",
        "weight": 50,
        "levels": [
          {"name": "Excelente", "score": 100},
          {"name": "Bueno", "score": 75},
          {"name": "Regular", "score": 50},
          {"name": "Deficiente", "score": 25}
        ]
      }
    ]
  }'

echo "Rúbrica creada"

# 3. Publicar ejercicio
echo "3. Publicando ejercicio..."
curl -s -X PUT "${API_URL}/exercises/${EXERCISE_ID}/publish" \
  -H "Authorization: Bearer ${TEACHER_TOKEN}"

echo "Ejercicio publicado"

# 4. Como estudiante: abrir workspace y entregar
echo "4. Simulando entrega de estudiante..."
SUBMISSION_RESP=$(curl -s -X POST "${API_URL}/submissions" \
  -H "Authorization: Bearer ${STUDENT_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "exercise_id": "'"${EXERCISE_ID}"'",
    "code": "# Código del estudiante\nfrom fastapi import FastAPI\napp = FastAPI()\n\n@app.get(\"/\")\ndef read_root():\n    return {\"message\": \"Hello World\"}"
  }')

SUBMISSION_ID=$(echo "$SUBMISSION_RESP" | jq -r '.id // .data.id')
echo "Entrega creada: ${SUBMISSION_ID}"

# 5. Como docente: calificar con rúbrica
echo "5. Calificando con rúbrica..."
curl -s -X POST "${API_URL}/submissions/${SUBMISSION_ID}/rubric-evaluation" \
  -H "Authorization: Bearer ${TEACHER_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "selections": {
      "criterion-1": "Excelente",
      "criterion-2": "Bueno"
    },
    "comments": "Buen trabajo en general"
  }'

echo "Calificación guardada"

# 6. Verificar score calculado
echo "6. Verificando score..."
SCORE=$(curl -s "${API_URL}/submissions/${SUBMISSION_ID}" \
  -H "Authorization: Bearer ${TEACHER_TOKEN}" | jq -r '.score // .data.score')

echo "Score calculado: ${SCORE}"

if [ "${SCORE}" = "87.5" ]; then
  echo "✓ Smoke test exitoso: score correcto (50% * 100 + 50% * 75 = 87.5)"
else
  echo "✗ Smoke test falló: score incorrecto (esperado 87.5, obtenido ${SCORE})"
  exit 1
fi
