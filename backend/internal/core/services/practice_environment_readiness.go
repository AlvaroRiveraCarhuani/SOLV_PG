package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// ErrEnvironmentNotReady señala que el contenedor del entorno existe pero el
// editor todavía no responde. No es un fallo del alta: el entorno sigue
// arrancando y el reintento lo encuentra vivo, así que el transporte responde
// 503 con un mensaje que invita a reintentar.
var ErrEnvironmentNotReady = errors.New("el entorno todavía no está listo")

// editorReadyTimeout acota la espera en la petición de apertura. La pestaña ya
// está abierta con la pantalla de carga, así que esperar acá es preferible a
// entregar una URL que el proxy todavía no puede alcanzar.
const editorReadyTimeout = 20 * time.Second

// editorProbeInterval es el ritmo con el que se reintenta la sonda mientras el
// proceso del editor termina de escuchar.
const editorProbeInterval = 400 * time.Millisecond

// editorProbeTimeout acota cada intento: un editor que no contesta en dos
// segundos en la red local no está listo todavía.
const editorProbeTimeout = 2 * time.Second

// editorProbeClient comparte el transporte para reutilizar conexiones entre
// reintentos; un cliente por sonda dejaría conexiones sin cerrar.
var editorProbeClient = &http.Client{Timeout: editorProbeTimeout}

// waitForEditorReady espera a que la URL pública del entorno sirva el editor
// antes de entregarla a la pestaña.
//
// Traefik responde 502/503/504 mientras el contenedor no atiende y 404 mientras
// todavía no conoce el router del entorno, así que esperar a que el proxy
// conteste no alcanza: solo un 2xx o un 3xx del editor cuenta como listo. Sin
// dominio público el acceso es directo al puerto publicado y no hay proxy que
// devuelva una página de error, por eso la espera solo aplica cuando
// BASE_DOMAIN está configurado.
func waitForEditorReady(ctx context.Context, accessURL string) error {
	return waitForEditorWithin(ctx, accessURL, editorReadyTimeout, editorProbeInterval)
}

func waitForEditorWithin(ctx context.Context, accessURL string, timeout, interval time.Duration) error {
	if getBaseDomain() == "" || accessURL == "" {
		return nil
	}

	deadline := time.Now().Add(timeout)
	for {
		if editorAnswers(ctx, accessURL) {
			return nil
		}
		if !time.Now().Before(deadline) {
			return fmt.Errorf("%w: %s no respondió en %s", ErrEnvironmentNotReady, accessURL, timeout)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
}

// editorAnswers hace una petición real al editor a través del proxy. Solo un
// 2xx o un 3xx significan que el editor ya está sirviendo: el 4xx también lo
// emite Traefik (404 sin router, 401/403 de sus middlewares) y tomarlo como
// listo entregaría una pestaña que no abre el IDE.
func editorAnswers(ctx context.Context, accessURL string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, accessURL, nil)
	if err != nil {
		return false
	}
	resp, err := editorProbeClient.Do(req)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusBadRequest
}
