package main

import (
	"time"
)

// A timeout starting at the object creation time.
// Precise to the second.
//
type Timeout struct {
	none     bool
	deadline time.Time
}

// Create a timeout with no time limit.
// The returned timeout never expires.
//
func NewTimeoutNone() *Timeout {
	var this Timeout

	this.none = true
	this.deadline = time.Unix(0, 0)

	return &this
}

// Create a timeout with the specified number of seconds after invocation.
//
func NewTimeoutFromSec(sec int) *Timeout {
	var this Timeout

	this.none = false
	this.deadline = time.Now().Add(time.Duration(sec * 1000000000))

	return &this
}

// Create a timeout with the specified number of days/hours/minutes/seconds
// after invocation.
// The timeout duration is specified with a human readable string like the
// following:
//
//     18          # 18 seconds (if only a number, the 's' suffix is optional)
//
//     30s         # 30 seconds (with explicit suffix)
//
//     5m          # 5 minutes (with another explicit suffix)
//
//     1m10        # 1 minute 10 seconds
//
//     5h45        # 5 hours 45 minutes (implicit number is unit immed. below)
//
//     5h13s       # 5 hours 13 seconds (explicit suffixes)
//
//     1d4h12m57s  # 1 day 4 hours 12 minutes 57 seconds (full spec)
//
// Spaces can be added between numbers and unit suffixes.
//
func NewTimeoutFromSpec(spec string) *Timeout {
	var mode int = 0 // 0 = num/sp, 1 = num, 2 = unit, 3 = sp
	var defaultMult int = 1
	var acc int = 0
	var ret int = 0
	var c rune

	// mode variation along string walk
	//
	// '  12d13  h14m  17   s   '
	//  001121133211200113332000
	//
	// '  h'
	//  00X
	//
	// '13 11'
	//  113X
	//
	// '11hm'
	//  112X
	//
	// '11h  m'
	//  11200X

	for _, c = range spec {
		if (c >= '0') && (c <= '9') {
			if mode == 3 {
				return nil
			} else {
				mode = 1
			}

			acc *= 10
			acc += int(c) - '0'
		} else if c == ' ' {
			if mode == 1 {
				mode = 3
			} else if mode == 2 {
				mode = 0
			}
		} else {
			if (mode == 0) || (mode == 2) {
				return nil
			} else {
				mode = 2
			}

			switch c {
			case 'd':
				ret += acc * 86400
				defaultMult = 3600
			case 'h':
				ret += acc * 3600
				defaultMult = 60
			case 'm':
				ret += acc * 60
				defaultMult = 1
			case 's':
				ret += acc
				defaultMult = -1
			default:
				return nil
			}

			acc = 0
		}
	}

	if acc != 0 {
		if defaultMult == -1 {
			return nil
		} else {
			ret += acc * defaultMult
		}
	}

	return NewTimeoutFromSec(ret)
}

// Return true if the timeout never expires.
//
func (this *Timeout) IsNone() bool {
	return this.none
}

// Return the number of seconds before expiration, or 0 if already expired.
//
func (this *Timeout) RemainingSeconds() int {
	var now = time.Now()
	var secthis = this.deadline.Unix()
	var secnow = now.Unix()

	if secthis > secnow {
		return int(secthis - secnow)
	} else {
		return 0
	}
}

// Return the date when this timeout expires.
//
func (this *Timeout) DeadlineDate() time.Time {
	return this.deadline
}

// Return true if the timeout expired.
//
func (this *Timeout) IsOver() bool {
	if this.none {
		return false
	}

	return (this.RemainingSeconds() > 0)
}
