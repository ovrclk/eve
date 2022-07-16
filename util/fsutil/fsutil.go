package fsutil

import "os"

// FileExists returns true if the file exists.
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}
