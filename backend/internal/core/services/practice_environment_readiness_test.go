package services

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// El editor responde 504 mientras arranca: la espera tiene que reintentar hasta
// que sirva contenido y no tomar el 5xx del proxy como entorno listo.
func TestWaitForEditorWithin_EsperaHastaQueElEditorResponde(t *testing.T) {
	t.Setenv("BASE_DOMAIN", "solv.example")

	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if atomic.AddInt32(&hits, 1) < 3 {
			w.WriteHeader(http.StatusGatewayTimeout)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	if err := waitForEditorWithin(context.Background(), server.URL, 2*time.Second, 20*time.Millisecond); err != nil {
		t.Fatalf("la espera debía terminar cuando el editor responde: %v", err)
	}
	if hits < 3 {
		t.Fatalf("esperaba al menos tres sondeos, hubo %d", hits)
	}
}

// Un entorno que nunca llega a servir no deja la petición colgada: devuelve el
// error que el transporte traduce a 503 environment_not_ready.
func TestWaitForEditorWithin_DevuelveNoListoAlVencerElPlazo(t *testing.T) {
	t.Setenv("BASE_DOMAIN", "solv.example")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	err := waitForEditorWithin(context.Background(), server.URL, 200*time.Millisecond, 20*time.Millisecond)
	if err == nil {
		t.Fatal("un entorno que solo responde 502 no está listo")
	}
	if !errors.Is(err, ErrEnvironmentNotReady) {
		t.Fatalf("esperaba ErrEnvironmentNotReady, obtuve: %v", err)
	}
}

// Traefik responde 404 mientras no conoce el router del entorno: ese 4xx no es
// el editor y la espera tiene que seguir hasta vencer el plazo.
func TestWaitForEditorWithin_Un404DeTraefikNoCuentaComoListo(t *testing.T) {
	t.Setenv("BASE_DOMAIN", "solv.example")

	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	err := waitForEditorWithin(context.Background(), server.URL, 200*time.Millisecond, 20*time.Millisecond)
	if !errors.Is(err, ErrEnvironmentNotReady) {
		t.Fatalf("un 404 del proxy no está listo; esperaba ErrEnvironmentNotReady, obtuve: %v", err)
	}
	if hits < 2 {
		t.Fatalf("esperaba varios sondeos antes de vencer el plazo, hubo %d", hits)
	}
}

// Un 3xx del editor sí cuenta como listo: es la redirección normal con la que
// OpenVSCode Server entrega la aplicación.
func TestWaitForEditorWithin_Un3xxDelEditorCuentaComoListo(t *testing.T) {
	t.Setenv("BASE_DOMAIN", "solv.example")

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ready", http.StatusFound)
	})
	mux.HandleFunc("/ready", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	if err := waitForEditorWithin(context.Background(), server.URL, time.Second, 20*time.Millisecond); err != nil {
		t.Fatalf("una redirección del editor significa que ya está sirviendo: %v", err)
	}
}

// Sin dominio público el acceso va directo al puerto publicado: no hay proxy
// que devuelva una página de error y la espera no aplica.
func TestWaitForEditorWithin_SinDominioPublicoNoEspera(t *testing.T) {
	t.Setenv("BASE_DOMAIN", "")

	if err := waitForEditorWithin(context.Background(), "http://localhost:3000", time.Second, time.Millisecond); err != nil {
		t.Fatalf("sin dominio público no debía esperar: %v", err)
	}
}
