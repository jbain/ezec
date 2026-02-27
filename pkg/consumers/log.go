package consumers

import "log"

// LineLogger is a LineConsumer that writes each received line to a *log.Logger.
// The prefix is prepended to every line and is also returned by Name().
type LineLogger struct {
	inputCh chan string
	Logger  *log.Logger
	prefix  string
	done    chan struct{}
}

func NewLineLogger(prefix string, queueSize int) *LineLogger {
	return &LineLogger{
		inputCh: make(chan string, queueSize),
		Logger:  log.Default(),
		prefix:  prefix,
		done:    make(chan struct{}),
	}
}

func (l *LineLogger) Name() string {
	return l.prefix
}

func (l *LineLogger) InputCh() chan string {
	return l.inputCh
}

func (l *LineLogger) Start() {
	defer close(l.done)
	for line := range l.inputCh {
		l.Logger.Printf("%s: %s", l.prefix, line)
	}
}

func (l *LineLogger) Wait() {
	<-l.done
}
