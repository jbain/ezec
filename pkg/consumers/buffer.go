package consumers

import (
	"strings"
	"sync"
)

type BufferedConsumer struct {
	inputCh chan string
	mu      sync.Mutex
	lines   []string
	done    chan struct{}
}

func NewBufferedConsumer(queueSize int) *BufferedConsumer {
	return &BufferedConsumer{
		inputCh: make(chan string, queueSize),
		lines:   make([]string, 0),
		done:    make(chan struct{}),
	}
}

func (b *BufferedConsumer) Name() string {
	return "BufferedConsumer"
}

func (b *BufferedConsumer) InputCh() chan string {
	return b.inputCh
}

func (b *BufferedConsumer) Start() {
	defer close(b.done)
	for line := range b.inputCh {
		b.mu.Lock()
		b.lines = append(b.lines, line)
		b.mu.Unlock()
	}
}

// Wait blocks until Start has finished processing all lines.
func (b *BufferedConsumer) Wait() {
	<-b.done
}

func (b *BufferedConsumer) Lines() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]string, len(b.lines))
	copy(out, b.lines)
	return out
}

func (b *BufferedConsumer) String() string {
	return strings.Join(b.Lines(), "\n")
}