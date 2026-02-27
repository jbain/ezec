package ezec

import (
	"testing"

	"github.com/jbain/ezec/pkg/consumers"
)

// sliceArgs implements the Args interface for tests.
type sliceArgs []string

func (s sliceArgs) Args() []string { return s }

func TestBasicExecution(t *testing.T) {
	c := Command("true", sliceArgs{})
	if err := c.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := c.Wait(); err != nil {
		t.Fatalf("Wait: %v", err)
	}
}

func TestStdoutConsumer(t *testing.T) {
	buf := consumers.NewBufferedConsumer(10)
	go buf.Start()

	c := Command("echo", sliceArgs{"hello", "world"})
	c.Stdout = []LineConsumer{buf}

	if err := c.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := c.Wait(); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	buf.Wait()

	lines := buf.Lines()
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d: %v", len(lines), lines)
	}
	if lines[0] != "hello world" {
		t.Errorf("expected %q, got %q", "hello world", lines[0])
	}
}

func TestStderrConsumer(t *testing.T) {
	buf := consumers.NewBufferedConsumer(10)
	go buf.Start()

	c := Command("sh", sliceArgs{"-c", "echo error line >&2"})
	c.Stderr = []LineConsumer{buf}

	if err := c.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := c.Wait(); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	buf.Wait()

	lines := buf.Lines()
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d: %v", len(lines), lines)
	}
	if lines[0] != "error line" {
		t.Errorf("expected %q, got %q", "error line", lines[0])
	}
}

func TestExtraFilesConsumer(t *testing.T) {
	buf := consumers.NewBufferedConsumer(10)
	go buf.Start()

	// fd 3 is the first ExtraFiles entry (stdin=0, stdout=1, stderr=2).
	c := Command("sh", sliceArgs{"-c", "echo extra line >&3"})
	c.AddFd([]LineConsumer{buf})

	if err := c.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := c.Wait(); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	buf.Wait()

	lines := buf.Lines()
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d: %v", len(lines), lines)
	}
	if lines[0] != "extra line" {
		t.Errorf("expected %q, got %q", "extra line", lines[0])
	}
}