package ezec

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
)

type Args interface {
	Args() []string
}

type Env interface {
	Env() []string
}

type LineConsumer interface {
	Name() string
	InputCh() chan string
}

type Cmd struct {
	cmd        *exec.Cmd
	Path       string
	Args       Args
	Env        Env
	Dir        string
	Stdin      io.Reader
	Stdout     []LineConsumer
	stdoutCh   chan string
	Stderr     []LineConsumer
	stderrCh   chan string
	ExtraFiles [][]LineConsumer
	wg         sync.WaitGroup
}

func Command(name string, args Args) *Cmd {
	cmd := exec.Command(name, args.Args()...)

	return &Cmd{
		cmd:        cmd,
		Path:       cmd.Path,
		stdoutCh:   make(chan string, 1000), //TODO make configurable
		stderrCh:   make(chan string, 1000),
		Stdout:     make([]LineConsumer, 0),
		Stderr:     make([]LineConsumer, 0),
		ExtraFiles: make([][]LineConsumer, 0),
	}
}

func (c *Cmd) Start() error {
	outPipe, err := c.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get StdoutPipe: %w", err)
	}
	errPipe, err := c.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get StderrPipe: %w", err)
	}

	if len(c.Stdout) > 0 {
		c.wg.Add(1)
		go func() { defer c.wg.Done(); consumePipe(outPipe, c.Stdout) }()
	}
	if len(c.Stderr) > 0 {
		c.wg.Add(1)
		go func() { defer c.wg.Done(); consumePipe(errPipe, c.Stderr) }()
	}

	writeEnds := make([]*os.File, 0, len(c.ExtraFiles))
	for _, consumers := range c.ExtraFiles {
		r, w, err := os.Pipe()
		if err != nil {
			return fmt.Errorf("failed to create pipe for extra file: %w", err)
		}
		c.cmd.ExtraFiles = append(c.cmd.ExtraFiles, w)
		writeEnds = append(writeEnds, w)
		c.wg.Add(1)
		go func(pipe *os.File, cons []LineConsumer) {
			defer c.wg.Done()
			consumePipe(pipe, cons)
		}(r, consumers)
	}

	if err := c.cmd.Start(); err != nil {
		return err
	}

	// Close the write ends in the parent — the child has inherited its own
	// copies. Without this the read goroutines will never see EOF.
	for _, w := range writeEnds {
		w.Close()
	}
	return nil
}

// Wait blocks until all pipe-consumer goroutines have finished, then reaps
// the child process. Callers should use this instead of accessing cmd.Wait
// directly to avoid closing pipes before consumers are done.
func (c *Cmd) Wait() error {
	c.wg.Wait()
	return c.cmd.Wait()
}

func (c *Cmd) AddFd(consumers []LineConsumer) {
	c.ExtraFiles = append(c.ExtraFiles, consumers)
}

func consumePipe(read io.ReadCloser, consumers []LineConsumer) {
	scanner := bufio.NewScanner(read)

	for {
		if scanner.Scan() {
			line := scanner.Text()
			for _, parser := range consumers {
				select {
				case parser.InputCh() <- line:
				default:
					log.Printf("failed to write to parser '%s', channel full", parser.Name())
				}
			}
		} else {
			if scanner.Err() != nil {
				log.Printf("consumer scan err: %s", scanner.Err().Error())
			}
			log.Printf("closing consumers")
			for _, parser := range consumers {
				log.Printf("closing parser: %s", parser.Name())
				close(parser.InputCh())
			}
			return
		}
	}
}