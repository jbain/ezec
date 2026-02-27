package consumers

import "log"

type StdoutLogger struct {
	inputCh chan string
	Logger  *log.Logger
	prefix  string
}

func NewStdoutLogger(prefix string, queueSize int) *StdoutLogger {
	return &StdoutLogger{
		inputCh: make(chan string, queueSize),
		Logger:  log.Default(),
		prefix:  prefix,
	}
}

func (l *StdoutLogger) Name() string {
	return "StdoutLogger"
}

func (l *StdoutLogger) InputCh() chan string {
	return l.inputCh
}

func (l *StdoutLogger) Start() {
	name := l.Name()
	for line := range l.inputCh {
		l.Logger.Printf("%s:%s:%s", name, l.prefix, line)
	}
	l.Logger.Printf("Shutting down logger")
}
