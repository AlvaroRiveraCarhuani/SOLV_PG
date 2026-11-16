package domain

import "testing"

func TestEnvironmentSnapshot_AsTemplateKeepsPinnedBindings(t *testing.T) {
	snapshot := &EnvironmentSnapshot{
		TemplateID:        "tpl-1",
		DockerImage:       "solv/ide:1.2.3",
		Entrypoint:        "python3 main.py",
		TimeoutMS:         5000,
		BaseRAMMB:         512,
		TargetEnvironment: "IDE_PERSISTENTE",
		SetupScript:       "echo setup",
		Bindings: []EnvironmentSnapshotBinding{
			{Role: ServiceRolePrimaryDatabase, Engine: "postgres", EnvAlias: "db", InitKind: ServiceInitSQL},
			{Role: ServiceRoleCache, Engine: "redis", EnvAlias: "cache", InitKind: ServiceInitRedisCmds},
		},
	}

	template := snapshot.AsTemplate()
	if template == nil || template.DockerImage != snapshot.DockerImage || template.SetupScript != snapshot.SetupScript {
		t.Fatalf("template reconstruido inesperado: %+v", template)
	}
	if len(template.ServicesConfig.Services) != 2 || template.ServicesConfig.Services[0].Engine != "postgres" {
		t.Fatalf("bindings reconstruidos inesperados: %+v", template.ServicesConfig.Services)
	}

	summary := snapshot.AsTemplateSummary()
	if summary == nil || summary.TimeoutMS != snapshot.TimeoutMS || summary.BaseRamMB != snapshot.BaseRAMMB {
		t.Fatalf("summary reconstruido inesperado: %+v", summary)
	}
}

func TestEnvironmentSnapshot_FingerprintIgnoresPreviousValue(t *testing.T) {
	snapshot := EnvironmentSnapshot{DockerImage: "solv/ide:1.2.3", BaseRAMMB: 512}
	first := snapshot.ComputeVerificationFingerprint()
	snapshot.VerificationFingerprint = "otra-huella"
	second := snapshot.ComputeVerificationFingerprint()
	if first == "" || first != second {
		t.Fatalf("la huella debe ser determinista: %q vs %q", first, second)
	}
}
