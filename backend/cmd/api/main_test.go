package main

import (
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

// El apagado ordenado deja terminar la petición en curso antes de cerrar el
// listener. La prueba usa un puerto efímero y libera el handler a mitad del
// apagado para comprobar que Shutdown espera.
func TestServeUntilShutdownDrainsInFlightRequest(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		w.WriteHeader(http.StatusNoContent)
	})}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("no se pudo abrir el listener: %v", err)
	}

	shutdown := make(chan struct{})
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- serveUntilShutdown(srv, listener, shutdown, 5*time.Second)
	}()

	requestDone := make(chan error, 1)
	go func() {
		response, err := http.Get("http://" + listener.Addr().String())
		if response != nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
		}
		requestDone <- err
	}()

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("la petición no llegó al servidor")
	}
	close(shutdown)
	close(release)

	if err := <-requestDone; err != nil {
		t.Fatalf("la petición en curso se interrumpió: %v", err)
	}
	if err := <-serverDone; err != nil {
		t.Fatalf("el servidor no se apagó de forma ordenada: %v", err)
	}
}
