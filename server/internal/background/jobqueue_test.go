package background

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestRegistry_RegisterAndLookup(t *testing.T) {
	r := NewRegistry()
	called := false
	r.Register("test.job", HandlerFunc(func(_ context.Context, _ json.RawMessage) error {
		called = true
		return nil
	}))
	h, ok := r.handler("test.job")
	if !ok {
		t.Fatal("handler not found")
	}
	if err := h.Execute(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("handler not invoked")
	}
	if _, ok := r.handler("missing"); ok {
		t.Fatal("unexpected handler for missing type")
	}
}

func TestRegistry_Types(t *testing.T) {
	r := NewRegistry()
	r.Register("a", HandlerFunc(func(_ context.Context, _ json.RawMessage) error { return nil }))
	r.Register("b", HandlerFunc(func(_ context.Context, _ json.RawMessage) error { return nil }))
	if got := len(r.Types()); got != 2 {
		t.Fatalf("want 2 types got %d", got)
	}
}

func TestSafeExecute_RecoversPanic(t *testing.T) {
	h := HandlerFunc(func(_ context.Context, _ json.RawMessage) error {
		panic("boom")
	})
	err := safeExecute(context.Background(), h, nil)
	if err == nil {
		t.Fatal("expected error from panic")
	}
}

func TestSafeExecute_PassesError(t *testing.T) {
	sentinel := errors.New("nope")
	h := HandlerFunc(func(_ context.Context, _ json.RawMessage) error { return sentinel })
	if err := safeExecute(context.Background(), h, nil); !errors.Is(err, sentinel) {
		t.Fatalf("got %v", err)
	}
}

func TestStartJobQueueWorker_DisabledStillRegisters(t *testing.T) {
	// With a nil pool and flag off, the worker does not start but built-in job
	// types are still registered so enqueue paths can reference them.
	reg := StartJobQueueWorker(context.Background(), nil, config.Config{BackgroundJobsEnabled: false})
	if _, ok := reg.handler(JobTypeEmailDelivery); !ok {
		t.Fatal("email.delivery not registered")
	}
}
