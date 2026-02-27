package consumers

type CallbackConsumer struct {
	inputCh  chan string
	callback func(string)
	name     string
	done     chan struct{}
}

func NewCallbackConsumer(name string, queueSize int, fn func(string)) *CallbackConsumer {
	return &CallbackConsumer{
		inputCh:  make(chan string, queueSize),
		callback: fn,
		name:     name,
		done:     make(chan struct{}),
	}
}

func (c *CallbackConsumer) Name() string       { return c.name }
func (c *CallbackConsumer) InputCh() chan string { return c.inputCh }

func (c *CallbackConsumer) Start() {
	defer close(c.done)
	for line := range c.inputCh {
		c.callback(line)
	}
}

func (c *CallbackConsumer) Wait() {
	<-c.done
}
