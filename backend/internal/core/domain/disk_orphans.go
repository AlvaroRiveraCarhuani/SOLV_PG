package domain

import (
	"errors"
	"sort"
	"strings"
)

// EnvironmentVolumePrefix es el prefijo de los volúmenes que la plataforma crea
// para el workspace de un entorno. La liberación de huérfanos no mira nada
// fuera de ese espacio de nombres: la base de notas (pg_data), los respaldos y
// los volúmenes de otras pilas comparten el daemon, y varios de ellos también
// quedan con LINK=0 cuando su contenedor está detenido.
const EnvironmentVolumePrefix = "solv_workspace_"

// OrphanVolumePrunePhrase es la frase exacta que confirma la liberación. Sin
// ella el servicio no borra nada, igual que el resto de las acciones
// destructivas del panel (ADR-032).
const OrphanVolumePrunePhrase = "LIBERAR VOLUMENES HUERFANOS"

// MaxOrphanVolumesPerRun acota cuántos volúmenes se pueden liberar en una
// confirmación. Una lista enorme es más probable que sea un error de selección
// que una limpieza deliberada.
const MaxOrphanVolumesPerRun = 200

// Razones por las que un volumen colgante se conserva. Son códigos estables:
// la interfaz los traduce y el contrato no cambia con el texto.
const (
	VolumeKeepRegisteredEnvironment = "entorno_registrado"
	VolumeKeepNotPlatformWorkspace  = "fuera_del_espacio_de_la_plataforma"
	VolumeKeepNotDangling           = "ya_no_es_huerfano"
	VolumeKeepRemovalFailed         = "fallo_al_borrar"
)

// ErrVolumeOutsidePlatformScope rechaza un borrado pedido sobre un volumen que
// la plataforma no creó.
var ErrVolumeOutsidePlatformScope = errors.New("volume is outside the platform workspace namespace")

// DanglingVolume es un volumen sin contenedor con su tamaño medido.
type DanglingVolume struct {
	Name   string `json:"name"`
	SizeMB int64  `json:"size_mb"`
}

// KeptVolume es un volumen que el inventario decidió no liberar, con el motivo.
type KeptVolume struct {
	Name   string `json:"name"`
	SizeMB int64  `json:"size_mb"`
	Reason string `json:"reason"`
}

// OrphanVolumePlan separa lo que se puede liberar de lo que no, sin tocar el
// daemon. La lista de conservados explica el motivo para que el administrador
// no tenga que adivinar por qué su volumen no aparece.
type OrphanVolumePlan struct {
	Orphans       []DanglingVolume `json:"orphans"`
	Kept          []KeptVolume     `json:"kept"`
	TotalOrphanMB int64            `json:"total_orphan_mb"`
	TotalKeptMB   int64            `json:"total_kept_mb"`
}

// OrphanVolumePruneRequest pide liberar volúmenes concretos. Nunca acepta un
// «borrar todo»: el nombre es parte de la confirmación.
type OrphanVolumePruneRequest struct {
	VolumeNames        []string `json:"volume_names"`
	ConfirmationPhrase string   `json:"confirmation_phrase"`
	Reason             string   `json:"reason,omitempty"`
}

// DeletedVolume es un volumen liberado con el espacio que ocupaba.
type DeletedVolume struct {
	Name   string `json:"name"`
	SizeMB int64  `json:"size_mb"`
}

// OrphanVolumePruneResult informa qué se borró, qué se conservó y cuánto quedó.
type OrphanVolumePruneResult struct {
	Deleted           []DeletedVolume `json:"deleted"`
	Skipped           []KeptVolume    `json:"skipped"`
	FreedMB           int64           `json:"freed_mb"`
	RemainingOrphanMB int64           `json:"remaining_orphan_mb"`
}

// PlanOrphanVolumes decide qué volúmenes colgantes se pueden liberar. Un
// volumen entra en `Orphans` solo si pertenece al espacio de nombres de la
// plataforma y ningún entorno registrado lo reclama. Cualquier duda conserva el
// volumen: borrar un workspace por error no tiene vuelta atrás.
func PlanOrphanVolumes(dangling []DanglingVolume, registered map[string]struct{}) OrphanVolumePlan {
	plan := OrphanVolumePlan{
		Orphans: make([]DanglingVolume, 0, len(dangling)),
		Kept:    make([]KeptVolume, 0, len(dangling)),
	}

	for _, volume := range dangling {
		name := strings.TrimSpace(volume.Name)
		if name == "" {
			continue
		}

		if !strings.HasPrefix(name, EnvironmentVolumePrefix) {
			plan.Kept = append(plan.Kept, KeptVolume{
				Name:   name,
				SizeMB: volume.SizeMB,
				Reason: VolumeKeepNotPlatformWorkspace,
			})
			plan.TotalKeptMB += volume.SizeMB
			continue
		}

		if _, claimed := registered[name]; claimed {
			plan.Kept = append(plan.Kept, KeptVolume{
				Name:   name,
				SizeMB: volume.SizeMB,
				Reason: VolumeKeepRegisteredEnvironment,
			})
			plan.TotalKeptMB += volume.SizeMB
			continue
		}

		plan.Orphans = append(plan.Orphans, DanglingVolume{Name: name, SizeMB: volume.SizeMB})
		plan.TotalOrphanMB += volume.SizeMB
	}

	sort.Slice(plan.Orphans, func(i, j int) bool { return plan.Orphans[i].SizeMB > plan.Orphans[j].SizeMB })
	sort.Slice(plan.Kept, func(i, j int) bool { return plan.Kept[i].Name < plan.Kept[j].Name })
	return plan
}

// ValidateOrphanVolumeName comprueba que un borrado pedido caiga dentro del
// espacio de nombres de la plataforma. Es la segunda barrera, detrás del
// plan: el adaptador de Docker la repite antes de llamar al daemon.
func ValidateOrphanVolumeName(name string) error {
	if !strings.HasPrefix(strings.TrimSpace(name), EnvironmentVolumePrefix) {
		return ErrVolumeOutsidePlatformScope
	}
	return nil
}
