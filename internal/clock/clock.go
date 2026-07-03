// Package clock defines the application's wall-clock boundary.
package clock

import "time"

// Clock supplies the current time to time-dependent application behavior.
type Clock interface {
	Now() time.Time
}

// Real uses the system wall clock.
type Real struct{}

func (Real) Now() time.Time { return time.Now() }

// Func adapts a function into a Clock.
type Func func() time.Time

func (f Func) Now() time.Time { return f() }
