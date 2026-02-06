//go:build windows

package main

import (
	"fmt"
	"os/exec"
)

// captureScreenshot takes a screenshot on Windows using PowerShell and .NET's System.Windows.Forms.
// mode: "full", "selection", "focused"
// savePath: absolute path to save the screenshot PNG.
// Returns whether the app window should be hidden during capture, and any error.
func captureScreenshot(mode string, savePath string) (hideWindow bool, err error) {
	switch mode {
	case "full":
		hideWindow = true
	case "selection":
		// Windows full-screen capture as fallback; interactive selection is not
		// natively available without a third-party tool.
		hideWindow = true
	case "focused":
		hideWindow = true
	default:
		return false, fmt.Errorf("invalid mode %q: use full, selection, or focused", mode)
	}

	var psScript string

	switch mode {
	case "full":
		psScript = fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
$screens = [System.Windows.Forms.Screen]::AllScreens
$minX = ($screens | ForEach-Object { $_.Bounds.X } | Measure-Object -Minimum).Minimum
$minY = ($screens | ForEach-Object { $_.Bounds.Y } | Measure-Object -Minimum).Minimum
$maxX = ($screens | ForEach-Object { $_.Bounds.X + $_.Bounds.Width } | Measure-Object -Maximum).Maximum
$maxY = ($screens | ForEach-Object { $_.Bounds.Y + $_.Bounds.Height } | Measure-Object -Maximum).Maximum
$totalWidth = $maxX - $minX
$totalHeight = $maxY - $minY
$bitmap = New-Object System.Drawing.Bitmap($totalWidth, $totalHeight)
$graphics = [System.Drawing.Graphics]::FromImage($bitmap)
$graphics.CopyFromScreen($minX, $minY, 0, 0, $bitmap.Size)
$bitmap.Save('%s', [System.Drawing.Imaging.ImageFormat]::Png)
$graphics.Dispose()
$bitmap.Dispose()
`, savePath)

	case "selection":
		// Capture full screen as fallback for selection mode on Windows.
		psScript = fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
$screen = [System.Windows.Forms.Screen]::PrimaryScreen
$bitmap = New-Object System.Drawing.Bitmap($screen.Bounds.Width, $screen.Bounds.Height)
$graphics = [System.Drawing.Graphics]::FromImage($bitmap)
$graphics.CopyFromScreen($screen.Bounds.Location, [System.Drawing.Point]::Empty, $screen.Bounds.Size)
$bitmap.Save('%s', [System.Drawing.Imaging.ImageFormat]::Png)
$graphics.Dispose()
$bitmap.Dispose()
`, savePath)

	case "focused":
		psScript = fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
Add-Type @"
using System;
using System.Runtime.InteropServices;
public class Win32 {
    [DllImport("user32.dll")]
    public static extern IntPtr GetForegroundWindow();
    [DllImport("user32.dll")]
    [return: MarshalAs(UnmanagedType.Bool)]
    public static extern bool GetWindowRect(IntPtr hWnd, out RECT lpRect);
}
[StructLayout(LayoutKind.Sequential)]
public struct RECT {
    public int Left;
    public int Top;
    public int Right;
    public int Bottom;
}
"@
$hwnd = [Win32]::GetForegroundWindow()
$rect = New-Object RECT
[Win32]::GetWindowRect($hwnd, [ref]$rect) | Out-Null
$width = $rect.Right - $rect.Left
$height = $rect.Bottom - $rect.Top
$bitmap = New-Object System.Drawing.Bitmap($width, $height)
$graphics = [System.Drawing.Graphics]::FromImage($bitmap)
$graphics.CopyFromScreen($rect.Left, $rect.Top, 0, 0, $bitmap.Size)
$bitmap.Save('%s', [System.Drawing.Imaging.ImageFormat]::Png)
$graphics.Dispose()
$bitmap.Dispose()
`, savePath)
	}

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psScript)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return hideWindow, fmt.Errorf("screenshot capture failed: %v - %s", err, string(output))
	}

	return hideWindow, nil
}
