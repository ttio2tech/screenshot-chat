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
	Done       bool   `json:"done"`
	DoneReason string `json:"done_reason"`
}

// --- Methods (all exported = bound to frontend) ---

// TakeScreenshot captures a screenshot using the platform-native tool.
// mode: "full", "selection", "focused"
func (a *App) TakeScreenshot(mode string) ScreenshotResult {
	timestamp := time.Now().Format("2006-01-02_150405")
	filename := fmt.Sprintf("screenshot_%s_%s.png", timestamp, mode)
	savePath := filepath.Join(a.screenshotDir, filename)

	// Peek at whether this mode requires hiding the window, without actually
	// capturing yet — captureScreenshot returns hideWindow as its first value.
	hideWindow := mode == "full" || mode == "focused"

	if hideWindow {
		wailsRuntime.WindowHide(a.ctx)
		time.Sleep(500 * time.Millisecond)
	}

	_, err := captureScreenshot(mode, savePath)

	if hideWindow {
		time.Sleep(100 * time.Millisecond)
		wailsRuntime.WindowShow(a.ctx)
	}

	if err != nil {
		return ScreenshotResult{Success: false, Error: err.Error()}
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
