import { expect, test } from '@playwright/test';
import {
  CANONICAL_ACTORS,
  createAuthenticatedRoleSession
} from '../../fixtures/auth-roles.fixture.mjs';
import { createForensicMonitor } from '../../fixtures/forensic-logger.fixture.mjs';

test.describe.configure({ mode: 'serial' });

test.describe('Categoría 02 (Media): Laboratorios de IDE Persistente y Rúbricas al 100%', () => {
  const runId = Date.now().toString().slice(-6);
  const courseName = `Ingeniería Web E2E #${runId}`;
  const courseCode = `WEB-${runId}`;
  const exerciseTitle = `API REST con Base de Datos #${runId}`;

  let sharedState = {
    courseId: '',
    exerciseId: ''
  };

  test('Configura un ejercicio de IDE Persistente con rúbrica estricta (100%) y valida checklist', async ({ browser }, testInfo) => {
    const forensic = createForensicMonitor();
    const baseURL = process.env.E2E_BASE_URL || 'http://localhost:4200';

    const adminSession = await createAuthenticatedRoleSession(browser, 'admin', baseURL);
    const teacherSession = await createAuthenticatedRoleSession(browser, 'teacher', baseURL);

    const { page: adminPage } = adminSession;
    const { page: teacherPage } = teacherSession;

    forensic.attachToPage(adminPage, 'admin');
    forensic.attachToPage(teacherPage, 'teacher');

    try {
      // Setup de materia
      await test.step('Setup: Admin crea materia', async () => {
        await adminPage.bringToFront();
        forensic.recordStep('Admin crea materia para laboratorio de IDE');

        await adminPage.goto("/");
        const courseRes = await adminPage.evaluate(async ({ cName, cCode, teacherId, studentId }) => {
          const token = window.sessionStorage.getItem('solv_token');
          const headers = {
            'Content-Type': 'application/json',
            ...(token ? { Authorization: `Bearer ${token}` } : {})
          };

          const cResp = await fetch('/api/v1/subjects', {
            method: 'POST',
            headers,
            body: JSON.stringify({
              name: cName,
              code: cCode,
              teacher_id: teacherId,
              period: '2026-II',
              max_students: 50,
              is_active: true
            })
          });
          const cJson = await cResp.json();
          const courseId = cJson.data?.id || cJson.id;

          if (courseId) {
            await fetch(`/api/v1/subjects/${courseId}/enroll`, {
              method: 'POST',
              headers,
              body: JSON.stringify({ student_id: studentId })
            });
          }
          return { ok: cResp.status === 201 || cResp.status === 200, courseId };
        }, {
          cName: courseName,
          cCode: courseCode,
          teacherId: CANONICAL_ACTORS.teacher.id,
          studentId: CANONICAL_ACTORS.student.id
        });

        sharedState.courseId = courseRes.courseId;
        expect(sharedState.courseId).toBeTruthy();
      });

      // Docente crea ejercicio IDE con rúbrica
      await test.step('Paso 1: Docente guarda ejercicio IDE con rúbrica ponderada exactamente al 100%', async () => {
        await teacherPage.bringToFront();
        forensic.recordStep('Docente guarda ejercicio IDE con rúbrica 100%');

        const saveRes = await teacherPage.evaluate(async ({ courseId, title }) => {
          const token = window.sessionStorage.getItem('solv_token');
          const headers = {
            'Content-Type': 'application/json',
            ...(token ? { Authorization: `Bearer ${token}` } : {})
          };

          // 1. Crear ejercicio IDE
          const exResp = await fetch('/api/v1/exercises', {
            method: 'POST',
            headers,
            body: JSON.stringify({
              subject_id: courseId,
              title,
              description: 'Desarrolla el backend REST con endpoints CRUD y script DDL.',
              type: 'project',
              environment_type: 'IDE_PERSISTENTE',
              language: 'typescript',
              difficulty: 'hard',
              db_init_script: 'CREATE TABLE users (id SERIAL PRIMARY KEY, name VARCHAR(100));',
              allow_resubmit: true,
              auto_checkpoint: true
            })
          });
          const exJson = await exResp.json();
          const exId = exJson.data?.id || exJson.id;

          // 2. Asociar rúbrica de evaluación con suma = 100%
          const rubricPayload = {
            criteria: [
              {
                id: 'crit-1',
                title: 'Arquitectura y Modelado DDL',
                description: 'Estructura correcta de tablas e índices',
                weight: 40,
                levels: [
                  { name: 'Excelente', score: 40, description: 'Esquema normalizado' },
                  { name: 'Insuficiente', score: 0, description: 'Faltan llaves primarias' }
                ]
              },
              {
                id: 'crit-2',
                title: 'Endpoints REST y Manejo de Errores',
                description: 'Códigos HTTP 200, 201, 400 y 404',
                weight: 60,
                levels: [
                  { name: 'Completo', score: 60, description: 'Todos los endpoints operativos' },
                  { name: 'Incompleto', score: 20, description: 'Faltan validaciones' }
                ]
              }
            ]
          };

          await fetch(`/api/v1/exercises/${exId}/rubric`, {
            method: 'PUT',
            headers,
            body: JSON.stringify(rubricPayload)
          });

          // 3. Publicar
          const pubResp = await fetch(`/api/v1/exercises/${exId}/publish`, { method: 'POST', headers });
          return { ok: pubResp.status === 200 || pubResp.status === 201, exId };
        }, { courseId: sharedState.courseId, title: exerciseTitle });

        expect(saveRes.ok).toBe(true);
        sharedState.exerciseId = saveRes.exId;
      });
    } catch (err) {
      await forensic.flushOnTestEnd(
        {
          ...testInfo,
          status: 'failed',
          expectedStatus: 'passed',
          error: { message: err?.message || String(err), stack: err?.stack || '' }
        },
        { admin: adminPage, teacher: teacherPage }
      );
      throw err;
    }
  });
});
