package docker_test

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/infrastructure/docker"
)

// Simula la lectura y deadline de inactividad del stream de Docker pull (C3)
func parseDockerPullStream(
	ctx context.Context,
	reader io.Reader,
	inactivityDeadline time.Duration,
	onProgress func(done, total int64, cur, tot int, action string),
) error {
	pullCtx, cancelPull := context.WithCancel(ctx)
	defer cancelPull()

	type layerStats struct {
		current int64
		total   int64
	}
	layers := make(map[string]*layerStats)

	lineCh := make(chan []byte)
	readErrCh := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Bytes()
			cp := make([]byte, len(line))
			copy(cp, line)
			select {
			case lineCh <- cp:
			case <-pullCtx.Done():
				return
			}
		}
		if scanErr := scanner.Err(); scanErr != nil {
			readErrCh <- scanErr
		} else {
			readErrCh <- io.EOF
		}
	}()

	timer := time.NewTimer(inactivityDeadline)
	defer timer.Stop()

	type pullMsg struct {
		Status         string `json:"status"`
		ID             string `json:"id"`
		ProgressDetail struct {
			Current int64 `json:"current"`
			Total   int64 `json:"total"`
		} `json:"progressDetail"`
	}

	for {
		select {
		case <-pullCtx.Done():
			return pullCtx.Err()

		case <-timer.C:
			cancelPull()
			return docker.ErrPullStalled

		case err := <-readErrCh:
			if err == io.EOF {
				return nil
			}
			return err

		case line := <-lineCh:
			var msg pullMsg
			if err := json.Unmarshal(line, &msg); err != nil {
				continue
			}

			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(inactivityDeadline)

			if msg.ID != "" {
				l, exists := layers[msg.ID]
				if !exists {
					l = &layerStats{}
					layers[msg.ID] = l
				}
				if msg.ProgressDetail.Current > 0 {
					l.current = msg.ProgressDetail.Current
				}
				if msg.ProgressDetail.Total > 0 {
					l.total = msg.ProgressDetail.Total
				}
			}

			var totalDone, totalBytes int64
			var completedLayers int
			for _, l := range layers {
				totalDone += l.current
				totalBytes += l.total
				if l.total > 0 && l.current >= l.total {
					completedLayers++
				}
			}

			if onProgress != nil {
				onProgress(totalDone, totalBytes, completedLayers, len(layers), msg.Status)
			}
		}
	}
}

func TestDockerPullParser_GoldenStream(t *testing.T) {
	// Golden stream simulando salida real de Docker Daemon
	goldenData := `{"status":"Pulling from library/python","id":"3.12-slim"}
{"status":"Pulling fs layer","id":"layer1"}
{"status":"Pulling fs layer","id":"layer2"}
{"status":"Downloading","progressDetail":{"current":500,"total":1000},"id":"layer1"}
{"status":"Downloading","progressDetail":{"current":1000,"total":1000},"id":"layer1"}
{"status":"Download complete","id":"layer1"}
{"status":"Downloading","progressDetail":{"current":2000,"total":2000},"id":"layer2"}
{"status":"Download complete","id":"layer2"}
{"status":"Digest: sha256:abcd1234"}
{"status":"Status: Downloaded newer image for python:3.12-slim"}
`

	reader := strings.NewReader(goldenData)
	var maxDone int64
	var maxLayers int

	err := parseDockerPullStream(
		context.Background(),
		reader,
		2*time.Second,
		func(done, total int64, cur, tot int, action string) {
			if done > maxDone {
				maxDone = done
			}
			if tot > maxLayers {
				maxLayers = tot
			}
		},
	)

	if err != nil {
		t.Fatalf("expected golden stream to complete without error, got: %v", err)
	}

	if maxDone != 3000 {
		t.Errorf("expected maxDone = 3000 bytes, got: %d", maxDone)
	}
	if maxLayers != 3 {
		t.Errorf("expected 3 layers detected (tag + 2 layers), got: %d", maxLayers)
	}
}

// Bloqueo / corte a la mitad de la red que dispara ErrPullStalled (DA-02, C3)
type blockingReader struct {
	data     []byte
	hasSent  bool
	blockSec time.Duration
}

func (b *blockingReader) Read(p []byte) (n int, err error) {
	if !b.hasSent {
		b.hasSent = true
		n = copy(p, b.data)
		return n, nil
	}
	// Pausa prolongada simulando corte de red
	time.Sleep(b.blockSec)
	return 0, io.EOF
}

func TestDockerPullParser_InactivityDeadlineExceeded(t *testing.T) {
	initialChunk := `{"status":"Pulling fs layer","id":"layer1"}
{"status":"Downloading","progressDetail":{"current":200,"total":1000},"id":"layer1"}
`
	// El reader envía el primer chunk y luego se congela 500ms
	br := &blockingReader{
		data:     []byte(initialChunk),
		blockSec: 500 * time.Millisecond,
	}

	// Deadline de 50ms (expira rápidamente ante inactividad)
	deadline := 50 * time.Millisecond

	err := parseDockerPullStream(context.Background(), br, deadline, nil)

	if err == nil {
		t.Fatalf("expected error due to inactivity deadline, got nil")
	}

	if !errors.Is(err, docker.ErrPullStalled) && !strings.Contains(err.Error(), domain.EnvTestErrPullStalled) {
		t.Errorf("expected ErrPullStalled, got: %v", err)
	}
}
