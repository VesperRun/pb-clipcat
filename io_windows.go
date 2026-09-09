//go:build windows

package main

import (
	"os"
	"syscall"

	"golang.org/x/sys/windows"
	"golang.org/x/term"
)

func writeOut(data []byte, textToConsole bool) int {
	if textToConsole && term.IsTerminal(int(os.Stdout.Fd())) {
		if err := writeConsole(string(data)); err == nil {
			return 0
		}
	}
	if _, err := os.Stdout.Write(data); err != nil {
		fail(errf("%v", err))
		return 1
	}
	return 0
}

func writeConsole(s string) error {
	if s == "" {
		return nil
	}
	h := windows.Handle(os.Stdout.Fd())
	u, err := syscall.UTF16FromString(s)
	if err != nil {
		return err
	}
	u = u[:len(u)-1]
	var written uint32
	return windows.WriteConsole(h, &u[0], uint32(len(u)), &written, nil)
}
