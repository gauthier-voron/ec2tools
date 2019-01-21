package main

import (
	"testing"
	"time"
)

func TestNewTimeoutNone(t *testing.T) {
	var to *Timeout = NewTimeoutNone()

	if to == nil {
		t.FailNow()
	} else if !to.IsNone() {
		t.FailNow()
	} else if to.IsOver() {
		t.FailNow()
	}
}

func TestNewTimeoutFromSec(t *testing.T) {
	var to *Timeout = NewTimeoutFromSec(2)

	if to == nil {
		t.FailNow()
	} else if to.IsNone() {
		t.FailNow()
	} else if to.IsOver() {
		t.FailNow()
	} else if to.RemainingSeconds() < 1 {
		t.FailNow()
	}

	time.Sleep(2 * time.Second)

	if to.IsNone() {
		t.FailNow()
	} else if !to.IsOver() {
		t.FailNow()
	} else if to.RemainingSeconds() != 0 {
		t.FailNow()
	}
}

func TestNewTimeoutFromEmptySpec(t *testing.T) {
	var to *Timeout = NewTimeoutFromSpec("")

	if to != nil {
		t.FailNow()
	}
}

func TestNewTimeoutFromIntSpec(t *testing.T) {
	var to *Timeout = NewTimeoutFromSpec("30")

	if to == nil {
		t.FailNow()
	} else if to.IsNone() {
		t.FailNow()
	} else if to.IsOver() {
		t.FailNow()
	} else if to.RemainingSeconds() < 29 {
		t.FailNow()
	}
}

func TestNewTimeoutFromPartialSpec(t *testing.T) {
	var to *Timeout = NewTimeoutFromSpec("1h30")

	if to == nil {
		t.FailNow()
	} else if to.IsNone() {
		t.FailNow()
	} else if to.IsOver() {
		t.FailNow()
	} else if to.RemainingSeconds() < 5399 {
		t.FailNow()
	}
}

func TestNewTimeoutFromFullSpec(t *testing.T) {
	var to *Timeout = NewTimeoutFromSpec("4d18h14m57s")

	if to == nil {
		t.FailNow()
	} else if to.IsNone() {
		t.FailNow()
	} else if to.IsOver() {
		t.FailNow()
	} else if to.RemainingSeconds() < 411296 {
		t.FailNow()
	}
}

func TestNewTimeoutFromSpacedSpec(t *testing.T) {
	var to *Timeout = NewTimeoutFromSpec("  4d 18 h 14m57s   ")

	if to == nil {
		t.FailNow()
	} else if to.IsNone() {
		t.FailNow()
	} else if to.IsOver() {
		t.FailNow()
	} else if to.RemainingSeconds() < 411296 {
		t.FailNow()
	}
}
