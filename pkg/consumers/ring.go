package consumers

import (
	"strings"
	"sync"
)

// LastNConsumer retains only the last [capacity] lines received. When the buffer
// is full, the oldest line is silently evicted to make room for the newest.
// Note that lines may also be dropped upstream (before reaching this consumer)
// if the input channel fills — see consumePipe.
type LastNConsumer struct {
	inputCh  chan string
	mu       sync.Mutex
	buf      []string
	head     int // index of oldest item
	count    int // number of valid items (0..capacity)
	capacity int
	name     string
	done     chan struct{}
}

func NewLastNConsumer(name string, queueSize int, capacity int) *LastNConsumer {
	return &LastNConsumer{
		inputCh:  make(chan string, queueSize),
		buf:      make([]string, capacity),
		capacity: capacity,
		name:     name,
		done:     make(chan struct{}),
	}
}

func (r *LastNConsumer) Name() string        { return r.name }
func (r *LastNConsumer) InputCh() chan string { return r.inputCh }

func (r *LastNConsumer) Start() {
	defer close(r.done)
	for line := range r.inputCh {
		r.mu.Lock()
		if r.count < r.capacity {
			r.buf[(r.head+r.count)%r.capacity] = line
			r.count++
		} else {
			// Buffer full: overwrite oldest slot and advance head.
			r.buf[r.head] = line
			r.head = (r.head + 1) % r.capacity
		}
		r.mu.Unlock()
	}
}

func (r *LastNConsumer) Wait() {
	<-r.done
}

// Lines returns a snapshot of the retained lines in arrival order (oldest first).
// Safe to call while Start is running.
func (r *LastNConsumer) Lines() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, r.count)
	for i := 0; i < r.count; i++ {
		out[i] = r.buf[(r.head+i)%r.capacity]
	}
	return out
}

func (r *LastNConsumer) String() string {
	return strings.Join(r.Lines(), "\n")
}
