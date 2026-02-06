//go:build linux

package main

import (
	"fmt"
	"os/exec"
)

// captureScreenshot takes a screenshot on Linux using scrot.
// mode: "full", "selection", "focused"
// savePath: absolute path to save the screenshot PNG.
// Returns whether the app window should be hidden during capture, and any error.
func captureScreenshot(mode string, savePath string) (hideWindow bool, err error) {
	var args []string

	switch mode {
	case "full":
		args = []string{"--silent", "--overwrite", savePath}
		hideWindow = true
	case "selection":
		args = []string{"--select", "--freeze", "--silent", "--overwrite", savePath}
	case "focused":
		args = []string{"--focused", "--silent", "--overwrite", savePath}
		hideWindow = true
	default:
		return false, fmt.Errorf("invalid mode %q: use full, selection, or focused", mode)
	}

	cmd := exec.Command("scrot", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return hideWindow, fmt.Errorf("scrot failed: %v - %s", err, string(output))
	}

	return hideWindow, nil
}
