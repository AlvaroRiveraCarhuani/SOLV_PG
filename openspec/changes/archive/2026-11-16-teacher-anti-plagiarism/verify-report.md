```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: working-tree-2026-11-16 (base 1a6cf8d)
verdict: superseded
superseded_by: ADR-062 (2026-11-02)
blockers: 0
critical_findings: 0
requirements: 0/4 vigentes; 4/4 retirados o reubicados
scenarios: 0/0; la spec original no define escenarios Given/When/Then
test_command: cd frontend && npx ng test --watch=false --include='**/teacher/supervision/anti-plagio/teacher-plagiarism.component.spec.ts'
test_exit_code: 0
test_result: 10 tests passed
test_output_hash: sha256:5ec539910983a7d87085f8d040691bf6cf4bc21a38e5835133bc39737fa552b1
lint_command: cd frontend && npm run lint:styles
lint_exit_code: 0
lint_output_hash: sha256:3134c732ffaa924c4ce09faa65e42fd9b30b3a85572877c65e626fce0e83ef64
```

## Verification Report

**Change**: teacher-anti-plagiarism
**Mode**: verificación de cambio superado
**Fecha**: 2026-11-16

### Resultado

El cambio implementó originalmente un centro global de supervisión en `/teacher/anti-plagio`. El ADR-062, del 2026-11-02, retiró ese listado global y trasladó la auditoría a la ficha de cada práctica:

- `/teacher/anti-plagio` redirige a `/teacher/por-calificar`.
- La sección `SUPERVISIÓN` sale del sidebar.
- La ruta vigente es `/teacher/cursos/:courseId/practicas/:exerciseId/similitud`.
- El componente `TeacherPlagiarismComponent` sigue existiendo, pero opera con materia y práctica fijadas por la ruta.

Por eso el cambio no se cierra como PASS contra su especificación original: la especificación fue reemplazada por una decisión arquitectónica posterior. Se archiva como SUPERADO, no como cumplido.

### Evidencia del componente vigente

La suite del componente pasa con 10 pruebas:
```text
escanea la práctica que llega por la ruta, sin elegir materia
calcula los indicadores del reporte
filtra los pares por nivel de riesgo
filtra los pares por nombre
cambia el par seleccionado
muestra los tramos compartidos con archivo y líneas
carga el par y arma el árbol de archivos afectados con la base descartada
resalta en cada lado solo el tramo del archivo elegido
resume la cadencia de escritura de cada entrega
traduce los veredictos del juez a texto para el docente
```

El gate de estilos pasa sin hallazgos.

### Correspondencia con la especificación original

| Requisito original | Estado actual | Tratamiento |
|---|---|---|
| R1: vista global en `/teacher/anti-plagio` | Retirada; la ruta redirige y la vista vive en la práctica | Superado por ADR-062 |
| R2: KPIs de integridad y riesgo | Implementado en el componente de práctica | Reubicado |
| R3: explorador y comparador lado a lado | Implementado en el componente de práctica | Reubicado |
| R4: estándares de diseño y pruebas | La suite y el gate de estilos pasan | Vigente |

### Issues Found

**CRITICAL**: ninguno.

**WARNING**:

- W1: La spec original describe una ruta y un alcance que ya no existen. No debe copiarse a `openspec/specs/` porque introduciría un contrato obsoleto.
- W2: La suite cubre el componente de práctica, no la redirección de `/teacher/anti-plagio` ni la ausencia de la sección `SUPERVISIÓN` en el sidebar. Esas conductas se verificaron por código y por el ADR-062.
- W3: No se ejecutó un flujo E2E de auditoría desde la ficha de práctica en esta sesión.

### Verdict

**SUPERSEDED**

El cambio queda cerrado como historia. Su reemplazo vigente es el ADR-062 y la auditoría por práctica. El spec histórico se conserva dentro del archivo, pero no se fusiona con las especificaciones actuales.
