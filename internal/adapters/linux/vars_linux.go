//go:build linux

package linux

import (
	"errors"
)

var errDeviceSkipped = errors.New("inputstats: linux device skipped")
