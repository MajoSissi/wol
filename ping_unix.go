//go:build !windows

package main

import (
	"os/exec"
)

func ping(ip string) bool {
	cmd := exec.Command("ping", "-c", "1", "-W", "1", ip)
	return cmd.Run() == nil
}
