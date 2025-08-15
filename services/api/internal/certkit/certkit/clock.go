package certkit

import "time"

type sysClock struct{}

func (sysClock) Now() time.Time { return time.Now().UTC() }
