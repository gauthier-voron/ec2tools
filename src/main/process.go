package main

import (
	"context"
	"os/exec"
	"sync"
	"time"
)

// A pipe buffer (LIFO) with unrestricted size.
// Acts pretty much like a never blocking `chan string`.
//
type Pipe struct {
	lock    sync.Mutex
	cond    *sync.Cond
	content []string
	closed  bool
}

// Create a new pipe buffer.
// The pipe buffer is initially open and empty.
//
func NewPipe() *Pipe {
	var pipe Pipe

	pipe.cond = sync.NewCond(&pipe.lock)
	pipe.content = make([]string, 0)
	pipe.closed = false

	return &pipe
}

// Push a string element at the tail of the pipe buffer.
// Always return without blocking.
// Has no effect if the pipe buffer is closed.
//
func (this *Pipe) Push(elem string) {
	this.lock.Lock()

	if !this.closed {
		this.content = append(this.content, elem)
		this.cond.Broadcast()
	}

	this.lock.Unlock()
}

// Pop the first element at the head of the pipe buffer.
// Block if the pipe buffer is open but there is no element to pop.
// If there is something to pop, return it with true.
// If the pipe buffer is closed (or get closed during the call), return an
// empty string with false.
//
func (this *Pipe) Pop() (string, bool) {
	var elem string
	var has bool

	this.lock.Lock()

	for {
		if len(this.content) > 0 {
			elem = this.content[0]
			this.content = this.content[1:]
			has = true
		} else if this.closed {
			elem = ""
			has = false
		} else {
			this.cond.Wait()
			continue
		}
		break
	}

	this.lock.Unlock()

	return elem, has
}

// Try to pop the first element at the head of the pipe without blocking.
// If there is no element to read or if the pipe has been closed, return an
// empty string with false.
// Otherwise, return the poped element with true.
//
func (this *Pipe) TryPop() (string, bool) {
	var elem string
	var has bool

	this.lock.Lock()

	if len(this.content) > 0 {
		elem = this.content[0]
		this.content = this.content[1:]
		has = true
	} else {
		elem = ""
		has = false
	}

	this.lock.Unlock()

	return elem, has
}

// Close the pipe buffer, preventing subsequent Pipe.Push().
// Element stored in the pipe buffer can still be poped.
// No method invocation can block after this method returns.
//
func (this *Pipe) Close() {
	this.lock.Lock()
	this.closed = true
	this.cond.Broadcast()
	this.lock.Unlock()
}

// An external process, monitored by the go process.
// The Go process read its stdout and stderr and write its stdin.
// The Go process also can wait for the termination of the external process and
// get its exit code.
// By contrast with standard exec.Cmd, the read/write operations may be
// non-blocking and happen before the start of the process or after its
// termination.
//
type Process struct {
	command  *exec.Cmd // internal Go representation of an external process
}

// Create a new Process structure with a nil Process.command
//
func newProcess() *Process {
	var this Process

	this.command = nil

	return &this
}

// Create a new process with the specified command line.
// The Process does not start immediately.
//
func NewProcess(cmdline []string) *Process {
	var this *Process = newProcess()

	this.command = exec.Command(cmdline[0], cmdline[1:]...)

	return this
}

// Create a new process with the specified command line, sopping after a given
// number of seconds.
// The Process does not start immediately.
//
func NewProcessTimeout(cmdline []string, timeout int) *Process {
	var this *Process = newProcess()
	var lifetime time.Duration
	var ctx context.Context

	lifetime = time.Duration(timeout) * time.Second

	ctx, _ = context.WithTimeout(context.Background(), lifetime)

	this.command = exec.CommandContext(ctx, cmdline[0], cmdline[1:]...)

	return this
}

// Start this process.
// Launch the internal Go command and start to transfer lines between streams
// and pipe buffers.
// Eventually call Process.wait() in a separate goroutine to update the state
// when the external process finishes.
// This method does not block.
//
func (this *Process) Start() {
	this.command.Start()
}

// Read the next line from this process standard output.
// Block if there is no line available yet.
// Return false in second value if the standard output is closed.
// Otherwise return the line (with the end-of-line character) and true.
//
func (this *Process) ReadStdout() (string, bool) {
	return "", false
}

// Try to read the next line from this process standard output, not blocking.
// If there is no line available yet or if the stream is closed, return false
// in second value.
// Otherwise return the line (with the end-of-line character) and true.
//
func (this *Process) TryReadStdout() (string, bool) {
	return "", false
}

// Read the next line from this process standard error.
// Block if there is no line available yet.
// Return false in second value if the standard error is closed.
// Otherwise return the line (with the end-of-line character) and true.
//
func (this *Process) ReadStderr() (string, bool) {
	return "", false
}

// Try to read the next line from this process standard error, not blocking.
// If there is no line available yet or if the stream is closed, return false
// in second value.
// Otherwise return the line (with the end-of-line character) and true.
//
func (this *Process) TryReadStderr() (string, bool) {
	return "", false
}

// Write the given string on this process input.
// No end-of-line character is added. The buffering behavior of the external
// process is unspecified.
//
func (this *Process) WriteStdin(str string) {
}

// Close the input stream of this process.
// Subsequent Process.WriteStdin() or Process.CloseStdin() invocations have no
// effect.
//
func (this *Process) CloseStdin() {
}

// Block until the external process finishes.
// After this method return, the Process.ExitCode() always return a valid exit
// code and true.
//
func (this *Process) WaitFinished() {
}

// Return the exit code of this process.
// If the process has not yet finished, return false in second value and an
// unspecified first value.
//
func (this *Process) ExitCode() (int, bool) {
	return -1, false
}
