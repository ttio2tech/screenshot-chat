# Screenshot Analyzer - Wails Desktop App Implementation Plan

## Overview
Build a desktop application on Ubuntu Linux that lets users capture screenshots and analyze them using a local Ollama `qwen3-vl:8b` vision model. Uses Wails v2 (Go backend + React/TypeScript frontend).

**Environment:** Go 1.23.9, Wails v2.9.2, Node.js v22.15.0, npm 11.5.2, Ollama with qwen3-vl:8b, `scrot` for screenshots.
**Working directory:** `/home/ti/CascadeProjects/screenshotLocal_02_go`

## Architecture

```
Go Backend (app.go)            <-->   React/TypeScript Frontend (App.tsx)
- Screenshot capture via scrot         - Capture buttons (full/selection/window)
- File management (~/Screenshots)      - Screenshot gallery list
- Ollama API calls (base64 image)      - Image preview panel
- Base64 image serving to frontend     - Prompt input + Analyze button
                                       - Markdown-rendered analysis results
```

---

## Phase 0: Project Initialization

### Step 0.1: Initialize Wails project
The working directory `/home/ti/CascadeProjects/screenshotLocal_02_go` exists but is empty. Since `wails init` requires an empty or non-existent target directory, and the directory has a `.claude` folder, we need to init into a temp name then move contents, OR use `wails init` pointing at the directory.

```bash
cd /home/ti/CascadeProjects/screenshotLocal_02_go
wails init -n screenshotLocal_02_go -t react-ts -d .
```

If that doesn't work (because dir is not empty due to `.claude`), alternative:
```bash
cd /home/ti/CascadeProjects
wails init -n _wails_temp -t react-ts
cp -r _wails_temp/* _wails_temp/.* screenshotLocal_02_go/ 2>/dev/null
rm -rf _wails_temp
```

### Step 0.2: Create screenshots directory
```bash
mkdir -p ~/Screenshots
```

---

## Phase 1: Go Backend

### File: `app.go` — REPLACE entirely

```go
package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx            context.Context
	screenshotDir  string
	ollamaEndpoint string
}

// NewApp creates a new App application struct
func NewApp() *App {
	homeDir, _ := os.UserHomeDir()
	return &App{
		screenshotDir:  filepath.Join(homeDir, "Screenshots"),
		ollamaEndpoint: "http://localhost:11434",
	}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	os.MkdirAll(a.screenshotDir, 0755)
}

// --- Result Types ---

type ScreenshotResult struct {
	FilePath string `json:"filePath"`
	FileName string `json:"fileName"`
	Success  bool   `json:"success"`
	Error    string `json:"error,omitempty"`
}

type ScreenshotInfo struct {
	FileName string `json:"fileName"`
	FilePath string `json:"filePath"`
	Size     int64  `json:"size"`
	ModTime  string `json:"modTime"`
}

type AnalysisResult struct {
	Content  string  `json:"content"`
	Thinking string  `json:"thinking,omitempty"`
	Success  bool    `json:"success"`
	Error    string  `json:"error,omitempty"`
	Duration float64 `json:"duration"`
}

// --- Ollama API Types ---

type OllamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []OllamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type OllamaMessage struct {
	Role    string   `json:"role"`
	Content string   `json:"content"`
	Images  []string `json:"images,omitempty"`
}

type OllamaChatResponse struct {
	Model   string `json:"model"`
	Message struct {
		Role     string `json:"role"`
		Content  string `json:"content"`
		Thinking string `json:"thinking,omitempty"`
	} `json:"message"`
	Done       bool  `json:"done"`
	DoneReason string `json:"done_reason"`
}

// --- Methods (all exported = bound to frontend) ---

// TakeScreenshot captures a screenshot using scrot
// mode: "full", "selection", "focused"
func (a *App) TakeScreenshot(mode string) ScreenshotResult {
	timestamp := time.Now().Format("2006-01-02_150405")
	filename := fmt.Sprintf("screenshot_%s_%s.png", timestamp, mode)
	savePath := filepath.Join(a.screenshotDir, filename)

	var args []string
	hideWindow := false

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
		return ScreenshotResult{Success: false, Error: "Invalid mode. Use: full, selection, focused"}
	}

	if hideWindow {
		wailsRuntime.WindowHide(a.ctx)
		time.Sleep(500 * time.Millisecond)
	}

	cmd := exec.Command("scrot", args...)
	output, err := cmd.CombinedOutput()

	if hideWindow {
		time.Sleep(100 * time.Millisecond)
		wailsRuntime.WindowShow(a.ctx)
	}

	if err != nil {
		return ScreenshotResult{Success: false, Error: fmt.Sprintf("scrot failed: %v - %s", err, string(output))}
	}

	return ScreenshotResult{FilePath: savePath, FileName: filename, Success: true}
}

// AnalyzeScreenshot sends a screenshot to Ollama for analysis
func (a *App) AnalyzeScreenshot(imagePath string, prompt string) AnalysisResult {
	startTime := time.Now()

	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return AnalysisResult{Success: false, Error: fmt.Sprintf("Failed to read image: %v", err)}
	}
	base64Image := base64.StdEncoding.EncodeToString(imageData)

	if prompt == "" {
		prompt = "Describe what you see in this screenshot in detail."
	}

	reqBody := OllamaChatRequest{
		Model: "qwen3-vl:8b",
		Messages: []OllamaMessage{
			{
				Role:    "user",
				Content: prompt,
				Images:  []string{base64Image},
			},
		},
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return AnalysisResult{Success: false, Error: fmt.Sprintf("Failed to marshal request: %v", err)}
	}

	client := &http.Client{Timeout: 180 * time.Second}
	resp, err := client.Post(
		a.ollamaEndpoint+"/api/chat",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return AnalysisResult{Success: false, Error: fmt.Sprintf("Ollama request failed: %v", err)}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return AnalysisResult{Success: false, Error: fmt.Sprintf("Failed to read response: %v", err)}
	}

	if resp.StatusCode != http.StatusOK {
		return AnalysisResult{Success: false, Error: fmt.Sprintf("Ollama returned status %d: %s", resp.StatusCode, string(body))}
	}

	var ollamaResp OllamaChatResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return AnalysisResult{Success: false, Error: fmt.Sprintf("Failed to parse response: %v", err)}
	}

	duration := time.Since(startTime).Seconds()
	return AnalysisResult{
		Content:  ollamaResp.Message.Content,
		Thinking: ollamaResp.Message.Thinking,
		Success:  true,
		Duration: duration,
	}
}

// GetScreenshots returns a list of screenshots sorted newest-first
func (a *App) GetScreenshots() []ScreenshotInfo {
	var screenshots []ScreenshotInfo

	entries, err := os.ReadDir(a.screenshotDir)
	if err != nil {
		return screenshots
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		screenshots = append(screenshots, ScreenshotInfo{
			FileName: name,
			FilePath: filepath.Join(a.screenshotDir, name),
			Size:     info.Size(),
			ModTime:  info.ModTime().Format(time.RFC3339),
		})
	}

	sort.Slice(screenshots, func(i, j int) bool {
		return screenshots[i].ModTime > screenshots[j].ModTime
	})

	return screenshots
}

// GetScreenshotBase64 returns a base64 data URI for displaying in the frontend
func (a *App) GetScreenshotBase64(filePath string) string {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}
	ext := strings.ToLower(filepath.Ext(filePath))
	mimeType := "image/png"
	if ext == ".jpg" || ext == ".jpeg" {
		mimeType = "image/jpeg"
	}
	return fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(data))
}

// DeleteScreenshot removes a screenshot file
func (a *App) DeleteScreenshot(filePath string) ScreenshotResult {
	absPath, err := filepath.Abs(filePath)
	if err != nil || !strings.HasPrefix(absPath, a.screenshotDir) {
		return ScreenshotResult{Success: false, Error: "Invalid file path"}
	}
	if err := os.Remove(absPath); err != nil {
		return ScreenshotResult{Success: false, Error: fmt.Sprintf("Failed to delete: %v", err)}
	}
	return ScreenshotResult{Success: true, FilePath: absPath}
}

// CheckOllamaStatus verifies Ollama connectivity and model availability
func (a *App) CheckOllamaStatus() map[string]interface{} {
	result := map[string]interface{}{
		"available":   false,
		"modelLoaded": false,
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(a.ollamaEndpoint + "/api/tags")
	if err != nil {
		result["error"] = "Ollama is not running"
		return result
	}
	defer resp.Body.Close()

	result["available"] = true

	body, _ := io.ReadAll(resp.Body)
	var tagsResp struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if json.Unmarshal(body, &tagsResp) == nil {
		for _, m := range tagsResp.Models {
			if strings.Contains(m.Name, "qwen3-vl") {
				result["modelLoaded"] = true
				break
			}
		}
	}

	return result
}
```

### File: `main.go` — REPLACE entirely

```go
package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:    "Screenshot Analyzer",
		Width:    1280,
		Height:   800,
		MinWidth: 900,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 255},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
```

---

## Phase 2: Frontend

### Step 2.1: Install dependency
```bash
cd /home/ti/CascadeProjects/screenshotLocal_02_go/frontend
npm install react-markdown
```

### File: `frontend/src/types.ts` — CREATE

```typescript
export interface ScreenshotResult {
  filePath: string;
  fileName: string;
  success: boolean;
  error?: string;
}

export interface ScreenshotInfo {
  fileName: string;
  filePath: string;
  size: number;
  modTime: string;
}

export interface AnalysisResult {
  content: string;
  thinking?: string;
  success: boolean;
  error?: string;
  duration: number;
}

export interface OllamaStatus {
  available: boolean;
  modelLoaded: boolean;
  error?: string;
}
```

### File: `frontend/src/App.tsx` — REPLACE entirely

Two-panel layout:
- **Header**: App title + Ollama status indicator (green/red dot)
- **Capture bar**: 3 buttons — Full Screen, Selection, Active Window
- **Left panel** (300px): Scrollable screenshot gallery list, each item shows filename, size, date, delete button
- **Right panel**:
  - Top: Image preview (large, object-fit contain)
  - Middle: Prompt textarea + Analyze button
  - Bottom: Analysis results with markdown rendering, "Show Thinking" toggle, duration display

State management:
- `screenshots: ScreenshotInfo[]` — loaded on mount and after capture/delete
- `selectedScreenshot: ScreenshotInfo | null` — currently selected
- `previewDataUri: string` — base64 data URI loaded when selection changes
- `analysisResult: AnalysisResult | null` — result from Ollama
- `prompt: string` — default "Describe what you see in this screenshot in detail."
- `isCapturing: boolean` — loading state for capture
- `isAnalyzing: boolean` — loading state for analysis
- `ollamaStatus: OllamaStatus | null` — checked on mount
- `showThinking: boolean` — toggle for chain-of-thought display
- `elapsedTime: number` — seconds counter during analysis (setInterval)

Import Go bindings from `../wailsjs/go/main/App` (auto-generated by Wails).

### File: `frontend/src/App.css` — REPLACE entirely

Dark theme design:
- Background: `#1b2636` (matches Go BackgroundColour)
- Text: `#e0e0e0`
- Cards/panels: `#243447` with subtle borders
- Accent color: `#4a9eff` for buttons and active states
- CSS Grid layout: `grid-template-columns: 300px 1fr`
- Screenshot list items: hover highlight, selected state with left border accent
- Image preview: `max-height: 400px`, `object-fit: contain`, centered
- Loading spinner: CSS animation with `@keyframes spin`
- Markdown output: styled `pre`, `code`, `h1-h3`, `ul/ol` within results area
- Thinking block: collapsible, dimmer text color, monospace font
- Responsive: scrollable panels with `overflow-y: auto`

---

## Phase 3: Build & Verify

```bash
cd /home/ti/CascadeProjects/screenshotLocal_02_go
wails dev
```

### Test checklist:
1. App window opens at 1280x800 with dark theme
2. Ollama status shows green (connected + model available)
3. "Full Screen" button: window hides, screenshot captured, appears in gallery
4. "Selection" button: screen freezes, draw rectangle, screenshot captured
5. "Active Window" button: captures focused window
6. Click screenshot in gallery: preview loads in right panel
7. Enter prompt, click "Analyze": loading spinner with timer, then markdown result
8. "Show Thinking" toggle reveals/hides chain-of-thought
9. Delete button removes screenshot from list and disk

### Production build:
```bash
wails build
# Binary at: build/bin/screenshotLocal_02_go
```
