package domain

import (
	"strings"
	"testing"
	"time"
)

func preparationRequest() DependencyPreparationRequest {
	return DependencyPreparationRequest{
		TenantID:       "00000000-0000-0000-0000-000000000001",
		VersionID:      "version-1",
		ComponentSlug:  "node-20",
		BaseImage:      "node:20-bookworm-slim",
		ImageDigest:    "sha256:aaaa",
		ProfileVersion: 1,
		ProjectPath:    "/home/workspace/proyecto",
		Install:        "npm ci",
		Verify:         "npm ls --depth=0",
		Network:        "build",
		LockfileHash:   "lock-1",
	}
}

func TestDependencyPreparationKeyIsStableAndSensitive(t *testing.T) {
	base := preparationRequest()
	key := base.Key()

	if len(key) != 64 {
		t.Fatalf("la clave debe ser sha256 en hexadecimal, obtuvo %q", key)
	}
	if key != preparationRequest().Key() {
		t.Fatal("la misma solicitud debe producir la misma clave")
	}

	cases := map[string]func(r *DependencyPreparationRequest){
		"version":  func(r *DependencyPreparationRequest) { r.VersionID = "version-2" },
		"imagen":   func(r *DependencyPreparationRequest) { r.ImageDigest = "sha256:bbbb" },
		"perfil":   func(r *DependencyPreparationRequest) { r.ProfileVersion = 2 },
		"lockfile": func(r *DependencyPreparationRequest) { r.LockfileHash = "lock-2" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			changed := preparationRequest()
			mutate(&changed)
			if changed.Key() == key {
				t.Fatalf("cambiar %s debe cambiar la clave de preparación", name)
			}
		})
	}

	sinDigest := preparationRequest()
	sinDigest.ImageDigest = ""
	if sinDigest.Key() != preparationRequest().Key() && sinDigest.Key() == "" {
		t.Fatal("sin digest la clave debe caer en la imagen base")
	}
}

func TestDependencyPreparationRequestValidate(t *testing.T) {
	if err := preparationRequest().Validate(); err != nil {
		t.Fatalf("la solicitud base debe ser válida: %v", err)
	}

	cases := map[string]struct {
		mutate func(r *DependencyPreparationRequest)
		want   string
	}{
		"sin tenant":       {func(r *DependencyPreparationRequest) { r.TenantID = "" }, "tenant"},
		"sin versión":      {func(r *DependencyPreparationRequest) { r.VersionID = " " }, "versión"},
		"sin imagen":       {func(r *DependencyPreparationRequest) { r.BaseImage = "" }, "imagen"},
		"perfil inválido":  {func(r *DependencyPreparationRequest) { r.ProfileVersion = 0 }, "perfil"},
		"ruta relativa":    {func(r *DependencyPreparationRequest) { r.ProjectPath = "proyecto" }, "ruta"},
		"ruta raíz":        {func(r *DependencyPreparationRequest) { r.ProjectPath = "/" }, "ruta"},
		"ruta con escape":  {func(r *DependencyPreparationRequest) { r.ProjectPath = "/home/../etc" }, "volumen"},
		"sin instalación":  {func(r *DependencyPreparationRequest) { r.Install = "" }, "instalación"},
		"sin comprobación": {func(r *DependencyPreparationRequest) { r.Verify = "" }, "comprobación"},
		"red inválida":     {func(r *DependencyPreparationRequest) { r.Network = "host" }, "network"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			req := preparationRequest()
			tc.mutate(&req)
			err := req.Validate()
			if err == nil {
				t.Fatalf("se esperaba error que mencione %q", tc.want)
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.want)) {
				t.Fatalf("el error debería mencionar %q, obtuvo: %v", tc.want, err)
			}
		})
	}
}

func TestDependencyPreparationInstallCommandUsesFallback(t *testing.T) {
	sinLock := preparationRequest()
	sinLock.InstallWithoutLock = "npm install --no-audit"
	if sinLock.InstallCommand() != "npm install --no-audit" {
		t.Fatalf("sin lockfile debe usar el comando de respaldo, obtuvo %q", sinLock.InstallCommand())
	}

	conLock := preparationRequest()
	conLock.HasLockfile = true
	conLock.InstallWithoutLock = "npm install --no-audit"
	if conLock.InstallCommand() != "npm ci" {
		t.Fatalf("con lockfile debe usar el comando principal, obtuvo %q", conLock.InstallCommand())
	}

	sinRespaldo := preparationRequest()
	if sinRespaldo.InstallCommand() != "npm ci" {
		t.Fatalf("sin respaldo declarado debe usar el principal, obtuvo %q", sinRespaldo.InstallCommand())
	}
}

func TestDependencyPreparationLimits(t *testing.T) {
	resolved := (DependencyPreparationLimits{}).WithDefaults()
	if resolved.Timeout != DefaultDependencyPreparationTimeout {
		t.Fatalf("el tiempo por defecto debe ser %s, obtuvo %s", DefaultDependencyPreparationTimeout, resolved.Timeout)
	}
	if resolved.MaxBytes != DefaultDependencyPreparationMaxBytes {
		t.Fatalf("el techo por defecto debe ser %d, obtuvo %d", DefaultDependencyPreparationMaxBytes, resolved.MaxBytes)
	}
	if resolved.MinFreeBytes != DefaultDependencyPreparationMinFreeBytes {
		t.Fatalf("el mínimo libre por defecto debe ser %d, obtuvo %d", DefaultDependencyPreparationMinFreeBytes, resolved.MinFreeBytes)
	}
	if err := resolved.Validate(); err != nil {
		t.Fatalf("los techos por defecto deben ser válidos: %v", err)
	}
	if err := (DependencyPreparationLimits{Timeout: -time.Second, MaxBytes: 1}).Validate(); err == nil {
		t.Fatal("un tiempo negativo debe rechazarse")
	}
	if err := (DependencyPreparationLimits{Timeout: time.Second, MaxBytes: -1}).Validate(); err == nil {
		t.Fatal("un techo de bytes negativo debe rechazarse")
	}
	if err := (DependencyPreparationLimits{Timeout: time.Second, MaxBytes: 1, MinFreeBytes: -1}).Validate(); err == nil {
		t.Fatal("un mínimo libre negativo debe rechazarse")
	}
}
