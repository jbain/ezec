package consumers

import (
	"fmt"
	"testing"
)

func TestLastNConsumer_UnderCapacity(t *testing.T) {
	r := NewLastNConsumer("test", 10, 5)
	go r.Start()

	lines := []string{"a", "b", "c"}
	for _, l := range lines {
		r.InputCh() <- l
	}
	close(r.InputCh())
	r.Wait()

	got := r.Lines()
	if len(got) != len(lines) {
		t.Fatalf("expected %d lines, got %d", len(lines), len(got))
	}
	for i, want := range lines {
		if got[i] != want {
			t.Errorf("line %d: want %q, got %q", i, want, got[i])
		}
	}
}

func TestLastNConsumer_AtCapacity(t *testing.T) {
	r := NewLastNConsumer("test", 10, 3)
	go r.Start()

	for _, l := range []string{"a", "b", "c"} {
		r.InputCh() <- l
	}
	close(r.InputCh())
	r.Wait()

	got := r.Lines()
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("expected %d lines, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d: want %q, got %q", i, want[i], got[i])
		}
	}
}

func TestLastNConsumer_OverCapacity(t *testing.T) {
	r := NewLastNConsumer("test", 20, 3)
	go r.Start()

	// Send 7 lines; only the last 3 should be retained.
	for i := 0; i < 7; i++ {
		r.InputCh() <- fmt.Sprintf("line%d", i)
	}
	close(r.InputCh())
	r.Wait()

	got := r.Lines()
	want := []string{"line4", "line5", "line6"}
	if len(got) != len(want) {
		t.Fatalf("expected %d lines, got %d: %v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d: want %q, got %q", i, want[i], got[i])
		}
	}
}

func TestLastNConsumer_WrapAround(t *testing.T) {
	// Verify ordering is correct across multiple full-buffer wrap cycles.
	capacity := 4
	r := NewLastNConsumer("test", 50, capacity)
	go r.Start()

	total := 13 // not a multiple of capacity
	for i := 0; i < total; i++ {
		r.InputCh() <- fmt.Sprintf("%d", i)
	}
	close(r.InputCh())
	r.Wait()

	got := r.Lines()
	if len(got) != capacity {
		t.Fatalf("expected %d lines, got %d: %v", capacity, len(got), got)
	}
	// Should be the last `capacity` lines in order.
	for i, want := range []string{"9", "10", "11", "12"} {
		if got[i] != want {
			t.Errorf("line %d: want %q, got %q", i, want, got[i])
		}
	}
}

func TestLastNConsumer_String(t *testing.T) {
	r := NewLastNConsumer("test", 10, 3)
	go r.Start()

	for _, l := range []string{"x", "y", "z"} {
		r.InputCh() <- l
	}
	close(r.InputCh())
	r.Wait()

	if s := r.String(); s != "x\ny\nz" {
		t.Errorf("String(): got %q", s)
	}
}

func TestLastNConsumer_ConcurrentRead(t *testing.T) {
	// Lines() must be safe to call while Start() is still running.
	r := NewLastNConsumer("test", 100, 10)
	go r.Start()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 50; i++ {
			r.InputCh() <- fmt.Sprintf("%d", i)
		}
		close(r.InputCh())
	}()

	// Read concurrently while the sender is still writing.
	for {
		_ = r.Lines()
		select {
		case <-done:
			r.Wait()
			return
		default:
		}
	}
}
