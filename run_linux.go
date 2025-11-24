//go:build !windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

const pidFile = "wol.pid"

func runPlatformSpecific() {
	if *killMode {
		killDaemon()
		return
	}

	if *daemonMode {
		// Check if already running as daemon (simple check to avoid infinite loop if logic is flawed)
		// But here we just re-execute self and exit.
		// To prevent infinite loop, we can check an env var or assume the user knows what they are doing.
		// A better way is to check if PPID is 1 (init), but that's not always true.
		// Let's use a simple environment variable marker.
		if os.Getenv("WOL_DAEMON_CHILD") == "1" {
			// We are the child, write PID file and run the server
			if err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644); err != nil {
				fmt.Printf("Failed to write PID file: %v\n", err)
			}
			defer os.Remove(pidFile)

			startServer()
			return
		}

		// We are the parent, spawn child
		exe, err := os.Executable()
		if err != nil {
			fmt.Println("Failed to determine executable path:", err)
			os.Exit(1)
		}

		cmd := exec.Command(exe)
		cmd.Env = append(os.Environ(), "WOL_DAEMON_CHILD=1")
		// Detach process
		cmd.Start()
		fmt.Println("WOL Manager started in background.")
		os.Exit(0)
	}

	// Foreground mode
	startServer()
}

func killDaemon() {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		fmt.Println("Failed to read PID file (is the daemon running?):", err)
		return
	}
	pid, err := strconv.Atoi(string(data))
	if err != nil {
		fmt.Println("Invalid PID file content:", err)
		return
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		fmt.Println("Failed to find process:", err)
		return
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		fmt.Println("Failed to kill process:", err)
		return
	}
	fmt.Println("Daemon stopped.")
	os.Remove(pidFile)
}
