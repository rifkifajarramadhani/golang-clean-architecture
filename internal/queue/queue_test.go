package queue

import (
	"context"
	"encoding/json"
	"testing"
)

func TestHandlerRegistryRejectsDuplicateTypes(t *testing.T) {
	registry := NewHandlerRegistry()
	handler := func(context.Context, json.RawMessage) error { return nil }

	if err := registry.Register("example", handler); err != nil {
		t.Fatalf("register first handler: %v", err)
	}
	if err := registry.Register("example", handler); err == nil {
		t.Fatal("expected duplicate registration error")
	}
}

func TestHandlerRegistryReturnsCopy(t *testing.T) {
	registry := NewHandlerRegistry()
	if err := registry.Register("example", func(context.Context, json.RawMessage) error { return nil }); err != nil {
		t.Fatal(err)
	}

	handlers := registry.Handlers()
	delete(handlers, "example")
	if len(registry.Handlers()) != 1 {
		t.Fatal("registry was mutated through returned map")
	}
}
