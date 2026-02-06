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
	switch mode {
	case "full":
		// Capture the entire screen non-interactively.
		hideWindow = true
		cmd := exec.Command("screencapture", "-x", savePath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return hideWindow, fmt.Errorf("screencapture failed: %v - %s", err, string(output))
		}
		return hideWindow, nil

	case "selection":
		// Interactive selection mode — user draws a rectangle.
		cmd := exec.Command("screencapture", "-i", "-x", savePath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return false, fmt.Errorf("screencapture failed: %v - %s", err, string(output))
		}
		return false, nil

	case "focused":
		// Interactive window selection — user clicks on a window.
		// Don't hide the app window so user can click on another window.
		cmd := exec.Command("screencapture", "-w", "-x", savePath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return false, fmt.Errorf("screencapture failed: %v - %s", err, string(output))
		}
		return false, nil

	default:
		return false, fmt.Errorf("invalid mode %q: use full, selection, or focused", mode)
	}
}
