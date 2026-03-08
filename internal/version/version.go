package version

import "fmt"

// Set via ldflags at build time.
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

// String returns a formatted version string.
func String() string {
	return fmt.Sprintf("beans %s (%s) built %s", Version, Commit, Date)
}
