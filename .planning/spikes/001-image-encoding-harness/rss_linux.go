package main

import (
	"os"
	"strconv"
	"strings"
	"syscall"
)

// Maxrss is in kilobytes on Linux.
func peakRSSMB() float64 {
	var ru syscall.Rusage
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &ru)
	return float64(ru.Maxrss) / 1024
}

func memAvailableMB() (int, bool) {
	b, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, false
	}
	for _, line := range strings.Split(string(b), "\n") {
		if f := strings.Fields(line); len(f) >= 2 && f[0] == "MemAvailable:" {
			kb, err := strconv.Atoi(f[1])
			return kb / 1024, err == nil
		}
	}
	return 0, false
}
