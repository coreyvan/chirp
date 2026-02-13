package cli

import "time"

const (
	defaultPort    = "/dev/cu.usbmodem101"
	defaultTimeout = 2 * time.Second
)

// Context contains process-wide CLI settings resolved from persistent flags.
type Context struct {
	Port    string
	Timeout time.Duration
	JSON    bool
	Verbose bool
}
