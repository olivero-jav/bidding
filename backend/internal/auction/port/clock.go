package port

import "time"

// Clock is the authoritative time source. Auction closing and bid validation
// depend on the server clock, never the client's; this port is that seam and
// keeps the application layer testable with a fixed time.
type Clock interface {
	Now() time.Time
}
