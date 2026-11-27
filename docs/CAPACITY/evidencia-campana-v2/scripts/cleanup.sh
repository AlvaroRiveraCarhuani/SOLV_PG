#!/usr/bin/env bash
# Limpieza de la campaña v2: contenedores, volúmenes, entornos, entregas,
# matrículas, usuarios temporales, ejercicios y materias de la campaña.
set -u

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RAW="$ROOT/raw"
ts() { date -u +%Y-%m-%dT%H:%M:%SZ; }

psql_run() {
  docker exec -i solv_db sh -lc 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -v ON_ERROR_STOP=1' "$@"
}

echo "cleanup_start_utc=$(ts)"

# 1. Contenedores de entorno de los estudiantes de campaña.
IDS=$(docker exec solv_db sh -lc 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -tA -c "SELECT container_id FROM environments WHERE student_id IN (SELECT id FROM users WHERE email LIKE '\''campaign-v2-%@test.local'\'') AND container_id IS NOT NULL"' 2>/dev/null)
for CID in $IDS; do
  [ -n "$CID" ] || continue
  docker rm -f "$CID" >/dev/null 2>&1 && echo "removed_container=$CID"
done
# Nota: no se toca ningún contenedor de entorno que no pertenezca a un
# documento de la campaña (el entorno heredado de la campaña v1 se conserva).

# 2. Volúmenes de los estudiantes de campaña.
for VOL in $(docker volume ls -q | grep -E 'c0000000-0000-0000-0002-' || true); do
  docker volume rm -f "$VOL" >/dev/null 2>&1 && echo "removed_volume=$VOL"
done

# 3. Base de datos.
psql_run <<'SQL'
BEGIN;
CREATE TEMP TABLE camp_users AS
  SELECT id FROM users WHERE email LIKE 'campaign-v2-%@test.local';
CREATE TEMP TABLE camp_exercises AS
  SELECT id FROM exercises WHERE id::text LIKE 'f0000000-%' OR id::text LIKE 'f1000000-%';
CREATE TEMP TABLE camp_subjects AS
  SELECT id FROM subjects WHERE id IN ('d0000000-0000-0000-0000-0000000000a1',
                                       'd0000000-0000-0000-0000-0000000000b1');
CREATE TEMP TABLE camp_submissions AS
  SELECT id FROM submissions WHERE student_id IN (SELECT id FROM camp_users);

DELETE FROM submission_keystroke_events WHERE submission_id IN (SELECT id FROM camp_submissions);
DELETE FROM submission_comments
 WHERE author_id IN (SELECT id FROM camp_users)
    OR submission_id IN (SELECT id FROM camp_submissions);
DELETE FROM exercise_grades
 WHERE student_id IN (SELECT id FROM camp_users)
    OR exercise_id IN (SELECT id FROM camp_exercises)
    OR subject_id IN (SELECT id FROM camp_subjects);
DELETE FROM deadline_reminders_sent
 WHERE student_id IN (SELECT id FROM camp_users)
    OR exercise_id IN (SELECT id FROM camp_exercises);
DELETE FROM evaluation_generated_cases
 WHERE student_id IN (SELECT id FROM camp_users)
    OR exercise_id IN (SELECT id FROM camp_exercises);
DELETE FROM run_metrics WHERE exercise_id IN (SELECT id FROM camp_exercises);
DELETE FROM dry_run_jobs WHERE exercise_id IN (SELECT id FROM camp_exercises);
DELETE FROM submissions
 WHERE student_id IN (SELECT id FROM camp_users)
    OR exercise_id IN (SELECT id FROM camp_exercises);
DELETE FROM environments
 WHERE student_id IN (SELECT id FROM camp_users)
    OR exercise_id IN (SELECT id FROM camp_exercises);
DELETE FROM notifications WHERE recipient_user_id IN (SELECT id FROM camp_users);
DELETE FROM enrollments
 WHERE student_id IN (SELECT id FROM camp_users)
    OR subject_id IN (SELECT id FROM camp_subjects);
DELETE FROM exercise_test_cases WHERE exercise_id IN (SELECT id FROM camp_exercises);
DELETE FROM exercise_db_init_scripts WHERE exercise_id IN (SELECT id FROM camp_exercises);
DELETE FROM exercises WHERE id IN (SELECT id FROM camp_exercises);
DELETE FROM subjects WHERE id IN (SELECT id FROM camp_subjects);
DELETE FROM users WHERE id IN (SELECT id FROM camp_users);
COMMIT;
SQL
RC=$?
echo "cleanup_sql_exit=$RC"

# 4. Verificación.
echo "--- verificacion ---"
docker exec solv_db sh -lc 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -tA -c "SELECT '\''campaign_users'\'', count(*) FROM users WHERE email LIKE '\''campaign-v2-%@test.local'\'' UNION ALL SELECT '\''campaign_subjects'\'', count(*) FROM subjects WHERE id IN ('\''d0000000-0000-0000-0000-0000000000a1'\'','\''d0000000-0000-0000-0000-0000000000b1'\'') UNION ALL SELECT '\''campaign_exercises'\'', count(*) FROM exercises WHERE subject_id IN ('\''d0000000-0000-0000-0000-0000000000a1'\'','\''d0000000-0000-0000-0000-0000000000b1'\'') UNION ALL SELECT '\''submissions'\'', count(*) FROM submissions WHERE student_id IN (SELECT id FROM users WHERE email LIKE '\''campaign-v2-%@test.local'\'');"'
echo "--- docker ps ---"
docker ps --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}' | tee "$RAW/docker_ps_after_cleanup.txt"
echo "containers_running=$(docker ps -q | wc -l) volumes=$(docker volume ls -q | wc -l)"
echo "cleanup_done_utc=$(ts)"
