-- +goose Up
-- Migrar casos de prueba desde exercises.config a la tabla relacional exercise_test_cases
INSERT INTO exercise_test_cases (
  exercise_id,
  order_index,
  input,
  expected_output,
  visibility,
  weight
)
SELECT 
  e.id,
  (tc.ordinality - 1)::int AS order_index,
  COALESCE(tc.value->>'input', '') AS input,
  tc.value->>'expected_output' AS expected_output,
  CASE
    WHEN (tc.value->>'is_hidden')::boolean = true THEN 'hidden'
    WHEN (tc.value->>'is_sample')::boolean = true THEN 'example'
    ELSE 'public'
  END AS visibility,
  CASE
    WHEN tc.value ? 'weight' AND (tc.value->>'weight') ~ '^[0-9]+(\.[0-9]+)?$' 
      THEN (tc.value->>'weight')::double precision
    ELSE 1.0
  END AS weight
FROM exercises e,
     jsonb_array_elements(
       CASE 
         WHEN e.config->'algorithm' ? 'test_cases' AND jsonb_typeof(e.config->'algorithm'->'test_cases') = 'array' 
           THEN e.config->'algorithm'->'test_cases'
         WHEN e.config ? 'test_cases' AND jsonb_typeof(e.config->'test_cases') = 'array'
           THEN e.config->'test_cases'
         ELSE '[]'::jsonb
       END
     ) WITH ORDINALITY AS tc(value, ordinality)
WHERE tc.value->>'expected_output' IS NOT NULL AND tc.value->>'expected_output' <> ''
ON CONFLICT (exercise_id, order_index) DO UPDATE
SET input = EXCLUDED.input,
    expected_output = EXCLUDED.expected_output,
    visibility = EXCLUDED.visibility,
    weight = EXCLUDED.weight;

-- +goose Down
DELETE FROM exercise_test_cases;
