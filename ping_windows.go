package main

import (
	"os/exec"
	"syscall"
)

func ping(ip string) bool {
	cmd := exec.Command("ping", "-n", "1", "-w", "1000", ip)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run() == nil
}
