package platform

import "time"

// SystemClock is the production Clock, backed by the OS wall clock in UTC.
type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now().UTC() }
