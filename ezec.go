package ezec

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

type Args interface {
	Args() []string
}

type LineConsumer interface {
	Name() string
	InputCh() chan string
}

type Cmd struct {
	cmd  *exec.Cmd
	ctx  context.Context
	wg   sync.WaitGroup

	// Fields mirroring exec.Cmd. All are wired to the underlying exec.Cmd
	// in Start(), so they can be set or modified after construction.
	Path         string
	Args         []string
	Env          []string
	Dir          string
	Stdin        io.Reader
	SysProcAttr  *syscall.SysProcAttr
	WaitDelay    time.Duration
	Cancel       func() error
	Err          error

	// Process is the underlying process. Set after a successful Start.
	Process *os.Process
	// ProcessState contains exit information. Set after Wait returns.
	ProcessState *os.ProcessState

	// ezec-specific: line consumers attached to each output stream.
	Stdout     []LineConsumer
	Stderr     []LineConsumer
	ExtraFiles [][]LineConsumer
}

func Command(name string, args Args) *Cmd {
	cmd := exec.Command(name, args.Args()...)
	return &Cmd{
		cmd:        cmd,
		ctx:        context.Background(),
		Path:       cmd.Path,
		Args:       cmd.Args,
		Err:        cmd.Err,
		Stdout:     make([]LineConsumer, 0),
		Stderr:     make([]LineConsumer, 0),
		ExtraFiles: make([][]LineConsumer, 0),
	}
}

func CommandContext(ctx context.Context, name string, args Args) *Cmd {
	cmd := exec.CommandContext(ctx, name, args.Args()...)
	return &Cmd{
		cmd:        cmd,
		ctx:        ctx,
		Path:       cmd.Path,
		Args:       cmd.Args,
		Err:        cmd.Err,
		Stdout:     make([]LineConsumer, 0),
		Stderr:     make([]LineConsumer, 0),
		ExtraFiles: make([][]LineConsumer, 0),
	}
}

func (c *Cmd) Start() error {
	// Wire all pass-through fields to the underlying exec.Cmd.
	// Zero/nil values are the exec.Cmd defaults, so unconditional assignment
	// is correct. Cancel and SysProcAttr are guarded: nil must not overwrite
	// the default Cancel that exec.CommandContext installs.
	c.cmd.Dir = c.Dir
	c.cmd.Stdin = c.Stdin
	c.cmd.Env = c.Env
	c.cmd.Args = c.Args
	if c.SysProcAttr != nil {
		c.cmd.SysProcAttr = c.SysProcAttr
	}
	if c.WaitDelay != 0 {
		c.cmd.WaitDelay = c.WaitDelay
	}
	if c.Cancel != nil {
		c.cmd.Cancel = c.Cancel
	}

	// Only attach a pipe when consumers are registered for that stream.
	// An unread pipe deadlocks the child once the kernel buffer (~64 KB) fills.
	var outPipe, errPipe io.ReadCloser
	if len(c.Stdout) > 0 {
		var err error
		outPipe, err = c.cmd.StdoutPipe()
		if err != nil {
			return fmt.Errorf("failed to get StdoutPipe: %w", err)
		}
	}
	if len(c.Stderr) > 0 {
		var err error
		errPipe, err = c.cmd.StderrPipe()
		if err != nil {
			if outPipe != nil {
				outPipe.Close()
			}
			return fmt.Errorf("failed to get StderrPipe: %w", err)
		}
	}

	type extraPipe struct {
		r    *os.File
		cons []LineConsumer
	}
	extras := make([]extraPipe, 0, len(c.ExtraFiles))
	writeEnds := make([]*os.File, 0, len(c.ExtraFiles))
	for _, cons := range c.ExtraFiles {
		r, w, err := os.Pipe()
		if err != nil {
			for _, ep := range extras {
				ep.r.Close()
			}
			for _, w2 := range writeEnds {
				w2.Close()
			}
			if outPipe != nil {
				outPipe.Close()
			}
			if errPipe != nil {
				errPipe.Close()
			}
			return fmt.Errorf("failed to create pipe for extra file: %w", err)
		}
		c.cmd.ExtraFiles = append(c.cmd.ExtraFiles, w)
		writeEnds = append(writeEnds, w)
		extras = append(extras, extraPipe{r, cons})
	}

	if err := c.cmd.Start(); err != nil {
		// exec closes stdout/stderr write ends via its internal cleanup;
		// we own the read ends and any ExtraFiles write ends.
		if outPipe != nil {
			outPipe.Close()
		}
		if errPipe != nil {
			errPipe.Close()
		}
		for _, ep := range extras {
			ep.r.Close()
		}
		for _, w := range writeEnds {
			w.Close()
		}
		return err
	}

	c.Process = c.cmd.Process

	// Close write ends in the parent — the child has inherited its own copies.
	// Without this the consumer goroutines will never see EOF.
	for _, w := range writeEnds {
		w.Close()
	}

	// Goroutines are launched only after a successful Start so they are never
	// orphaned by a Start failure.
	if outPipe != nil {
		c.wg.Add(1)
		go func() { defer c.wg.Done(); consumePipe(c.ctx, outPipe, c.Stdout) }()
	}
	if errPipe != nil {
		c.wg.Add(1)
		go func() { defer c.wg.Done(); consumePipe(c.ctx, errPipe, c.Stderr) }()
	}
	for _, ep := range extras {
		c.wg.Add(1)
		go func(pipe *os.File, cons []LineConsumer) {
			defer c.wg.Done()
			defer pipe.Close() // we own this fd; exec does not close it
			consumePipe(c.ctx, pipe, cons)
		}(ep.r, ep.cons)
	}

	return nil
}

// Wait blocks until all pipe-consumer goroutines have finished, then reaps
// the child process. Callers must use this instead of the underlying cmd.Wait
// to avoid closing pipes before consumers have drained them.
func (c *Cmd) Wait() error {
	c.wg.Wait()
	err := c.cmd.Wait()
	c.ProcessState = c.cmd.ProcessState
	return err
}

// Run starts the command and waits for it to complete.
func (c *Cmd) Run() error {
	if err := c.Start(); err != nil {
		return err
	}
	return c.Wait()
}

// StdinPipe returns a pipe connected to the command's standard input.
func (c *Cmd) StdinPipe() (io.WriteCloser, error) {
	return c.cmd.StdinPipe()
}

// Environ returns the environment that would be used to run the command,
// following exec.Cmd.Environ semantics.
func (c *Cmd) Environ() []string {
	// Sync fields that Environ reads before delegating.
	c.cmd.Env = c.Env
	c.cmd.Dir = c.Dir
	return c.cmd.Environ()
}

// String returns a human-readable description of the command.
func (c *Cmd) String() string {
	c.cmd.Args = c.Args
	return c.cmd.String()
}

func (c *Cmd) AddFd(consumers []LineConsumer) {
	c.ExtraFiles = append(c.ExtraFiles, consumers)
}

// consumePipe reads lines from read and fans them out to each consumer's input
// channel. When a consumer's channel is full the line is dropped and a warning
// is logged — there is no backpressure. Size consumer channels to handle peak
// throughput or accept that lines may be skipped under load.
// Consumer channels are closed via defer when the pipe reaches EOF or ctx is
// cancelled.
func consumePipe(ctx context.Context, read io.ReadCloser, consumers []LineConsumer) {
	scanner := bufio.NewScanner(read)
	defer func() {
		for _, c := range consumers {
			close(c.InputCh())
		}
	}()
	for scanner.Scan() {
		line := scanner.Text()
		for _, consumer := range consumers {
			select {
			case consumer.InputCh() <- line:
			case <-ctx.Done():
				return
			default:
				log.Printf("failed to write to consumer '%s', channel full", consumer.Name())
			}
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("consumer scan err: %s", err.Error())
	}
}
