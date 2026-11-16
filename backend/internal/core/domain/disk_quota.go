package domain

import (
	"errors"
	"fmt"
)

// DefaultHostDiskFloorPct es el espacio libre por debajo del cual el nodo deja
// de admitir entornos nuevos. Es la única situación en la que el disco justifica
// frenar el trabajo: sin espacio no puede escribir ni el propio contenedor.
const DefaultHostDiskFloorPct = 5.0

// ErrHostDiskExhausted rechaza un alta cuando el nodo se quedó sin espacio. El
// entorno existente no se toca: la negativa es para los nuevos.
var ErrHostDiskExhausted = errors.New("host disk exhausted")

// IsHostDiskCritical compara el espacio libre con el piso del nodo.
func IsHostDiskCritical(freePct, floorPct float64) bool {
	if floorPct <= 0 {
		floorPct = DefaultHostDiskFloorPct
	}
	return freePct < floorPct
}

// ErrDiskQuotaExceeded marca la entrega rechazada porque el volumen superó la
// cuota blanda del nodo. El transporte lo traduce a un 409 accionable.
var ErrDiskQuotaExceeded = errors.New("disk quota exceeded")

// DiskQuotaStatus es el veredicto de la cuota blanda.
type DiskQuotaStatus string

const (
	DiskQuotaOK      DiskQuotaStatus = "ok"
	DiskQuotaWarning DiskQuotaStatus = "warning"
	DiskQuotaFull    DiskQuotaStatus = "full"
)

// DiskQuotaVerdict describe el tamaño medido contra la cuota vigente.
type DiskQuotaVerdict struct {
	Status  DiskQuotaStatus `json:"status"`
	UsedMB  int64           `json:"used_mb"`
	LimitMB int64           `json:"limit_mb"`
	Message string          `json:"message,omitempty"`
}

// EvaluateDiskQuota compara un tamaño medido con la cuota del nodo. La cuota es
// blanda: se mide en los puntos donde el sistema escribe (entrega y siembra), no
// en cada escritura del editor, y por eso el diseño la documenta como tal.
func EvaluateDiskQuota(usedBytes, quotaMB int64, warnPct, blockPct int) DiskQuotaVerdict {
	verdict := DiskQuotaVerdict{Status: DiskQuotaOK}
	if quotaMB <= 0 {
		return verdict
	}
	if blockPct <= 0 || blockPct > 100 {
		blockPct = 100
	}
	if warnPct <= 0 || warnPct > blockPct {
		warnPct = 80
	}

	const mb = int64(1024 * 1024)
	usedMB := usedBytes / mb
	if usedBytes%mb != 0 {
		usedMB++
	}

	verdict.UsedMB = usedMB
	verdict.LimitMB = quotaMB
	switch {
	case usedMB*100 >= quotaMB*int64(blockPct):
		verdict.Status = DiskQuotaFull
		verdict.Message = fmt.Sprintf("el entorno ocupa %d MB y la cuota es de %d MB", usedMB, quotaMB)
	case usedMB*100 >= quotaMB*int64(warnPct):
		verdict.Status = DiskQuotaWarning
		verdict.Message = fmt.Sprintf("el entorno ocupa %d MB de una cuota de %d MB", usedMB, quotaMB)
	}
	return verdict
}
