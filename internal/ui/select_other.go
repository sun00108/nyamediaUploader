//go:build !darwin && !linux

package ui

import "fmt"

func isTerminal(fd uintptr) bool {
	return false
}

func makeRaw(fd uintptr) (rawState, error) {
	return nil, fmt.Errorf("raw terminal mode is not supported on this platform")
}

func restore(fd uintptr, state rawState) error {
	return nil
}
