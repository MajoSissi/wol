//go:build !windows

package main

func setAutoStart(enable bool) error {
	return nil // Not implemented for non-windows
}

func isAutoStartEnabled() bool {
	return false
}
