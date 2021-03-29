package ioutils

import (
	"github.com/lifei6671/fsnotify-sync/internal/log"
	"io"
	"os"
	"strings"
	"time"
)

func Close(closer io.Closer) {
	if closer != nil {
		_ = closer.Close()
	}
}
func IsFile(filename string) bool {
	if f, err := os.Stat(filename); err == nil && !f.IsDir() {
		return true
	}
	return false
}
func IsDir(filename string) bool {
	if f, err := os.Stat(filename); err == nil && f.IsDir() {
		return true
	}
	return false
}
func FileExist(filename string) bool {
	if _, err := os.Stat(filename); err == nil || os.IsExist(err) {
		return true
	}
	return false
}
func CopyFile(dstName, srcName string, perm os.FileMode) (written int64, err error) {
	src, err := os.Open(srcName)
	if err != nil {
		return
	}
	defer Close(src)
	dst, err := os.OpenFile(dstName, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, perm)
	if err != nil {
		return
	}
	defer Close(dst)
	return io.Copy(dst, src)
}
func GetFileModTime(path string) int64 {
	path = strings.Replace(path, "\\", "/", -1)
	f, err := os.Open(path)
	if err != nil {
		log.Logger.Errorf("Failed to open file on '%s': %s", path, err)
		return time.Now().Unix()
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		log.Logger.Errorf("Failed to get file stats: %s", err)
		return time.Now().Unix()
	}

	return fi.ModTime().Unix()
}
