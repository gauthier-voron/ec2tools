package main

import (
	"testing"
)

func TestNewProcess(t *testing.T) {
	var proc *Process = NewProcess([]string{"true"})
	var has bool

	_, has = proc.TryReadStdout()
	if has {
		t.FailNow()
	}

	_, has = proc.TryReadStderr()
	if has {
		t.FailNow()
	}

	proc.WriteStdin("never read\n")

	_, has = proc.ExitCode()
	if has {
		t.FailNow()
	}
}

func TestExitCodeSuccess(t *testing.T) {
	var proc *Process = NewProcess([]string{"true"})
	var code int

	proc.Start()
	proc.WaitFinished()

	code, _ = proc.ExitCode()
	if code != 0 {
		t.FailNow()
	}
}

func TestExitCodeFailure(t *testing.T) {
	var proc *Process = NewProcess([]string{"false"})
	var code int

	proc.Start()
	proc.WaitFinished()

	code, _ = proc.ExitCode()
	if code == 0 {
		t.FailNow()
	}
}

func TestReadStdout(t *testing.T) {
	var proc *Process = NewProcess([]string{"printf", "l0\\nl1\\n"})
	var line string
	var has bool

	proc.Start()

	line, has = proc.ReadStdout()
	if !has {
		t.FailNow()
	} else if line != "l0\n" {
		t.FailNow()
	}

	proc.WaitFinished()

	line, has = proc.ReadStdout()
	if !has {
		t.FailNow()
	} else if line != "l1\n" {
		t.FailNow()
	}

	line, has = proc.ReadStdout()
	if has {
		t.FailNow()
	}
}

func TestWriteStdin(t *testing.T) {
	var proc *Process = NewProcess([]string{"cat"})
	var line string
	var has bool

	proc.WriteStdin("l0\n")
	proc.WriteStdin("l1\n")

	proc.Start()

	line, has = proc.ReadStdout()
	if !has {
		t.FailNow()
	} else if line != "l0\n" {
		t.FailNow()
	}

	proc.WriteStdin("l2\n")

	line, has = proc.ReadStdout()
	if !has {
		t.FailNow()
	} else if line != "l1\n" {
		t.FailNow()
	}

	line, has = proc.ReadStdout()
	if !has {
		t.FailNow()
	} else if line != "l2\n" {
		t.FailNow()
	}

	proc.CloseStdin()
	proc.WaitFinished()
}
