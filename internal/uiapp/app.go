package uiapp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	appnode "github.com/coreyvan/chirp/internal/app/node"
	"github.com/coreyvan/chirp/pkg/radio"
	"github.com/coreyvan/chirp/pkg/serial"
)

type App struct {
	ctx context.Context

	mu    sync.Mutex
	port  string
	radio *radio.Radio
}

type Status struct {
	Connected bool   `json:"connected"`
	Port      string `json:"port"`
}

type InfoView struct {
	Summary appnode.InfoSummary `json:"summary"`
}

func NewApp() *App {
	return &App{}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) Shutdown(_ context.Context) {
	_ = a.Disconnect()
}

func (a *App) Health() string {
	return "ok"
}

func (a *App) ListPorts() ([]string, error) {
	ports, err := serial.GetPorts()
	if err != nil {
		return nil, err
	}

	trimmed := make([]string, 0, len(ports))
	for _, p := range ports {
		port := strings.TrimSpace(p)
		if port != "" {
			trimmed = append(trimmed, port)
		}
	}

	return trimmed, nil
}

func (a *App) Connect(port string) error {
	selectedPort := strings.TrimSpace(port)
	if selectedPort == "" {
		return errors.New("port is required")
	}

	newRadio := &radio.Radio{}
	if err := newRadio.Init(selectedPort); err != nil {
		return fmt.Errorf("failed to connect to %q: %w", selectedPort, err)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.radio != nil {
		_ = a.radio.Close()
	}
	a.radio = newRadio
	a.port = selectedPort

	return nil
}

func (a *App) Disconnect() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.radio == nil {
		a.port = ""
		return nil
	}

	err := a.radio.Close()
	a.radio = nil
	a.port = ""
	if err != nil {
		return fmt.Errorf("disconnect radio: %w", err)
	}
	return nil
}

func (a *App) ConnectionStatus() Status {
	a.mu.Lock()
	defer a.mu.Unlock()

	return Status{
		Connected: a.radio != nil,
		Port:      a.port,
	}
}

func (a *App) LoadInfo() (InfoView, error) {
	r, err := a.currentRadio()
	if err != nil {
		return InfoView{}, err
	}

	service := appnode.NewService(r)
	info, err := service.Info(a.currentContext())
	if err != nil {
		return InfoView{}, err
	}

	return InfoView{Summary: info.Summary}, nil
}

func (a *App) currentRadio() (*radio.Radio, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.radio == nil {
		return nil, errors.New("not connected")
	}
	return a.radio, nil
}

func (a *App) currentContext() context.Context {
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}
