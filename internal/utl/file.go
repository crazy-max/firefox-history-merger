package utl

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

// CopyFile copies src file to dst file
func CopyFile(src, dst string) (err error) {
	srcStats, err := os.Stat(src)
	if err != nil {
		return
	}
	if !srcStats.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}
	dstStats, err := os.Stat(dst)
	if err != nil {
		if !os.IsNotExist(err) {
			return
		}
	} else {
		// The file already exists
		if !dstStats.Mode().IsRegular() {
			return fmt.Errorf("%s already exists", dst)
		}
	}
	return copyFileContents(src, dst)
}

func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}

// FileHash returns hash of a file
func FileHash(filename string) (hash string, err error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(data)), nil
}

// FileExists checks if given file exists
func FileExists(file string) bool {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return false
	}
	return true
}
