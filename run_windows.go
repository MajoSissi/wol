package main

import (
	"fmt"
	"os"

	"wol/icon"

	"github.com/getlantern/systray"
	"github.com/skratchdot/open-golang/open"
)

func runPlatformSpecific() {
	// On Windows, we use systray.
	// Note: systray.Run blocks, so we start the server in a goroutine.
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetIcon(icon.Data)
	systray.SetTitle("WOL")
	systray.SetTooltip("Wake-on-LAN")

	mOpen := systray.AddMenuItem("Open Web Page", "Open the management interface in browser")
	mAutoStart := systray.AddMenuItemCheckbox("Run at Startup", "Start automatically with Windows", isAutoStartEnabled())
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Exit", "Quit the application")

	// Start the server in a goroutine
	go startServer()

	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				port := store.GetPort()
				url := fmt.Sprintf("http://localhost:%d", port)
				open.Run(url)
			case <-mAutoStart.ClickedCh:
				if mAutoStart.Checked() {
					if err := setAutoStart(false); err == nil {
						mAutoStart.Uncheck()
					}
				} else {
					if err := setAutoStart(true); err == nil {
						mAutoStart.Check()
					}
				}
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() {
	// Cleanup if needed
	os.Exit(0)
}
