package core

import (
	"context"
	"errors"
	"testing"
)

type fakeTransport struct {
	name       string
	startErr   error
	stopErr    error
	startCalls int
	stopCalls  int
}

func (f *fakeTransport) Name() string { return f.name }

func (f *fakeTransport) Start(ctx context.Context) error {
	f.startCalls++
	return f.startErr
}

func (f *fakeTransport) Stop(ctx context.Context) error {
	f.stopCalls++
	return f.stopErr
}

func TestTransportManagerRegisterStartStop(t *testing.T) {
	mgr := NewTransportManager()
	tr := &fakeTransport{name: "cli"}
	if err := mgr.Register(tr); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := mgr.StartAll(context.Background()); err != nil {
		t.Fatalf("start all: %v", err)
	}
	if err := mgr.StopAll(context.Background()); err != nil {
		t.Fatalf("stop all: %v", err)
	}
	if tr.startCalls != 1 || tr.stopCalls != 1 {
		t.Fatalf("unexpected calls: start=%d stop=%d", tr.startCalls, tr.stopCalls)
	}
}

func TestTransportManagerDuplicateRegister(t *testing.T) {
	mgr := NewTransportManager()
	if err := mgr.Register(&fakeTransport{name: "cli"}); err != nil {
		t.Fatalf("register first: %v", err)
	}
	if err := mgr.Register(&fakeTransport{name: "cli"}); err == nil {
		t.Fatalf("expected duplicate error")
	}
}

func TestTransportManagerStopOneUnknown(t *testing.T) {
	mgr := NewTransportManager()
	err := mgr.StopOne(context.Background(), "missing")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, errUnknownTransport) {
		t.Fatalf("expected errUnknownTransport, got: %v", err)
	}
}
