package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"solv-backend/internal/core/domain"
)

// DependencyPreparationWorker materializa las dependencias de una versión del
// proyecto una sola vez por clave, con un solo turno por nodo, techos de tiempo
// y de disco, y poda por referencia.
type DependencyPreparationWorker struct {
	runner domain.DependencyPreparationRunnerPort
	root   string
	limits domain.DependencyPreparationLimits
	slot   chan struct{}
	now    func() time.Time
}

// NewDependencyPreparationWorker valida la raíz y los techos antes de aceptar trabajo.
func NewDependencyPreparationWorker(
	runner domain.DependencyPreparationRunnerPort,
	root string,
	limits domain.DependencyPreparationLimits,
) (*DependencyPreparationWorker, error) {
	if runner == nil {
		return nil, errors.New("el worker de preparación necesita un runner")
	}
	cleanRoot := filepath.Clean(strings.TrimSpace(root))
	if !filepath.IsAbs(cleanRoot) || cleanRoot == "/" {
		return nil, fmt.Errorf("%w: la raíz debe ser una ruta absoluta distinta de /", domain.ErrInvalidDependencyPreparation)
	}
	resolved := limits.WithDefaults()
	if err := resolved.Validate(); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(cleanRoot, 0o750); err != nil {
		return nil, fmt.Errorf("no se pudo crear la raíz de preparados: %w", err)
	}
	return &DependencyPreparationWorker{
		runner: runner,
		root:   cleanRoot,
		limits: resolved,
		slot:   make(chan struct{}, 1),
		now:    time.Now,
	}, nil
}

// Root es el directorio donde viven los árboles preparados.
func (w *DependencyPreparationWorker) Root() string { return w.root }

// Limits devuelve los techos efectivos.
func (w *DependencyPreparationWorker) Limits() domain.DependencyPreparationLimits { return w.limits }

// Prepare instala, comprueba sin red y publica el árbol. Si la clave ya existe,
// no vuelve a preparar: devuelve el árbol existente marcado como reutilizado.
func (w *DependencyPreparationWorker) Prepare(
	ctx context.Context,
	req domain.DependencyPreparationRequest,
) (domain.DependencyPreparationResult, error) {
	result := domain.DependencyPreparationResult{}
	if err := req.Validate(); err != nil {
		return result, err
	}

	key := req.Key()
	dir := filepath.Join(w.root, key)
	result.Key = key
	result.Dir = dir

	if size, ok := preparedTree(dir); ok {
		return w.reusedResult(result, size), nil
	}

	select {
	case w.slot <- struct{}{}:
	case <-ctx.Done():
		return result, fmt.Errorf("esperando el turno de preparación: %w", ctx.Err())
	}
	defer func() { <-w.slot }()

	if size, ok := preparedTree(dir); ok {
		return w.reusedResult(result, size), nil
	}

	free, err := freeBytes(w.root)
	if err != nil {
		return w.failed(result, "", fmt.Errorf("no se pudo leer el espacio libre: %w", err))
	}
	if free < w.limits.MinFreeBytes {
		return w.failed(result, "", fmt.Errorf(
			"espacio insuficiente para preparar: quedan %d MB y el mínimo es %d MB",
			free/(1<<20), w.limits.MinFreeBytes/(1<<20)))
	}

	logPath, logFile, err := w.openLog(key)
	if err != nil {
		return w.failed(result, "", err)
	}
	defer func() { _ = logFile.Close() }()
	result.LogPath = logPath

	startedAt := w.now()
	fmt.Fprintf(logFile, "preparación %s\nimagen: %s\nversión: %s\nperfil: %d\nproyecto: %s\n\n",
		key, req.BaseImage, req.VersionID, req.ProfileVersion, req.ProjectPath)

	runCtx, cancel := context.WithTimeout(ctx, w.limits.Timeout)
	defer cancel()

	tmpDir, attempts, err := w.installWithRetry(runCtx, req, key, logFile)
	result.Attempts = attempts
	if err != nil {
		return w.failed(result, logPath, err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()
	if err := w.runner.RunVerify(runCtx, req, tmpDir, logFile); err != nil {
		_ = removePreparedTree(tmpDir)
		return w.failed(result, logPath, fmt.Errorf("la comprobación sin red falló: %w", err))
	}

	size, err := dirBytes(tmpDir)
	if err != nil {
		_ = removePreparedTree(tmpDir)
		return w.failed(result, logPath, fmt.Errorf("no se pudo medir el árbol preparado: %w", err))
	}
	if size > w.limits.MaxBytes {
		_ = removePreparedTree(tmpDir)
		return w.failed(result, logPath, fmt.Errorf(
			"el árbol ocupa %d MB y supera el techo de %d MB",
			size/(1<<20), w.limits.MaxBytes/(1<<20)))
	}

	if err := os.Rename(tmpDir, dir); err != nil {
		if existing, ok := preparedTree(dir); ok {
			return w.reusedResult(result, existing), nil
		}
		return w.failed(result, logPath, fmt.Errorf("no se pudo publicar el árbol: %w", err))
	}

	result.Status = domain.DependencyPreparationPrepared
	result.Bytes = size
	result.Duration = w.now().Sub(startedAt)
	result.Message = "árbol preparado y comprobado sin red"
	fmt.Fprintf(logFile, "\nresultado: preparado, %d bytes en %s\n", size, result.Duration.Round(time.Millisecond))
	return result, nil
}

// installWithRetry repite la instalación solo cuando el fallo es transitorio.
// Cada intento arranca sobre un directorio limpio y todos comparten el techo de
// tiempo de la preparación, así que un reintento no alarga el plazo total.
func (w *DependencyPreparationWorker) installWithRetry(
	ctx context.Context,
	req domain.DependencyPreparationRequest,
	key string,
	logWriter io.Writer,
) (string, int, error) {
	attempts := 0
	for {
		attempts++
		dir, err := os.MkdirTemp(w.root, ".tmp-"+key+"-")
		if err != nil {
			return "", attempts, fmt.Errorf("no se pudo crear el directorio temporal: %w", err)
		}
		owner, err := preparationOwnerFromUser(req.User)
		if err != nil {
			_ = removePreparedTree(dir)
			return "", attempts, err
		}
		if err := prepareTreeRoot(dir, owner); err != nil {
			_ = removePreparedTree(dir)
			return "", attempts, fmt.Errorf("no se pudo asignar el árbol al usuario del contenedor: %w", err)
		}
		if err := writePreparationFiles(dir, req.Files, owner); err != nil {
			_ = removePreparedTree(dir)
			return "", attempts, fmt.Errorf("no se pudieron escribir los archivos del proyecto: %w", err)
		}
		installErr := w.runner.RunInstall(ctx, req, dir, logWriter)
		if installErr == nil {
			return dir, attempts, nil
		}
		_ = os.RemoveAll(dir)
		if !errors.Is(installErr, domain.ErrDependencyPreparationTransient) {
			return "", attempts, fmt.Errorf("la instalación falló: %w", installErr)
		}
		if attempts >= w.limits.MaxAttempts {
			return "", attempts, fmt.Errorf("la instalación falló tras %d intentos: %w", attempts, installErr)
		}
		delay := w.retryDelay(attempts)
		fmt.Fprintf(logWriter, "fallo transitorio en el intento %d de %d; reintento en %s\n",
			attempts, w.limits.MaxAttempts, delay)
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return "", attempts, fmt.Errorf("la instalación no terminó dentro del plazo: %w", ctx.Err())
		}
	}
}

const (
	maxPreparationFiles     = 300
	maxPreparationFileBytes = 1024 * 1024
)

// writePreparationFiles escribe la versión publicada dentro del árbol antes de
// instalar. Las rutas se validan otra vez acá: el snapshot viene de contenido
// que escribió el docente, y una ruta que se escape del árbol escribiría donde
// no debe. Un error acá es determinista y no se reintenta.
// preparationOwner es el dueño con el que se crea el árbol. Sin usuario
// declarado queda deshabilitado y el árbol pertenece al proceso que prepara.
type preparationOwner struct {
	uid     int
	gid     int
	enabled bool
}

func preparationOwnerFromUser(user string) (preparationOwner, error) {
	trimmed := strings.TrimSpace(user)
	if trimmed == "" {
		return preparationOwner{}, nil
	}
	parts := strings.SplitN(trimmed, ":", 2)
	uid, err := strconv.Atoi(parts[0])
	if err != nil || uid < 0 {
		return preparationOwner{}, fmt.Errorf("usuario de preparación no válido: %q", user)
	}
	gid := uid
	if len(parts) == 2 {
		gid, err = strconv.Atoi(parts[1])
		if err != nil || gid < 0 {
			return preparationOwner{}, fmt.Errorf("grupo de preparación no válido: %q", user)
		}
	}
	return preparationOwner{uid: uid, gid: gid, enabled: true}, nil
}

// prepareTreeRoot deja el árbol escribible por el usuario del contenedor. Se
// hace antes de escribir los archivos: el contenedor instala como ese usuario y
// un directorio 0700 de root le negaría la escritura.
func prepareTreeRoot(dir string, owner preparationOwner) error {
	if !owner.enabled {
		return os.Chmod(dir, 0o750)
	}
	if err := os.Chown(dir, owner.uid, owner.gid); err != nil {
		return err
	}
	return os.Chmod(dir, 0o770)
}

func writePreparationFiles(root string, files []domain.WorkspaceSeedFile, owner preparationOwner) error {
	if len(files) > maxPreparationFiles {
		return fmt.Errorf("%d archivos superan el máximo de %d", len(files), maxPreparationFiles)
	}
	total := 0
	for _, file := range files {
		clean := filepath.Clean(strings.TrimSpace(file.Path))
		if clean == "." || filepath.IsAbs(clean) || !filepath.IsLocal(clean) {
			return fmt.Errorf("ruta de archivo no válida: %q", file.Path)
		}
		total += len(file.Content)
		if total > maxPreparationFileBytes {
			return fmt.Errorf("los archivos superan el máximo de %d bytes", maxPreparationFileBytes)
		}
		target := filepath.Join(root, clean)
		directory := filepath.Dir(target)
		if err := os.MkdirAll(directory, 0o750); err != nil {
			return err
		}
		if err := os.WriteFile(target, []byte(file.Content), 0o640); err != nil {
			return err
		}
		if !owner.enabled {
			continue
		}
		// El dueño se aplica también a cada archivo y subdirectorio: el
		// contenedor puede necesitar reescribir un lockfile durante la
		// instalación.
		if err := os.Chown(directory, owner.uid, owner.gid); err != nil {
			return err
		}
		if err := os.Chmod(directory, 0o770); err != nil {
			return err
		}
		if err := os.Chown(target, owner.uid, owner.gid); err != nil {
			return err
		}
		if err := os.Chmod(target, 0o660); err != nil {
			return err
		}
	}
	return nil
}

func (w *DependencyPreparationWorker) retryDelay(attempt int) time.Duration {
	if w.limits.RetryBackoff <= 0 {
		return 0
	}
	return w.limits.RetryBackoff << (attempt - 1)
}

func (w *DependencyPreparationWorker) reusedResult(
	result domain.DependencyPreparationResult,
	size int64,
) domain.DependencyPreparationResult {
	result.Status = domain.DependencyPreparationPrepared
	result.Bytes = size
	result.Reused = true
	result.Message = "el árbol ya estaba preparado"
	return result
}

func (w *DependencyPreparationWorker) failed(
	result domain.DependencyPreparationResult,
	logPath string,
	err error,
) (domain.DependencyPreparationResult, error) {
	result.Status = domain.DependencyPreparationFailed
	result.LogPath = logPath
	result.Message = err.Error()
	if logPath != "" {
		if file, openErr := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0o640); openErr == nil {
			fmt.Fprintf(file, "\nresultado: falló: %v\n", err)
			_ = file.Close()
		}
	}
	return result, err
}

func (w *DependencyPreparationWorker) openLog(key string) (string, *os.File, error) {
	logsDir := filepath.Join(w.root, "logs")
	if err := os.MkdirAll(logsDir, 0o750); err != nil {
		return "", nil, fmt.Errorf("no se pudo crear el directorio de logs: %w", err)
	}
	name := fmt.Sprintf("%s-%s.log", key, w.now().UTC().Format("20060102T150405Z"))
	path := filepath.Join(logsDir, name)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return "", nil, fmt.Errorf("no se pudo abrir el log de preparación: %w", err)
	}
	return path, file, nil
}

// removePreparedTree borra un árbol aunque tenga directorios de solo lectura.
// El caché de módulos de Go los deja así y RemoveAll no puede desenlazar dentro
// de un directorio sin permiso de escritura: la poda fallaría justo en la
// familia que más ocupa.
func removePreparedTree(path string) error {
	_ = filepath.Walk(path, func(current string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return nil
		}
		_ = os.Chmod(current, 0o750)
		return nil
	})
	return os.RemoveAll(path)
}

// PruneOrphanPreparedTrees borra los árboles que ninguna versión publicada
// referencia. Exige el conjunto de claves activas: sin él no borra nada, porque
// un nil significaría borrar todos los preparados del nodo.
func (w *DependencyPreparationWorker) PruneOrphanPreparedTrees(
	ctx context.Context,
	activeKeys map[string]bool,
) ([]string, error) {
	if activeKeys == nil {
		return nil, errors.New("la poda exige el conjunto de claves activas; nil borraría todos los preparados")
	}
	entries, err := os.ReadDir(w.root)
	if err != nil {
		return nil, fmt.Errorf("no se pudo listar la raíz de preparados: %w", err)
	}
	pruned := make([]string, 0)
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return pruned, err
		}
		name := entry.Name()
		if !entry.IsDir() || strings.HasPrefix(name, ".") || name == "logs" {
			continue
		}
		if activeKeys[name] {
			continue
		}
		if err := removePreparedTree(filepath.Join(w.root, name)); err != nil {
			return pruned, fmt.Errorf("no se pudo podar %s: %w", name, err)
		}
		pruned = append(pruned, name)
	}
	return pruned, nil
}

// TreePresent dice si el árbol de una ruta preparada sigue en disco. La ruta
// tiene que estar dentro de la raíz: la base guarda la ubicación, no la decide.
// Sirve para distinguir un trabajo preparado de uno que solo lo dice.
func (w *DependencyPreparationWorker) TreePresent(path string) bool {
	clean := filepath.Clean(strings.TrimSpace(path))
	if clean == "." || !filepath.IsAbs(clean) {
		return false
	}
	rel, err := filepath.Rel(w.root, clean)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return false
	}
	info, err := os.Stat(clean)
	return err == nil && info.IsDir()
}

// Usage informa cuántos árboles hay y cuánto ocupan, para la cuota del nodo.
func (w *DependencyPreparationWorker) Usage() (int, int64, error) {
	entries, err := os.ReadDir(w.root)
	if err != nil {
		return 0, 0, fmt.Errorf("no se pudo leer la raíz de preparados: %w", err)
	}
	count := 0
	var total int64
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() || strings.HasPrefix(name, ".") || name == "logs" {
			continue
		}
		count++
		if size, err := dirBytes(filepath.Join(w.root, name)); err == nil {
			total += size
		}
	}
	return count, total, nil
}

// CheckSameFilesystem verifica que dos rutas vivan en el mismo sistema de
// archivos. Es la condición para que el enlace duro funcione: si no se cumple,
// la preparación termina en una copia física por alumno.
func CheckSameFilesystem(a, b string) error {
	devA, err := deviceOf(a)
	if err != nil {
		return err
	}
	if _, err := os.Stat(b); err != nil {
		return fmt.Errorf("no se pudo leer %s: %w", b, err)
	}
	devB, err := deviceOf(b)
	if err != nil {
		return err
	}
	if devA != devB {
		return fmt.Errorf("%s y %s viven en sistemas de archivos distintos: el enlace duro no va a funcionar", a, b)
	}
	return nil
}

func deviceOf(path string) (uint64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, fmt.Errorf("no se pudo leer %s: %w", path, err)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("no se pudo leer el dispositivo de %s", path)
	}
	return uint64(stat.Dev), nil
}

func preparedTree(dir string) (int64, bool) {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return 0, false
	}
	size, err := dirBytes(dir)
	if err != nil {
		return 0, false
	}
	return size, true
}

func dirBytes(root string) (int64, error) {
	var total int64
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		return nil
	})
	if err != nil {
		return 0, err
	}
	return total, nil
}

func freeBytes(path string) (int64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, err
	}
	return int64(stat.Bavail) * int64(stat.Bsize), nil
}

var _ io.Writer = (*os.File)(nil)
