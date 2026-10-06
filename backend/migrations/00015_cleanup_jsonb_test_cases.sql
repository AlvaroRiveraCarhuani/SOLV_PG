-- +goose Up
-- Eliminar la clave test_cases de exercises.config ya que reside en exercise_test_cases
UPDATE exercises
SET config = config #- '{algorithm,test_cases}'
WHERE config->'algorithm' ? 'test_cases';

UPDATE exercises
SET config = config - 'test_cases'
WHERE config ? 'test_cases';

-- +goose Down
-- Restaurar test_cases dentro de exercises.config a partir de la tabla exercise_test_cases
WITH tc_agg AS (
  SELECT 
    exercise_id,
    jsonb_agg(
      jsonb_build_object(
        'input', input,
        'expected_output', expected_output,
        'is_hidden', (visibility = 'hidden'),
        'is_sample', (visibility = 'example'),
        'visibility', visibility,
        'weight', weight
      ) ORDER BY order_index
    ) AS test_cases_json
  FROM exercise_test_cases
  GROUP BY exercise_id
)
UPDATE exercises e
SET config = CASE
  WHEN e.type = 'algorithm' OR e.config ? 'algorithm' THEN
    jsonb_set(
      COALESCE(e.config, '{}'::jsonb),
      '{algorithm,test_cases}',
      COALESCE(t.test_cases_json, '[]'::jsonb),
      true
    )
  ELSE
    jsonb_set(
      COALESCE(e.config, '{}'::jsonb),
      '{test_cases}',
      COALESCE(t.test_cases_json, '[]'::jsonb),
      true
    )
END
FROM tc_agg t
WHERE e.id = t.exercise_id;
