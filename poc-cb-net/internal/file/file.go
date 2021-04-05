package file

import "os"

// Exists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func Exists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
