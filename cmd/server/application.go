package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"benzhi-project-93ebb82c-088e-4628-a722-962914c0d2d7/internal/store"
	"benzhi-project-93ebb82c-088e-4628-a722-962914c0d2d7/internal/web"
	"benzhi-project-93ebb82c-088e-4628-a722-962914c0d2d7/internal/workflow"
)

type application struct {
	store  *store.DiskStore
	server *http.Server
}

func newApplication(cfg config) (*application, error) {
	repository, err := store.Open(cfg.dataDir)
	if err != nil {
		return nil, fmt.Errorf("恢复本地项目: %w", err)
	}
	service := workflow.New(repository)
	handler := web.New(service)
	server := &http.Server{
		Addr: cfg.addr, Handler: handler,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second,
		WriteTimeout: 20 * time.Second, IdleTimeout: 45 * time.Second,
	}
	return &application{store: repository, server: server}, nil
}

func (a *application) listen() (net.Listener, error) {
	listener, err := net.Listen("tcp", a.server.Addr)
	if err != nil {
		return nil, fmt.Errorf("监听 %s: %w", a.server.Addr, err)
	}
	return listener, nil
}

func (a *application) serve(listener net.Listener) error {
	err := a.server.Serve(listener)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (a *application) shutdown(ctx context.Context) error {
	serverErr := a.server.Shutdown(ctx)
	storeErr := a.store.Close()
	if serverErr != nil {
		return serverErr
	}
	return storeErr
}
