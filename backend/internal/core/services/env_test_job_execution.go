package services

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/mem"

	"solv-backend/internal/core/domain"
)

const (
	// openVSCodeBinaryPath es la ruta del binario del editor inyectado en las
	// imágenes compuestas de IDE persistente.
	openVSCodeBinaryPath = "/home/.openvscode-server/bin/openvscode-server"
	// openVSCodeToolName es el nombre estable con el que el reporte registra
	// la verificación del editor.
	openVSCodeToolName = "openvscode-server"
)

// isNotFoundError verifica si el error indica que la imagen o tag no existen en el registro
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") ||
		strings.Contains(msg, "manifest unknown") ||
		strings.Contains(msg, "name unknown") ||
		strings.Contains(msg, "404")
}

// isNetworkOrDNSError verifica si el error se debe a problemas de red, DNS o socket
func isNetworkOrDNSError(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "i/o timeout") ||
		strings.Contains(msg, "network is unreachable") ||
		strings.Contains(msg, "temporary failure in name resolution")
}

// executeJob maneja el ciclo de vida, semáforo y lógica de degradación de la prueba
func (s *EnvTestService) executeJob(ctx context.Context, jobID, imageRef, dockerfileContent string, tools []string, targetEnv, entrypoint, sampleInput string, timeoutMS int, baseRamMB int) {
	// Limpieza de función cancel al terminar
	defer func() {
		s.cancelMu.Lock()
		delete(s.cancelFuncs, jobID)
		s.cancelMu.Unlock()
	}()

	// Pre-chequeo: Techo estructural de RAM contra el host (negativa al inicio del job, antes de reservar recursos)
	if baseRamMB > 0 {
		totalHostMB := 8192
		if v, err := mem.VirtualMemoryWithContext(ctx); err == nil && v != nil {
			totalHostMB = int(v.Total / (1024 * 1024))
		}
		maxAllowedRAM := domain.CalculateHostMaxAllowedRAM(totalHostMB)
		if err := domain.ValidateRamAgainstHost(baseRamMB, maxAllowedRAM); err != nil {
			_ = s.repo.Fail(context.Background(), jobID, "ram_exceeds_host", fmt.Sprintf("memoria declarada (%d MB) supera el techo permitido por el host (%d MB)", baseRamMB, maxAllowedRAM))
			return
		}
	}

	// Adquisición de semáforo de concurrencia
	select {
	case s.sem <- struct{}{}:
		// Slot adquirido
		defer func() {
			<-s.sem // Invariante: liberación garantizada en cualquier camino terminal
		}()
	case <-ctx.Done():
		_ = s.repo.Cancel(context.Background(), jobID)
		return
	}

	// 1. Degradación de Digest en 3 niveles (DA-03) o Construcción Local Determinista si es una pila compuesta
	isLocal, _, localDigest, err := s.registry.InspectLocal(ctx, imageRef)
	if err != nil && ctx.Err() != nil {
		_ = s.repo.Cancel(context.Background(), jobID)
		return
	}

	digestUnverified := false
	if domain.IsBuiltStackImageTag(imageRef) {
		if !isLocal {
			if s.builder == nil || dockerfileContent == "" {
				_ = s.repo.Fail(context.Background(), jobID, domain.EnvTestErrInternal, "No se pudo construir el entorno compuesto: definición de Dockerfile no disponible")
				return
			}
			_ = s.repo.UpdateProgress(context.Background(), jobID, domain.EnvTestProgress{
				Percent:       25,
				CurrentAction: "Integrando tecnologías seleccionadas en el servidor...",
			})
			buildErr := s.builder.BuildLocalStackImage(ctx, imageRef, dockerfileContent, func(action string) {
				_ = s.repo.UpdateProgress(context.Background(), jobID, domain.EnvTestProgress{
					Percent:       50,
					CurrentAction: action,
				})
			})
			if buildErr != nil {
				if errors.Is(buildErr, context.Canceled) {
					_ = s.repo.Cancel(context.Background(), jobID)
					return
				}
				_ = s.repo.Fail(context.Background(), jobID, domain.EnvTestErrTestCrash, fmt.Sprintf("Fallo al construir las capas del entorno: %v", buildErr))
				return
			}
			isLocal = true
		}
		job, _ := s.repo.GetByID(context.Background(), jobID)
		if job != nil {
			job.IsLocalImage = true
			_ = s.repo.Save(context.Background(), job)
		}
	} else {
		job, _ := s.repo.GetByID(context.Background(), jobID)
		if job != nil {
			job.IsLocalImage = isLocal
			_ = s.repo.Save(context.Background(), job)
		}

		remoteDigest, distErr := s.registry.InspectRemoteDigest(ctx, imageRef)

		needPull := false

		if distErr != nil {
			if isLocal {
				// Nivel B: Fallo de red/registro pero existe imagen local -> proceder con advertencia en modo degradado
				digestUnverified = true
				needPull = false
			} else {
				// Nivel C: Fallo de red/registro y no está local -> Error terminal taxativo
				if isNotFoundError(distErr) {
					_ = s.repo.Fail(context.Background(), jobID, domain.EnvTestErrNotFoundRemote, "Error 404: La imagen o el tag no existen en el registro remoto")
				} else if isNetworkOrDNSError(distErr) {
					_ = s.repo.Fail(context.Background(), jobID, domain.EnvTestErrNetworkFailure, "Fallo de red o DNS: no es posible alcanzar el registro remoto y la imagen no está en disco")
				} else {
					_ = s.repo.Fail(context.Background(), jobID, domain.EnvTestErrRegistryUnreach, fmt.Sprintf("No es posible acceder al registro de imágenes para verificar el manifiesto: %v", distErr))
				}
				return
			}
		} else {
			// Nivel A: Registro accesible
			if !isLocal {
				needPull = true
			} else if localDigest != "" && remoteDigest != "" && localDigest != remoteDigest {
				needPull = true
			}
		}

		// 2. Descarga si es requerida
		if needPull {
			_ = s.repo.UpdateProgress(context.Background(), jobID, domain.EnvTestProgress{
				CurrentAction: "Iniciando descarga de capas...",
			})

			pullErr := s.registry.PullImage(ctx, imageRef, func(doneBytes, totalBytes int64, currentLayer, totalLayers int, action string) {
				var pct float64
				if totalBytes > 0 {
					pct = float64(doneBytes) / float64(totalBytes) * 100
					if pct > 100 {
						pct = 100
					}
				}
				_ = s.repo.UpdateProgress(context.Background(), jobID, domain.EnvTestProgress{
					BytesDone:     doneBytes,
					BytesTotal:    totalBytes,
					LayerCurrent:  currentLayer,
					LayersTotal:   totalLayers,
					Percent:       pct,
					CurrentAction: action,
				})
			})

			if pullErr != nil {
				if errors.Is(pullErr, context.Canceled) {
					_ = s.repo.Cancel(context.Background(), jobID)
					return
				}
				if strings.Contains(pullErr.Error(), domain.EnvTestErrPullStalled) {
					_ = s.repo.Fail(context.Background(), jobID, domain.EnvTestErrPullStalled, "Descarga detenida: sin actividad de red por más de 60 segundos")
					return
				}
				_ = s.repo.Fail(context.Background(), jobID, domain.EnvTestErrPullTimeout, fmt.Sprintf("Error durante la descarga de la imagen: %v", pullErr))
				return
			}
		}
	}

	// 3. Ejecución de Smoke Test en Contenedor Efímero Aislado
	_ = s.repo.UpdateProgress(context.Background(), jobID, domain.EnvTestProgress{
		CurrentAction: "Ejecutando smoke test en contenedor efímero...",
	})

	memLimitMB := s.config.MemoryLimitMB
	if baseRamMB > 0 {
		memLimitMB = int64(baseRamMB)
	}

	if targetEnv == "JUEZ_EFIMERO" {
		var toolResults []domain.ToolResult
		if len(tools) > 0 {
			tRes, tExit, tErr := s.runner.RunSmokeTest(ctx, imageRef, tools, memLimitMB)
			if tErr == nil {
				toolResults = append(toolResults, tRes...)
				for _, tr := range tRes {
					if !tr.Present || tExit != 0 {
						testResult := &domain.EnvTestResult{
							Tools:      toolResults,
							ExitCode:   tExit,
							DurationMs: 0,
						}
						_ = s.repo.Fail(context.Background(), jobID, domain.EnvTestErrTestCrash, "Una o más herramientas requeridas no se encuentran instaladas en la imagen")
						job, _ := s.repo.GetByID(context.Background(), jobID)
						if job != nil {
							job.Result = testResult
							job.DigestUnverified = digestUnverified
							_ = s.repo.Save(context.Background(), job)
						}
						return
					}
				}
			}
		}

		output, durationMs, exitCode, runnerErr := s.runner.RunJudgeSmokeTest(ctx, imageRef, entrypoint, sampleInput, timeoutMS, memLimitMB)
		if runnerErr != nil {
			if errors.Is(runnerErr, context.Canceled) {
				_ = s.repo.Cancel(context.Background(), jobID)
				return
			}
			if strings.Contains(runnerErr.Error(), "timeout") || exitCode == 124 {
				_ = s.repo.Fail(context.Background(), jobID, domain.EnvTestErrTestCrash, fmt.Sprintf("Tiempo límite de ejecución excedido (%d ms)", timeoutMS))
				return
			}
			_ = s.repo.Fail(context.Background(), jobID, domain.EnvTestErrTestCrash, fmt.Sprintf("Fallo en ejecución de juez: %v", runnerErr))
			return
		}

		toolResults = append(toolResults, domain.ToolResult{Name: "entrypoint", Present: exitCode == 0, Path: entrypoint, Version: output})
		testResult := &domain.EnvTestResult{
			Tools:      toolResults,
			ExitCode:   exitCode,
			DurationMs: durationMs,
		}

		if exitCode != 0 {
			_ = s.repo.Fail(context.Background(), jobID, domain.EnvTestErrTestCrash, fmt.Sprintf("El comando de juez finalizó con código de salida %d: %s", exitCode, output))
			job, _ := s.repo.GetByID(context.Background(), jobID)
			if job != nil {
				job.Result = testResult
				job.DigestUnverified = digestUnverified
				_ = s.repo.Save(context.Background(), job)
			}
			return
		}

		_ = s.repo.Complete(context.Background(), jobID, testResult)
		if digestUnverified {
			job, _ := s.repo.GetByID(context.Background(), jobID)
			if job != nil {
				job.Result = testResult
				job.DigestUnverified = true
				_ = s.repo.Save(context.Background(), job)
			}
		}
		return
	}

	// Para entornos de IDE persistente el editor forma parte del contrato de la
	// imagen: se verifica su presencia junto a las herramientas pedidas.
	smokeTools := tools
	if targetEnv == domain.TargetEnvironmentIDEPersistente || targetEnv == domain.TargetEnvironmentAmbos {
		smokeTools = append(append([]string{}, tools...), openVSCodeBinaryPath)
	}

	startTime := time.Now()
	results, exitCode, runnerErr := s.runner.RunSmokeTest(ctx, imageRef, smokeTools, memLimitMB)
	durationMs := time.Since(startTime).Milliseconds()

	if runnerErr != nil {
		if errors.Is(runnerErr, context.Canceled) {
			_ = s.repo.Cancel(context.Background(), jobID)
			return
		}
		if strings.Contains(runnerErr.Error(), domain.EnvTestErrTestOOM) || exitCode == 137 {
			_ = s.repo.Fail(context.Background(), jobID, domain.EnvTestErrTestOOM, "El contenedor superó el límite de memoria asignado durante la prueba")
			return
		}
		_ = s.repo.Fail(context.Background(), jobID, domain.EnvTestErrTestCrash, fmt.Sprintf("Fallo en la ejecución del contenedor de prueba: %v", runnerErr))
		return
	}

	// El reporte registra el editor bajo su nombre corto, no la ruta completa.
	for i := range results {
		if results[i].Name == openVSCodeBinaryPath || results[i].Path == openVSCodeBinaryPath {
			results[i].Name = openVSCodeToolName
		}
	}

	testResult := &domain.EnvTestResult{
		Tools:      results,
		ExitCode:   exitCode,
		DurationMs: durationMs,
	}

	// Si alguna herramienta requerida faltó o el exit code fue distinto de cero, marcamos el job como fallido
	hasMissing := false
	for _, res := range results {
		if !res.Present {
			hasMissing = true
			break
		}
	}

	if hasMissing || exitCode != 0 {
		_ = s.repo.Fail(context.Background(), jobID, domain.EnvTestErrTestCrash, "Una o más herramientas requeridas no se encuentran instaladas en la imagen")
		// Adosar el resultado para que el frontend pueda mostrar el desglose por herramienta
		job, _ := s.repo.GetByID(context.Background(), jobID)
		if job != nil {
			job.Result = testResult
			job.DigestUnverified = digestUnverified
			_ = s.repo.Save(context.Background(), job)
		}
		return
	}

	// Si el entorno es AMBOS y se especificó comando de evaluador, ejecutar también la prueba de humo de juez
	if targetEnv == domain.TargetEnvironmentAmbos && strings.TrimSpace(entrypoint) != "" {
		_ = s.repo.UpdateProgress(context.Background(), jobID, domain.EnvTestProgress{
			CurrentAction: "Ejecutando prueba de comando de evaluación para entorno dual...",
		})

		judgeOutput, judgeDurationMs, judgeExitCode, judgeErr := s.runner.RunJudgeSmokeTest(ctx, imageRef, entrypoint, sampleInput, timeoutMS, memLimitMB)
		if judgeErr != nil {
			if errors.Is(judgeErr, context.Canceled) {
				_ = s.repo.Cancel(context.Background(), jobID)
				return
			}
			if strings.Contains(judgeErr.Error(), "timeout") || judgeExitCode == 124 {
				_ = s.repo.Fail(context.Background(), jobID, domain.EnvTestErrTestCrash, fmt.Sprintf("Tiempo límite de ejecución excedido (%d ms)", timeoutMS))
				return
			}
			_ = s.repo.Fail(context.Background(), jobID, domain.EnvTestErrTestCrash, fmt.Sprintf("Fallo en ejecución de juez: %v", judgeErr))
			return
		}

		testResult.Tools = append(testResult.Tools, domain.ToolResult{
			Name:    "entrypoint",
			Present: judgeExitCode == 0,
			Path:    entrypoint,
			Version: judgeOutput,
		})
		testResult.DurationMs += judgeDurationMs

		if judgeExitCode != 0 {
			testResult.ExitCode = judgeExitCode
			_ = s.repo.Fail(context.Background(), jobID, domain.EnvTestErrTestCrash, fmt.Sprintf("El comando de juez finalizó con código de salida %d: %s", judgeExitCode, judgeOutput))
			job, _ := s.repo.GetByID(context.Background(), jobID)
			if job != nil {
				job.Result = testResult
				job.DigestUnverified = digestUnverified
				_ = s.repo.Save(context.Background(), job)
			}
			return
		}
	}

	// Éxito completo
	_ = s.repo.Complete(context.Background(), jobID, testResult)
	if digestUnverified {
		job, _ := s.repo.GetByID(context.Background(), jobID)
		if job != nil {
			job.DigestUnverified = true
			_ = s.repo.Save(context.Background(), job)
		}
	}
}
