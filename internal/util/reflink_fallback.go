//go:build !darwin && !linux

package util

import (
	"errors"
)

// CloneFile is not supported on other platforms. Returning an error triggers the fallback.
func CloneFile(src, dst string) error {
	return errors.New("reflink: not supported on this platform")
}
