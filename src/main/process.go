package main

// An external process, monitored by the go process.
// The Go process read its stdout and stderr and write its stdin.
// The Go process also can wait for the termination of the external process and
// get its exit code.
// By contrast with standard exec.Cmd, the read/write operations may be
// non-blocking and happen before the start of the process or after its
// termination.
//
type Process struct {
}

// Create a new process with the specified command line.
// The Process does not start immediately.
//
func NewProcess(cmdline []string) *Process {
	var this Process

	return &this
}

// Create a new process with the specified command line, sopping after a given
// number of seconds.
// The Process does not start immediately.
//
func NewProcessTimeout(cmdline []string, timeout int) *Process {
	var this Process

	return &this
}

// Start this process.
// Launch the internal Go command and start to transfer lines between streams
// and pipe buffers.
// Eventually call Process.wait() in a separate goroutine to update the state
// when the external process finishes.
// This method does not block.
//
func (this *Process) Start() {
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
