package clock

import "time"

type Clock interface {
	Now() time.Time
}

type Real struct{}

func (Real) Now() time.Time { return time.Now() }

type Func func() time.Time

func (f Func) Now() time.Time { return f() }
