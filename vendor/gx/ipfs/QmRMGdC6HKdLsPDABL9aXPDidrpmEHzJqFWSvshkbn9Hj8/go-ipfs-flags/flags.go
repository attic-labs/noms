package flags

import (
	"os"
)

// LowMemMode specifies if we should try to reduce the memory footprint
var LowMemMode bool

func init() {
	if os.Getenv("IPFS_LOW_MEM") != "" {
		LowMemMode = true
	}
}
