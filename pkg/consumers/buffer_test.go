package consumers

import (
	"testing"
)

func TestBufferedConsumer(t *testing.T) {
	b := NewBufferedConsumer(10)
	go b.Start()

	lines := []string{"foo", "bar", "baz"}
	for _, l := range lines {
		b.InputCh() <- l
	}
	close(b.InputCh())

	// Wait for Start to drain the channel by checking Lines until settled.
	// Since the channel is closed, the goroutine will exit shortly after.
	// Use a simple spin to avoid importing sync/time primitives.
	for {
		if got := b.Lines(); len(got) == len(lines) {
			break
		}
	}

	got := b.Lines()
	if len(got) != len(lines) {
		t.Fatalf("expected %d lines, got %d", len(lines), len(got))
	}
	for i, want := range lines {
		if got[i] != want {
			t.Errorf("line %d: want %q, got %q", i, want, got[i])
		}
	}

	wantStr := "foo\nbar\nbaz"
	if s := b.String(); s != wantStr {
		t.Errorf("String(): want %q, got %q", wantStr, s)
	}
}