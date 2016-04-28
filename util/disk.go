package util

import (
	"log"
	"os"
	"syscall"
)

var (
	FreeDiskThreshold = 1024 * 1024 * 1024
)

func GetFreeDisk() int {
	var stat syscall.Statfs_t
	wd, err := os.Getwd()
	if err != nil {
		log.Println(err)
		return 0
	}
	syscall.Statfs(wd, &stat)
	// Available blocks * size per block = available space in bytes
	return int(stat.Bavail * uint64(stat.Bsize))
}

func IsLowDiskSpace() bool {
	return GetFreeDisk() < FreeDiskThreshold
}
