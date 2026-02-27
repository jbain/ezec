package consumers

import (
	"sync"
	"testing"
)

func TestCallbackConsumer(t *testing.T) {
	var mu sync.Mutex
	var got []string

	cb := NewCallbackConsumer("test", 10, func(line string) {
		mu.Lock()
		got = append(got, line)
		mu.Unlock()
	})
	go cb.Start()

	lines := []string{"foo", "bar", "baz"}
	for _, l := range lines {
		cb.InputCh() <- l
	}
	close(cb.InputCh())
	cb.Wait()

	mu.Lock()
	defer mu.Unlock()
	if len(got) != len(lines) {
		t.Fatalf("expected %d lines, got %d", len(lines), len(got))
	}
	for i, want := range lines {
		if got[i] != want {
			t.Errorf("line %d: want %q, got %q", i, want, got[i])
		}
	}
}

func TestCallbackConsumerName(t *testing.T) {
	cb := NewCallbackConsumer("my-consumer", 1, func(string) {})
	if cb.Name() != "my-consumer" {
		t.Errorf("Name(): want %q, got %q", "my-consumer", cb.Name())
	}
}

func TestCallbackConsumerWaitUnblocksAfterClose(t *testing.T) {
	cb := NewCallbackConsumer("test", 1, func(string) {})
	go cb.Start()
	close(cb.InputCh())
	cb.Wait() // must not block
}

func TestCallbackConsumerEmpty(t *testing.T) {
	var called bool
	cb := NewCallbackConsumer("test", 1, func(string) { called = true })
	go cb.Start()
	close(cb.InputCh())
	cb.Wait()
	if called {
		t.Error("callback should not be called for empty input")
	}
}
