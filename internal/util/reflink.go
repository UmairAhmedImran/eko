package util

import (
	"io"
	"os"
)

// CopyOrCloneFile attempts to create a copy-on-write clone of src at dst.
// If cloning is unsupported by the platform, filesystem, or device boundaries,
// it falls back to a standard block-by-block copy (with minimal memory allocations).
func CopyOrCloneFile(src, dst string) error {
	// 1. Try platform-specific reflink cloning
	err := CloneFile(src, dst)
	if err == nil {
		return nil
	}

	// 2. Fallback to standard copy
	return CopyFile(src, dst)
}

// CopyFile performs a standard buffered file copy.
func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	info, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
