//go:build !windows

package main

import "os"

func writeOut(data []byte, _ bool) int {
	if _, err := os.Stdout.Write(data); err != nil {
		fail(errf("%v", err))
		return 1
	}
	return 0
}
