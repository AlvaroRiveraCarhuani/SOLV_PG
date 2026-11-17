package docker

import (
	"fmt"
	"strings"
	"testing"
)

func TestExamNetwork_ConstantsAndNaming(t *testing.T) {
	if examNetworkName != "solv-exam-net" {
		t.Errorf("examNetworkName inesperado: %q", examNetworkName)
	}
	if examSatellitesNetworkName != "solv-exam-satellites-net" {
		t.Errorf("examSatellitesNetworkName inesperado: %q", examSatellitesNetworkName)
	}
	if examProxyImage != "alpine/socat:1.8.0.0" {
		t.Errorf("examProxyImage debe estar fijada con version estricta: %q", examProxyImage)
	}
	if strings.Contains(examProxyImage, ":latest") {
		t.Errorf("la imagen del proxy no puede usar :latest: %q", examProxyImage)
	}
	if examNoEgressLabel != "solv.exam.no-egress" {
		t.Errorf("examNoEgressLabel inesperado: %q", examNoEgressLabel)
	}
	if examProxyLabel != "solv.exam.proxy" {
		t.Errorf("examProxyLabel inesperado: %q", examProxyLabel)
	}
}

func TestExamNetwork_ProxyCommandFormulation(t *testing.T) {
	containerName := "solv_lab_std123_ex456"
	expectedProxyName := containerName + examProxySuffix
	if expectedProxyName != "solv_lab_std123_ex456-proxy" {
		t.Errorf("nombre de proxy inesperado: %q", expectedProxyName)
	}

	cmd := []string{
		"TCP-LISTEN:3000,fork,reuseaddr",
		fmt.Sprintf("TCP:%s:3000", containerName),
	}
	if len(cmd) != 2 {
		t.Fatalf("se esperaban 2 argumentos para socat, obtenidos %d", len(cmd))
	}
	if cmd[0] != "TCP-LISTEN:3000,fork,reuseaddr" {
		t.Errorf("comando socat listen inesperado: %q", cmd[0])
	}
	if cmd[1] != "TCP:solv_lab_std123_ex456:3000" {
		t.Errorf("comando socat forward inesperado: %q", cmd[1])
	}
}
