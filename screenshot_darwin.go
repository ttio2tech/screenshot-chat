//go:build darwin

package main

import (
	"fmt"
	"os/exec"
	"strings"
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
		// Capture the frontmost window non-interactively using its CGWindowID.
		hideWindow = true
		windowID, err := getFrontWindowID()
		if err != nil {
			return hideWindow, fmt.Errorf("could not get focused window: %v", err)
		}
		cmd := exec.Command("screencapture", "-l"+windowID, "-x", savePath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return hideWindow, fmt.Errorf("screencapture failed: %v - %s", err, string(output))
		}
		return hideWindow, nil

	default:
		return false, fmt.Errorf("invalid mode %q: use full, selection, or focused", mode)
	}
}

// getFrontWindowID returns the CGWindowID of the frontmost window (excluding
// the current app's window) by querying the window list via AppleScript and
// the CGWindowList API through Python.
func getFrontWindowID() (string, error) {
	script := `
import json, subprocess, sys
from Quartz import CGWindowListCopyWindowInfo, kCGWindowListOptionOnScreenOnly, kCGWindowListExcludeDesktopElements, kCGNullWindowID
windows = CGWindowListCopyWindowInfo(kCGWindowListOptionOnScreenOnly | kCGWindowListExcludeDesktopElements, kCGNullWindowID)
for w in windows:
    layer = w.get("kCGWindowLayer", 999)
    name = w.get("kCGWindowOwnerName", "")
    if layer == 0 and name not in ("", "Window Server"):
        print(w["kCGWindowNumber"])
        sys.exit(0)
print("")
`
	cmd := exec.Command("python3", "-c", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("python3 CGWindowList query failed: %v - %s", err, string(output))
	}
	wid := strings.TrimSpace(string(output))
	if wid == "" {
		return "", fmt.Errorf("no focusable window found")
	}
	return wid, nil
}
