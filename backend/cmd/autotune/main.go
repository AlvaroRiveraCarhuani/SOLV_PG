package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/shirou/gopsutil/v3/mem"
)

type HardwareStats struct {
	TotalRAMMB   uint64 `json:"total_ram_mb"`
	TotalSwapMB  uint64 `json:"total_swap_mb"`
	BaselineRAM  uint64 `json:"baseline_ram_used_mb"`
	BaselineSwap uint64 `json:"baseline_swap_used_mb"`
}

type StepMetric struct {
	ContainerCount int     `json:"container_count"`
	AvailableRAMMB uint64  `json:"available_ram_mb"`
	UsedRAMMB      uint64  `json:"used_ram_mb"`
	SwapUsedMB     uint64  `json:"swap_used_mb"`
	RAMFreePct     float64 `json:"ram_free_pct"`
}

type DensityMetrics struct {
	MaxWorkspacesIdle256MB int `json:"max_workspaces_idle_256mb"`
	MaxWorkspacesPeak2GB   int `json:"max_workspaces_peak_2gb"`
}

type AutotuneReport struct {
	Timestamp            string         `json:"timestamp"`
	Hardware             HardwareStats  `json:"hardware"`
	CircuitBreakerTrigger string        `json:"circuit_breaker_trigger"`
	ThrashingSwapDeltaMB uint64         `json:"thrashing_swap_delta_mb"`
	CalculatedOOMGuardMB uint64         `json:"calculated_oom_guard_mb"`
	AllocatableRAMMB     uint64         `json:"allocatable_ram_mb"`
	TheoreticalDensity   DensityMetrics `json:"theoretical_density"`
	StepHistory          []StepMetric   `json:"step_history"`
}

const (
	SafetyFloorRAMMB   uint64 = 1000 // Freno de emergencia: no bajar de 1000MB libres
	MaxSwapDeltaMB     uint64 = 10   // Freno de emergencia: no permitir incremento de Swap > 10MB
	MaxDummyContainers int    = 50   // Techo seguro de iteración
)

func main() {
	log.Println("=========================================================")
	log.Println("SOLV Auto-Tuner & Hardware Profiler (Kubernetes Model)")
	log.Println("=========================================================")

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Fatal: failed to initialize Docker client: %v", err)
	}
	defer cli.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Manejo de interrupción por Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	createdContainers := make([]string, 0)
	defer func() {
		log.Println("\n[CleanUp] Destruyendo contenedores dummy creados en el test...")
		for _, cid := range createdContainers {
			_ = cli.ContainerRemove(context.Background(), cid, container.RemoveOptions{Force: true})
		}
		log.Println("[CleanUp] Limpieza completada exitosamente.")
	}()

	go func() {
		<-sigChan
		log.Println("\n[Warning] Interrupción detectada. Cancelando prueba y ejecutando limpieza...")
		cancel()
	}()

	// Fase 1: Línea Base del Sistema
	log.Println("\n[Fase 1] Midiendo huella en reposo del host y servicios base...")
	vInit, err := mem.VirtualMemory()
	if err != nil {
		log.Fatalf("Fatal: failed to read memory: %v", err)
	}
	sInit, err := mem.SwapMemory()
	if err != nil {
		log.Fatalf("Fatal: failed to read swap: %v", err)
	}

	hw := HardwareStats{
		TotalRAMMB:   vInit.Total / (1024 * 1024),
		TotalSwapMB:  sInit.Total / (1024 * 1024),
		BaselineRAM:  vInit.Used / (1024 * 1024),
		BaselineSwap: sInit.Used / (1024 * 1024),
	}

	log.Printf("-> RAM Total Host: %d MB", hw.TotalRAMMB)
	log.Printf("-> Swap Total Host: %d MB", hw.TotalSwapMB)
	log.Printf("-> RAM Usada en Reposo (OS + Postgres + Traefik + Docker): %d MB", hw.BaselineRAM)
	log.Printf("-> RAM Disponible en Reposo: %d MB", vInit.Available/(1024*1024))
	log.Printf("-> Swap Usado en Reposo: %d MB", hw.BaselineSwap)

	// Asegurar imagen alpine para contenedores dummy
	imageName := "alpine:latest"
	_, _, err = cli.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		log.Printf("Pulling dummy image %s...", imageName)
		out, pErr := cli.ImagePull(ctx, imageName, image.PullOptions{})
		if pErr == nil {
			_, _ = io.Copy(io.Discard, out)
			out.Close()
		}
	}

	// Fase 2 y 3: Prueba de Estrés e Inyección Progresiva con Circuit-Breaker
	log.Println("\n[Fase 2 & 3] Iniciando prueba de estrés progresiva con Circuit-Breaker...")
	history := make([]StepMetric, 0)
	circuitTrigger := "Reached max dummy container iteration count without reaching physical limit"
	var thrashingSwapDelta uint64 = 0

	for i := 1; i <= MaxDummyContainers; i++ {
		select {
		case <-ctx.Done():
			circuitTrigger = "Interrupted by user signal"
			goto FinishTest
		default:
		}

		// Crear contenedor dummy consume 128MB
		resp, cErr := cli.ContainerCreate(ctx, &container.Config{
			Image: imageName,
			Cmd:   []string{"sh", "-c", "tail -f /dev/null"},
		}, &container.HostConfig{
			Resources: container.Resources{
				Memory: 128 * 1024 * 1024,
			},
		}, nil, nil, fmt.Sprintf("solv-autotune-dummy-%d-%d", time.Now().Unix(), i))

		if cErr != nil {
			circuitTrigger = fmt.Sprintf("Docker container creation error: %v", cErr)
			break
		}

		createdContainers = append(createdContainers, resp.ID)
		if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
			circuitTrigger = fmt.Sprintf("Docker container start error: %v", err)
			break
		}

		time.Sleep(500 * time.Millisecond)

		vCurr, _ := mem.VirtualMemory()
		sCurr, _ := mem.SwapMemory()

		availMB := vCurr.Available / (1024 * 1024)
		usedMB := vCurr.Used / (1024 * 1024)
		swapUsedMB := sCurr.Used / (1024 * 1024)
		freePct := (float64(vCurr.Available) / float64(vCurr.Total)) * 100.0

		swapDelta := uint64(0)
		if swapUsedMB > hw.BaselineSwap {
			swapDelta = swapUsedMB - hw.BaselineSwap
		}

		step := StepMetric{
			ContainerCount: i,
			AvailableRAMMB: availMB,
			UsedRAMMB:      usedMB,
			SwapUsedMB:     swapUsedMB,
			RAMFreePct:     freePct,
		}
		history = append(history, step)

		log.Printf("Step %02d | Contenedores: %02d | RAM Libre: %d MB (%.1f%%) | Swap Delta: %d MB", i, i, availMB, freePct, swapDelta)

		// Evaluaciones del Circuit-Breaker
		if swapDelta >= MaxSwapDeltaMB {
			thrashingSwapDelta = swapDelta
			circuitTrigger = fmt.Sprintf("Thrashing Protection Triggered: Swap usage increased by %d MB (>= %d MB limit)", swapDelta, MaxSwapDeltaMB)
			log.Printf("[Circuit-Breaker] STOP: %s", circuitTrigger)
			break
		}

		if availMB <= SafetyFloorRAMMB {
			circuitTrigger = fmt.Sprintf("Safety Floor Triggered: Available RAM dropped to %d MB (<= %d MB limit)", availMB, SafetyFloorRAMMB)
			log.Printf("[Circuit-Breaker] STOP: %s", circuitTrigger)
			break
		}
	}

FinishTest:
	log.Println("\n=========================================================")
	log.Println("[Fase 4] Cálculo de Memoria Asignable y Densidad Teórica")
	log.Println("=========================================================")

	// OOM_GUARD_MB = SystemOverhead (Baseline) + SafetyFloor (1000MB)
	oomGuardMB := hw.BaselineRAM + SafetyFloorRAMMB
	allocatableMB := uint64(0)
	if hw.TotalRAMMB > oomGuardMB {
		allocatableMB = hw.TotalRAMMB - oomGuardMB
	}

	idleDensity := int(math.Floor(float64(allocatableMB) / 256.0))
	peakDensity := int(math.Floor(float64(allocatableMB) / 2048.0))

	report := AutotuneReport{
		Timestamp:            time.Now().Format(time.RFC3339),
		Hardware:             hw,
		CircuitBreakerTrigger: circuitTrigger,
		ThrashingSwapDeltaMB: thrashingSwapDelta,
		CalculatedOOMGuardMB: oomGuardMB,
		AllocatableRAMMB:     allocatableMB,
		TheoreticalDensity: DensityMetrics{
			MaxWorkspacesIdle256MB: idleDensity,
			MaxWorkspacesPeak2GB:   peakDensity,
		},
		StepHistory: history,
	}

	log.Printf("-> Motivo de parada: %s", report.CircuitBreakerTrigger)
	log.Printf("-> OOM_GUARD_MB Calculado (Límite de Admisión): %d MB", report.CalculatedOOMGuardMB)
	log.Printf("-> Memoria Asignable (Allocatable RAM): %d MB", report.AllocatableRAMMB)
	log.Printf("-> Densidad Reposo (256MB/alumno): Hasta %d alumnos simultáneos", report.TheoreticalDensity.MaxWorkspacesIdle256MB)
	log.Printf("-> Densidad Pico (2048MB/alumno): Hasta %d alumnos simultáneos", report.TheoreticalDensity.MaxWorkspacesPeak2GB)

	// Guardar reporte JSON para Anexo de Tesis
	reportJSON, err := json.MarshalIndent(report, "", "  ")
	if err == nil {
		_ = os.WriteFile("autotune_report.json", reportJSON, 0644)
		log.Println("\n-> Reporte de tesis guardado en: autotune_report.json")
	}

	log.Println("\n---------------------------------------------------------")
	log.Println("Línea sugerida para tu archivo .env:")
	log.Printf("OOM_GUARD_MB=%d", report.CalculatedOOMGuardMB)
	log.Println("---------------------------------------------------------")
}
