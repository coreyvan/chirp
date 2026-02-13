package uiapp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	appnode "github.com/coreyvan/chirp/internal/app/node"
	"github.com/coreyvan/chirp/pkg/radio"
	"github.com/coreyvan/chirp/pkg/serial"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const listenerEventName = "listener:line"

type App struct {
	ctx context.Context

	mu    sync.Mutex
	port  string
	radio *radio.Radio

	listenerCancel context.CancelFunc
	listenerDone   chan struct{}
}

type Status struct {
	Connected bool   `json:"connected"`
	Port      string `json:"port"`
}

type InfoView struct {
	Summary appnode.InfoSummary `json:"summary"`
}

type ListenerStatus struct {
	Running bool `json:"running"`
}

type ListenerLine struct {
	Timestamp string `json:"timestamp"`
	Label     string `json:"label"`
	Message   string `json:"message"`
	Category  string `json:"category"`
}

func NewApp() *App {
	return &App{}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) Shutdown(_ context.Context) {
	_ = a.StopListener()
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

	if err := a.StopListener(); err != nil {
		_ = newRadio.Close()
		return err
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
	if err := a.StopListener(); err != nil {
		return err
	}

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

func (a *App) StartListener() error {
	a.mu.Lock()
	if a.radio == nil {
		a.mu.Unlock()
		return errors.New("not connected")
	}
	if a.listenerCancel != nil {
		a.mu.Unlock()
		return errors.New("listener already running")
	}

	r := a.radio
	port := a.port
	base := a.currentContext()
	runCtx, cancel := context.WithCancel(base)
	done := make(chan struct{})
	a.listenerCancel = cancel
	a.listenerDone = done
	a.mu.Unlock()

	go a.runListener(runCtx, done, r, port)
	return nil
}

func (a *App) StopListener() error {
	a.mu.Lock()
	cancel := a.listenerCancel
	done := a.listenerDone
	a.listenerCancel = nil
	a.listenerDone = nil
	a.mu.Unlock()

	if cancel == nil {
		return nil
	}

	cancel()
	if done != nil {
		<-done
	}

	return nil
}

func (a *App) GetListenerStatus() ListenerStatus {
	a.mu.Lock()
	defer a.mu.Unlock()

	return ListenerStatus{Running: a.listenerCancel != nil}
}

func (a *App) runListener(ctx context.Context, done chan struct{}, r *radio.Radio, port string) {
	defer close(done)

	a.emitListenerLine("EVT", fmt.Sprintf("rx listener started on %s", port), appnode.StreamCategoryEvent)

	if responses, err := r.GetRadioInfo(); err != nil {
		a.emitListenerLine("ERR", fmt.Sprintf("get radio info: %v", err), appnode.StreamCategoryEvent)
	} else {
		for _, fr := range responses {
			for _, line := range appnode.RenderFromRadio(fr) {
				a.emitListenerLine(line.Label, line.Message, line.Category)
			}
		}
	}

	for {
		select {
		case <-ctx.Done():
			a.emitListenerLine("EVT", "rx listener stopped", appnode.StreamCategoryEvent)
			return
		default:
		}

		fromRadioPackets, err := r.ReadResponse(true)
		if err != nil {
			a.emitListenerLine("ERR", fmt.Sprintf("read response: %v", err), appnode.StreamCategoryEvent)
			select {
			case <-ctx.Done():
				a.emitListenerLine("EVT", "rx listener stopped", appnode.StreamCategoryEvent)
				return
			case <-time.After(300 * time.Millisecond):
			}
			continue
		}

		if len(fromRadioPackets) == 0 {
			continue
		}

		for _, fr := range fromRadioPackets {
			for _, line := range appnode.RenderFromRadio(fr) {
				a.emitListenerLine(line.Label, line.Message, line.Category)
			}
		}
	}
}

func (a *App) emitListenerLine(label string, message string, category appnode.StreamCategory) {
	ctx := a.currentContext()
	if ctx == nil {
		return
	}

	wailsruntime.EventsEmit(ctx, listenerEventName, ListenerLine{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Label:     label,
		Message:   message,
		Category:  streamCategoryName(category),
	})
}

func streamCategoryName(category appnode.StreamCategory) string {
	switch category {
	case appnode.StreamCategoryEvent:
		return "event"
	case appnode.StreamCategoryPacket:
		return "packet"
	case appnode.StreamCategoryTelemetry:
		return "telemetry"
	case appnode.StreamCategoryMessage:
		return "message"
	default:
		return "event"
	}
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
