package main

import (
	"os"

	"golang.org/x/sys/windows/registry"
)

func setAutoStart(enable bool) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	appName := "WOL"
	if enable {
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		// Ensure we quote the path in case of spaces
		cmd := "\"" + exe + "\""
		return k.SetStringValue(appName, cmd)
	} else {
		return k.DeleteValue(appName)
	}
}

func isAutoStartEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()

	appName := "WOL"
	_, _, err = k.GetStringValue(appName)
	return err == nil
}
