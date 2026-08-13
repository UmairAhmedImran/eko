//go:build darwin

package util

import (
	"golang.org/x/sys/unix"
)

// CloneFile attempts to create a copy-on-write clone of src at dst using clonefile(2).
func CloneFile(src, dst string) error {
	return unix.Clonefile(src, dst, 0)
}
