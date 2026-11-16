package system

import (
	"fmt"
	"syscall"
)

// HostFreeDiskPercent devuelve el porcentaje libre del sistema de archivos que
// contiene la ruta indicada (por defecto la raíz). El alta de entornos lo usa
// para no sumar contenedores a un nodo que se quedó sin espacio.
func HostFreeDiskPercent(path string) (float64, error) {
	if path == "" {
		path = "/"
	}
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, fmt.Errorf("no se pudo leer el espacio libre de %s: %w", path, err)
	}
	if stat.Blocks == 0 {
		return 0, fmt.Errorf("el sistema de archivos de %s no reporta bloques", path)
	}
	free := float64(stat.Bavail) / float64(stat.Blocks) * 100.0
	return free, nil
}
