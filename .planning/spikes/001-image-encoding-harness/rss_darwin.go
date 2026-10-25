package main

import "syscall"

// Maxrss is in bytes on macOS.
func peakRSSMB() float64 {
	var ru syscall.Rusage
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &ru)
	return float64(ru.Maxrss) / (1 << 20)
}

func memAvailableMB() (int, bool) { return 0, false }
