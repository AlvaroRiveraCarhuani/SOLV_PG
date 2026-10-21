package docker

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/errdefs"

	"solv-backend/internal/core/domain"
)

type dockerPullMsg struct {
	Status         string `json:"status"`
	ID             string `json:"id"`
	ProgressDetail struct {
		Current int64 `json:"current"`
		Total   int64 `json:"total"`
	} `json:"progressDetail"`
}

// InspectLocal comprueba si la imagen está presente en el daemon local y retorna su tamaño y digest
func (a *EnvTestDockerAdapter) InspectLocal(ctx context.Context, imageRef string) (bool, int64, string, error) {
	inspect, _, err := a.cli.ImageInspectWithRaw(ctx, imageRef)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return false, 0, "", nil
		}
		return false, 0, "", err
	}

	localDigest := ""
	if len(inspect.RepoDigests) > 0 {
		// RepoDigests suele tener la forma "repo@sha256:..."
		parts := strings.Split(inspect.RepoDigests[0], "@")
		if len(parts) > 1 {
			localDigest = parts[1]
		} else {
			localDigest = inspect.RepoDigests[0]
		}
	} else {
		localDigest = inspect.ID
	}

	return true, inspect.Size, localDigest, nil
}

// InspectRemoteDigest consulta el registro OCI para obtener el digest del descriptor del manifiesto
func (a *EnvTestDockerAdapter) InspectRemoteDigest(ctx context.Context, imageRef string) (string, error) {
	dist, err := a.cli.DistributionInspect(ctx, imageRef, "")
	if err != nil {
		return "", err
	}
	return string(dist.Descriptor.Digest), nil
}

// PullImage realiza el pull de la imagen con deadline de inactividad (DA-02) y reporte continuo de progreso
func (a *EnvTestDockerAdapter) PullImage(ctx context.Context, imageRef string, onProgress func(doneBytes, totalBytes int64, currentLayer, totalLayers int, action string)) error {
	pullCtx, cancelPull := context.WithCancel(ctx)
	defer cancelPull()

	reader, err := a.cli.ImagePull(pullCtx, imageRef, image.PullOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()

	type layerStats struct {
		current int64
		total   int64
	}
	layers := make(map[string]*layerStats)

	lineCh := make(chan []byte)
	readErrCh := make(chan error, 1)

	// Goroutine que lee líneas JSON del stream
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

	timer := time.NewTimer(a.inactivityDeadline)
	defer timer.Stop()

	for {
		select {
		case <-pullCtx.Done():
			return pullCtx.Err()

		case <-timer.C:
			// Inactivity deadline expirado (DA-02)
			cancelPull()
			return ErrPullStalled

		case err := <-readErrCh:
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err

		case line := <-lineCh:
			var msg dockerPullMsg
			if jsonErr := json.Unmarshal(line, &msg); jsonErr != nil {
				continue
			}

			// Reseteamos el timer de inactividad con cada avance
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(a.inactivityDeadline)

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
			var completedLayers, totalLayerCount int
			totalLayerCount = len(layers)

			for _, l := range layers {
				totalDone += l.current
				totalBytes += l.total
				if l.total > 0 && l.current >= l.total {
					completedLayers++
				}
			}

			if onProgress != nil {
				action := msg.Status
				if msg.ID != "" {
					action = fmt.Sprintf("%s: %s", msg.ID, msg.Status)
				}
				onProgress(totalDone, totalBytes, completedLayers, totalLayerCount, action)
			}
		}
	}
}

// BuildLocalStackImage construye una imagen local inmutable a partir de un Dockerfile en memoria (ImageBuilderPort)
func (a *EnvTestDockerAdapter) BuildLocalStackImage(ctx context.Context, tag string, dockerfileContent string, onProgress func(action string)) error {
	var tarBuf bytes.Buffer
	tw := tar.NewWriter(&tarBuf)
	dfBytes := []byte(dockerfileContent)
	hdr := &tar.Header{
		Name: "Dockerfile",
		Mode: 0600,
		Size: int64(len(dfBytes)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("error preparando contexto tar de Dockerfile: %w", err)
	}
	if _, err := tw.Write(dfBytes); err != nil {
		return fmt.Errorf("error escribiendo Dockerfile en contexto tar: %w", err)
	}
	if err := tw.Close(); err != nil {
		return fmt.Errorf("error cerrando contexto tar: %w", err)
	}

	resp, err := a.cli.ImageBuild(ctx, &tarBuf, types.ImageBuildOptions{
		Tags:        []string{tag},
		Dockerfile:  "Dockerfile",
		Remove:      true,
		ForceRemove: true,
	})
	if err != nil {
		return fmt.Errorf("error iniciando construcción de imagen local %s: %w", tag, err)
	}
	defer func() { _ = resp.Body.Close() }()

	type buildStreamMsg struct {
		Stream string `json:"stream"`
		Error  string `json:"error"`
	}
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		var msg buildStreamMsg
		if err := json.Unmarshal(scanner.Bytes(), &msg); err == nil {
			if msg.Error != "" {
				return fmt.Errorf("error en capa de construcción: %s", strings.TrimSpace(msg.Error))
			}
			line := strings.TrimSpace(msg.Stream)
			if strings.HasPrefix(line, "Step ") && onProgress != nil {
				onProgress("Construyendo entorno: " + line)
			}
		}
	}
	return scanner.Err()
}

// PruneOrphanStackImages elimina del daemon local las imágenes solv-local/stack:* que no estén en activeTags
func (a *EnvTestDockerAdapter) PruneOrphanStackImages(ctx context.Context, activeTags map[string]bool) ([]string, error) {
	imgs, err := a.cli.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return nil, err
	}
	var pruned []string
	for _, img := range imgs {
		for _, repoTag := range img.RepoTags {
			if domain.IsBuiltStackImageTag(repoTag) && !activeTags[repoTag] {
				if _, errRm := a.cli.ImageRemove(ctx, repoTag, image.RemoveOptions{Force: false, PruneChildren: true}); errRm == nil {
					pruned = append(pruned, repoTag)
				}
			}
		}
	}
	return pruned, nil
}
