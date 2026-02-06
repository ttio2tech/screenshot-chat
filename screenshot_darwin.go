//go:build darwin

package main

import (
	"fmt"
	"os/exec"
)

// captureScreenshot takes a screenshot on macOS using the built-in screencapture command.
// mode: "full", "selection", "focused"
// savePath: absolute path to save the screenshot PNG.
// Returns whether the app window should be hidden during capture, and any error.
func captureScreenshot(mode string, savePath string) (hideWindow bool, err error) {
	var args []string

	switch mode {
	case "full":
		args = []string{"-x", savePath}
		hideWindow = true
	case "selection":
		args = []string{"-i", "-x", savePath}
	case "focused":
		args = []string{"-w", "-x", savePath}
		hideWindow = true
	default:
		return false, fmt.Errorf("invalid mode %q: use full, selection, or focused", mode)
	}

	cmd := exec.Command("screencapture", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return hideWindow, fmt.Errorf("screencapture failed: %v - %s", err, string(output))
	}

	return hideWindow, nil
}
