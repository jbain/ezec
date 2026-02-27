package consumers

import (
	"testing"
)

func TestBufferedConsumer(t *testing.T) {
	b := NewBufferedConsumer("test", 10)
	go b.Start()

	lines := []string{"foo", "bar", "baz"}
	for _, l := range lines {
		b.InputCh() <- l
	}
	close(b.InputCh())

	b.Wait()

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