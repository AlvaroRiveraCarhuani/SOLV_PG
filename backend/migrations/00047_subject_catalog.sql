-- +goose Up
-- Catálogo de materias.
--
-- Hasta ahora `subjects` cumplía dos papeles a la vez: era el catálogo (código y
-- nombre de la asignatura) y la oferta (docente, periodo e inscritos). Eso
-- impedía responder preguntas que cruzan periodos, como el histórico de un
-- alumno en una materia, y dejaba sin defensa los códigos repetidos: nada
-- impedía dos filas "SIS-211" en el mismo periodo.
--
-- Este catálogo separa la identidad de la asignatura de su dictado concreto.
-- `subjects` sigue siendo la oferta y conserva `code` y `name` como snapshot del
-- curso en su periodo: si el catálogo se renombra, los cursos anteriores
-- mantienen el nombre con el que se dictaron. La edición del catálogo propaga el
-- nombre sólo a los cursos no archivados.
--
-- `section` identifica el paralelo o comisión dentro del mismo periodo. Es
-- opcional porque hoy hay una sola oferta por materia y periodo; el índice
-- único la usa como cadena vacía para que dos cursos sin sección tampoco se
-- dupliquen.

-- El catálogo pertenece al tenant y se va con él: la baja de un tenant ya está
-- protegida por `subjects`, así que retener el catálogo solo agregaría un motivo
-- más para que esa baja falle a mitad de camino.
CREATE TABLE IF NOT EXISTS subject_catalog (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    code VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- El código identifica la materia dentro del tenant y se compara sin mayúsculas:
-- "sis-211" y "SIS-211" son la misma asignatura escrita de dos formas.
CREATE UNIQUE INDEX IF NOT EXISTS ux_subject_catalog_tenant_code
    ON subject_catalog (tenant_id, upper(code));

CREATE INDEX IF NOT EXISTS idx_subject_catalog_tenant ON subject_catalog(tenant_id);

-- Alta del catálogo a partir de lo que ya existe. Cuando un código aparece en
-- varios cursos se prefiere el curso vigente (no archivado) y, entre iguales, el
-- más reciente: es el nombre que el docente está usando hoy. El código se guarda
-- en mayúsculas porque es la forma en que se muestra en toda la interfaz.
INSERT INTO subject_catalog (tenant_id, code, name)
SELECT tenant_id, upper(code), name
FROM (
    SELECT DISTINCT ON (tenant_id, upper(code))
        tenant_id, code, name
    FROM subjects
    WHERE code <> ''
    ORDER BY tenant_id, upper(code), is_archived ASC, created_at DESC
) AS vigente
ON CONFLICT DO NOTHING;

ALTER TABLE subjects ADD COLUMN IF NOT EXISTS subject_catalog_id UUID
    REFERENCES subject_catalog(id) ON DELETE RESTRICT;

ALTER TABLE subjects ADD COLUMN IF NOT EXISTS section VARCHAR(20);

UPDATE subjects s
SET subject_catalog_id = c.id
FROM subject_catalog c
WHERE c.tenant_id = s.tenant_id
  AND upper(c.code) = upper(s.code)
  AND s.subject_catalog_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_subjects_subject_catalog ON subjects(subject_catalog_id);

-- Los duplicados se revisan después del backfill, cuando la materia ya está
-- asignada y dos cursos del mismo periodo se vuelven comparables. Si existen, la
-- migración se detiene con el detalle y la lista para corregirlos, en lugar de
-- elegir uno y perder el otro en silencio. Este es el único paso que puede
-- bloquear el despliegue, y es a propósito: el índice único no admite atajos.
-- +goose StatementBegin
DO $$
DECLARE
    duplicados INT;
BEGIN
    SELECT COUNT(*) INTO duplicados
    FROM (
        SELECT 1
        FROM subjects
        WHERE subject_catalog_id IS NOT NULL AND is_archived = FALSE
        GROUP BY tenant_id,
                 subject_catalog_id,
                 COALESCE(academic_period_id, '00000000-0000-0000-0000-000000000000'::uuid),
                 upper(COALESCE(section, ''))
        HAVING COUNT(*) > 1
    ) AS repetidos;

    IF duplicados > 0 THEN
        RAISE EXCEPTION
            'Hay % combinaciones repetidas de materia, periodo y sección en subjects. Revisalas con: SELECT tenant_id, subject_catalog_id, academic_period_id, section, COUNT(*) FROM subjects WHERE subject_catalog_id IS NOT NULL AND is_archived = FALSE GROUP BY 1,2,3,4 HAVING COUNT(*) > 1; archivá o fusioná las sobrantes antes de migrar.',
            duplicados;
    END IF;
END $$;
-- +goose StatementEnd

-- Un curso se identifica por materia, periodo y sección dentro del tenant. Los
-- cursos archivados quedan fuera para no bloquear el alta del periodo siguiente.
CREATE UNIQUE INDEX IF NOT EXISTS ux_subjects_course_identity
    ON subjects (
        tenant_id,
        subject_catalog_id,
        COALESCE(academic_period_id, '00000000-0000-0000-0000-000000000000'::uuid),
        upper(COALESCE(section, ''))
    )
    WHERE subject_catalog_id IS NOT NULL AND is_archived = FALSE;

-- +goose Down
DROP INDEX IF EXISTS ux_subjects_course_identity;
DROP INDEX IF EXISTS idx_subjects_subject_catalog;
ALTER TABLE subjects DROP COLUMN IF EXISTS section;
ALTER TABLE subjects DROP COLUMN IF EXISTS subject_catalog_id;
DROP TABLE IF EXISTS subject_catalog;
